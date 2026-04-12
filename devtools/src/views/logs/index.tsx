import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './logs.css';

interface LogEntry {
  timestamp: string;
  service: string;
  severity: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
  message: string;
}

export function LogsView() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [severity, setSeverity] = useState('');
  const [service, setService] = useState('');

  useEffect(() => { fetchLogs(); }, []);

  async function fetchLogs() {
    setLoading(true);
    try {
      const data = await api<{ logs: LogEntry[] }>('/api/lambda/logs');
      setLogs(data.logs || []);
    } catch {
      setLogs([]);
    }
    setLoading(false);
  }

  const filtered = logs.filter((l) => {
    if (severity && l.severity !== severity) return false;
    if (service && !l.service.toLowerCase().includes(service.toLowerCase())) return false;
    if (search && !l.message.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  return (
    <div class="logs-view">
      <div class="logs-toolbar">
        <input placeholder="Search logs..." value={search}
          onInput={(e) => setSearch((e.target as HTMLInputElement).value)} />
        <input placeholder="Service..." value={service} style="flex:none;width:120px"
          onInput={(e) => setService((e.target as HTMLInputElement).value)} />
        <select value={severity} onChange={(e) => setSeverity((e.target as HTMLSelectElement).value)}>
          <option value="">All levels</option>
          <option value="DEBUG">DEBUG</option>
          <option value="INFO">INFO</option>
          <option value="WARN">WARN</option>
          <option value="ERROR">ERROR</option>
        </select>
        <button class="btn btn-ghost btn-sm" onClick={fetchLogs}>Refresh</button>
      </div>
      {loading ? (
        <div class="logs-empty">Loading logs...</div>
      ) : filtered.length === 0 ? (
        <div class="logs-empty">No log entries found</div>
      ) : (
        <div class="logs-table-wrap">
          <table class="logs-table">
            <thead>
              <tr><th>Timestamp</th><th>Service</th><th>Severity</th><th>Message</th></tr>
            </thead>
            <tbody>
              {filtered.map((l, i) => (
                <tr key={i}>
                  <td class="logs-ts">{new Date(l.timestamp).toLocaleString()}</td>
                  <td class="logs-svc">{l.service}</td>
                  <td>
                    <span class={`severity-badge severity-${l.severity.toLowerCase()}`}>
                      {l.severity}
                    </span>
                  </td>
                  <td class="logs-msg">{l.message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
