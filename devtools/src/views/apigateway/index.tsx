import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './apigateway.css';

interface Route {
  method: string;
  path: string;
  integration: string;
}

interface RestApi {
  id: string;
  name: string;
  routes: Route[];
}

export function ApiGatewayView() {
  const [apis, setApis] = useState<RestApi[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadApis(); }, []);

  async function loadApis() {
    setLoading(true);
    try {
      const data = await api<{ apis: RestApi[] }>('/api/apigateway/apis');
      setApis(data.apis || []);
    } catch {
      setApis([]);
    }
    setLoading(false);
  }

  if (loading) {
    return <div class="apigw-view"><div class="apigw-empty">Loading APIs...</div></div>;
  }

  return (
    <div class="apigw-view">
      <div class="apigw-header">
        <h2>API Gateway</h2>
        <button class="btn btn-ghost btn-sm" onClick={loadApis}>Refresh</button>
      </div>
      <div class="apigw-list">
        {apis.length === 0 && <div class="apigw-empty">No REST APIs found</div>}
        {apis.map((a) => (
          <div class="apigw-api" key={a.id}>
            <div class="apigw-api-header">
              <div class="apigw-api-name">{a.name}</div>
              <div class="apigw-api-id">{a.id}</div>
            </div>
            {a.routes.length > 0 && (
              <table class="apigw-table">
                <thead>
                  <tr><th>Method</th><th>Path</th><th>Integration</th></tr>
                </thead>
                <tbody>
                  {a.routes.map((r, i) => (
                    <tr key={i}>
                      <td><span class="apigw-method">{r.method}</span></td>
                      <td class="apigw-path">{r.path}</td>
                      <td style="color:var(--text-secondary)">{r.integration}</td>
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
