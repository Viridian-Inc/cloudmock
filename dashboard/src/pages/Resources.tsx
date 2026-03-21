import { useState, useEffect } from 'preact/hooks';
import { api } from '../api';
import { JsonView } from '../components/JsonView';

interface ResourcesPageProps {
  services: any[];
}

export function ResourcesPage({ services }: ResourcesPageProps) {
  const [selected, setSelected] = useState<string | null>(null);
  const [resources, setResources] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(location.hash.split('?')[1] || '');
    const svc = params.get('service');
    if (svc) selectService(svc);
  }, []);

  function selectService(name: string) {
    setSelected(name);
    setLoading(true);
    api(`/api/resources/${name}`).then((r: any) => {
      setResources(r.resources);
      setLoading(false);
    }).catch(() => { setResources(null); setLoading(false); });
  }

  return (
    <div>
      <div class="mb-6">
        <h1 class="page-title">Resource Explorer</h1>
        <p class="page-desc">Browse resources across all AWS services</p>
      </div>

      <div class="flex gap-4" style="height:calc(100vh - var(--header-height) - 120px)">
        <div style="width:220px;flex-shrink:0;overflow-y:auto">
          <div class="card" style="height:100%">
            <div class="card-body" style="padding:8px">
              {services.map((svc: any) => (
                <div
                  class={`nav-item ${selected === svc.name ? 'active' : ''}`}
                  style="border-radius:var(--radius-md);border-left:none;padding:8px 12px"
                  onClick={() => selectService(svc.name)}
                >
                  <span>{svc.name}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div style="flex:1;overflow-y:auto">
          {!selected ? (
            <div class="empty-state">Select a service to browse resources</div>
          ) : loading ? (
            <div class="empty-state">Loading...</div>
          ) : (
            <div class="card">
              <div class="card-header">
                <h3 style="font-weight:700">{selected} Resources</h3>
              </div>
              <div class="card-body">
                <JsonView data={resources} />
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
