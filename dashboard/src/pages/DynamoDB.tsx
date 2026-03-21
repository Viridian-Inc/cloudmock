import { useState, useEffect, useCallback } from 'preact/hooks';
import { ddbRequest } from '../api';
import { Modal } from '../components/Modal';
import { DatabaseIcon, PlusIcon, RefreshIcon, TrashIcon } from '../components/Icons';
import { DDBItem, TableDescription } from './dynamodb/types';
import { TableList } from './dynamodb/TableList';
import { ItemBrowser } from './dynamodb/ItemBrowser';
import { QueryBuilder } from './dynamodb/QueryBuilder';
import { TableInfo } from './dynamodb/TableInfo';
import { ItemEditor } from './dynamodb/ItemEditor';
import { CreateTable } from './dynamodb/CreateTable';
import { ExportMenu } from './dynamodb/ExportMenu';

interface DynamoDBPageProps {
  showToast: (msg: string) => void;
}

export function DynamoDBPage({ showToast }: DynamoDBPageProps) {
  const [tables, setTables] = useState<string[]>([]);
  const [tableCounts, setTableCounts] = useState<Record<string, number>>({});
  const [selectedTable, setSelectedTable] = useState<string | null>(null);
  const [tableDesc, setTableDesc] = useState<TableDescription | null>(null);
  const [items, setItems] = useState<DDBItem[]>([]);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(25);
  const [lastKeys, setLastKeys] = useState<any[]>([]);
  const [activeTab, setActiveTab] = useState<'items' | 'query' | 'scan' | 'info'>('items');
  const [editItem, setEditItem] = useState<DDBItem | null>(null);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showCreateItem, setShowCreateItem] = useState(false);
  const [showCreateTable, setShowCreateTable] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [deleteItemsConfirm, setDeleteItemsConfirm] = useState<DDBItem[] | null>(null);

  // Load tables
  const loadTables = useCallback(async () => {
    try {
      const r = await ddbRequest('ListTables', {});
      const names: string[] = r.TableNames || [];
      setTables(names);
      // Fetch item counts
      const counts: Record<string, number> = {};
      for (const name of names) {
        try {
          const desc = await ddbRequest('DescribeTable', { TableName: name });
          counts[name] = desc.Table?.ItemCount ?? 0;
        } catch { counts[name] = 0; }
      }
      setTableCounts(counts);
    } catch {
      setTables([]);
    }
  }, []);

  useEffect(() => { loadTables(); }, [loadTables]);

  // Scan items
  const scanItems = useCallback(async (tableName: string, startKey: any, size: number = pageSize) => {
    const params: any = { TableName: tableName, Limit: size };
    if (startKey) params.ExclusiveStartKey = startKey;
    try {
      const r = await ddbRequest('Scan', params);
      setItems(r.Items || []);
      if (r.LastEvaluatedKey) {
        setLastKeys(prev => [...prev, r.LastEvaluatedKey]);
      }
    } catch {
      setItems([]);
    }
  }, [pageSize]);

  // Select table
  function selectTable(name: string) {
    setSelectedTable(name);
    setPage(0);
    setLastKeys([]);
    setActiveTab('items');
    ddbRequest('DescribeTable', { TableName: name }).then((r: any) => {
      setTableDesc(r.Table);
    }).catch(() => {});
    scanItems(name, null);
  }

  function refresh() {
    if (selectedTable) {
      setPage(0);
      setLastKeys([]);
      scanItems(selectedTable, null);
      ddbRequest('DescribeTable', { TableName: selectedTable }).then((r: any) => {
        setTableDesc(r.Table);
      }).catch(() => {});
    }
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

  function handlePageSizeChange(size: number) {
    setPageSize(size);
    setPage(0);
    setLastKeys([]);
    if (selectedTable) scanItems(selectedTable, null, size);
  }

  // Item operations
  async function saveItem(item: DDBItem) {
    try {
      await ddbRequest('PutItem', { TableName: selectedTable, Item: item });
      showToast('Item saved');
      setPage(0);
      setLastKeys([]);
      scanItems(selectedTable!, null);
      setShowEditModal(false);
      setShowCreateItem(false);
    } catch {
      showToast('Save failed');
    }
  }

  async function deleteItem(item: DDBItem) {
    if (!tableDesc) return;
    const key: any = {};
    tableDesc.KeySchema.forEach(k => {
      key[k.AttributeName] = item[k.AttributeName];
    });
    try {
      await ddbRequest('DeleteItem', { TableName: selectedTable, Key: key });
      showToast('Item deleted');
      setPage(0);
      setLastKeys([]);
      scanItems(selectedTable!, null);
      setShowEditModal(false);
    } catch {
      showToast('Delete failed');
    }
  }

  async function deleteItems(itemsToDelete: DDBItem[]) {
    setDeleteItemsConfirm(itemsToDelete);
  }

  async function confirmDeleteItems() {
    if (!deleteItemsConfirm || !tableDesc) return;
    let failed = 0;
    for (const item of deleteItemsConfirm) {
      const key: any = {};
      tableDesc.KeySchema.forEach(k => {
        key[k.AttributeName] = item[k.AttributeName];
      });
      try {
        await ddbRequest('DeleteItem', { TableName: selectedTable, Key: key });
      } catch { failed++; }
    }
    showToast(`Deleted ${deleteItemsConfirm.length - failed} items${failed ? `, ${failed} failed` : ''}`);
    setDeleteItemsConfirm(null);
    setPage(0);
    setLastKeys([]);
    scanItems(selectedTable!, null);
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

  function handleEditItem(item: DDBItem) {
    setEditItem(item);
    setShowEditModal(true);
  }

  function handleDuplicate(item: DDBItem) {
    // Open as new item (user must change keys)
    setEditItem(null);
    setShowCreateItem(true);
    // The CreateItem will pre-populate from the last edited state
  }

  function handleDescribeTable(name: string) {
    setSelectedTable(name);
    setActiveTab('info');
    ddbRequest('DescribeTable', { TableName: name }).then((r: any) => {
      setTableDesc(r.Table);
    }).catch(() => {});
    scanItems(name, null);
  }

  // Keyboard shortcuts
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement || e.target instanceof HTMLSelectElement) return;
      if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault();
        if (selectedTable && tableDesc) setShowCreateItem(true);
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
        // Let table search handle it — noop here
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
        e.preventDefault();
        refresh();
      }
      if (e.key === 'Escape') {
        setShowEditModal(false);
        setShowCreateItem(false);
        setShowCreateTable(false);
        setDeleteConfirm(null);
        setDeleteItemsConfirm(null);
      }
      if (e.key === 'Delete' && deleteItemsConfirm) {
        confirmDeleteItems();
      }
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [selectedTable, tableDesc, deleteItemsConfirm]);

  return (
    <div class="ddb-layout">
      <TableList
        tables={tables}
        tableCounts={tableCounts}
        selectedTable={selectedTable}
        onSelect={selectTable}
        onCreateTable={() => setShowCreateTable(true)}
        onRefresh={loadTables}
        onDeleteTable={(name) => setDeleteConfirm(name)}
        onDescribeTable={handleDescribeTable}
        showToast={showToast}
      />

      <div class="ddb-main">
        {!selectedTable ? (
          <div class="empty-state">
            <DatabaseIcon width="48" height="48" />
            <div style="margin-top:12px;font-size:16px;font-weight:500">Select a table to browse items</div>
            <div style="margin-top:4px;font-size:13px;color:var(--n400)">Or create a new table to get started</div>
          </div>
        ) : (
          <div>
            {/* Header */}
            <div class="ddb-header">
              <div>
                <h2 style="font-size:20px;font-weight:700;margin-bottom:4px">{selectedTable}</h2>
                {tableDesc && (
                  <div class="ddb-key-schema">
                    {tableDesc.KeySchema.map(k => {
                      const attrDef = tableDesc.AttributeDefinitions.find(a => a.AttributeName === k.AttributeName);
                      return (
                        <span key={k.AttributeName}>
                          {k.AttributeName}
                          <span style="opacity:0.6;margin-left:4px">{k.KeyType === 'HASH' ? 'PK' : 'SK'}</span>
                          {attrDef && <span style="opacity:0.5;margin-left:2px">({attrDef.AttributeType})</span>}
                        </span>
                      );
                    })}
                    <span style="color:var(--n400)">{tableDesc.ItemCount ?? 0} items</span>
                  </div>
                )}
              </div>
              <div class="flex gap-2">
                {activeTab === 'items' && (
                  <ExportMenu
                    items={items}
                    selectedItems={[]}
                    tableName={selectedTable}
                    showToast={showToast}
                  />
                )}
                <button class="btn btn-primary btn-sm" onClick={() => setShowCreateItem(true)}>
                  <PlusIcon /> New Item
                </button>
                <button class="btn btn-ghost btn-sm" onClick={refresh}>
                  <RefreshIcon /> Refresh
                </button>
                <button class="btn btn-danger btn-sm" onClick={() => setDeleteConfirm(selectedTable)}>
                  <TrashIcon />
                </button>
              </div>
            </div>

            {/* Tabs */}
            <div class="tabs">
              <button class={`tab ${activeTab === 'items' ? 'active' : ''}`} onClick={() => setActiveTab('items')}>Items</button>
              <button class={`tab ${activeTab === 'query' ? 'active' : ''}`} onClick={() => setActiveTab('query')}>Query</button>
              <button class={`tab ${activeTab === 'scan' ? 'active' : ''}`} onClick={() => setActiveTab('scan')}>Scan</button>
              <button class={`tab ${activeTab === 'info' ? 'active' : ''}`} onClick={() => setActiveTab('info')}>Table Info</button>
            </div>

            {/* Tab content */}
            {activeTab === 'items' && tableDesc && (
              <ItemBrowser
                items={items}
                tableDesc={tableDesc}
                tableName={selectedTable}
                pageSize={pageSize}
                onPageSizeChange={handlePageSizeChange}
                page={page}
                hasNext={!!lastKeys[page]}
                onNextPage={nextPage}
                onPrevPage={prevPage}
                onEditItem={handleEditItem}
                onDeleteItems={deleteItems}
                showToast={showToast}
              />
            )}

            {(activeTab === 'query' || activeTab === 'scan') && tableDesc && (
              <QueryBuilder
                tableName={selectedTable}
                tableDesc={tableDesc}
                showToast={showToast}
                onEditItem={handleEditItem}
              />
            )}

            {activeTab === 'info' && tableDesc && (
              <TableInfo tableDesc={tableDesc} />
            )}
          </div>
        )}
      </div>

      {/* Item editor modal */}
      {showEditModal && tableDesc && (
        <ItemEditor
          item={editItem}
          tableDesc={tableDesc}
          onSave={saveItem}
          onDelete={deleteItem}
          onDuplicate={handleDuplicate}
          onClose={() => setShowEditModal(false)}
        />
      )}

      {/* Create item modal */}
      {showCreateItem && tableDesc && (
        <ItemEditor
          item={null}
          tableDesc={tableDesc}
          onSave={saveItem}
          onClose={() => setShowCreateItem(false)}
        />
      )}

      {/* Create table modal */}
      {showCreateTable && (
        <CreateTable
          onClose={() => setShowCreateTable(false)}
          onCreated={loadTables}
          showToast={showToast}
        />
      )}

      {/* Delete table confirmation */}
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

      {/* Delete items confirmation */}
      {deleteItemsConfirm && (
        <Modal
          title="Delete Items"
          size="sm"
          onClose={() => setDeleteItemsConfirm(null)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setDeleteItemsConfirm(null)}>Cancel</button>
              <button class="btn btn-danger btn-sm" onClick={confirmDeleteItems}>
                Delete {deleteItemsConfirm.length} item{deleteItemsConfirm.length !== 1 ? 's' : ''}
              </button>
            </>
          }
        >
          <p>Are you sure you want to delete {deleteItemsConfirm.length} item{deleteItemsConfirm.length !== 1 ? 's' : ''}? This action cannot be undone.</p>
        </Modal>
      )}
    </div>
  );
}
