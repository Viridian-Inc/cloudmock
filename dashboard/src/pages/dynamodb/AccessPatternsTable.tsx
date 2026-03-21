import { useState, useMemo, useCallback, useRef, useEffect } from 'preact/hooks';
import { ddbRequest } from '../../api';
import { Modal } from '../../components/Modal';
import { PlusIcon, RefreshIcon } from '../../components/Icons';
import { TableDescription, DDBItem } from './types';
import { CodeGenerator } from './CodeGenerator';

// --- Types ---

interface QueryPattern {
  id: string;
  pkValue: string;
  skCondition: '=' | 'begins_with' | 'between' | '>' | '<';
  skValue: string;
  skValue2: string; // for "between"
}

interface IndexInfo {
  name: string;
  displayName: string;
  type: 'Table' | 'GSI' | 'LSI';
  pk: string;
  pkType: string;
  sk: string;
  skType: string;
}

interface CellResult {
  matches: boolean;
  count: number;
  items: DDBItem[];
  queried: boolean;
}

interface AccessPatternsTableProps {
  tableDesc: TableDescription;
  tableName: string;
  showToast: (msg: string) => void;
}

// --- Helpers ---

const SK_CONDITIONS: { value: QueryPattern['skCondition']; label: string }[] = [
  { value: '=', label: '= (equals)' },
  { value: 'begins_with', label: 'begins_with' },
  { value: 'between', label: 'between' },
  { value: '>', label: '> (greater than)' },
  { value: '<', label: '< (less than)' },
];

function makePatternLabel(p: QueryPattern, pkAttr: string, skAttr: string): { pk: string; sk: string } {
  const pk = `${pkAttr}="${p.pkValue}"`;
  let sk = '';
  if (p.skValue && skAttr) {
    if (p.skCondition === 'begins_with') {
      sk = `${skAttr} begins "${p.skValue}"`;
    } else if (p.skCondition === 'between') {
      sk = `${skAttr} between "${p.skValue}" and "${p.skValue2}"`;
    } else {
      sk = `${skAttr} ${p.skCondition} "${p.skValue}"`;
    }
  }
  return { pk, sk };
}

function doesPatternMatchIndex(pattern: QueryPattern, index: IndexInfo): boolean {
  // A pattern matches an index if the pattern's PK attribute concept could
  // be queried on that index. We check: does the pattern supply a PK value
  // that semantically targets this index's PK? Since patterns are user-defined
  // with explicit PK values, we match if the user said this pattern's pkValue
  // is for the *same attribute* as the index PK. But since we have column-level
  // patterns, each pattern specifies which attribute it targets. For simplicity
  // in this matrix: a pattern is "compatible" with an index if its pkValue is
  // non-empty (always true in our case) -- the USER decides intent.
  //
  // Real logic: the pattern is compatible if the PK attribute name referenced
  // matches the index's PK attribute. We store the target PK attr on the pattern.
  return true; // Compatibility is computed via `patternTargetAttr`
}

function buildQueryApiCall(
  tableName: string,
  index: IndexInfo,
  pattern: QueryPattern,
): string {
  const params: any = {
    TableName: tableName,
    KeyConditionExpression: `#pk = :pk`,
    ExpressionAttributeNames: { '#pk': index.pk },
    ExpressionAttributeValues: { ':pk': { S: pattern.pkValue } },
  };
  if (index.type !== 'Table') {
    params.IndexName = index.name;
  }
  if (pattern.skValue && index.sk) {
    params.ExpressionAttributeNames['#sk'] = index.sk;
    if (pattern.skCondition === '=') {
      params.KeyConditionExpression += ' AND #sk = :sk';
      params.ExpressionAttributeValues[':sk'] = { S: pattern.skValue };
    } else if (pattern.skCondition === 'begins_with') {
      params.KeyConditionExpression += ' AND begins_with(#sk, :sk)';
      params.ExpressionAttributeValues[':sk'] = { S: pattern.skValue };
    } else if (pattern.skCondition === 'between') {
      params.KeyConditionExpression += ' AND #sk BETWEEN :sk1 AND :sk2';
      params.ExpressionAttributeValues[':sk1'] = { S: pattern.skValue };
      params.ExpressionAttributeValues[':sk2'] = { S: pattern.skValue2 };
    } else if (pattern.skCondition === '>') {
      params.KeyConditionExpression += ' AND #sk > :sk';
      params.ExpressionAttributeValues[':sk'] = { S: pattern.skValue };
    } else if (pattern.skCondition === '<') {
      params.KeyConditionExpression += ' AND #sk < :sk';
      params.ExpressionAttributeValues[':sk'] = { S: pattern.skValue };
    }
  }
  return JSON.stringify(params, null, 2);
}

// --- Storage ---

function storageKey(tableName: string): string {
  return `ap-patterns-${tableName}`;
}

function loadPatterns(tableName: string): QueryPattern[] {
  try {
    const raw = localStorage.getItem(storageKey(tableName));
    return raw ? JSON.parse(raw) : [];
  } catch { return []; }
}

function savePatterns(tableName: string, patterns: QueryPattern[]) {
  localStorage.setItem(storageKey(tableName), JSON.stringify(patterns));
}

// --- Component ---

export function AccessPatternsTable({ tableDesc, tableName, showToast }: AccessPatternsTableProps) {
  const [patterns, setPatterns] = useState<QueryPattern[]>(() => loadPatterns(tableName));
  const [showAddModal, setShowAddModal] = useState(false);
  const [results, setResults] = useState<Record<string, CellResult>>({});
  const [activeCell, setActiveCell] = useState<{ indexName: string; patternId: string } | null>(null);
  const [tooltip, setTooltip] = useState<{ x: number; y: number; text: string } | null>(null);
  const [showCodeGen, setShowCodeGen] = useState(false);
  const [codeGenParams, setCodeGenParams] = useState<any>(null);
  const [suggesting, setSuggesting] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);

  // Build indexes list
  const indexes = useMemo((): IndexInfo[] => {
    const list: IndexInfo[] = [];
    const pkAttr = tableDesc.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '';
    const skAttr = tableDesc.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '';
    list.push({
      name: '__table__',
      displayName: 'Table',
      type: 'Table',
      pk: pkAttr,
      pkType: tableDesc.AttributeDefinitions.find(a => a.AttributeName === pkAttr)?.AttributeType || 'S',
      sk: skAttr,
      skType: skAttr ? (tableDesc.AttributeDefinitions.find(a => a.AttributeName === skAttr)?.AttributeType || 'S') : '',
    });
    if (tableDesc.GlobalSecondaryIndexes) {
      for (const gsi of tableDesc.GlobalSecondaryIndexes) {
        const gPk = gsi.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '';
        const gSk = gsi.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '';
        list.push({
          name: gsi.IndexName,
          displayName: gsi.IndexName,
          type: 'GSI',
          pk: gPk,
          pkType: tableDesc.AttributeDefinitions.find(a => a.AttributeName === gPk)?.AttributeType || 'S',
          sk: gSk,
          skType: gSk ? (tableDesc.AttributeDefinitions.find(a => a.AttributeName === gSk)?.AttributeType || 'S') : '',
        });
      }
    }
    if (tableDesc.LocalSecondaryIndexes) {
      for (const lsi of tableDesc.LocalSecondaryIndexes) {
        const lPk = lsi.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '';
        const lSk = lsi.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '';
        list.push({
          name: lsi.IndexName,
          displayName: lsi.IndexName,
          type: 'LSI',
          pk: lPk,
          pkType: tableDesc.AttributeDefinitions.find(a => a.AttributeName === lPk)?.AttributeType || 'S',
          sk: lSk,
          skType: lSk ? (tableDesc.AttributeDefinitions.find(a => a.AttributeName === lSk)?.AttributeType || 'S') : '',
        });
      }
    }
    return list;
  }, [tableDesc]);

  // Check if a pattern is compatible with an index (PK attr matches)
  const isCompatible = useCallback((pattern: QueryPattern, index: IndexInfo): boolean => {
    // A pattern implicitly targets all indexes. Compatibility is:
    // The pattern's pkValue is meaningful for that index's PK attribute.
    // Since patterns don't carry the target PK attr name, we check if any
    // existing items have this PK value on this index. For the matrix,
    // the simplest approach: a pattern is compatible with an index if the
    // PK attr on the pattern's "parent" matches the index PK.
    // For the general case (user picks PK value freely), we attempt the query.
    // Mark it as "unknown" until queried, but let the user click any cell.

    // If pattern has an SK condition but the index has no SK, it can't match.
    if (pattern.skValue && !index.sk) return false;
    return true;
  }, []);

  // Persist patterns
  useEffect(() => {
    savePatterns(tableName, patterns);
  }, [patterns, tableName]);

  // Run a query for a specific cell
  async function runCellQuery(index: IndexInfo, pattern: QueryPattern) {
    const cellKey = `${index.name}::${pattern.id}`;
    try {
      const params: any = {
        TableName: tableName,
        KeyConditionExpression: '#pk = :pk',
        ExpressionAttributeNames: { '#pk': index.pk } as Record<string, string>,
        ExpressionAttributeValues: { ':pk': { [index.pkType]: pattern.pkValue } } as Record<string, any>,
        Limit: 50,
      };
      if (index.type !== 'Table') {
        params.IndexName = index.name;
      }
      if (pattern.skValue && index.sk) {
        params.ExpressionAttributeNames['#sk'] = index.sk;
        const skTypeLetter = index.skType || 'S';
        if (pattern.skCondition === '=') {
          params.KeyConditionExpression += ' AND #sk = :sk';
          params.ExpressionAttributeValues[':sk'] = { [skTypeLetter]: pattern.skValue };
        } else if (pattern.skCondition === 'begins_with') {
          params.KeyConditionExpression += ' AND begins_with(#sk, :sk)';
          params.ExpressionAttributeValues[':sk'] = { [skTypeLetter]: pattern.skValue };
        } else if (pattern.skCondition === 'between') {
          params.KeyConditionExpression += ' AND #sk BETWEEN :sk1 AND :sk2';
          params.ExpressionAttributeValues[':sk1'] = { [skTypeLetter]: pattern.skValue };
          params.ExpressionAttributeValues[':sk2'] = { [skTypeLetter]: pattern.skValue2 };
        } else if (pattern.skCondition === '>') {
          params.KeyConditionExpression += ' AND #sk > :sk';
          params.ExpressionAttributeValues[':sk'] = { [skTypeLetter]: pattern.skValue };
        } else if (pattern.skCondition === '<') {
          params.KeyConditionExpression += ' AND #sk < :sk';
          params.ExpressionAttributeValues[':sk'] = { [skTypeLetter]: pattern.skValue };
        }
      }
      const r = await ddbRequest('Query', params);
      const items = r.Items || [];
      setResults(prev => ({
        ...prev,
        [cellKey]: { matches: items.length > 0, count: r.Count ?? items.length, items, queried: true },
      }));
    } catch {
      setResults(prev => ({
        ...prev,
        [cellKey]: { matches: false, count: 0, items: [], queried: true },
      }));
    }
  }

  function handleCellClick(index: IndexInfo, pattern: QueryPattern) {
    const cellKey = `${index.name}::${pattern.id}`;
    setActiveCell({ indexName: index.name, patternId: pattern.id });
    if (!results[cellKey]?.queried) {
      runCellQuery(index, pattern);
    }
  }

  function handleCellHover(e: MouseEvent, index: IndexInfo, pattern: QueryPattern) {
    const text = buildQueryApiCall(tableName, index, pattern);
    const rect = (e.target as HTMLElement).getBoundingClientRect();
    const wrapRect = wrapRef.current?.getBoundingClientRect();
    if (wrapRect) {
      setTooltip({
        x: rect.left - wrapRect.left + rect.width / 2,
        y: rect.top - wrapRect.top - 8,
        text: `DynamoDB Query:\n${text}`,
      });
    }
  }

  function handleAddPattern(pattern: QueryPattern) {
    setPatterns(prev => [...prev, pattern]);
    setShowAddModal(false);
  }

  function removePattern(id: string) {
    setPatterns(prev => prev.filter(p => p.id !== id));
    // Clean up results
    setResults(prev => {
      const next = { ...prev };
      for (const key of Object.keys(next)) {
        if (key.endsWith(`::${id}`)) delete next[key];
      }
      return next;
    });
  }

  // Auto-suggest patterns from existing items
  async function suggestPatterns() {
    setSuggesting(true);
    try {
      const r = await ddbRequest('Scan', { TableName: tableName, Limit: 10 });
      const items: DDBItem[] = r.Items || [];
      if (items.length === 0) {
        showToast('No items found to suggest patterns');
        setSuggesting(false);
        return;
      }
      const primaryPk = indexes[0].pk;
      const primarySk = indexes[0].sk;
      const seen = new Set<string>();
      const newPatterns: QueryPattern[] = [];

      for (const item of items) {
        const pkVal = item[primaryPk];
        const pkStr = pkVal?.S || pkVal?.N || '';
        if (!pkStr || seen.has(pkStr)) continue;
        seen.add(pkStr);

        const skVal = primarySk ? (item[primarySk]?.S || item[primarySk]?.N || '') : '';
        newPatterns.push({
          id: `auto-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
          pkValue: pkStr,
          skCondition: '=',
          skValue: skVal,
          skValue2: '',
        });
        if (newPatterns.length >= 5) break;
      }

      if (newPatterns.length > 0) {
        setPatterns(prev => [...prev, ...newPatterns]);
        showToast(`Added ${newPatterns.length} suggested patterns`);
      }
    } catch {
      showToast('Failed to scan items for suggestions');
    }
    setSuggesting(false);
  }

  function handleExportQuery(index: IndexInfo, pattern: QueryPattern) {
    setCodeGenParams({
      mode: 'query',
      pkValue: pattern.pkValue,
      skOp: pattern.skCondition,
      skValue: pattern.skValue,
      skValue2: pattern.skValue2,
      indexName: index.type === 'Table' ? '' : index.name,
    });
    setShowCodeGen(true);
  }

  // Active cell results
  const activeCellResult = activeCell
    ? results[`${activeCell.indexName}::${activeCell.patternId}`]
    : null;

  const activeIndex = activeCell ? indexes.find(i => i.name === activeCell.indexName) : null;
  const activePattern = activeCell ? patterns.find(p => p.id === activeCell.patternId) : null;

  if (patterns.length === 0 && !showAddModal) {
    return (
      <div style="margin-bottom:16px">
        <div style="display:flex;align-items:center;gap:8px;margin-bottom:8px">
          <span style="font-size:14px;font-weight:700;color:var(--n800)">Access Patterns Matrix</span>
        </div>
        <div style="padding:24px;text-align:center;background:var(--n50);border:1px dashed var(--n300);border-radius:var(--radius-md)">
          <div style="font-size:13px;color:var(--n500);margin-bottom:12px">
            No access patterns defined yet. Add patterns to visualize which queries work on which indexes.
          </div>
          <div style="display:flex;gap:8px;justify-content:center">
            <button class="btn btn-secondary btn-sm" onClick={() => setShowAddModal(true)}>
              <PlusIcon /> Add Pattern
            </button>
            <button class="btn btn-ghost btn-sm" onClick={suggestPatterns} disabled={suggesting}>
              {suggesting ? 'Scanning...' : 'Suggest Patterns'}
            </button>
          </div>
        </div>

        {showAddModal && (
          <AddPatternModal
            indexes={indexes}
            onAdd={handleAddPattern}
            onClose={() => setShowAddModal(false)}
          />
        )}
      </div>
    );
  }

  return (
    <div style="margin-bottom:16px;position:relative" ref={wrapRef}>
      <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:8px">
        <span style="font-size:14px;font-weight:700;color:var(--n800)">Access Patterns Matrix</span>
        <div style="display:flex;gap:6px">
          <button class="btn btn-ghost btn-sm" onClick={suggestPatterns} disabled={suggesting}>
            {suggesting ? 'Scanning...' : 'Suggest'}
          </button>
          <button class="ap-add-pattern-btn" onClick={() => setShowAddModal(true)}>
            <PlusIcon /> Add Pattern
          </button>
        </div>
      </div>

      {/* Matrix table */}
      <div class="ap-matrix-wrap">
        <table class="ap-matrix">
          <thead>
            <tr>
              <th class="ap-corner">Index</th>
              {patterns.map(p => {
                const label = makePatternLabel(p, indexes[0].pk, indexes[0].sk);
                return (
                  <th key={p.id}>
                    <div style="display:flex;flex-direction:column;gap:2px;min-width:100px">
                      <span class="font-mono" style="font-size:11px">{label.pk}</span>
                      {label.sk && <span class="font-mono" style="font-size:10px;color:var(--n400)">{label.sk}</span>}
                      <button
                        style="background:none;border:none;color:var(--error);cursor:pointer;font-size:10px;align-self:flex-end;padding:0"
                        onClick={() => removePattern(p.id)}
                        title="Remove pattern"
                      >remove</button>
                    </div>
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody>
            {indexes.map(index => (
              <tr key={index.name}>
                <td class="ap-index-cell">
                  <div>
                    <div style="display:flex;align-items:center;gap:6px">
                      <span class="font-mono" style="font-weight:700">{index.displayName}</span>
                      <span class={`ddb-ap-badge ddb-ap-badge-${index.type.toLowerCase()}`}>{index.type}</span>
                    </div>
                    <div style="font-size:10px;color:var(--n400);margin-top:2px">
                      PK:{index.pk}{index.sk ? ` SK:${index.sk}` : ''}
                    </div>
                  </div>
                </td>
                {patterns.map(pattern => {
                  const cellKey = `${index.name}::${pattern.id}`;
                  const result = results[cellKey];
                  const compatible = isCompatible(pattern, index);
                  const isActive = activeCell?.indexName === index.name && activeCell?.patternId === pattern.id;

                  return (
                    <td
                      key={pattern.id}
                      class={compatible ? 'ap-cell-match' : 'ap-cell-miss'}
                      style={isActive ? 'background:rgba(9,127,245,0.08);outline:2px solid var(--brand-blue)' : ''}
                      onClick={() => compatible && handleCellClick(index, pattern)}
                      onMouseEnter={(e) => compatible && handleCellHover(e as any, index, pattern)}
                      onMouseLeave={() => setTooltip(null)}
                    >
                      {!compatible ? (
                        <span class="ap-cell-cross" title="Pattern SK not supported on this index">&#10007;</span>
                      ) : !result?.queried ? (
                        <span style="color:var(--n400);font-size:12px;cursor:pointer" title="Click to run query">? click</span>
                      ) : result.matches ? (
                        <span class="ap-cell-check">&#10003; {result.count} item{result.count !== 1 ? 's' : ''}</span>
                      ) : (
                        <span class="ap-cell-cross">&#10007; 0</span>
                      )}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Tooltip */}
      {tooltip && (
        <div class="ap-tooltip" style={`left:${tooltip.x}px;top:${tooltip.y}px;transform:translate(-50%,-100%)`}>
          {tooltip.text}
        </div>
      )}

      {/* Results panel */}
      {activeCell && activeCellResult?.queried && (
        <div class="ap-results-panel">
          <div class="ap-results-header">
            <span>
              Results: {activeCellResult.count} item{activeCellResult.count !== 1 ? 's' : ''}
              {activeIndex && activePattern && (
                <span style="color:var(--n400);margin-left:8px;font-weight:400">
                  on {activeIndex.displayName}
                </span>
              )}
            </span>
            <div style="display:flex;gap:6px">
              {activeIndex && activePattern && (
                <button
                  class="btn btn-ghost btn-sm"
                  onClick={() => handleExportQuery(activeIndex, activePattern)}
                >
                  Export Query
                </button>
              )}
              <button class="btn btn-ghost btn-sm" onClick={() => setActiveCell(null)}>Close</button>
            </div>
          </div>
          <div class="ap-results-body">
            {activeCellResult.items.length > 0 ? (
              <div class="json-view" style="max-height:280px;border-radius:0">
                {JSON.stringify(activeCellResult.items, null, 2)}
              </div>
            ) : (
              <div style="padding:16px;text-align:center;color:var(--n400);font-size:13px">
                No matching items
              </div>
            )}
          </div>
        </div>
      )}

      {/* Add pattern modal */}
      {showAddModal && (
        <AddPatternModal
          indexes={indexes}
          onAdd={handleAddPattern}
          onClose={() => setShowAddModal(false)}
        />
      )}

      {/* Code generator modal */}
      {showCodeGen && codeGenParams && (
        <CodeGenerator
          tableName={tableName}
          tableDesc={tableDesc}
          mode="query"
          pkValue={codeGenParams.pkValue}
          skOp={codeGenParams.skOp}
          skValue={codeGenParams.skValue}
          skValue2={codeGenParams.skValue2}
          indexName={codeGenParams.indexName}
          filters={[]}
          limit=""
          onClose={() => setShowCodeGen(false)}
          showToast={showToast}
        />
      )}
    </div>
  );
}

// --- Add Pattern Modal ---

function AddPatternModal({
  indexes,
  onAdd,
  onClose,
}: {
  indexes: IndexInfo[];
  onAdd: (p: QueryPattern) => void;
  onClose: () => void;
}) {
  const [pkValue, setPkValue] = useState('');
  const [skCondition, setSkCondition] = useState<QueryPattern['skCondition']>('=');
  const [skValue, setSkValue] = useState('');
  const [skValue2, setSkValue2] = useState('');

  const primaryIndex = indexes[0];

  function handleSubmit(e: Event) {
    e.preventDefault();
    if (!pkValue.trim()) return;
    onAdd({
      id: `p-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
      pkValue: pkValue.trim(),
      skCondition,
      skValue: skValue.trim(),
      skValue2: skValue2.trim(),
    });
  }

  return (
    <Modal title="Add Query Pattern" size="md" onClose={onClose}>
      <form onSubmit={handleSubmit}>
        <div style="margin-bottom:12px">
          <label class="label">
            Partition Key ({primaryIndex.pk}) <span class="ddb-ap-type">{primaryIndex.pkType}</span>
          </label>
          <input
            class="input w-full"
            placeholder={`Enter ${primaryIndex.pk} value`}
            value={pkValue}
            onInput={(e) => setPkValue((e.target as HTMLInputElement).value)}
            autoFocus
          />
        </div>

        {primaryIndex.sk && (
          <>
            <div style="margin-bottom:12px">
              <label class="label">Sort Key Condition ({primaryIndex.sk})</label>
              <select
                class="select w-full"
                value={skCondition}
                onChange={(e) => setSkCondition((e.target as HTMLSelectElement).value as any)}
              >
                {SK_CONDITIONS.map(c => (
                  <option key={c.value} value={c.value}>{c.label}</option>
                ))}
              </select>
            </div>
            <div style="margin-bottom:12px">
              <label class="label">Sort Key Value</label>
              <input
                class="input w-full"
                placeholder={`Enter ${primaryIndex.sk} value`}
                value={skValue}
                onInput={(e) => setSkValue((e.target as HTMLInputElement).value)}
              />
            </div>
            {skCondition === 'between' && (
              <div style="margin-bottom:12px">
                <label class="label">Sort Key Value 2 (upper bound)</label>
                <input
                  class="input w-full"
                  placeholder="Upper bound value"
                  value={skValue2}
                  onInput={(e) => setSkValue2((e.target as HTMLInputElement).value)}
                />
              </div>
            )}
          </>
        )}

        <div style="display:flex;justify-content:flex-end;gap:8px;margin-top:16px">
          <button type="button" class="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
          <button type="submit" class="btn btn-primary btn-sm" disabled={!pkValue.trim()}>
            Add Pattern
          </button>
        </div>
      </form>
    </Modal>
  );
}
