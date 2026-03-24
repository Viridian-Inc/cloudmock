import { useState, useEffect, useCallback, useRef, useReducer, useContext } from 'preact/hooks';
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
import { AccessPatterns } from './dynamodb/AccessPatterns';
import { AccessPatternsTable } from './dynamodb/AccessPatternsTable';
import { PartiQL } from './dynamodb/PartiQL';
import { Terminal } from './dynamodb/Terminal';
import { DDBContext, ddbReducer, initialState, loadPersistedState, usePersistState } from './dynamodb/store';

interface DynamoDBPageProps {
  showToast: (msg: string) => void;
}

// Outer shell: owns the reducer and provides context + session persistence
export function DynamoDBPage({ showToast }: DynamoDBPageProps) {
  const persisted = loadPersistedState();
  const [state, dispatch] = useReducer(ddbReducer, persisted ?? initialState);
  usePersistState(state);

  return (
    <DDBContext.Provider value={{ state, dispatch }}>
      <DynamoDBCore showToast={showToast} />
    </DDBContext.Provider>
  );
}

// Inner component: consumes context, holds UI-only modal state
function DynamoDBCore({ showToast }: DynamoDBPageProps) {
  const { state, dispatch } = useContext(DDBContext);
  const [tables, setTables] = useState<string[]>([]);
  const [tableCounts, setTableCounts] = useState<Record<string, number>>({});
  const [editItem, setEditItem] = useState<DDBItem | null>(null);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showCreateItem, setShowCreateItem] = useState(false);
  const [showCreateTable, setShowCreateTable] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [deleteItemsConfirm, setDeleteItemsConfirm] = useState<DDBItem[] | null>(null);
  const [truncateConfirm, setTruncateConfirm] = useState<string | null>(null);
  const [truncateProgress, setTruncateProgress] = useState<{ done: number; total: number } | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  const { tabs, activeTabIndex } = state;
  const currentTab = activeTabIndex >= 0 && activeTabIndex < tabs.length ? tabs[activeTabIndex] : null;
  const selectedTable = currentTab?.tableName ?? null;
  const tableDesc = currentTab?.tableDesc ?? null;
  const items = currentTab?.items ?? [];
  const page = currentTab?.page ?? 0;
  const pageSize = currentTab?.pageSize ?? 25;
  const lastKeys = currentTab?.lastEvaluatedKeys ?? [];
  const activeSubTab = currentTab?.activeSubTab ?? 'items';

  // Load tables list
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

  // Scan items for a specific tab index
  // NOTE: tabIdx and tabPageSize must be captured before dispatch changes them
  const scanItems = useCallback(async (
    tableName: string,
    startKey: any,
    tabIdx: number,
    tabPage: number,
    size: number,
  ) => {
    dispatch({ type: 'SET_LOADING', index: tabIdx, loading: true });
    const params: any = { TableName: tableName, Limit: size };
    if (startKey) params.ExclusiveStartKey = startKey;
    try {
      const r = await ddbRequest('Scan', params);
      dispatch({
        type: 'SET_ITEMS',
        index: tabIdx,
        items: r.Items || [],
        lastKey: r.LastEvaluatedKey ?? null,
        page: tabPage,
      });
    } catch {
      dispatch({ type: 'SET_LOADING', index: tabIdx, loading: false });
    }
  }, [dispatch]);

  // Select / open table as tab
  function selectTable(name: string) {
    const existingIdx = tabs.findIndex(t => t.tableName === name);
    if (existingIdx >= 0) {
      dispatch({ type: 'SET_ACTIVE_TAB', index: existingIdx });
      return;
    }
    // OPEN_TAB will append at index = tabs.length
    const newIdx = tabs.length;
    dispatch({ type: 'OPEN_TAB', tableName: name });
    ddbRequest('DescribeTable', { TableName: name }).then((r: any) => {
      dispatch({ type: 'SET_TABLE_DESC', index: newIdx, desc: r.Table });
    }).catch(() => {});
    scanItems(name, null, newIdx, 0, 25);
  }

  function closeTab(index: number, e?: Event) {
    if (e) { e.stopPropagation(); e.preventDefault(); }
    dispatch({ type: 'CLOSE_TAB', index });
  }

  function setContentTab(tab: 'items' | 'query' | 'scan' | 'patterns' | 'sql' | 'terminal' | 'info') {
    if (activeTabIndex >= 0) {
      dispatch({ type: 'UPDATE_TAB', index: activeTabIndex, patch: { activeSubTab: tab } });
    }
  }

  function refresh() {
    if (selectedTable && activeTabIndex >= 0 && currentTab) {
      dispatch({ type: 'UPDATE_TAB', index: activeTabIndex, patch: { page: 0, lastEvaluatedKeys: [null] } });
      scanItems(selectedTable, null, activeTabIndex, 0, currentTab.pageSize);
      ddbRequest('DescribeTable', { TableName: selectedTable }).then((r: any) => {
        dispatch({ type: 'SET_TABLE_DESC', index: activeTabIndex, desc: r.Table });
      }).catch(() => {});
    }
  }

  function nextPage() {
    if (!currentTab) return;
    const nextPageIndex = currentTab.page + 1;
    const lastKey = currentTab.lastEvaluatedKeys[nextPageIndex];
    if (!lastKey) return;
    dispatch({ type: 'UPDATE_TAB', index: activeTabIndex, patch: { page: nextPageIndex } });
    scanItems(selectedTable!, lastKey, activeTabIndex, nextPageIndex, currentTab.pageSize);
  }

  function prevPage() {
    if (!currentTab || currentTab.page <= 0) return;
    const newPage = currentTab.page - 1;
    dispatch({ type: 'UPDATE_TAB', index: activeTabIndex, patch: { page: newPage } });
    const startKey = newPage === 0 ? null : currentTab.lastEvaluatedKeys[newPage];
    scanItems(selectedTable!, startKey, activeTabIndex, newPage, currentTab.pageSize);
  }

  function handlePageSizeChange(size: number) {
    if (selectedTable && activeTabIndex >= 0) {
      dispatch({ type: 'UPDATE_TAB', index: activeTabIndex, patch: { page: 0, pageSize: size, lastEvaluatedKeys: [null] } });
      scanItems(selectedTable, null, activeTabIndex, 0, size);
    }
  }

  // Item operations
  async function saveItem(item: DDBItem) {
    try {
      await ddbRequest('PutItem', { TableName: selectedTable, Item: item });
      showToast('Item saved');
      if (activeTabIndex >= 0 && currentTab) {
        dispatch({ type: 'UPDATE_TAB', index: activeTabIndex, patch: { page: 0, lastEvaluatedKeys: [null] } });
        scanItems(selectedTable!, null, activeTabIndex, 0, currentTab.pageSize);
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
    tableDesc.KeySchema.forEach(k => { key[k.AttributeName] = item[k.AttributeName]; });
    try {
      await ddbRequest('DeleteItem', { TableName: selectedTable, Key: key });
      showToast('Item deleted');
      if (activeTabIndex >= 0 && currentTab) {
        dispatch({ type: 'UPDATE_TAB', index: activeTabIndex, patch: { page: 0, lastEvaluatedKeys: [null] } });
        scanItems(selectedTable!, null, activeTabIndex, 0, currentTab.pageSize);
      }
      setShowEditModal(false);
    } catch {
      showToast('Delete failed');
    }
  }

  function deleteItems(itemsToDelete: DDBItem[]) {
    setDeleteItemsConfirm(itemsToDelete);
  }

  async function confirmDeleteItems() {
    if (!deleteItemsConfirm || !tableDesc) return;
    let failed = 0;
    for (const item of deleteItemsConfirm) {
      const key: any = {};
      tableDesc.KeySchema.forEach(k => { key[k.AttributeName] = item[k.AttributeName]; });
      try {
        await ddbRequest('DeleteItem', { TableName: selectedTable, Key: key });
      } catch { failed++; }
    }
    showToast(`Deleted ${deleteItemsConfirm.length - failed} items${failed ? `, ${failed} failed` : ''}`);
    setDeleteItemsConfirm(null);
    if (activeTabIndex >= 0 && currentTab) {
      dispatch({ type: 'UPDATE_TAB', index: activeTabIndex, patch: { page: 0, lastEvaluatedKeys: [null] } });
      scanItems(selectedTable!, null, activeTabIndex, 0, currentTab.pageSize);
    }
  }

  function deleteTable(name: string) {
    ddbRequest('DeleteTable', { TableName: name }).then(() => {
      showToast('Table deleted');
      loadTables();
      const tabIdx = tabs.findIndex(t => t.tableName === name);
      if (tabIdx >= 0) closeTab(tabIdx);
      setDeleteConfirm(null);
    }).catch(() => showToast('Delete table failed'));
  }

  async function truncateTable(name: string) {
    try {
      const desc = await ddbRequest('DescribeTable', { TableName: name });
      const keySchema = desc.Table?.KeySchema || [];

      let allItems: DDBItem[] = [];
      let lastKey: any = undefined;
      do {
        const params: any = { TableName: name };
        if (lastKey) params.ExclusiveStartKey = lastKey;
        params.ProjectionExpression = keySchema.map((_: any, i: number) => `#k${i}`).join(', ');
        params.ExpressionAttributeNames = {};
        keySchema.forEach((k: any, i: number) => {
          params.ExpressionAttributeNames[`#k${i}`] = k.AttributeName;
        });
        const r = await ddbRequest('Scan', params);
        allItems = allItems.concat(r.Items || []);
        lastKey = r.LastEvaluatedKey;
      } while (lastKey);

      setTruncateProgress({ done: 0, total: allItems.length });
      let done = 0;
      for (let i = 0; i < allItems.length; i += 25) {
        const batch = allItems.slice(i, i + 25);
        const deleteRequests = batch.map(item => {
          const key: any = {};
          keySchema.forEach((k: any) => { key[k.AttributeName] = item[k.AttributeName]; });
          return { DeleteRequest: { Key: key } };
        });
        try {
          await ddbRequest('BatchWriteItem', { RequestItems: { [name]: deleteRequests } });
        } catch {
          for (const req of deleteRequests) {
            try { await ddbRequest('DeleteItem', { TableName: name, Key: req.DeleteRequest.Key }); } catch { /* skip */ }
          }
        }
        done += batch.length;
        setTruncateProgress({ done, total: allItems.length });
      }

      showToast(`Truncated ${allItems.length} items from ${name}`);
      setTruncateConfirm(null);
      setTruncateProgress(null);
      loadTables();
      const tabIdx = tabs.findIndex(t => t.tableName === name);
      if (tabIdx >= 0) {
        const tabPageSize = tabs[tabIdx]?.pageSize ?? 25;
        dispatch({ type: 'UPDATE_TAB', index: tabIdx, patch: { page: 0, lastEvaluatedKeys: [null] } });
        scanItems(name, null, tabIdx, 0, tabPageSize);
      }
    } catch (e: any) {
      showToast('Truncate failed: ' + (e.message || 'unknown error'));
      setTruncateConfirm(null);
      setTruncateProgress(null);
    }
  }

  function handleEditItem(item: DDBItem) {
    setEditItem(item);
    setShowEditModal(true);
  }

  function handleDuplicate(_item: DDBItem) {
    setEditItem(null);
    setShowCreateItem(true);
  }

  function handleDescribeTable(name: string) {
    selectTable(name);
    setTimeout(() => {
      const idx = tabs.findIndex(t => t.tableName === name);
      if (idx >= 0) dispatch({ type: 'UPDATE_TAB', index: idx, patch: { activeSubTab: 'info' } });
    }, 50);
  }

  function handleQueryIndex(indexName: string, _pkAttr: string) {
    if (activeTabIndex >= 0) {
      dispatch({
        type: 'UPDATE_TAB',
        index: activeTabIndex,
        patch: { activeSubTab: 'query', queryInitialIndex: indexName, queryInitialPk: '' },
      });
    }
  }

  const contentTabs: { key: 'items' | 'query' | 'scan' | 'patterns' | 'sql' | 'terminal' | 'info'; label: string; shortcut?: string }[] = [
    { key: 'items', label: 'Items' },
    { key: 'query', label: 'Query' },
    { key: 'scan', label: 'Scan' },
    { key: 'patterns', label: 'Access Patterns' },
    { key: 'sql', label: 'SQL' },
    { key: 'terminal', label: 'Terminal', shortcut: 'Ctrl+T' },
    { key: 'info', label: 'Table Info' },
  ];

  // Keyboard shortcuts
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement || e.target instanceof HTMLSelectElement) {
        if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') return;
        return;
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'n') { e.preventDefault(); if (selectedTable && tableDesc) setShowCreateItem(true); }
      if ((e.ctrlKey || e.metaKey) && e.key === 'f') { e.preventDefault(); searchRef.current?.focus(); }
      if ((e.ctrlKey || e.metaKey) && e.key === 'r') { e.preventDefault(); refresh(); }
      if ((e.ctrlKey || e.metaKey) && e.key === 't') { e.preventDefault(); if (selectedTable) setContentTab('terminal'); }
      if (e.key === 'Escape') {
        setShowEditModal(false);
        setShowCreateItem(false);
        setShowCreateTable(false);
        setDeleteConfirm(null);
        setDeleteItemsConfirm(null);
      }
      if ((e.key === 'Delete' || e.key === 'Backspace') && deleteItemsConfirm) { confirmDeleteItems(); }
      if (e.key === 'Tab' && !e.ctrlKey && !e.metaKey && !e.shiftKey && selectedTable) {
        e.preventDefault();
        const currentIdx = contentTabs.findIndex(t => t.key === activeSubTab);
        setContentTab(contentTabs[(currentIdx + 1) % contentTabs.length].key);
      }
    }
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [selectedTable, tableDesc, deleteItemsConfirm, activeTabIndex, activeSubTab]);

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
        onTruncateTable={(name) => setTruncateConfirm(name)}
        showToast={showToast}
      />

      <div class="ddb-main">
        {tabs.length > 0 && (
          <div class="ddb-tab-bar">
            {tabs.map((tab, idx) => (
              <div
                key={tab.tableName}
                class={`ddb-tab-item ${idx === activeTabIndex ? 'active' : ''}`}
                onClick={() => dispatch({ type: 'SET_ACTIVE_TAB', index: idx })}
              >
                <span class="ddb-tab-name">{tab.tableName}</span>
                <button class="ddb-tab-close" onClick={(e) => closeTab(idx, e)} title="Close tab">
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
            <div style="margin-top:4px;font-size:13px;color:var(--text-tertiary)">Or create a new table to get started</div>
            <div style="margin-top:16px;font-size:12px;color:var(--text-tertiary)">
              <kbd style="padding:2px 6px;background:var(--bg-secondary);border-radius:4px;font-family:var(--font-mono)">Ctrl+N</kbd> New Item
              &nbsp;&middot;&nbsp;
              <kbd style="padding:2px 6px;background:var(--bg-secondary);border-radius:4px;font-family:var(--font-mono)">Ctrl+F</kbd> Search
              &nbsp;&middot;&nbsp;
              <kbd style="padding:2px 6px;background:var(--bg-secondary);border-radius:4px;font-family:var(--font-mono)">Ctrl+R</kbd> Refresh
              &nbsp;&middot;&nbsp;
              <kbd style="padding:2px 6px;background:var(--bg-secondary);border-radius:4px;font-family:var(--font-mono)">Ctrl+T</kbd> Terminal
              &nbsp;&middot;&nbsp;
              <kbd style="padding:2px 6px;background:var(--bg-secondary);border-radius:4px;font-family:var(--font-mono)">Tab</kbd> Switch Tabs
            </div>
          </div>
        ) : (
          <div>
            {tableDesc && (
              <AccessPatterns tableDesc={tableDesc} onQueryIndex={handleQueryIndex} />
            )}

            <div class="ddb-header">
              <div>
                <h2 style="font-size:20px;font-weight:700;margin-bottom:4px">{selectedTable}</h2>
              </div>
              <div class="flex gap-2">
                {activeSubTab === 'items' && (
                  <>
                    <ExportMenu items={items} selectedItems={[]} tableName={selectedTable} showToast={showToast} />
                    <ImportMenu tableName={selectedTable} showToast={showToast} onComplete={refresh} />
                    <BatchWrite tableName={selectedTable} showToast={showToast} onComplete={refresh} />
                  </>
                )}
                <button class="btn btn-primary btn-sm" onClick={() => setShowCreateItem(true)} title="New Item (Ctrl+N)">
                  <PlusIcon /> New Item
                </button>
                <button class="btn btn-ghost btn-sm" onClick={refresh} title="Refresh (Ctrl+R)">
                  <RefreshIcon /> Refresh
                </button>
                <button class="btn btn-danger btn-sm" onClick={() => setDeleteConfirm(selectedTable)}>
                  <TrashIcon />
                </button>
              </div>
            </div>

            <div class="tabs">
              {contentTabs.map(ct => (
                <button
                  key={ct.key}
                  class={`tab ${activeSubTab === ct.key ? 'active' : ''}`}
                  onClick={() => setContentTab(ct.key)}
                  title={ct.shortcut || undefined}
                >
                  {ct.label}
                </button>
              ))}
            </div>

            {activeSubTab === 'items' && tableDesc && (
              <ItemBrowser
                items={items}
                tableDesc={tableDesc}
                tableName={selectedTable}
                pageSize={pageSize}
                onPageSizeChange={handlePageSizeChange}
                page={page}
                hasNext={!!lastKeys[page + 1]}
                onNextPage={nextPage}
                onPrevPage={prevPage}
                onEditItem={handleEditItem}
                onDeleteItems={deleteItems}
                showToast={showToast}
                searchRef={searchRef}
              />
            )}

            {(activeSubTab === 'query' || activeSubTab === 'scan') && tableDesc && currentTab && (
              <QueryBuilder
                tableName={selectedTable}
                tableDesc={tableDesc}
                showToast={showToast}
                onEditItem={handleEditItem}
                initialIndexName={currentTab.queryInitialIndex}
                initialPkValue={currentTab.queryInitialPk}
                tabIndex={activeTabIndex}
              />
            )}

            {activeSubTab === 'patterns' && tableDesc && (
              <AccessPatternsTable
                tableName={selectedTable}
                tableDesc={tableDesc}
                showToast={showToast}
              />
            )}

            {activeSubTab === 'sql' && tableDesc && (
              <PartiQL
                tableName={selectedTable}
                tableDesc={tableDesc}
                showToast={showToast}
                onEditItem={handleEditItem}
                tabIndex={activeTabIndex}
              />
            )}

            {activeSubTab === 'terminal' && tableDesc && (
              <Terminal
                tableName={selectedTable}
                showToast={showToast}
                tabIndex={activeTabIndex}
              />
            )}

            {activeSubTab === 'info' && tableDesc && (
              <TableInfo tableDesc={tableDesc} />
            )}
          </div>
        )}
      </div>

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

      {showCreateItem && tableDesc && (
        <ItemEditor
          item={null}
          tableDesc={tableDesc}
          onSave={saveItem}
          onClose={() => setShowCreateItem(false)}
        />
      )}

      {showCreateTable && (
        <CreateTable
          onClose={() => setShowCreateTable(false)}
          onCreated={loadTables}
          showToast={showToast}
        />
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

      {truncateConfirm && (
        <Modal
          title="Truncate Table"
          size="sm"
          onClose={() => { if (!truncateProgress) setTruncateConfirm(null); }}
          footer={
            truncateProgress ? undefined : (
              <>
                <button class="btn btn-ghost btn-sm" onClick={() => setTruncateConfirm(null)}>Cancel</button>
                <button class="btn btn-danger btn-sm" onClick={() => truncateTable(truncateConfirm)}>Truncate</button>
              </>
            )
          }
        >
          {truncateProgress ? (
            <div>
              <p style="margin-bottom:12px">Deleting items from <strong>{truncateConfirm}</strong>...</p>
              <div style="background:var(--bg-tertiary);border-radius:4px;height:8px;overflow:hidden">
                <div style={`background:var(--error);height:100%;width:${truncateProgress.total > 0 ? (truncateProgress.done / truncateProgress.total) * 100 : 0}%;transition:width 0.2s`} />
              </div>
              <div style="font-size:12px;color:var(--text-secondary);margin-top:6px">
                {truncateProgress.done} / {truncateProgress.total} items deleted
              </div>
            </div>
          ) : (
            <p>Are you sure you want to delete <strong>all {tableCounts[truncateConfirm] ?? '?'} items</strong> from <strong>{truncateConfirm}</strong>? The table structure will be preserved.</p>
          )}
        </Modal>
      )}

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
