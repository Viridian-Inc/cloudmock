import { useState, useEffect } from 'preact/hooks';
import { getConfig } from '../../lib/api';

export function Config() {
  const [config, setConfig] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    getConfig()
      .then((data) => {
        setConfig(data);
        setError(null);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : String(err));
        setConfig(null);
      })
      .finally(() => setLoading(false));
  }, []);

  return (
    <div class="settings-section">
      <h3 class="settings-section-title">CloudMock Configuration</h3>
      <p class="settings-section-desc">
        Read-only view of the current cloudmock config.
      </p>

      {loading && (
        <div class="settings-placeholder">Loading configuration...</div>
      )}

      {error && <div class="settings-error">{error}</div>}

      {config && (
        <pre class="settings-config-block">
          {JSON.stringify(config, null, 2)}
        </pre>
      )}
    </div>
  );
}
