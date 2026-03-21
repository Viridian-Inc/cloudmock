import { useState, useMemo } from 'preact/hooks';

interface ServicesPageProps {
  services: any[];
  stats: any;
  health: any;
}

export function ServicesPage({ services, stats, health }: ServicesPageProps) {
  const [search, setSearch] = useState('');

  const filtered = useMemo(() => {
    if (!search) return services;
    const q = search.toLowerCase();
    return services.filter((s: any) => s.name.toLowerCase().includes(q));
  }, [services, search]);

  const totalRequests = useMemo(() => {
    if (!stats || !stats.services) return 0;
    return Object.values(stats.services).reduce((sum: number, s: any) => sum + (s.total || 0), 0);
  }, [stats]);

  const healthyCount = useMemo(() => {
    if (!health || !health.services) return 0;
    return Object.values(health.services).filter(Boolean).length;
  }, [health]);

  return (
    <div>
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="page-title">Services</h1>
          <p class="page-desc">Registered AWS service mocks</p>
        </div>
      </div>

      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-label">Total Services</div>
          <div class="stat-value">{services.length}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Total Requests</div>
          <div class="stat-value">{totalRequests.toLocaleString()}</div>
          <div class="stat-sub">Requests/min tracked in /api/stats</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Healthy</div>
          <div class="stat-value">{healthyCount} / {services.length}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Uptime</div>
          <div class="stat-value">100%</div>
        </div>
      </div>

      <div class="mb-4">
        <input
          class="input input-search"
          style="width:320px"
          placeholder="Filter services..."
          value={search}
          onInput={(e) => setSearch((e.target as HTMLInputElement).value)}
        />
      </div>

      <div class="services-grid">
        {filtered.map((svc: any) => {
          const tier = svc.action_count > 5 ? 'T1' : 'T2';
          return (
            <div class="svc-card" onClick={() => (location.hash = `/resources?service=${svc.name}`)}>
              <div class="svc-card-head">
                <span class="svc-card-name">{svc.name}</span>
                <span class={`svc-card-tier ${tier === 'T1' ? 'tier-t1' : 'tier-t2'}`}>{tier}</span>
                <div class="svc-card-status">
                  <span class={`dot ${svc.healthy ? 'dot-green' : 'dot-red'}`} />
                  {svc.healthy ? 'Healthy' : 'Unhealthy'}
                </div>
              </div>
              <div class="svc-card-meta">
                <span>{svc.action_count} actions</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
