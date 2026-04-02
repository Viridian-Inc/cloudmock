import { useState, useEffect, useMemo } from 'preact/hooks';
import { api } from '../../lib/api';

interface DomainGroup {
  name: string;
  services: string[];
}

interface DomainsConfig {
  domains: DomainGroup[];
  userFacingOverrides?: Record<string, boolean>;
  routing?: Record<string, Record<string, string>>;
}

const STORAGE_KEY = 'neureaux-devtools:domains';

async function fetchOriginalConfig(): Promise<DomainsConfig | null> {
  try {
    const res = await fetch('/service-domains.json');
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

function loadSavedConfig(): DomainsConfig | null {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) return JSON.parse(saved);
  } catch (e) { console.warn('[Domains] Failed to parse saved config:', e); }
  return null;
}

function saveDomainConfig(config: DomainsConfig) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
}

interface DiffLine {
  type: 'added' | 'removed' | 'unchanged';
  text: string;
}

function computeDiff(original: DomainsConfig, modified: DomainsConfig): DiffLine[] {
  const origLines = JSON.stringify(original, null, 2).split('\n');
  const modLines = JSON.stringify(modified, null, 2).split('\n');
  const lines: DiffLine[] = [];

  const maxLen = Math.max(origLines.length, modLines.length);
  // Simple line-by-line diff
  const origSet = new Set(origLines);
  const modSet = new Set(modLines);

  for (const line of origLines) {
    if (!modSet.has(line)) {
      lines.push({ type: 'removed', text: line });
    }
  }
  for (const line of modLines) {
    if (!origSet.has(line)) {
      lines.push({ type: 'added', text: line });
    }
  }

  if (lines.length === 0) {
    lines.push({ type: 'unchanged', text: '(no changes)' });
  }

  return lines;
}

export function Domains() {
  const [originalConfig, setOriginalConfig] = useState<DomainsConfig | null>(null);
  const [config, setConfig] = useState<DomainsConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showDiff, setShowDiff] = useState(false);
  const [saved, setSaved] = useState(false);
  const [editingName, setEditingName] = useState<number | null>(null);
  const [apiServices, setApiServices] = useState<string[]>([]);

  // Fetch known services from the IaC config API
  useEffect(() => {
    api<any[]>('/api/services')
      .then((data) => setApiServices(data.map((s) => s.name || s.Name).filter(Boolean)))
      .catch(() => {
        // Try topology config as fallback
        api<{ nodes: any[] }>('/api/topology/config')
          .then((config) => setApiServices((config.nodes || []).map((n: any) => n.label || n.id)))
          .catch(() => setApiServices([]));
      });
  }, []);

  useEffect(() => {
    async function load() {
      setLoading(true);
      const fetched = await fetchOriginalConfig();
      if (!fetched) {
        setError('Could not load /service-domains.json');
        setLoading(false);
        return;
      }
      setOriginalConfig(fetched);
      const savedConfig = loadSavedConfig();
      setConfig(savedConfig ?? JSON.parse(JSON.stringify(fetched)));
      setLoading(false);
    }
    load();
  }, []);

  const allAssignedServices = useMemo(() => {
    if (!config) return new Set<string>();
    const assigned = new Set<string>();
    for (const group of config.domains) {
      for (const svc of group.services) {
        assigned.add(svc);
      }
    }
    return assigned;
  }, [config]);

  const allKnownServices = useMemo(() => {
    const svcSet = new Set<string>(apiServices);
    if (originalConfig) {
      for (const group of originalConfig.domains) {
        for (const svc of group.services) {
          svcSet.add(svc);
        }
      }
    }
    return Array.from(svcSet).sort();
  }, [originalConfig, apiServices]);

  const availableServices = useMemo(() => {
    return allKnownServices.filter((s) => !allAssignedServices.has(s));
  }, [allKnownServices, allAssignedServices]);

  const hasChanges = useMemo(() => {
    if (!originalConfig || !config) return false;
    const savedConfig = loadSavedConfig();
    const compareTo = savedConfig ?? originalConfig;
    return JSON.stringify(compareTo) !== JSON.stringify(config);
  }, [originalConfig, config]);

  function updateDomainName(index: number, name: string) {
    if (!config) return;
    const updated = { ...config, domains: config.domains.map((d, i) =>
      i === index ? { ...d, name } : d
    )};
    setConfig(updated);
    setSaved(false);
  }

  function removeService(domainIndex: number, service: string) {
    if (!config) return;
    const updated = { ...config, domains: config.domains.map((d, i) =>
      i === domainIndex ? { ...d, services: d.services.filter((s) => s !== service) } : d
    )};
    setConfig(updated);
    setSaved(false);
  }

  function addServiceToDomain(domainIndex: number, service: string) {
    if (!config || !service) return;
    const updated = { ...config, domains: config.domains.map((d, i) =>
      i === domainIndex ? { ...d, services: [...d.services, service] } : d
    )};
    setConfig(updated);
    setSaved(false);
  }

  function addDomain() {
    if (!config) return;
    const updated = {
      ...config,
      domains: [...config.domains, { name: 'New Domain', services: [] }],
    };
    setConfig(updated);
    setSaved(false);
    setEditingName(updated.domains.length - 1);
  }

  function deleteDomain(index: number) {
    if (!config) return;
    const updated = {
      ...config,
      domains: config.domains.filter((_, i) => i !== index),
    };
    setConfig(updated);
    setSaved(false);
  }

  function resetToOriginal() {
    if (!originalConfig) return;
    setConfig(JSON.parse(JSON.stringify(originalConfig)));
    localStorage.removeItem(STORAGE_KEY);
    setSaved(false);
    setShowDiff(false);
  }

  function handleSave() {
    if (!config) return;
    saveDomainConfig(config);
    setSaved(true);
    setShowDiff(false);
    setTimeout(() => setSaved(false), 2000);
  }

  if (loading) {
    return <div class="settings-placeholder">Loading domain configuration...</div>;
  }

  if (error || !config || !originalConfig) {
    return <div class="settings-error">{error || 'Failed to load configuration'}</div>;
  }

  const diffLines = showDiff ? computeDiff(originalConfig, config) : [];

  return (
    <div class="settings-section" style="max-width: 700px;">
      <h3 class="settings-section-title">Service Domains</h3>
      <p class="settings-section-desc">
        Organize services into domain groups. Changes are saved to localStorage (cannot write files from the browser).
      </p>

      <div class="domains-cards">
        {config.domains.map((group, gi) => (
          <div key={gi} class="domain-card">
            <div class="domain-card-header">
              {editingName === gi ? (
                <input
                  class="input domain-name-input"
                  value={group.name}
                  onInput={(e) => updateDomainName(gi, (e.target as HTMLInputElement).value)}
                  onBlur={() => setEditingName(null)}
                  onKeyDown={(e) => { if (e.key === 'Enter') setEditingName(null); }}
                  autoFocus
                />
              ) : (
                <span
                  class="domain-card-name"
                  onClick={() => setEditingName(gi)}
                  title="Click to edit name"
                >
                  {group.name}
                </span>
              )}
              <button
                class="btn btn-ghost domain-delete-btn"
                onClick={() => deleteDomain(gi)}
                title="Delete domain group"
              >
                Delete
              </button>
            </div>

            <div class="domain-services">
              {group.services.length === 0 && (
                <span class="domain-empty">No services assigned</span>
              )}
              {group.services.map((svc) => (
                <span key={svc} class="domain-service-chip">
                  {svc}
                  <button
                    class="domain-service-remove"
                    onClick={() => removeService(gi, svc)}
                    title={`Remove ${svc}`}
                  >
                    x
                  </button>
                </span>
              ))}
            </div>

            {availableServices.length > 0 && (
              <div class="domain-add-service">
                <select
                  class="input domain-add-select"
                  onChange={(e) => {
                    const val = (e.target as HTMLSelectElement).value;
                    if (val) {
                      addServiceToDomain(gi, val);
                      (e.target as HTMLSelectElement).value = '';
                    }
                  }}
                >
                  <option value="">+ Add service...</option>
                  {availableServices.map((s) => (
                    <option key={s} value={s}>{s}</option>
                  ))}
                </select>
              </div>
            )}
          </div>
        ))}
      </div>

      <div class="domains-actions">
        <button class="btn" onClick={addDomain}>Add Domain</button>
        <button
          class="btn btn-ghost"
          onClick={() => setShowDiff(!showDiff)}
          disabled={!hasChanges && !showDiff}
        >
          {showDiff ? 'Hide Diff' : 'Show Diff'}
        </button>
        <button class="btn btn-ghost" onClick={resetToOriginal}>Reset</button>
        <button class="btn btn-primary" onClick={handleSave} disabled={!hasChanges && !saved}>
          {saved ? 'Saved' : 'Save'}
        </button>
      </div>

      {showDiff && (
        <div class="domains-diff">
          <div class="domains-diff-title">Changes from original</div>
          <pre class="domains-diff-block">
            {diffLines.map((line, i) => (
              <div
                key={i}
                class={`domains-diff-line ${line.type}`}
              >
                {line.type === 'added' ? '+ ' : line.type === 'removed' ? '- ' : '  '}
                {line.text}
              </div>
            ))}
          </pre>
        </div>
      )}
    </div>
  );
}
