import { useState, useMemo, useContext } from 'preact/hooks';
import { DDBItem, TableDescription } from './types';
import { extractValue, getType, typeBadgeColor, collectColumns, ddbRequest } from './utils';
import { PlayIcon } from '../../components/icons';
import { DDBContext } from './store';

interface PartiQLProps {
  tableName: string;
  tableDesc: TableDescription;
  showToast: (msg: string) => void;
  onEditItem: (item: DDBItem) => void;
  tabIndex: number;
}

const SQL_EXAMPLES = [
  { label: 'Select all', sql: (t: string) => `SELECT * FROM "${t}"` },
  { label: 'Select by PK', sql: (t: string, pk: string) => `SELECT * FROM "${t}" WHERE ${pk} = 'value'` },
  { label: 'Select columns', sql: (t: string) => `SELECT pk, name FROM "${t}"` },
  { label: 'PK + SK prefix', sql: (t: string, pk: string, sk: string) => sk ? `SELECT * FROM "${t}" WHERE ${pk} = 'value' AND begins_with(${sk}, 'prefix')` : `SELECT * FROM "${t}" WHERE ${pk} = 'value'` },
];

const SQL_KEYWORDS = /\b(SELECT|FROM|WHERE|AND|OR|INSERT|INTO|VALUES|UPDATE|SET|DELETE|BEGINS_WITH|BETWEEN|IN|NOT|NULL|EXISTS|LIMIT)\b/gi;

interface ParsedCondition {
  attr: string;
  op: string;
  value: string;
  value2?: string;
}

function parseSQL(sql: string, tableDesc: TableDescription): { action: string; params: any } | null {
  const trimmed = sql.trim().replace(/;$/, '');
  const upper = trimmed.toUpperCase();
  if (upper.startsWith('SELECT')) return parseSelect(trimmed, tableDesc);
  return null;
}

function parseSelect(sql: string, tableDesc: TableDescription): { action: string; params: any } | null {
  const match = sql.match(/^SELECT\s+(.+?)\s+FROM\s+"?([^"]+)"?\s*(WHERE\s+(.+))?$/i);
  if (!match) return null;

  const columns = match[1].trim();
  const tableName = match[2].trim();
  const whereClause = match[4]?.trim();

  const pkAttr = tableDesc.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '';
  const skAttr = tableDesc.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '';
  const pkType = tableDesc.AttributeDefinitions.find(a => a.AttributeName === pkAttr)?.AttributeType || 'S';
  const skType = tableDesc.AttributeDefinitions.find(a => a.AttributeName === skAttr)?.AttributeType || 'S';

  const params: any = { TableName: tableName };

  if (columns !== '*') {
    const cols = columns.split(',').map(c => c.trim());
    const exprNames: Record<string, string> = {};
    const projParts: string[] = [];
    cols.forEach((c, i) => {
      const key = `#p${i}`;
      exprNames[key] = c;
      projParts.push(key);
    });
    params.ProjectionExpression = projParts.join(', ');
    params.ExpressionAttributeNames = { ...(params.ExpressionAttributeNames || {}), ...exprNames };
  }

  if (!whereClause) return { action: 'Scan', params };

  const conditions = parseWhereConditions(whereClause);
  if (!conditions || conditions.length === 0) return { action: 'Scan', params };

  const pkCondition = conditions.find(c => c.attr === pkAttr && c.op === '=');

  if (pkCondition) {
    const exprNames: Record<string, string> = { ...(params.ExpressionAttributeNames || {}), '#pk': pkAttr };
    const exprValues: Record<string, any> = { ':pkv': pkType === 'N' ? { N: pkCondition.value } : { S: pkCondition.value } };
    let keyExpr = '#pk = :pkv';

    const skCondition = conditions.find(c => c.attr === skAttr);
    if (skCondition && skAttr) {
      exprNames['#sk'] = skAttr;
      const skTyped = skType === 'N' ? { N: skCondition.value } : { S: skCondition.value };
      exprValues[':skv'] = skTyped;
      if (skCondition.op === 'begins_with') {
        keyExpr += ' AND begins_with(#sk, :skv)';
      } else if (skCondition.op === 'between' && skCondition.value2) {
        exprValues[':skv2'] = skType === 'N' ? { N: skCondition.value2 } : { S: skCondition.value2 };
        keyExpr += ' AND #sk BETWEEN :skv AND :skv2';
      } else {
        keyExpr += ` AND #sk ${skCondition.op} :skv`;
      }
    }

    const filterConditions = conditions.filter(c => c.attr !== pkAttr && c.attr !== skAttr);
    if (filterConditions.length > 0) {
      const { expr, names, values } = buildFilterFromConditions(filterConditions);
      params.FilterExpression = expr;
      Object.assign(exprNames, names);
      Object.assign(exprValues, values);
    }

    params.KeyConditionExpression = keyExpr;
    params.ExpressionAttributeNames = exprNames;
    params.ExpressionAttributeValues = exprValues;
    return { action: 'Query', params };
  }

  const { expr, names, values } = buildFilterFromConditions(conditions);
  params.FilterExpression = expr;
  params.ExpressionAttributeNames = { ...(params.ExpressionAttributeNames || {}), ...names };
  params.ExpressionAttributeValues = values;
  return { action: 'Scan', params };
}

function parseWhereConditions(where: string): ParsedCondition[] {
  const conditions: ParsedCondition[] = [];
  const parts = where.split(/\s+AND\s+/i);

  for (const part of parts) {
    const trimmed = part.trim();

    const bwMatch = trimmed.match(/^begins_with\s*\(\s*(\w+)\s*,\s*'([^']*)'\s*\)$/i);
    if (bwMatch) { conditions.push({ attr: bwMatch[1], op: 'begins_with', value: bwMatch[2] }); continue; }

    const betweenMatch = trimmed.match(/^(\w+)\s+BETWEEN\s+'([^']*)'\s+AND\s+'([^']*)'/i);
    if (betweenMatch) { conditions.push({ attr: betweenMatch[1], op: 'between', value: betweenMatch[2], value2: betweenMatch[3] }); continue; }

    const opMatch = trimmed.match(/^(\w+)\s*(=|!=|<>|<=|>=|<|>)\s*'?([^']*?)'?$/);
    if (opMatch) { conditions.push({ attr: opMatch[1], op: opMatch[2], value: opMatch[3] }); continue; }
  }

  return conditions;
}

function buildFilterFromConditions(conditions: ParsedCondition[]): { expr: string; names: Record<string, string>; values: Record<string, any> } {
  const names: Record<string, string> = {};
  const values: Record<string, any> = {};
  const parts: string[] = [];

  conditions.forEach((c, i) => {
    const nk = `#fc${i}`;
    const vk = `:fv${i}`;
    names[nk] = c.attr;
    values[vk] = { S: c.value };

    if (i > 0) parts.push('AND');
    if (c.op === 'begins_with') {
      parts.push(`begins_with(${nk}, ${vk})`);
    } else if (c.op === 'between') {
      values[`${vk}b`] = { S: c.value2 || '' };
      parts.push(`${nk} BETWEEN ${vk} AND ${vk}b`);
    } else {
      parts.push(`${nk} ${c.op} ${vk}`);
    }
  });

  return { expr: parts.join(' '), names, values };
}

export function PartiQL({ tableName, tableDesc, showToast, onEditItem, tabIndex }: PartiQLProps) {
  const { state, dispatch } = useContext(DDBContext);
  const tab = state.tabs[tabIndex];

  const pkAttr = tableDesc.KeySchema.find(k => k.KeyType === 'HASH')?.AttributeName || '';
  const skAttr = tableDesc.KeySchema.find(k => k.KeyType === 'RANGE')?.AttributeName || '';

  const sql = tab?.sqlQuery ?? `SELECT * FROM "${tableName}"`;
  const results = tab?.sqlResults ?? null;

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [showExamples, setShowExamples] = useState(false);

  const resultColumns = useMemo(() => {
    if (!results || results.length === 0) return [];
    const keyAttrs = tableDesc.KeySchema.map(k => k.AttributeName);
    return collectColumns(results, keyAttrs);
  }, [results, tableDesc]);

  function setSQL(value: string) {
    dispatch({ type: 'UPDATE_TAB', index: tabIndex, patch: { sqlQuery: value } });
  }

  async function runSQL() {
    setLoading(true);
    setError('');
    try {
      const parsed = parseSQL(sql, tableDesc);
      if (!parsed) {
        setError('Could not parse SQL statement. Only SELECT queries are currently supported.');
        return;
      }
      const r = await ddbRequest(parsed.action, parsed.params);
      dispatch({ type: 'SET_SQL_RESULTS', index: tabIndex, results: r.Items || [] });
      showToast(`${parsed.action}: ${(r.Items || []).length} results`);
    } catch (e: any) {
      setError(e.message || 'Query execution failed');
    } finally {
      setLoading(false);
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault();
      runSQL();
    }
  }

  return (
    <div>
      <div class="ddb-query-card">
        <div class="flex items-center justify-between mb-4">
          <div class="label" style="margin-bottom:0">PartiQL / SQL Query</div>
          <div class="flex gap-2">
            <button class="btn btn-ghost btn-sm" onClick={() => setShowExamples(!showExamples)}>
              Examples
            </button>
          </div>
        </div>

        {showExamples && (
          <div class="ddb-partiql-examples mb-4">
            {SQL_EXAMPLES.map((ex, i) => (
              <div
                key={i}
                class="ddb-partiql-example"
                onClick={() => { setSQL(ex.sql(tableName, pkAttr, skAttr)); setShowExamples(false); }}
              >
                <span class="text-sm" style="font-weight:600">{ex.label}</span>
                <span class="font-mono text-sm text-muted">{ex.sql(tableName, pkAttr, skAttr)}</span>
              </div>
            ))}
          </div>
        )}

        <div class="ddb-partiql-editor-wrap">
          <textarea
            class="ddb-partiql-input"
            value={sql}
            onInput={(e) => setSQL((e.target as HTMLTextAreaElement).value)}
            onKeyDown={handleKeyDown}
            spellcheck={false}
            rows={4}
            placeholder="Enter SQL query..."
          />
        </div>

        {error && (
          <div style="color:var(--error);font-size:13px;margin-top:8px">{error}</div>
        )}

        <div class="flex items-center gap-2 mt-4">
          <button
            class="btn btn-primary btn-sm"
            onClick={runSQL}
            disabled={loading || !sql.trim()}
          >
            <PlayIcon /> {loading ? 'Running...' : 'Run SQL'}
          </button>
          <span class="text-sm text-muted">Ctrl+Enter to run</span>
        </div>
      </div>

      {results && (
        <div style="margin-top:16px">
          <div class="mb-4">
            <span style="font-weight:600">{results.length} results</span>
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
