import { useState, useEffect, useMemo } from 'preact/hooks';
import { ddbRequest } from '../api';
import { JsonView } from '../components/JsonView';
import { Modal } from '../components/Modal';
import { DatabaseIcon, PlusIcon, RefreshIcon, TrashIcon, ExpandIcon, PlayIcon } from '../components/Icons';
import { copyToClipboard } from '../utils';

interface DynamoDBPageProps {
  showToast: (msg: string) => void;
}

const PAGE_SIZE = 25;

export function DynamoDBPage({ showToast }: DynamoDBPageProps) {
  const [tables, setTables] = useState<string[]>([]);
  const [selectedTable, setSelectedTable] = useState<string | null>(null);
  const [tableDesc, setTableDesc] = useState<any>(null);
  const [items, setItems] = useState<any[]>([]);
  const [page, setPage] = useState(0);
  const [lastKeys, setLastKeys] = useState<any[]>([]);
  const [tableSearch, setTableSearch] = useState('');
  const [activeTab, setActiveTab] = useState('browse');
  const [editModal, setEditModal] = useState<any>(null);
  const [createModal, setCreateModal] = useState(false);
  const [createTableModal, setCreateTableModal] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [queryMode, setQueryMode] = useState('scan');
  const [queryExpr, setQueryExpr] = useState('');
  const [filterExpr, setFilterExpr] = useState('');
  const [exprAttrValues, setExprAttrValues] = useState('{}');
  const [queryResults, setQueryResults] = useState<any[] | null>(null);
  const [newTableName, setNewTableName] = useState('');
  const [newTablePK, setNewTablePK] = useState('id');
  const [newTablePKType, setNewTablePKType] = useState('S');
  const [newTableSK, setNewTableSK] = useState('');
  const [newTableSKType, setNewTableSKType] = useState('S');
  const [itemJson, setItemJson] = useState('{}');

  function loadTables() {
    ddbRequest('ListTables', {}).then((r: any) => {
      setTables(r.TableNames || []);
    }).catch(() => {});
  }

  useEffect(() => { loadTables(); }, []);

  function selectTable(name: string) {
    setSelectedTable(name);
    setPage(0);
    setLastKeys([]);
    setActiveTab('browse');
    ddbRequest('DescribeTable', { TableName: name }).then((r: any) => {
      setTableDesc(r.Table);
    }).catch(() => {});
    scanItems(name, null);
  }

  function scanItems(tableName: string, exclusiveStartKey: any) {
    const params: any = { TableName: tableName, Limit: PAGE_SIZE };
    if (exclusiveStartKey) params.ExclusiveStartKey = exclusiveStartKey;
    ddbRequest('Scan', params).then((r: any) => {
      setItems(r.Items || []);
      if (r.LastEvaluatedKey) {
        setLastKeys(prev => [...prev, r.LastEvaluatedKey]);
      }
    }).catch(() => setItems([]));
  }

  function nextPage() {
    const lastKey = lastKeys[page];
    if (!lastKey) return;
    setPage(p => p + 1);
    scanItems(selectedTable!, lastKey);
  }

  function prevPage() {
    if (page <= 0) return;
    const newPage = page - 1;
    setPage(newPage);
    if (newPage === 0) {
      scanItems(selectedTable!, null);
    } else {
      scanItems(selectedTable!, lastKeys[newPage - 1]);
    }
  }

  function deleteItem(item: any) {
    if (!tableDesc) return;
    const key: any = {};
    tableDesc.KeySchema.forEach((k: any) => {
      key[k.AttributeName] = item[k.AttributeName];
    });
    ddbRequest('DeleteItem', { TableName: selectedTable, Key: key }).then(() => {
      showToast('Item deleted');
      scanItems(selectedTable!, null);
      setPage(0);
      setLastKeys([]);
      setEditModal(null);
    }).catch(() => showToast('Delete failed'));
  }

  function saveItem(jsonStr: string) {
    try {
      const item = JSON.parse(jsonStr);
      ddbRequest('PutItem', { TableName: selectedTable, Item: item }).then(() => {
        showToast('Item saved');
        scanItems(selectedTable!, null);
        setPage(0);
        setLastKeys([]);
        setEditModal(null);
        setCreateModal(false);
      }).catch(() => showToast('Save failed'));
    } catch {
      showToast('Invalid JSON');
    }
  }

  function createTable() {
    const params: any = {
      TableName: newTableName,
      KeySchema: [{ AttributeName: newTablePK, KeyType: 'HASH' }],
      AttributeDefinitions: [{ AttributeName: newTablePK, AttributeType: newTablePKType }],
      BillingMode: 'PAY_PER_REQUEST',
    };
    if (newTableSK) {
      params.KeySchema.push({ AttributeName: newTableSK, KeyType: 'RANGE' });
      params.AttributeDefinitions.push({ AttributeName: newTableSK, AttributeType: newTableSKType });
    }
    ddbRequest('CreateTable', params).then(() => {
      showToast('Table created');
      loadTables();
      setCreateTableModal(false);
      setNewTableName('');
      setNewTablePK('id');
      setNewTableSK('');
    }).catch(() => showToast('Create table failed'));
  }

  function deleteTable(name: string) {
    ddbRequest('DeleteTable', { TableName: name }).then(() => {
      showToast('Table deleted');
      loadTables();
      if (selectedTable === name) {
        setSelectedTable(null);
        setItems([]);
        setTableDesc(null);
      }
      setDeleteConfirm(null);
    }).catch(() => showToast('Delete table failed'));
  }

  function runQuery() {
    const params: any = { TableName: selectedTable, Limit: PAGE_SIZE };
    try {
      if (exprAttrValues && exprAttrValues !== '{}') {
        params.ExpressionAttributeValues = JSON.parse(exprAttrValues);
      }
    } catch {
      showToast('Invalid expression attribute values JSON');
      return;
    }
    if (queryMode === 'query') {
      if (!queryExpr) { showToast('Key condition expression required'); return; }
      params.KeyConditionExpression = queryExpr;
      if (filterExpr) params.FilterExpression = filterExpr;
      ddbRequest('Query', params).then((r: any) => {
        setQueryResults(r.Items || []);
      }).catch(() => showToast('Query failed'));
    } else {
      if (filterExpr) params.FilterExpression = filterExpr;
      ddbRequest('Scan', params).then((r: any) => {
        setQueryResults(r.Items || []);
      }).catch(() => showToast('Scan failed'));
    }
  }

  const filteredTables = useMemo(() => {
    if (!tableSearch) return tables;
    const q = tableSearch.toLowerCase();
    return tables.filter(t => t.toLowerCase().includes(q));
  }, [tables, tableSearch]);

  const columns = useMemo(() => {
    if (!items || items.length === 0) return [];
    const cols = new Set<string>();
    items.forEach((item: any) => Object.keys(item).forEach(k => cols.add(k)));
    const keyAttrs = tableDesc ? tableDesc.KeySchema.map((k: any) => k.AttributeName) : [];
    const sorted = [...keyAttrs.filter((k: string) => cols.has(k)), ...[...cols].filter(k => !keyAttrs.includes(k)).sort()];
    return sorted;
  }, [items, tableDesc]);

  function formatDDBValue(val: any): string {
    if (!val) return '';
    if (val.S !== undefined) return val.S;
    if (val.N !== undefined) return val.N;
    if (val.BOOL !== undefined) return String(val.BOOL);
    if (val.NULL) return 'null';
    if (val.L) return JSON.stringify(val.L);
    if (val.M) return JSON.stringify(val.M);
    if (val.SS) return val.SS.join(', ');
    if (val.NS) return val.NS.join(', ');
    return JSON.stringify(val);
  }

  return (
    <div class="ddb-layout">
      <div class="ddb-sidebar">
        <div class="ddb-sidebar-header">
          <div class="flex items-center justify-between mb-4">
            <span style="font-weight:700;font-size:15px">Tables</span>
            <button class="btn btn-primary btn-sm" onClick={() => setCreateTableModal(true)}>
              <PlusIcon /> New
            </button>
          </div>
          <input
            class="input input-search w-full"
            placeholder="Filter tables..."
            value={tableSearch}
            onInput={(e) => setTableSearch((e.target as HTMLInputElement).value)}
            style="height:32px;font-size:13px"
          />
        </div>
        <div class="ddb-sidebar-list">
          {filteredTables.length === 0 ? (
            <div style="padding:24px;text-align:center;color:var(--n400);font-size:13px">No tables found</div>
          ) : filteredTables.map(t => (
            <div class={`ddb-table-item ${selectedTable === t ? 'active' : ''}`} onClick={() => selectTable(t)}>
              <span class="name">{t}</span>
            </div>
          ))}
        </div>
      </div>

      <div class="ddb-main">
        {!selectedTable ? (
          <div class="empty-state">
            <DatabaseIcon width="48" height="48" />
            <div>Select a table to browse items</div>
          </div>
        ) : (
          <div>
            <div class="ddb-header">
              <div>
                <h2 style="font-size:20px;font-weight:700;margin-bottom:4px">{selectedTable}</h2>
                {tableDesc && (
                  <div class="ddb-key-schema">
                    {tableDesc.KeySchema.map((k: any) => (
                      <span>{k.AttributeName} ({k.KeyType})</span>
                    ))}
                    <span style="color:var(--n400)">{tableDesc.ItemCount || 0} items</span>
                  </div>
                )}
              </div>
              <div class="flex gap-2">
                <button class="btn btn-primary btn-sm" onClick={() => {
                  const template: any = {};
                  if (tableDesc) {
                    tableDesc.KeySchema.forEach((k: any) => {
                      const attrDef = tableDesc.AttributeDefinitions.find((a: any) => a.AttributeName === k.AttributeName);
                      const type = attrDef ? attrDef.AttributeType : 'S';
                      template[k.AttributeName] = { [type]: '' };
                    });
                  }
                  setItemJson(JSON.stringify(template, null, 2));
                  setCreateModal(true);
                }}>
                  <PlusIcon /> New Item
                </button>
                <button class="btn btn-ghost btn-sm" onClick={() => scanItems(selectedTable!, null)}>
                  <RefreshIcon /> Refresh
                </button>
                <button class="btn btn-danger btn-sm" onClick={() => setDeleteConfirm(selectedTable)}>
                  <TrashIcon /> Delete Table
                </button>
              </div>
            </div>

            <div class="tabs">
              <button class={`tab ${activeTab === 'browse' ? 'active' : ''}`} onClick={() => setActiveTab('browse')}>Browse Items</button>
              <button class={`tab ${activeTab === 'query' ? 'active' : ''}`} onClick={() => setActiveTab('query')}>Query / Scan</button>
              <button class={`tab ${activeTab === 'info' ? 'active' : ''}`} onClick={() => setActiveTab('info')}>Table Info</button>
            </div>

            {activeTab === 'browse' && (
              <div>
                <div class="card">
                  <div class="table-wrap">
                    <table>
                      <thead>
                        <tr>
                          {columns.map(c => <th>{c}</th>)}
                          <th style="width:40px"></th>
                        </tr>
                      </thead>
                      <tbody>
                        {items.length === 0 ? (
                          <tr><td colSpan={columns.length + 1} class="empty-state">No items</td></tr>
                        ) : items.map((item: any) => (
                          <tr class="clickable" onClick={() => { setItemJson(JSON.stringify(item, null, 2)); setEditModal(item); }}>
                            {columns.map(c => (
                              <td class="font-mono text-sm truncate" style="max-width:250px">{formatDDBValue(item[c])}</td>
                            ))}
                            <td>
                              <button class="btn-icon btn-sm btn-ghost" title="Edit" onClick={(e) => { e.stopPropagation(); setItemJson(JSON.stringify(item, null, 2)); setEditModal(item); }}>
                                <ExpandIcon />
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
                <div class="pagination">
                  <button onClick={prevPage} disabled={page === 0}>Previous</button>
                  <span class="page-info">Page {page + 1}</span>
                  <button onClick={nextPage} disabled={!lastKeys[page]}>Next</button>
                </div>
              </div>
            )}

            {activeTab === 'query' && (
              <div>
                <div class="card" style="margin-bottom:16px">
                  <div class="card-body">
                    <div class="flex gap-3 mb-4">
                      <button class={`btn btn-sm ${queryMode === 'query' ? 'btn-secondary' : 'btn-ghost'}`} onClick={() => setQueryMode('query')}>Query</button>
                      <button class={`btn btn-sm ${queryMode === 'scan' ? 'btn-secondary' : 'btn-ghost'}`} onClick={() => setQueryMode('scan')}>Scan</button>
                    </div>
                    {queryMode === 'query' && (
                      <div class="mb-4">
                        <div class="label">Key Condition Expression</div>
                        <input class="input w-full" placeholder="pk = :pk AND sk BEGINS_WITH :prefix" value={queryExpr} onInput={(e) => setQueryExpr((e.target as HTMLInputElement).value)} />
                      </div>
                    )}
                    <div class="mb-4">
                      <div class="label">Filter Expression</div>
                      <input class="input w-full" placeholder="attribute_exists(name)" value={filterExpr} onInput={(e) => setFilterExpr((e.target as HTMLInputElement).value)} />
                    </div>
                    <div class="mb-4">
                      <div class="label">Expression Attribute Values (JSON)</div>
                      <textarea class="textarea" style="min-height:80px" value={exprAttrValues} onInput={(e) => setExprAttrValues((e.target as HTMLTextAreaElement).value)} />
                    </div>
                    <div class="flex gap-2">
                      <button class="btn btn-primary btn-sm" onClick={runQuery}>
                        <PlayIcon /> Run {queryMode === 'query' ? 'Query' : 'Scan'}
                      </button>
                      {queryResults && (
                        <button class="btn btn-ghost btn-sm" onClick={() => { copyToClipboard(JSON.stringify(queryResults, null, 2)); showToast('Exported to clipboard'); }}>
                          Export JSON
                        </button>
                      )}
                    </div>
                  </div>
                </div>

                {queryResults && (
                  <div class="card">
                    <div class="card-header">
                      <span style="font-weight:600">{queryResults.length} results</span>
                    </div>
                    <div class="card-body">
                      <JsonView data={queryResults} />
                    </div>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'info' && tableDesc && (
              <div class="card">
                <div class="card-body">
                  <JsonView data={tableDesc} />
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {editModal && (
        <Modal
          title="Edit Item"
          size="lg"
          onClose={() => setEditModal(null)}
          footer={
            <>
              <button class="btn btn-danger btn-sm" onClick={() => deleteItem(editModal)}>Delete</button>
              <button class="btn btn-ghost btn-sm" onClick={() => setEditModal(null)}>Cancel</button>
              <button class="btn btn-primary btn-sm" onClick={() => saveItem(itemJson)}>Save</button>
            </>
          }
        >
          <textarea class="textarea" style="min-height:300px" value={itemJson} onInput={(e) => setItemJson((e.target as HTMLTextAreaElement).value)} />
        </Modal>
      )}

      {createModal && (
        <Modal
          title="New Item"
          size="lg"
          onClose={() => setCreateModal(false)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setCreateModal(false)}>Cancel</button>
              <button class="btn btn-primary btn-sm" onClick={() => saveItem(itemJson)}>Create</button>
            </>
          }
        >
          <textarea class="textarea" style="min-height:300px" value={itemJson} onInput={(e) => setItemJson((e.target as HTMLTextAreaElement).value)} />
        </Modal>
      )}

      {createTableModal && (
        <Modal
          title="Create Table"
          size="md"
          onClose={() => setCreateTableModal(false)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setCreateTableModal(false)}>Cancel</button>
              <button class="btn btn-primary btn-sm" onClick={createTable} disabled={!newTableName}>Create Table</button>
            </>
          }
        >
          <div class="label">Table Name</div>
          <input class="input w-full mb-4" value={newTableName} onInput={(e) => setNewTableName((e.target as HTMLInputElement).value)} placeholder="my-table" />

          <div class="label">Partition Key</div>
          <div class="field-row mb-4">
            <input class="input" value={newTablePK} onInput={(e) => setNewTablePK((e.target as HTMLInputElement).value)} placeholder="id" />
            <select class="select" value={newTablePKType} onChange={(e) => setNewTablePKType((e.target as HTMLSelectElement).value)}>
              <option value="S">String</option>
              <option value="N">Number</option>
              <option value="B">Binary</option>
            </select>
          </div>

          <div class="label">Sort Key (optional)</div>
          <div class="field-row">
            <input class="input" value={newTableSK} onInput={(e) => setNewTableSK((e.target as HTMLInputElement).value)} placeholder="sk" />
            <select class="select" value={newTableSKType} onChange={(e) => setNewTableSKType((e.target as HTMLSelectElement).value)}>
              <option value="S">String</option>
              <option value="N">Number</option>
              <option value="B">Binary</option>
            </select>
          </div>
        </Modal>
      )}

      {deleteConfirm && (
        <Modal
          title="Delete Table"
          size="sm"
          onClose={() => setDeleteConfirm(null)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setDeleteConfirm(null)}>Cancel</button>
              <button class="btn btn-danger btn-sm" onClick={() => deleteTable(deleteConfirm)}>Delete</button>
            </>
          }
        >
          <p>Are you sure you want to delete <strong>{deleteConfirm}</strong>? This action cannot be undone.</p>
        </Modal>
      )}
    </div>
  );
}
