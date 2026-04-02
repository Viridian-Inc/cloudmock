interface ResourceBrowserProps {
  service: string | null;
  resources: any;
  loading: boolean;
}

function JsonTree({ data, depth = 0 }: { data: any; depth?: number }) {
  if (data === null || data === undefined) {
    return <span class="json-null">null</span>;
  }
  if (typeof data === 'boolean') {
    return <span class="json-bool">{String(data)}</span>;
  }
  if (typeof data === 'number') {
    return <span class="json-number">{data}</span>;
  }
  if (typeof data === 'string') {
    return <span class="json-string">"{data}"</span>;
  }
  if (Array.isArray(data)) {
    if (data.length === 0) return <span class="json-empty">[]</span>;
    return (
      <div class="json-array" style={{ marginLeft: depth > 0 ? 16 : 0 }}>
        {data.map((item, i) => (
          <div key={i} class="json-array-item">
            <span class="json-index">[{i}]</span>
            <JsonTree data={item} depth={depth + 1} />
          </div>
        ))}
      </div>
    );
  }
  if (typeof data === 'object') {
    const entries = Object.entries(data).filter(
      ([_, v]) => v !== '' && v !== null && !(typeof v === 'object' && v !== null && Object.keys(v).length === 0),
    );
    if (entries.length === 0) return <span class="json-empty">{'{}'}</span>;
    return (
      <div class="json-object" style={{ marginLeft: depth > 0 ? 16 : 0 }}>
        {entries.map(([key, val]) => (
          <div key={key} class="json-entry">
            <span class="json-key">{key}:</span>{' '}
            <JsonTree data={val} depth={depth + 1} />
          </div>
        ))}
      </div>
    );
  }
  return <span>{String(data)}</span>;
}

export function ResourceBrowser({
  service,
  resources,
  loading,
}: ResourceBrowserProps) {
  if (!service) {
    return (
      <div class="resource-browser resource-browser-placeholder">
        <div class="resource-browser-placeholder-text">
          Select a service to browse resources
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <div class="resource-browser resource-browser-placeholder">
        <div class="resource-browser-placeholder-text">Loading resources...</div>
      </div>
    );
  }

  if (!resources) {
    return (
      <div class="resource-browser resource-browser-placeholder">
        <div class="resource-browser-placeholder-text">No resources</div>
      </div>
    );
  }

  return (
    <div class="resource-browser">
      <div class="resource-browser-header">
        <span class="resource-browser-title">{service}</span>
      </div>
      <div class="resource-browser-body">
        <JsonTree data={resources} />
      </div>
    </div>
  );
}
