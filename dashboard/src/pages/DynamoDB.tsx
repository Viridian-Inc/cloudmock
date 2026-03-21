import { useState, useEffect, useCallback, useRef } from 'preact/hooks';
import { ddbRequest } from '../api';
import { Modal } from '../components/Modal';
import { DatabaseIcon, PlusIcon, RefreshIcon, TrashIcon, XIcon } from '../components/Icons';
import { DDBItem, TableDescription } from './dynamodb/types';
import { TableList } from './dynamodb/TableList';
import { ItemBrowser } from './dynamodb/ItemBrowser';
import { QueryBuilder } from './dynamodb/QueryBuilder';
import { TableInfo } from './dynamodb/TableInfo';
import { ItemEditor } from './dynamodb/ItemEditor';
import { CreateTable } from './dynamodb/CreateTable';
import { ExportMenu } from './dynamodb/ExportMenu';
import { ImportMenu } from './dynamodb/ImportMenu';
import { BatchWrite } from './dynamodb/BatchWrite';

interface DynamoDBPageProps {
  showToast: (msg: string) => void;
}

interface OpenTab {
  name: string;
  tableDesc: TableDescription | null;
  items: DDBItem[];
  page: number;
  lastKeys: any[];
  activeTab: 'items' | 'query' | 'scan' | 'info';
}

export function DynamoDBPage({ showToast }: DynamoDBPageProps) {
  const [tables, setTables] = useState<string[]>([]);
  const [tableCounts, setTableCounts] = useState<Record<string, number>>({});
  const [openTabs, setOpenTabs] = useState<OpenTab[]>([]);
  const [activeTabIndex, setActiveTabIndex] = useState<number>(-1);
  const [pageSize, setPageSize] = useState(25);
  const [editItem, setEditItem] = useState<DDBItem | null>(null);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showCreateItem, setShowCreateItem] = useState(false);
  const [showCreateTable, setShowCreateTable] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [deleteItemsConfirm, setDeleteItemsConfirm] = useState<DDBItem[] | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  const currentTab = activeTabIndex >= 0 && activeTabIndex < openTabs.length ? openTabs[activeTabIndex] : null;
  const selectedTable = currentTab?.name || null;
  const tableDesc = currentTab?.tableDesc || null;
  const items = currentTab?.items || [];
  const page = currentTab?.page || 0;
  const lastKeys = currentTab?.lastKeys || [];
  const activeTab = currentTab?.activeTab || 'items';

  // Helper to update current tab state
  function updateTab(index: number, patch: Partial<OpenTab>) {
    setOpenTabs(prev => prev.map((t, i) => i === index ? { ...t, ...patch } : t));
  }

  // Load tables
  const loadTables = useCallback(async () => {
    try {
      const r = await ddbRequest('ListTables', {});
      const names: string[] = r.TableNames || [];
      setTables(names);
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
  const scanItems = useCallback(async (tableName: string, startKey: any, tabIdx: number, size: number = pageSize) => {
    const params: any = { TableName: tableName, Limit: size };
    if (startKey) params.ExclusiveStartKey = startKey;
    try {
      const r = await ddbRequest('Scan', params);
      setOpenTabs(prev => prev.map((t, i) => {
        if (i !== tabIdx) return t;
        const newLastKeys = r.LastEvaluatedKey
          ? [...t.lastKeys, r.LastEvaluatedKey]
          : t.lastKeys;
        return { ...t, items: r.Items || [], lastKeys: startKey ? newLastKeys : (r.LastEvaluatedKey ? [r.LastEvaluatedKey] : []) };
      }));
    } catch {
      setOpenTabs(prev => prev.map((t, i) => i === tabIdx ? { ...t, items: [] } : t));
    }
  }, [pageSize]);

  // Select table — open as tab or switch to existing
  function selectTable(name: string) {
    const existingIdx = openTabs.findIndex(t => t.name === name);
    if (existingIdx >= 0) {
      setActiveTabIndex(existingIdx);
      return;
    }
    const newTab: OpenTab = {
      name,
      tableDesc: null,
      items: [],
      page: 0,
      lastKeys: [],
      activeTab: 'items',
    };
    const newTabs = [...openTabs, newTab];
    const newIdx = newTabs.length - 1;
    setOpenTabs(newTabs);
    setActiveTabIndex(newIdx);
    ddbRequest('DescribeTable', { TableName: name }).then((r: any) => {
      setOpenTabs(prev => prev.map((t, i) => i === newIdx ? { ...t, tableDesc: r.Table } : t));
    }).catch(() => {});
    scanItems(name, null, newIdx);
  }

  function closeTab(index: number, e?: Event) {
    if (e) { e.stopPropagation(); e.preventDefault(); }
    const newTabs = openTabs.filter((_, i) => i !== index);
    setOpenTabs(newTabs);
    if (newTabs.length === 0) {
      setActiveTabIndex(-1);
    } else if (activeTabIndex === index) {
      setActiveTabIndex(Math.min(index, newTabs.length - 1));
    } else if (activeTabIndex > index) {
      setActiveTabIndex(activeTabIndex - 1);
    }
  }

  function setContentTab(tab: 'items' | 'query' | 'scan' | 'info') {
    if (activeTabIndex >= 0) updateTab(activeTabIndex, { activeTab: tab });
  }

  function refresh() {
    if (selectedTable && activeTabIndex >= 0) {
      updateTab(activeTabIndex, { page: 0, lastKeys: [] });
      scanItems(selectedTable, null, activeTabIndex);
      ddbRequest('DescribeTable', { TableName: selectedTable }).then((r: any) => {
        if (activeTabIndex >= 0) updateTab(activeTabIndex, { tableDesc: r.Table });
      }).catch(() => {});
    }
  }

  function nextPage() {
    if (!currentTab) return;
    const lastKey = currentTab.lastKeys[currentTab.page];
    if (!lastKey) return;
    const newPage = currentTab.page + 1;
    updateTab(activeTabIndex, { page: newPage });
    scanItems(selectedTable!, lastKey, activeTabIndex);
  }

  function prevPage() {
    if (!currentTab || currentTab.page <= 0) return;
    const newPage = currentTab.page - 1;
    updateTab(activeTabIndex, { page: newPage });
    if (newPage === 0) {
      scanItems(selectedTable!, null, activeTabIndex);
    } else {
      scanItems(selectedTable!, currentTab.lastKeys[newPage - 1], activeTabIndex);
    }
  }

  function handlePageSizeChange(size: number) {
    setPageSize(size);
    if (selectedTable && activeTabIndex >= 0) {
      updateTab(activeTabIndex, { page: 0, lastKeys: [] });
      scanItems(selectedTable, null, activeTabIndex, size);
    }
  }

  // Item operations
  async function saveItem(item: DDBItem) {
    try {
      await ddbRequest('PutItem', { TableName: selectedTable, Item: item });
      showToast('Item saved');
      if (activeTabIndex >= 0) {
        updateTab(activeTabIndex, { page: 0, lastKeys: [] });
        scanItems(selectedTable!, null, activeTabIndex);
      }
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
      if (activeTabIndex >= 0) {
        updateTab(activeTabIndex, { page: 0, lastKeys: [] });
        scanItems(selectedTable!, null, activeTabIndex);
      }
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
    if (activeTabIndex >= 0) {
      updateTab(activeTabIndex, { page: 0, lastKeys: [] });
      scanItems(selectedTable!, null, activeTabIndex);
    }
  }

  function deleteTable(name: string) {
    ddbRequest('DeleteTable', { TableName: name }).then(() => {
      showToast('Table deleted');
      loadTables();
      // Close tab if open
      const tabIdx = openTabs.findIndex(t => t.name === name);
      if (tabIdx >= 0) closeTab(tabIdx);
      setDeleteConfirm(null);
    }).catch(() => showToast('Delete table failed'));
  }

  function handleEditItem(item: DDBItem) {
    setEditItem(item);
    setShowEditModal(true);
  }

  function handleDuplicate(item: DDBItem) {
    setEditItem(null);
    setShowCreateItem(true);
  }

  function handleDescribeTable(name: string) {
    selectTable(name);
    // Set to info tab after opening
    setTimeout(() => {
      setOpenTabs(prev => {
        const idx = prev.findIndex(t => t.name === name);
        if (idx >= 0) return prev.map((t, i) => i === idx ? { ...t, activeTab: 'info' as const } : t);
        return prev;
      });
    }, 50);
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
        e.preventDefault();
        searchRef.current?.focus();
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
      if ((e.key === 'Delete' || e.key === 'Backspace') && deleteItemsConfirm) {
        confirmDeleteItems();
      }
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [selectedTable, tableDesc, deleteItemsConfirm, activeTabIndex]);

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
        {/* Table Tabs Bar */}
        {openTabs.length > 0 && (
          <div class="ddb-tab-bar">
            {openTabs.map((tab, idx) => (
              <div
                key={tab.name}
                class={`ddb-tab-item ${idx === activeTabIndex ? 'active' : ''}`}
                onClick={() => setActiveTabIndex(idx)}
              >
                <span class="ddb-tab-name">{tab.name}</span>
                <button
                  class="ddb-tab-close"
                  onClick={(e) => closeTab(idx, e)}
                  title="Close tab"
                >
                  <XIcon width="12" height="12" />
                </button>
              </div>
            ))}
          </div>
        )}

        {!selectedTable ? (
          <div class="empty-state">
            <DatabaseIcon width="48" height="48" />
            <div style="margin-top:12px;font-size:16px;font-weight:500">Select a table to browse items</div>
            <div style="margin-top:4px;font-size:13px;color:var(--n400)">Or create a new table to get started</div>
            <div style="margin-top:16px;font-size:12px;color:var(--n400)">
              <kbd style="padding:2px 6px;background:var(--n100);border-radius:4px;font-family:var(--font-mono)">Ctrl+N</kbd> New Item
              &nbsp;&middot;&nbsp;
              <kbd style="padding:2px 6px;background:var(--n100);border-radius:4px;font-family:var(--font-mono)">Ctrl+F</kbd> Search
              &nbsp;&middot;&nbsp;
              <kbd style="padding:2px 6px;background:var(--n100);border-radius:4px;font-family:var(--font-mono)">Ctrl+R</kbd> Refresh
            </div>
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
                  <>
                    <ExportMenu
                      items={items}
                      selectedItems={[]}
                      tableName={selectedTable}
                      showToast={showToast}
                    />
                    <ImportMenu
                      tableName={selectedTable}
                      showToast={showToast}
                      onComplete={refresh}
                    />
                    <BatchWrite
                      tableName={selectedTable}
                      showToast={showToast}
                      onComplete={refresh}
                    />
                  </>
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

            {/* Content Tabs */}
            <div class="tabs">
              <button class={`tab ${activeTab === 'items' ? 'active' : ''}`} onClick={() => setContentTab('items')}>Items</button>
              <button class={`tab ${activeTab === 'query' ? 'active' : ''}`} onClick={() => setContentTab('query')}>Query</button>
              <button class={`tab ${activeTab === 'scan' ? 'active' : ''}`} onClick={() => setContentTab('scan')}>Scan</button>
              <button class={`tab ${activeTab === 'info' ? 'active' : ''}`} onClick={() => setContentTab('info')}>Table Info</button>
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
                searchRef={searchRef}
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
