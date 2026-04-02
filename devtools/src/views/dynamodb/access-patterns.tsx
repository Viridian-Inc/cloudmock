import { useMemo } from 'preact/hooks';
import { TableDescription } from './types';

interface AccessPatternsProps {
  tableDesc: TableDescription;
  onQueryIndex: (indexName: string, pkAttr: string) => void;
}

export function AccessPatterns({ tableDesc, onQueryIndex }: AccessPatternsProps) {
  const pkAttr = tableDesc.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '';
  const skAttr = tableDesc.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '';
  const pkType = tableDesc.AttributeDefinitions.find(a => a.AttributeName === pkAttr)?.AttributeType || 'S';
  const skType = tableDesc.AttributeDefinitions.find(a => a.AttributeName === skAttr)?.AttributeType || '';

  const indexes = useMemo(() => {
    const list: { name: string; type: 'Table' | 'GSI' | 'LSI'; pk: string; pkType: string; sk: string; skType: string; projection: string }[] = [];
    list.push({
      name: 'Table',
      type: 'Table',
      pk: pkAttr,
      pkType,
      sk: skAttr,
      skType: skType || '',
      projection: 'ALL',
    });
    if (tableDesc.GlobalSecondaryIndexes) {
      for (const gsi of tableDesc.GlobalSecondaryIndexes) {
        const gPk = gsi.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '';
        const gSk = gsi.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '';
        list.push({
          name: gsi.IndexName,
          type: 'GSI',
          pk: gPk,
          pkType: tableDesc.AttributeDefinitions.find(a => a.AttributeName === gPk)?.AttributeType || 'S',
          sk: gSk,
          skType: gSk ? (tableDesc.AttributeDefinitions.find(a => a.AttributeName === gSk)?.AttributeType || 'S') : '',
          projection: gsi.Projection.ProjectionType,
        });
      }
    }
    if (tableDesc.LocalSecondaryIndexes) {
      for (const lsi of tableDesc.LocalSecondaryIndexes) {
        const lPk = lsi.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '';
        const lSk = lsi.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '';
        list.push({
          name: lsi.IndexName,
          type: 'LSI',
          pk: lPk,
          pkType: tableDesc.AttributeDefinitions.find(a => a.AttributeName === lPk)?.AttributeType || 'S',
          sk: lSk,
          skType: lSk ? (tableDesc.AttributeDefinitions.find(a => a.AttributeName === lSk)?.AttributeType || 'S') : '',
          projection: lsi.Projection.ProjectionType,
        });
      }
    }
    return list;
  }, [tableDesc]);

  return (
    <div class="ddb-access-patterns">
      <div class="ddb-ap-summary">
        <span class="ddb-ap-label">Table:</span>
        <span class="ddb-ap-value font-mono">{tableDesc.TableName}</span>
        <span class="ddb-ap-sep">|</span>
        <span class="ddb-ap-label">PK:</span>
        <span class="ddb-ap-value font-mono">{pkAttr} <span class="ddb-ap-type">({pkType})</span></span>
        <span class="ddb-ap-sep">|</span>
        <span class="ddb-ap-label">SK:</span>
        <span class="ddb-ap-value font-mono">{skAttr ? `${skAttr} (${skType})` : '\u2014'}</span>
        <span class="ddb-ap-sep">|</span>
        <span class="ddb-ap-label">Items:</span>
        <span class="ddb-ap-value">{tableDesc.ItemCount ?? 0}</span>
        <span class="ddb-ap-sep">|</span>
        <span class="ddb-ap-label">GSIs:</span>
        <span class="ddb-ap-value">{tableDesc.GlobalSecondaryIndexes?.length ?? 0}</span>
      </div>

      <div class="ddb-ap-grid">
        {indexes.map(idx => (
          <div key={idx.name} class="ddb-ap-card">
            <div class="ddb-ap-card-header">
              <div class="flex items-center gap-2">
                <span class="font-mono" style="font-weight:700;font-size:14px">{idx.name}</span>
                <span class={`ddb-ap-badge ddb-ap-badge-${idx.type.toLowerCase()}`}>{idx.type}</span>
              </div>
              <button
                class="btn btn-ghost btn-sm"
                onClick={() => onQueryIndex(idx.type === 'Table' ? '' : idx.name, idx.pk)}
                title="Query this index"
              >
                Query
              </button>
            </div>
            <div class="ddb-ap-card-body">
              <div class="ddb-ap-key-row">
                <span class="ddb-ap-key-badge ddb-ap-key-pk">PK</span>
                <span class="font-mono">{idx.pk}</span>
                <span class="ddb-ap-type">{idx.pkType}</span>
              </div>
              {idx.sk && (
                <div class="ddb-ap-key-row">
                  <span class="ddb-ap-key-badge ddb-ap-key-sk">SK</span>
                  <span class="font-mono">{idx.sk}</span>
                  <span class="ddb-ap-type">{idx.skType}</span>
                </div>
              )}
              <div class="ddb-ap-projection">
                Projection: <span class="font-mono">{idx.projection}</span>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
