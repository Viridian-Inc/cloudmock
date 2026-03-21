import { TableDescription } from './types';
import { formatBytes, formatDate } from './utils';

interface TableInfoProps {
  tableDesc: TableDescription;
}

export function TableInfo({ tableDesc }: TableInfoProps) {
  const t = tableDesc;

  return (
    <div class="ddb-info-grid">
      {/* General Info */}
      <div class="card">
        <div class="card-header">
          <span style="font-weight:600">General Information</span>
        </div>
        <div class="card-body">
          <div class="ddb-info-table">
            <div class="ddb-info-row">
              <span class="ddb-info-label">Table Name</span>
              <span class="ddb-info-value font-mono">{t.TableName}</span>
            </div>
            <div class="ddb-info-row">
              <span class="ddb-info-label">ARN</span>
              <span class="ddb-info-value font-mono text-sm">{t.TableArn || 'N/A'}</span>
            </div>
            <div class="ddb-info-row">
              <span class="ddb-info-label">Status</span>
              <span class="ddb-info-value">
                <span class={`status-pill ${t.TableStatus === 'ACTIVE' ? 'status-2xx' : 'status-4xx'}`}>
                  {t.TableStatus}
                </span>
              </span>
            </div>
            <div class="ddb-info-row">
              <span class="ddb-info-label">Item Count</span>
              <span class="ddb-info-value">{t.ItemCount ?? 'N/A'}</span>
            </div>
            <div class="ddb-info-row">
              <span class="ddb-info-label">Table Size</span>
              <span class="ddb-info-value">{t.TableSizeBytes !== undefined ? formatBytes(t.TableSizeBytes) : 'N/A'}</span>
            </div>
            <div class="ddb-info-row">
              <span class="ddb-info-label">Billing Mode</span>
              <span class="ddb-info-value">{t.BillingModeSummary?.BillingMode || 'PAY_PER_REQUEST'}</span>
            </div>
            <div class="ddb-info-row">
              <span class="ddb-info-label">Created</span>
              <span class="ddb-info-value">{formatDate(t.CreationDateTime)}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Key Schema */}
      <div class="card">
        <div class="card-header">
          <span style="font-weight:600">Key Schema</span>
        </div>
        <div class="card-body">
          <table class="ddb-info-keys-table">
            <thead>
              <tr>
                <th>Attribute</th>
                <th>Key Type</th>
                <th>Data Type</th>
              </tr>
            </thead>
            <tbody>
              {t.KeySchema.map(k => {
                const attrDef = t.AttributeDefinitions.find(a => a.AttributeName === k.AttributeName);
                return (
                  <tr key={k.AttributeName}>
                    <td class="font-mono">{k.AttributeName}</td>
                    <td>
                      <span class={`status-pill ${k.KeyType === 'HASH' ? 'status-2xx' : 'status-3xx'}`}>
                        {k.KeyType === 'HASH' ? 'Partition Key' : 'Sort Key'}
                      </span>
                    </td>
                    <td>{attrDef?.AttributeType || 'N/A'}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>

          {t.AttributeDefinitions.length > t.KeySchema.length && (
            <>
              <div class="label mt-4">Additional Attribute Definitions</div>
              <table class="ddb-info-keys-table">
                <thead>
                  <tr>
                    <th>Attribute</th>
                    <th>Data Type</th>
                  </tr>
                </thead>
                <tbody>
                  {t.AttributeDefinitions
                    .filter(a => !t.KeySchema.some(k => k.AttributeName === a.AttributeName))
                    .map(a => (
                      <tr key={a.AttributeName}>
                        <td class="font-mono">{a.AttributeName}</td>
                        <td>{a.AttributeType}</td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </>
          )}
        </div>
      </div>

      {/* GSIs */}
      {t.GlobalSecondaryIndexes && t.GlobalSecondaryIndexes.length > 0 && (
        <div class="card">
          <div class="card-header">
            <span style="font-weight:600">Global Secondary Indexes ({t.GlobalSecondaryIndexes.length})</span>
          </div>
          <div class="card-body">
            {t.GlobalSecondaryIndexes.map(gsi => (
              <div key={gsi.IndexName} class="ddb-index-card">
                <div class="flex items-center justify-between mb-2">
                  <span class="font-mono" style="font-weight:600">{gsi.IndexName}</span>
                  <span class={`status-pill ${gsi.IndexStatus === 'ACTIVE' ? 'status-2xx' : 'status-4xx'}`}>
                    {gsi.IndexStatus}
                  </span>
                </div>
                <div class="text-sm text-muted">
                  Keys: {gsi.KeySchema.map(k => `${k.AttributeName} (${k.KeyType})`).join(', ')}
                </div>
                <div class="text-sm text-muted">
                  Projection: {gsi.Projection.ProjectionType}
                </div>
                {gsi.ItemCount !== undefined && (
                  <div class="text-sm text-muted">Items: {gsi.ItemCount}</div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* LSIs */}
      {t.LocalSecondaryIndexes && t.LocalSecondaryIndexes.length > 0 && (
        <div class="card">
          <div class="card-header">
            <span style="font-weight:600">Local Secondary Indexes ({t.LocalSecondaryIndexes.length})</span>
          </div>
          <div class="card-body">
            {t.LocalSecondaryIndexes.map(lsi => (
              <div key={lsi.IndexName} class="ddb-index-card">
                <div style="font-weight:600;margin-bottom:4px" class="font-mono">{lsi.IndexName}</div>
                <div class="text-sm text-muted">
                  Keys: {lsi.KeySchema.map(k => `${k.AttributeName} (${k.KeyType})`).join(', ')}
                </div>
                <div class="text-sm text-muted">
                  Projection: {lsi.Projection.ProjectionType}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Streams */}
      {t.StreamSpecification && (
        <div class="card">
          <div class="card-header">
            <span style="font-weight:600">DynamoDB Streams</span>
          </div>
          <div class="card-body">
            <div class="ddb-info-table">
              <div class="ddb-info-row">
                <span class="ddb-info-label">Enabled</span>
                <span class="ddb-info-value">{String(t.StreamSpecification.StreamEnabled)}</span>
              </div>
              {t.StreamSpecification.StreamViewType && (
                <div class="ddb-info-row">
                  <span class="ddb-info-label">View Type</span>
                  <span class="ddb-info-value">{t.StreamSpecification.StreamViewType}</span>
                </div>
              )}
              {t.LatestStreamArn && (
                <div class="ddb-info-row">
                  <span class="ddb-info-label">Stream ARN</span>
                  <span class="ddb-info-value font-mono text-sm">{t.LatestStreamArn}</span>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
