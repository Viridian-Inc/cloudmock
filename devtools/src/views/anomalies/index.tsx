import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './anomalies.css';

interface Anomaly {
  id: string;
  type: 'latency_spike' | 'error_spike' | 'throughput_drop';
  severity: 'warning' | 'critical';
  service: string;
  message: string;
  detectedAt: string;
}

export function AnomaliesView() {
  const [anomalies, setAnomalies] = useState<Anomaly[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadAnomalies(); }, []);

  async function loadAnomalies() {
    setLoading(true);
    try {
      const data = await api<{ anomalies: Anomaly[] }>('/api/anomalies');
      setAnomalies(data.anomalies || []);
    } catch {
      setAnomalies([]);
    }
    setLoading(false);
  }

  const countByType = (type: string) => anomalies.filter((a) => a.type === type).length;

  if (loading) {
    return <div class="anomalies-view"><div class="anomalies-empty">Loading anomalies...</div></div>;
  }

  return (
    <div class="anomalies-view">
      <div class="anomalies-header">
        <h2>Anomalies</h2>
        <button class="btn btn-ghost btn-sm" onClick={loadAnomalies}>Refresh</button>
      </div>
      <div class="anomalies-cards">
        <div class="anomalies-summary-card">
          <div class="count">{countByType('latency_spike')}</div>
          <div class="label">Latency Spikes</div>
        </div>
        <div class="anomalies-summary-card">
          <div class="count">{countByType('error_spike')}</div>
          <div class="label">Error Spikes</div>
        </div>
        <div class="anomalies-summary-card">
          <div class="count">{countByType('throughput_drop')}</div>
          <div class="label">Throughput Drops</div>
        </div>
      </div>
      <div class="anomalies-list">
        {anomalies.length === 0 && (
          <div class="anomalies-empty">No anomalies detected</div>
        )}
        {anomalies.map((a) => (
          <div class="anomaly-item" key={a.id}>
            <div class="anomaly-item-header">
              <span class="anomaly-type">{a.type.replace(/_/g, ' ')}</span>
              <span class={`anomaly-sev ${a.severity}`}>{a.severity}</span>
            </div>
            <div style="font-size:13px;color:var(--text-primary)">{a.message}</div>
            <div class="anomaly-meta">{a.service} &middot; {new Date(a.detectedAt).toLocaleString()}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
