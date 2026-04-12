import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './route53.css';

interface RecordSet {
  name: string;
  type: string;
  ttl: number;
  values: string[];
}

interface HostedZone {
  id: string;
  name: string;
  recordSets: RecordSet[];
}

export function Route53View() {
  const [zones, setZones] = useState<HostedZone[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadZones(); }, []);

  async function loadZones() {
    setLoading(true);
    try {
      const data = await api<{ zones: HostedZone[] }>('/api/services/route53');
      setZones(data.zones || []);
    } catch {
      setZones([]);
    }
    setLoading(false);
  }

  if (loading) {
    return <div class="r53-view"><div class="r53-empty">Loading hosted zones...</div></div>;
  }

  return (
    <div class="r53-view">
      <div class="r53-header">
        <h2>Route 53</h2>
        <button class="btn btn-ghost btn-sm" onClick={loadZones}>Refresh</button>
      </div>
      <div class="r53-list">
        {zones.length === 0 && <div class="r53-empty">No hosted zones found</div>}
        {zones.map((zone) => (
          <div class="r53-zone" key={zone.id}>
            <div class="r53-zone-header">
              <span class="r53-zone-name">{zone.name}</span>
              <span class="r53-zone-id">{zone.id}</span>
            </div>
            {zone.recordSets.length > 0 && (
              <table class="r53-table">
                <thead>
                  <tr><th>Name</th><th>Type</th><th>TTL</th><th>Values</th></tr>
                </thead>
                <tbody>
                  {zone.recordSets.map((r, i) => (
                    <tr key={i}>
                      <td>{r.name}</td>
                      <td class="r53-type">{r.type}</td>
                      <td>{r.ttl}</td>
                      <td>{r.values.join(', ')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
