import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './iac-diff.css';

interface DiffEntry {
  service: string;
  name: string;
  type: string;
  status: 'synced' | 'missing' | 'orphaned' | 'drift';
  details?: string;
}

interface DiffSummary {
  total: number;
  synced: number;
  missing: number;
  orphaned: number;
  drift: number;
}

interface DiffResult {
  entries: DiffEntry[];
  summary: DiffSummary;
  message?: string;
}

const SERVICE_ICONS: Record<string, string> = {
  dynamodb: '📊',
  lambda: '⚡',
  sqs: '📨',
  sns: '📢',
  s3: '🪣',
  iam: '🔐',
  cognito: '👤',
  apigateway: '🔀',
};

const STATUS_LABELS: Record<string, string> = {
  synced: 'Synced',
  missing: 'Missing',
  orphaned: 'Orphaned',
  drift: 'Drift',
};

export default function IaCDiffView() {
  const [diff, setDiff] = useState<DiffResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string>('all');

  useEffect(() => {
    loadDiff();
    const interval = setInterval(loadDiff, 10000); // refresh every 10s
    return () => clearInterval(interval);
  }, []);

  async function loadDiff() {
    try {
      const result = await api<DiffResult>('/api/iac/diff');
      setDiff(result);
    } catch {
      // API not available — show empty state
    } finally {
      setLoading(false);
    }
  }

  if (loading) {
    return <div class="iac-diff"><p>Loading IaC diff...</p></div>;
  }

  if (!diff || diff.message) {
    return (
      <div class="iac-diff">
        <h2>IaC vs Runtime</h2>
        <div class="no-iac-message">
          <p>No IaC project detected.</p>
          <p>
            Start CloudMock with <code>--iac path/to/project</code> to compare
            your Terraform, CDK, SAM, or Pulumi resources against what's running.
          </p>
        </div>
      </div>
    );
  }

  const filtered = filter === 'all'
    ? diff.entries
    : diff.entries.filter(e => e.status === filter);

  return (
    <div class="iac-diff">
      <h2>IaC vs Runtime</h2>

      <div class="diff-summary">
        <button class={`diff-stat ${filter === 'all' ? 'active' : ''}`} onClick={() => setFilter('all')}>
          <div class="count">{diff.summary.total}</div>
          <div class="label">Total</div>
        </button>
        <button class={`diff-stat synced ${filter === 'synced' ? 'active' : ''}`} onClick={() => setFilter('synced')}>
          <div class="count">{diff.summary.synced}</div>
          <div class="label">Synced</div>
        </button>
        <button class={`diff-stat missing ${filter === 'missing' ? 'active' : ''}`} onClick={() => setFilter('missing')}>
          <div class="count">{diff.summary.missing}</div>
          <div class="label">Missing</div>
        </button>
        <button class={`diff-stat orphaned ${filter === 'orphaned' ? 'active' : ''}`} onClick={() => setFilter('orphaned')}>
          <div class="count">{diff.summary.orphaned}</div>
          <div class="label">Orphaned</div>
        </button>
        <button class={`diff-stat drift ${filter === 'drift' ? 'active' : ''}`} onClick={() => setFilter('drift')}>
          <div class="count">{diff.summary.drift}</div>
          <div class="label">Drift</div>
        </button>
      </div>

      {filtered.length === 0 ? (
        <p style={{ color: 'var(--text-secondary, #888)' }}>
          {filter === 'all' ? 'No resources found.' : `No ${filter} resources.`}
        </p>
      ) : (
        <table class="diff-table">
          <thead>
            <tr>
              <th>Service</th>
              <th>Resource</th>
              <th>Type</th>
              <th>Status</th>
              <th>Details</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((entry, i) => (
              <tr key={i}>
                <td>
                  <span class="service-icon">{SERVICE_ICONS[entry.service] || '☁️'}</span>
                  {entry.service}
                </td>
                <td><strong>{entry.name}</strong></td>
                <td>{entry.type}</td>
                <td>
                  <span class={`status-badge ${entry.status}`}>
                    {STATUS_LABELS[entry.status] || entry.status}
                  </span>
                </td>
                <td>{entry.details || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
