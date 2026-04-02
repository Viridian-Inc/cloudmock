import { useState, useEffect, useMemo } from 'preact/hooks';
import type { DeployEvent, ServiceMetrics } from '../../lib/health';
import { api } from '../../lib/api';

/* ====== Types ====== */

interface TraceSummaryForDeploy {
  TraceID: string;
  RootService: string;
  DurationMs: number;
  StatusCode: number;
  HasError: boolean;
  StartTime: string;
}

interface ContainerInfo {
  name: string;
  status: 'Running' | 'Pending' | 'Failed' | 'Terminated';
  restartCount: number;
  cpuPercent: number;
  memoryPercent: number;
  age: string;
  ready: boolean;
}

/** Whether container data came from a live API */
type ContainerDataSource = 'k8s' | 'ecs' | 'none';

interface MetricsComparison {
  beforeErrorRate: number;
  afterErrorRate: number;
  beforeP99: number;
  afterP99: number;
  beforeVolume: number;
  afterVolume: number;
}

interface DeployDetailProps {
  deploy: DeployEvent;
  onClose: () => void;
}

/* ====== Helpers ====== */

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '--';
  return d.toLocaleString([], {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const secs = Math.floor(diff / 1000);
  if (secs < 60) return `${secs}s ago`;
  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

function formatDuration(ms: number): string {
  if (ms < 1) return `${ms.toFixed(2)}ms`;
  if (ms < 1000) return `${Math.round(ms)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) return 0;
  const idx = Math.ceil((p / 100) * sorted.length) - 1;
  return sorted[Math.max(0, idx)];
}

/**
 * Try fetching real pod data from cloudmock's K8s plugin.
 * Returns containers + source, or null if unavailable.
 */
async function fetchK8sPods(serviceName: string): Promise<{ containers: ContainerInfo[]; source: ContainerDataSource } | null> {
  try {
    const res = await fetch('http://localhost:4566/api/v1/namespaces/default/pods');
    if (!res.ok) return null;
    const data = await res.json();
    const items: any[] = data?.items ?? [];
    if (items.length === 0) return null;

    const svcLower = serviceName.replace(/^svc:/, '').toLowerCase();
    // Filter pods related to this service (by label or name prefix)
    const related = items.filter((pod: any) => {
      const podName: string = pod?.metadata?.name ?? '';
      const appLabel: string = pod?.metadata?.labels?.app ?? '';
      return podName.toLowerCase().includes(svcLower) || appLabel.toLowerCase().includes(svcLower);
    });

    const pods = related.length > 0 ? related : items;
    const containers: ContainerInfo[] = pods.map((pod: any) => {
      const phase: string = pod?.status?.phase ?? 'Unknown';
      const containerStatuses: any[] = pod?.status?.containerStatuses ?? [];
      const restarts = containerStatuses.reduce((sum: number, cs: any) => sum + (cs?.restartCount ?? 0), 0);
      const allReady = containerStatuses.length > 0 && containerStatuses.every((cs: any) => cs?.ready === true);

      const createdAt = pod?.metadata?.creationTimestamp;
      let age = '--';
      if (createdAt) {
        const ageSecs = Math.floor((Date.now() - new Date(createdAt).getTime()) / 1000);
        if (ageSecs < 60) age = `${ageSecs}s`;
        else if (ageSecs < 3600) age = `${Math.floor(ageSecs / 60)}m`;
        else if (ageSecs < 86400) age = `${Math.floor(ageSecs / 3600)}h`;
        else age = `${Math.floor(ageSecs / 86400)}d`;
      }

      let status: ContainerInfo['status'] = 'Running';
      if (phase === 'Pending') status = 'Pending';
      else if (phase === 'Failed') status = 'Failed';
      else if (phase === 'Succeeded') status = 'Terminated';

      return {
        name: pod?.metadata?.name ?? 'unknown-pod',
        status,
        restartCount: restarts,
        cpuPercent: 0, // K8s pods API doesn't include metrics; would need metrics-server
        memoryPercent: 0,
        age,
        ready: allReady,
      };
    });

    return containers.length > 0 ? { containers, source: 'k8s' } : null;
  } catch {
    return null;
  }
}

/**
 * Try fetching real ECS task data from cloudmock admin API.
 * Returns containers + source, or null if unavailable.
 */
async function fetchEcsTasks(serviceName: string): Promise<{ containers: ContainerInfo[]; source: ContainerDataSource } | null> {
  try {
    const data = await api<any>('/api/resources/ecs');
    const tasks: any[] = Array.isArray(data) ? data : (data?.tasks ?? data?.taskArns ?? []);
    if (tasks.length === 0) return null;

    const containers: ContainerInfo[] = tasks.map((task: any, i: number) => {
      const taskId: string = task?.taskArn?.split('/').pop() ?? task?.taskId ?? `task-${i}`;
      const lastStatus: string = task?.lastStatus ?? 'RUNNING';
      const desiredStatus: string = task?.desiredStatus ?? 'RUNNING';

      let status: ContainerInfo['status'] = 'Running';
      if (lastStatus === 'PENDING' || lastStatus === 'PROVISIONING') status = 'Pending';
      else if (lastStatus === 'STOPPED') status = lastStatus === desiredStatus ? 'Terminated' : 'Failed';

      const createdAt = task?.createdAt ?? task?.startedAt;
      let age = '--';
      if (createdAt) {
        const ageSecs = Math.floor((Date.now() - new Date(createdAt).getTime()) / 1000);
        if (ageSecs < 60) age = `${ageSecs}s`;
        else if (ageSecs < 3600) age = `${Math.floor(ageSecs / 60)}m`;
        else if (ageSecs < 86400) age = `${Math.floor(ageSecs / 3600)}h`;
        else age = `${Math.floor(ageSecs / 86400)}d`;
      }

      return {
        name: taskId.slice(0, 20),
        status,
        restartCount: 0,
        cpuPercent: 0,
        memoryPercent: 0,
        age,
        ready: status === 'Running',
      };
    });

    return containers.length > 0 ? { containers, source: 'ecs' } : null;
  } catch {
    return null;
  }
}

/**
 * Compute before/after metrics comparison from trace data around the deploy timestamp.
 */
function computeMetricsComparison(
  traces: TraceSummaryForDeploy[],
  deployTimestamp: string,
  serviceName: string,
): MetricsComparison {
  const deployTs = new Date(deployTimestamp).getTime();
  const windowMs = 5 * 60 * 1000; // 5 minute window

  const svc = serviceName.replace(/^svc:/, '');

  const beforeTraces = traces.filter((t) => {
    const ts = new Date(t.StartTime).getTime();
    return ts >= deployTs - windowMs && ts < deployTs && t.RootService === svc;
  });

  const afterTraces = traces.filter((t) => {
    const ts = new Date(t.StartTime).getTime();
    return ts >= deployTs && ts <= deployTs + windowMs && t.RootService === svc;
  });

  function calcMetrics(ts: TraceSummaryForDeploy[]): { errorRate: number; p99: number; volume: number } {
    if (ts.length === 0) return { errorRate: 0, p99: 0, volume: 0 };
    const errors = ts.filter((t) => t.HasError || t.StatusCode >= 500).length;
    const durations = ts.map((t) => t.DurationMs).sort((a, b) => a - b);
    return {
      errorRate: errors / ts.length,
      p99: percentile(durations, 99),
      volume: ts.length,
    };
  }

  const before = calcMetrics(beforeTraces);
  const after = calcMetrics(afterTraces);

  return {
    beforeErrorRate: before.errorRate,
    afterErrorRate: after.errorRate,
    beforeP99: before.p99,
    afterP99: after.p99,
    beforeVolume: before.volume,
    afterVolume: after.volume,
  };
}

/* ====== Sub-components ====== */

function StatusBadge({ deploy }: { deploy: DeployEvent }) {
  // Infer status from deploy age
  const ageSecs = (Date.now() - new Date(deploy.timestamp).getTime()) / 1000;
  let status: 'success' | 'rolling' | 'failed';
  let label: string;

  if (ageSecs < 120) {
    status = 'rolling';
    label = 'Rolling';
  } else {
    status = 'success';
    label = 'Success';
  }

  return (
    <span class={`dd-status-badge dd-status-${status}`}>{label}</span>
  );
}

function ContainerRow({ container }: { container: ContainerInfo }) {
  const statusColor = container.status === 'Running'
    ? '#22c55e'
    : container.status === 'Pending'
      ? '#fbbf24'
      : '#ef4444';

  return (
    <div class="dd-container-row">
      <div class="dd-container-info">
        <span class="dd-container-status-dot" style={{ background: statusColor }} />
        <span class="dd-container-name">{container.name}</span>
        <span class="dd-container-status" style={{ color: statusColor }}>
          {container.status}
        </span>
        {container.ready ? (
          <span class="dd-container-ready dd-ready-yes">Ready</span>
        ) : (
          <span class="dd-container-ready dd-ready-no">Not Ready</span>
        )}
      </div>
      <div class="dd-container-metrics">
        <div class="dd-container-bar-group">
          <span class="dd-container-bar-label">CPU</span>
          <div class="dd-container-bar-track">
            <div
              class="dd-container-bar-fill dd-bar-cpu"
              style={{ width: `${container.cpuPercent}%` }}
            />
          </div>
          <span class="dd-container-bar-value">{container.cpuPercent}%</span>
        </div>
        <div class="dd-container-bar-group">
          <span class="dd-container-bar-label">Mem</span>
          <div class="dd-container-bar-track">
            <div
              class="dd-container-bar-fill dd-bar-mem"
              style={{ width: `${container.memoryPercent}%` }}
            />
          </div>
          <span class="dd-container-bar-value">{container.memoryPercent}%</span>
        </div>
        <span class="dd-container-age">{container.age}</span>
        {container.restartCount > 0 && (
          <span class="dd-container-restarts">{container.restartCount} restart{container.restartCount !== 1 ? 's' : ''}</span>
        )}
      </div>
    </div>
  );
}

function MetricChangeIndicator({
  label,
  before,
  after,
  formatFn,
  invertDesired,
}: {
  label: string;
  before: number;
  after: number;
  formatFn: (v: number) => string;
  invertDesired?: boolean;
}) {
  const delta = after - before;
  const pctChange = before > 0 ? ((delta / before) * 100) : 0;
  const improved = invertDesired ? delta < 0 : delta > 0;
  const degraded = invertDesired ? delta > 0 : delta < 0;
  const changeClass = improved ? 'dd-change-good' : degraded ? 'dd-change-bad' : 'dd-change-neutral';
  const arrow = delta > 0 ? '\u2191' : delta < 0 ? '\u2193' : '\u2192';

  return (
    <div class="dd-metric-change">
      <span class="dd-metric-change-label">{label}</span>
      <div class="dd-metric-change-values">
        <span class="dd-metric-change-before">{formatFn(before)}</span>
        <span class={`dd-metric-change-arrow ${changeClass}`}>{arrow}</span>
        <span class="dd-metric-change-after">{formatFn(after)}</span>
      </div>
      {pctChange !== 0 && (
        <span class={`dd-metric-change-pct ${changeClass}`}>
          {pctChange > 0 ? '+' : ''}{pctChange.toFixed(1)}%
        </span>
      )}
    </div>
  );
}

/* ====== Main Component ====== */

export function DeployDetail({ deploy, onClose }: DeployDetailProps) {
  const [traces, setTraces] = useState<TraceSummaryForDeploy[]>([]);
  const [loadingTraces, setLoadingTraces] = useState(true);
  const [showRollbackConfirm, setShowRollbackConfirm] = useState(false);
  const [rollbackStatus, setRollbackStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');
  const [rollbackError, setRollbackError] = useState<string | null>(null);
  const [containers, setContainers] = useState<ContainerInfo[]>([]);
  const [containerSource, setContainerSource] = useState<ContainerDataSource>('none');
  const [loadingContainers, setLoadingContainers] = useState(true);

  useEffect(() => {
    setLoadingTraces(true);
    api<TraceSummaryForDeploy[]>('/api/traces')
      .then((t) => setTraces(Array.isArray(t) ? t : []))
      .catch(() => setTraces([]))
      .finally(() => setLoadingTraces(false));
  }, [deploy.id]);

  // Try real K8s pods, then ECS tasks, then show empty state
  useEffect(() => {
    setLoadingContainers(true);
    (async () => {
      // Try K8s first
      const k8sResult = await fetchK8sPods(deploy.service);
      if (k8sResult) {
        setContainers(k8sResult.containers);
        setContainerSource(k8sResult.source);
        setLoadingContainers(false);
        return;
      }

      // Try ECS
      const ecsResult = await fetchEcsTasks(deploy.service);
      if (ecsResult) {
        setContainers(ecsResult.containers);
        setContainerSource(ecsResult.source);
        setLoadingContainers(false);
        return;
      }

      // No container data available from either source
      setContainers([]);
      setContainerSource('none');
      setLoadingContainers(false);
    })();
  }, [deploy.id, deploy.service]);

  const comparison = useMemo(
    () => computeMetricsComparison(traces, deploy.timestamp, deploy.service),
    [traces, deploy.timestamp, deploy.service],
  );

  const commitDisplay = deploy.commit ? deploy.commit.slice(0, 8) : '--';

  async function handleRollback() {
    setRollbackStatus('loading');
    setRollbackError(null);
    try {
      await api<unknown>('/api/deploys', {
        method: 'POST',
        body: JSON.stringify({
          service: deploy.service,
          commit: `rollback-${deploy.commit.slice(0, 8)}`,
          author: 'devtools',
          message: `Rollback from ${deploy.commit.slice(0, 8)}`,
          branch: deploy.branch || 'main',
        }),
      });
      setRollbackStatus('success');
      setShowRollbackConfirm(false);
    } catch (err: any) {
      setRollbackStatus('error');
      setRollbackError(err.message || 'Rollback failed');
    }
  }

  return (
    <div class="dd-overlay" onClick={onClose}>
      <div class="dd-panel" onClick={(e) => e.stopPropagation()}>
        {/* Header */}
        <div class="dd-header">
          <div class="dd-header-top">
            <div class="dd-header-info">
              <div class="dd-header-service">{deploy.service}</div>
              <div class="dd-header-meta">
                <span class="dd-header-commit">{commitDisplay}</span>
                <span class="dd-header-sep">&middot;</span>
                <span>{deploy.author || 'unknown'}</span>
                {deploy.branch && (
                  <>
                    <span class="dd-header-sep">&middot;</span>
                    <span class="dd-header-branch">{deploy.branch}</span>
                  </>
                )}
              </div>
              <div class="dd-header-time">
                {formatTimestamp(deploy.timestamp)} ({relativeTime(deploy.timestamp)})
              </div>
            </div>
            <div class="dd-header-actions">
              <StatusBadge deploy={deploy} />
              <button class="dd-close-btn" onClick={onClose} title="Close">
                &times;
              </button>
            </div>
          </div>
          {deploy.message && (
            <div class="dd-header-message">{deploy.message}</div>
          )}
        </div>

        {/* Containers / Pods */}
        <div class="dd-section">
          <div class="dd-section-title">
            Containers / Pods
            {containerSource === 'k8s' && (
              <span class="dd-source-badge dd-source-live">K8s</span>
            )}
            {containerSource === 'ecs' && (
              <span class="dd-source-badge dd-source-live">ECS</span>
            )}
          </div>
          {loadingContainers ? (
            <div class="dd-loading">Loading container data...</div>
          ) : containers.length > 0 ? (
            <div class="dd-container-list">
              {containers.map((c) => (
                <ContainerRow key={c.name} container={c} />
              ))}
            </div>
          ) : (
            <div class="dd-loading">
              No container data available — deploy to K8s or ECS to see pod/task status
            </div>
          )}
        </div>

        {/* Metrics comparison */}
        <div class="dd-section">
          <div class="dd-section-title">Before / After Deploy</div>
          {loadingTraces ? (
            <div class="dd-loading">Loading metrics...</div>
          ) : (
            <div class="dd-metrics-grid">
              <MetricChangeIndicator
                label="Error Rate"
                before={comparison.beforeErrorRate}
                after={comparison.afterErrorRate}
                formatFn={(v) => `${(v * 100).toFixed(2)}%`}
                invertDesired={false}
              />
              <MetricChangeIndicator
                label="p99 Latency"
                before={comparison.beforeP99}
                after={comparison.afterP99}
                formatFn={(v) => formatDuration(v)}
                invertDesired={false}
              />
              <MetricChangeIndicator
                label="Request Volume"
                before={comparison.beforeVolume}
                after={comparison.afterVolume}
                formatFn={(v) => `${v} req`}
              />
            </div>
          )}
        </div>

        {/* Rollback section */}
        <div class="dd-section">
          <div class="dd-section-title">Rollback</div>

          {rollbackStatus === 'success' && (
            <div class="dd-rollback-feedback dd-rollback-success">
              Rollback initiated for {deploy.service}
            </div>
          )}

          {rollbackStatus === 'error' && (
            <div class="dd-rollback-feedback dd-rollback-error">
              Failed: {rollbackError}
            </div>
          )}

          {showRollbackConfirm ? (
            <div class="dd-rollback-confirm">
              <p class="dd-rollback-confirm-text">
                Rollback <strong>{deploy.service}</strong> to previous version?
              </p>
              <div class="dd-rollback-confirm-actions">
                <button
                  class="btn btn-primary dd-rollback-confirm-btn"
                  onClick={handleRollback}
                  disabled={rollbackStatus === 'loading'}
                >
                  {rollbackStatus === 'loading' ? 'Rolling back...' : 'Confirm Rollback'}
                </button>
                <button
                  class="btn btn-ghost"
                  onClick={() => setShowRollbackConfirm(false)}
                  disabled={rollbackStatus === 'loading'}
                >
                  Cancel
                </button>
              </div>
              <p class="dd-rollback-note">
                In production, this would trigger your CI/CD pipeline.
              </p>
            </div>
          ) : (
            rollbackStatus !== 'success' && (
              <button
                class="btn btn-ghost dd-rollback-btn"
                onClick={() => setShowRollbackConfirm(true)}
              >
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                  <path d="M2 7h6M2 7l3-3M2 7l3 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                  <path d="M5 3h4a3 3 0 010 6H8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
                </svg>
                Rollback to previous version
              </button>
            )
          )}
        </div>
      </div>
    </div>
  );
}
