import { useState, useMemo, useContext } from 'preact/hooks';
import { DDBItem, TableDescription, FilterCondition, QueryHistoryEntry } from './types';
import { extractValue, getType, typeBadgeColor, collectColumns, ddbRequest } from './utils';
import { PlayIcon, TrashIcon } from '../../components/icons';
import { CodeGenerator } from './code-generator';
import { DDBContext, TabState } from './store';

interface QueryBuilderProps {
  tableName: string;
  tableDesc: TableDescription;
  showToast: (msg: string) => void;
  onEditItem: (item: DDBItem) => void;
  initialIndexName?: string;
  initialPkValue?: string;
  tabIndex: number;
}

const SORT_OPS = ['=', '<', '>', '<=', '>=', 'begins_with', 'between'];

export function QueryBuilder({ tableName, tableDesc, showToast, onEditItem, initialIndexName, initialPkValue, tabIndex }: QueryBuilderProps) {
  const { state, dispatch } = useContext(DDBContext);
  const tab = state.tabs[tabIndex];

  const mode = tab?.queryMode ?? 'query';
  const pkValue = tab?.queryPK ?? (initialPkValue || '');
  const skOp = tab?.querySKOp ?? '=';
  const skValue = tab?.querySK ?? '';
  const indexName = tab?.queryIndex ?? (initialIndexName || '');
  const scanForward = tab?.queryScanForward ?? true;
  const limit = tab?.queryLimit ?? '';
  const results = tab?.queryResults ?? null;
  const scannedCount = tab?.queryScannedCount ?? 0;
  const filters = tab?.queryFilters ?? [];

  const [skValue2, setSkValue2] = useState('');
  const [loading, setLoading] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [showCodeGen, setShowCodeGen] = useState(false);
  const [optimizerHint, setOptimizerHint] = useState<{ indexName: string; pkAttr: string; pkValue: string } | null>(null);
  const [historyCache, setHistoryCache] = useState<QueryHistoryEntry[]>([]);

  function patch(p: Partial<TabState>) {
    dispatch({ type: 'UPDATE_TAB', index: tabIndex, patch: p });
  }

  const tablePkAttr = tableDesc.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '';
  const tableSkAttr = tableDesc.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '';

  const indexes = useMemo(() => {
    const list: { name: string; pk: string; sk: string }[] = [
      { name: '', pk: tablePkAttr, sk: tableSkAttr },
    ];
    if (tableDesc.GlobalSecondaryIndexes) {
      for (const gsi of tableDesc.GlobalSecondaryIndexes) {
        list.push({
          name: gsi.IndexName,
          pk: gsi.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '',
          sk: gsi.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '',
        });
      }
    }
    if (tableDesc.LocalSecondaryIndexes) {
      for (const lsi of tableDesc.LocalSecondaryIndexes) {
        list.push({
          name: lsi.IndexName,
          pk: lsi.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '',
          sk: lsi.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '',
        });
      }
    }
    return list;
  }, [tableDesc]);

  const activeIndex = indexes.find(i => i.name === indexName) || indexes[0];
  const activePkType = tableDesc.AttributeDefinitions.find(a => a.AttributeName === activeIndex.pk)?.AttributeType || 'S';
  const activeSkType = tableDesc.AttributeDefinitions.find(a => a.AttributeName === activeIndex.sk)?.AttributeType || 'S';

  useMemo(() => {
    if (mode !== 'scan') { setOptimizerHint(null); return; }
    const validFilters = filters.filter(f => f.attribute && f.value && f.operator === '=');
    if (validFilters.length === 0) { setOptimizerHint(null); return; }
    for (const idx of indexes) {
      const matchingFilter = validFilters.find(f => f.attribute === idx.pk);
      if (matchingFilter) {
        setOptimizerHint({ indexName: idx.name, pkAttr: idx.pk, pkValue: matchingFilter.value });
        return;
      }
    }
    setOptimizerHint(null);
  }, [mode, filters, indexes]);

  function addFilter() {
    patch({ queryFilters: [...filters, { attribute: '', operator: '=', value: '', value2: '', connector: 'AND' }] });
  }

  function updateFilter(idx: number, p: Partial<FilterCondition>) {
    patch({ queryFilters: filters.map((f, i) => i === idx ? { ...f, ...p } : f) });
  }

  function removeFilter(idx: number) {
    patch({ queryFilters: filters.filter((_, i) => i !== idx) });
  }

  function buildFilterExpression(): { expr: string; values: Record<string, any>; names: Record<string, string> } | null {
    const validFilters = filters.filter(f => f.attribute && f.value);
    if (validFilters.length === 0) return null;
    const names: Record<string, string> = {};
    const values: Record<string, any> = {};
    const parts: string[] = [];
    validFilters.forEach((f, i) => {
      const nameKey = `#fa${i}`;
      const valKey = `:fv${i}`;
      names[nameKey] = f.attribute;
      values[valKey] = { S: f.value };
      if (i > 0) parts.push(f.connector);
      if (f.operator === 'between') {
        const valKey2 = `:fv${i}b`;
        values[valKey2] = { S: f.value2 };
        parts.push(`${nameKey} BETWEEN ${valKey} AND ${valKey2}`);
      } else if (f.operator === 'begins_with') {
        parts.push(`begins_with(${nameKey}, ${valKey})`);
      } else if (f.operator === 'attribute_exists') {
        parts.push(`attribute_exists(${nameKey})`);
      } else if (f.operator === 'attribute_not_exists') {
        parts.push(`attribute_not_exists(${nameKey})`);
      } else {
        parts.push(`${nameKey} ${f.operator} ${valKey}`);
      }
    });
    return { expr: parts.join(' '), values, names };
  }

  async function runQuery() {
    setLoading(true);
    try {
      const params: any = { TableName: tableName };
      if (limit) params.Limit = Number(limit);

      if (mode === 'query') {
        const exprNames: Record<string, string> = {};
        const exprValues: Record<string, any> = {};

        exprNames['#pk'] = activeIndex.pk;
        exprValues[':pkv'] = activePkType === 'N' ? { N: pkValue } : { S: pkValue };

        let keyExpr = '#pk = :pkv';
        if (activeIndex.sk && skValue) {
          exprNames['#sk'] = activeIndex.sk;
          const skTyped = activeSkType === 'N' ? { N: skValue } : { S: skValue };
          exprValues[':skv'] = skTyped;
          if (skOp === 'between') {
            exprValues[':skv2'] = activeSkType === 'N' ? { N: skValue2 } : { S: skValue2 };
            keyExpr += ' AND #sk BETWEEN :skv AND :skv2';
          } else if (skOp === 'begins_with') {
            keyExpr += ' AND begins_with(#sk, :skv)';
          } else {
            keyExpr += ` AND #sk ${skOp} :skv`;
          }
        }

        params.KeyConditionExpression = keyExpr;
        params.ExpressionAttributeNames = exprNames;
        params.ExpressionAttributeValues = exprValues;
        params.ScanIndexForward = scanForward;
        if (indexName) params.IndexName = indexName;

        const filter = buildFilterExpression();
        if (filter) {
          params.FilterExpression = filter.expr;
          Object.assign(params.ExpressionAttributeNames, filter.names);
          Object.assign(params.ExpressionAttributeValues, filter.values);
        }

        const r = await ddbRequest('Query', params);
        dispatch({ type: 'SET_QUERY_RESULTS', index: tabIndex, results: r.Items || [], scannedCount: r.ScannedCount || r.Count || 0 });
        saveHistory('query', r.Items?.length || 0);
      } else {
        const filter = buildFilterExpression();
        if (filter) {
          params.FilterExpression = filter.expr;
          params.ExpressionAttributeNames = filter.names;
          params.ExpressionAttributeValues = filter.values;
        }
        const r = await ddbRequest('Scan', params);
        dispatch({ type: 'SET_QUERY_RESULTS', index: tabIndex, results: r.Items || [], scannedCount: r.ScannedCount || r.Count || 0 });
        saveHistory('scan', r.Items?.length || 0);
      }
    } catch (e: any) {
      showToast('Query failed: ' + (e.message || 'unknown error'));
    } finally {
      setLoading(false);
    }
  }

  function saveHistory(type: 'query' | 'scan', count: number) {
    const entry: QueryHistoryEntry = {
      id: Date.now().toString(36),
      timestamp: Date.now(),
      table: tableName,
      type,
      partitionKey: activeIndex.pk,
      partitionValue: pkValue,
      sortCondition: skOp,
      sortValue: skValue,
      indexName,
      resultCount: count,
    };
    try {
      const raw = localStorage.getItem('ddb-query-history');
      const list: QueryHistoryEntry[] = raw ? JSON.parse(raw) : [];
      list.unshift(entry);
      localStorage.setItem('ddb-query-history', JSON.stringify(list.slice(0, 10)));
    } catch { /* ignore */ }
  }

  function loadHistory(): QueryHistoryEntry[] {
    return historyCache;
  }

  function applyHistory(entry: QueryHistoryEntry) {
    patch({
      queryMode: entry.type,
      queryPK: entry.partitionValue,
      querySKOp: entry.sortCondition,
      querySK: entry.sortValue,
      queryIndex: entry.indexName,
    });
    setShowHistory(false);
  }

  const resultColumns = useMemo(() => {
    if (!results || results.length === 0) return [];
    const keyAttrs = tableDesc.KeySchema.map(k => k.AttributeName);
    return collectColumns(results, keyAttrs);
  }, [results, tableDesc]);

  return (
    <div>
      <div class="ddb-query-card">
        <div class="flex items-center justify-between mb-4">
          <div class="flex gap-2">
            <button
              class={`btn btn-sm ${mode === 'query' ? 'btn-secondary' : 'btn-ghost'}`}
              onClick={() => patch({ queryMode: 'query' })}
            >Query</button>
            <button
              class={`btn btn-sm ${mode === 'scan' ? 'btn-secondary' : 'btn-ghost'}`}
              onClick={() => patch({ queryMode: 'scan' })}
            >Scan</button>
          </div>
          <button class="btn btn-ghost btn-sm" onClick={() => {
            const next = !showHistory;
            setShowHistory(next);
            if (next) {
              try {
                const raw = localStorage.getItem('ddb-query-history');
                setHistoryCache(raw ? JSON.parse(raw) : []);
              } catch { setHistoryCache([]); }
            }
          }}>
            History
          </button>
        </div>

        {showHistory && (
          <div class="ddb-history-panel mb-4">
            {loadHistory().length === 0 ? (
              <div class="text-sm text-muted" style="padding:12px">No query history</div>
            ) : loadHistory().map(h => (
              <div key={h.id} class="ddb-history-item" onClick={() => applyHistory(h)}>
                <span class="ddb-type-badge" style={`background:${h.type === 'query' ? 'rgba(9,127,245,0.1)' : 'rgba(2,150,98,0.1)'};color:${h.type === 'query' ? '#097FF5' : '#029662'}`}>
                  {h.type}
                </span>
                <span class="font-mono text-sm">{h.partitionKey}={h.partitionValue}</span>
                <span class="text-muted text-sm ml-auto">{h.resultCount} results</span>
              </div>
            ))}
          </div>
        )}

        {mode === 'query' && (
          <>
            <div class="mb-4">
              <div class="label">Index</div>
              <select
                class="select w-full"
                value={indexName}
                onChange={(e) => {
                  patch({ queryIndex: (e.target as HTMLSelectElement).value, queryPK: '', querySK: '' });
                  setSkValue2('');
                }}
              >
                {indexes.map(idx => {
                  const isGSI = tableDesc.GlobalSecondaryIndexes?.some(g => g.IndexName === idx.name);
                  const isLSI = tableDesc.LocalSecondaryIndexes?.some(l => l.IndexName === idx.name);
                  const tag = isGSI ? ' [GSI]' : isLSI ? ' [LSI]' : '';
                  return (
                    <option key={idx.name} value={idx.name}>
                      {idx.name ? `${idx.name}${tag} (${idx.pk}${idx.sk ? ', ' + idx.sk : ''})` : `Table (${idx.pk}${idx.sk ? ', ' + idx.sk : ''})`}
                    </option>
                  );
                })}
              </select>
            </div>

            <div class="mb-4">
              <div class="label">Partition Key ({activeIndex.pk})</div>
              <input
                class="input w-full"
                placeholder={`Enter ${activeIndex.pk} value...`}
                value={pkValue}
                onInput={(e) => patch({ queryPK: (e.target as HTMLInputElement).value })}
              />
            </div>

            {activeIndex.sk && (
              <div class="mb-4">
                <div class="label">Sort Key ({activeIndex.sk})</div>
                <div class="flex gap-2">
                  <select
                    class="select"
                    style="width:160px"
                    value={skOp}
                    onChange={(e) => patch({ querySKOp: (e.target as HTMLSelectElement).value })}
                  >
                    {SORT_OPS.map(op => <option key={op} value={op}>{op}</option>)}
                  </select>
                  <input
                    class="input"
                    style="flex:1"
                    placeholder="Value..."
                    value={skValue}
                    onInput={(e) => patch({ querySK: (e.target as HTMLInputElement).value })}
                  />
                  {skOp === 'between' && (
                    <input
                      class="input"
                      style="flex:1"
                      placeholder="Value 2..."
                      value={skValue2}
                      onInput={(e) => setSkValue2((e.target as HTMLInputElement).value)}
                    />
                  )}
                </div>
              </div>
            )}

            <div class="mb-4">
              <label class="flex items-center gap-2" style="cursor:pointer">
                <input
                  type="checkbox"
                  checked={scanForward}
                  onChange={() => patch({ queryScanForward: !scanForward })}
                />
                <span class="text-sm">Ascending order (ScanIndexForward)</span>
              </label>
            </div>
          </>
        )}

        {/* Filter conditions */}
        <div class="mb-4">
          <div class="flex items-center justify-between mb-2">
            <div class="label" style="margin-bottom:0">Filter Conditions</div>
            <button class="btn btn-ghost btn-sm" onClick={addFilter}>+ Add Filter</button>
          </div>
          {filters.map((f, i) => (
            <div key={i} class="ddb-filter-row">
              {i > 0 && (
                <select
                  class="select"
                  style="width:70px"
                  value={f.connector}
                  onChange={(e) => updateFilter(i, { connector: (e.target as HTMLSelectElement).value as 'AND' | 'OR' })}
                >
                  <option value="AND">AND</option>
                  <option value="OR">OR</option>
                </select>
              )}
              <input
                class="input"
                style="width:140px"
                placeholder="Attribute"
                value={f.attribute}
                onInput={(e) => updateFilter(i, { attribute: (e.target as HTMLInputElement).value })}
              />
              <select
                class="select"
                style="width:140px"
                value={f.operator}
                onChange={(e) => updateFilter(i, { operator: (e.target as HTMLSelectElement).value })}
              >
                <option value="=">=</option>
                <option value="!=">!=</option>
                <option value="<">&lt;</option>
                <option value=">">&gt;</option>
                <option value="<=">&lt;=</option>
                <option value=">=">&gt;=</option>
                <option value="begins_with">begins_with</option>
                <option value="between">between</option>
                <option value="attribute_exists">exists</option>
                <option value="attribute_not_exists">not exists</option>
              </select>
              {f.operator !== 'attribute_exists' && f.operator !== 'attribute_not_exists' && (
                <input
                  class="input"
                  style="flex:1"
                  placeholder="Value"
                  value={f.value}
                  onInput={(e) => updateFilter(i, { value: (e.target as HTMLInputElement).value })}
                />
              )}
              {f.operator === 'between' && (
                <input
                  class="input"
                  style="flex:1"
                  placeholder="Value 2"
                  value={f.value2}
                  onInput={(e) => updateFilter(i, { value2: (e.target as HTMLInputElement).value })}
                />
              )}
              <button class="btn-icon btn-sm btn-ghost" onClick={() => removeFilter(i)} title="Remove">
                <TrashIcon />
              </button>
            </div>
          ))}
        </div>

        <div class="mb-4">
          <div class="label">Limit</div>
          <input
            class="input"
            type="number"
            style="width:120px"
            placeholder="No limit"
            value={limit}
            onInput={(e) => patch({ queryLimit: (e.target as HTMLInputElement).value })}
          />
        </div>

        {mode === 'scan' && optimizerHint && (
          <div class="ddb-optimizer-banner mb-4">
            <div class="ddb-optimizer-text">
              This scan can be optimized to a <strong>Query</strong> using{' '}
              <strong>{optimizerHint.indexName || 'the table primary key'}</strong>
            </div>
            <button
              class="btn btn-secondary btn-sm"
              onClick={() => {
                patch({
                  queryMode: 'query',
                  queryIndex: optimizerHint.indexName,
                  queryPK: optimizerHint.pkValue,
                  queryFilters: filters.filter(f => f.attribute !== optimizerHint.pkAttr),
                });
                setOptimizerHint(null);
              }}
            >
              Optimize
            </button>
          </div>
        )}

        <div class="flex items-center gap-2">
          <button
            class="btn btn-primary btn-sm"
            onClick={runQuery}
            disabled={loading || (mode === 'query' && !pkValue)}
          >
            <PlayIcon /> {loading ? 'Running...' : `Run ${mode === 'query' ? 'Query' : 'Scan'}`}
          </button>
          {results && results.length > 0 && (
            <button class="btn btn-ghost btn-sm" onClick={() => setShowCodeGen(true)}>
              Generate Code
            </button>
          )}
        </div>
      </div>

      {showCodeGen && (
        <CodeGenerator
          tableName={tableName}
          tableDesc={tableDesc}
          mode={mode}
          pkValue={pkValue}
          skOp={skOp}
          skValue={skValue}
          skValue2={skValue2}
          indexName={indexName}
          filters={filters}
          limit={limit}
          onClose={() => setShowCodeGen(false)}
          showToast={showToast}
        />
      )}

      {results && (
        <div style="margin-top:16px">
          <div class="flex items-center gap-2 mb-4">
            <span style="font-weight:600">{results.length} results</span>
            {scannedCount > 0 && <span class="text-sm text-muted">({scannedCount} scanned)</span>}
          </div>
          {results.length > 0 ? (
            <div class="ddb-table-wrap">
              <table class="ddb-items-table">
                <thead>
                  <tr>
                    {resultColumns.map(c => <th key={c}>{c}</th>)}
                  </tr>
                </thead>
                <tbody>
                  {results.map((item, idx) => (
                    <tr key={idx} class="ddb-item-row" style="cursor:pointer" onClick={() => onEditItem(item)}>
                      {resultColumns.map(c => {
                        const val = item[c];
                        const t = getType(val);
                        const tc = typeBadgeColor(t);
                        return (
                          <td key={c} class="ddb-cell" style="max-width:250px">
                            <div class="ddb-cell-content">
                              {val && <span class="ddb-type-badge" style={`background:${tc.bg};color:${tc.fg}`}>{t}</span>}
                              <span class="ddb-cell-value truncate">{extractValue(val)}</span>
                            </div>
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div class="text-muted text-sm" style="text-align:center;padding:32px">No matching items</div>
          )}
        </div>
      )}
    </div>
  );
}
