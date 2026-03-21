import { useState, useMemo, useCallback } from 'preact/hooks';
import { DDBItem, TableDescription } from './types';
import { extractValue, getType, typeBadgeColor, collectColumns } from './utils';
import { ddbRequest } from '../../api';
import { JsonView } from '../../components/JsonView';

interface ItemBrowserProps {
  items: DDBItem[];
  tableDesc: TableDescription;
  tableName: string;
  pageSize: number;
  onPageSizeChange: (size: number) => void;
  page: number;
  hasNext: boolean;
  onNextPage: () => void;
  onPrevPage: () => void;
  onEditItem: (item: DDBItem) => void;
  onDeleteItems: (items: DDBItem[]) => void;
  showToast: (msg: string) => void;
}

export function ItemBrowser({
  items, tableDesc, tableName, pageSize, onPageSizeChange,
  page, hasNext, onNextPage, onPrevPage, onEditItem, onDeleteItems, showToast,
}: ItemBrowserProps) {
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [expandedRow, setExpandedRow] = useState<number | null>(null);
  const [sortCol, setSortCol] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const [hiddenCols, setHiddenCols] = useState<Set<string>>(new Set());
  const [showColMenu, setShowColMenu] = useState(false);
  const [editingCell, setEditingCell] = useState<{ row: number; col: string } | null>(null);
  const [editingValue, setEditingValue] = useState('');
  const [colWidths, setColWidths] = useState<Record<string, number>>({});
  const [resizing, setResizing] = useState<{ col: string; startX: number; startW: number } | null>(null);

  const keyAttrs = useMemo(() =>
    tableDesc.KeySchema.map(k => k.AttributeName),
    [tableDesc]
  );

  const allColumns = useMemo(() => collectColumns(items, keyAttrs), [items, keyAttrs]);
  const columns = useMemo(() => allColumns.filter(c => !hiddenCols.has(c)), [allColumns, hiddenCols]);

  const sortedItems = useMemo(() => {
    if (!sortCol) return items;
    return [...items].sort((a, b) => {
      const va = extractValue(a[sortCol]);
      const vb = extractValue(b[sortCol]);
      const cmp = va.localeCompare(vb, undefined, { numeric: true });
      return sortDir === 'asc' ? cmp : -cmp;
    });
  }, [items, sortCol, sortDir]);

  function toggleSort(col: string) {
    if (sortCol === col) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    } else {
      setSortCol(col);
      setSortDir('asc');
    }
  }

  function toggleSelect(idx: number) {
    setSelected(prev => {
      const next = new Set(prev);
      if (next.has(idx)) next.delete(idx); else next.add(idx);
      return next;
    });
  }

  function toggleSelectAll() {
    if (selected.size === sortedItems.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(sortedItems.map((_, i) => i)));
    }
  }

  function handleDeleteSelected() {
    const toDelete = sortedItems.filter((_, i) => selected.has(i));
    if (toDelete.length === 0) return;
    onDeleteItems(toDelete);
    setSelected(new Set());
  }

  function startCellEdit(row: number, col: string, val: string) {
    setEditingCell({ row, col });
    setEditingValue(val);
  }

  async function saveCellEdit() {
    if (!editingCell) return;
    const item = { ...sortedItems[editingCell.row] };
    const oldVal = item[editingCell.col];
    const type = getType(oldVal);
    // Build new attribute value
    let newVal: any;
    if (type === 'S') newVal = { S: editingValue };
    else if (type === 'N') newVal = { N: editingValue };
    else if (type === 'BOOL') newVal = { BOOL: editingValue === 'true' };
    else {
      try { newVal = JSON.parse(editingValue); } catch { showToast('Invalid value'); return; }
    }
    item[editingCell.col] = newVal;
    try {
      await ddbRequest('PutItem', { TableName: tableName, Item: item });
      showToast('Cell updated');
    } catch {
      showToast('Update failed');
    }
    setEditingCell(null);
  }

  function handleResizeStart(e: MouseEvent, col: string) {
    e.preventDefault();
    e.stopPropagation();
    const startX = e.clientX;
    const startW = colWidths[col] || 150;
    setResizing({ col, startX, startW });

    function onMove(ev: MouseEvent) {
      const diff = ev.clientX - startX;
      setColWidths(prev => ({ ...prev, [col]: Math.max(60, startW + diff) }));
    }
    function onUp() {
      setResizing(null);
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    }
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  }

  const typeBadge = useCallback((val: any) => {
    const t = getType(val);
    const c = typeBadgeColor(t);
    return (
      <span class="ddb-type-badge" style={`background:${c.bg};color:${c.fg}`}>{t}</span>
    );
  }, []);

  return (
    <div>
      <div class="ddb-browser-toolbar">
        <div class="flex items-center gap-2">
          {selected.size > 0 && (
            <>
              <span class="text-sm text-muted">{selected.size} selected</span>
              <button class="btn btn-danger btn-sm" onClick={handleDeleteSelected}>Delete Selected</button>
            </>
          )}
        </div>
        <div class="flex items-center gap-2">
          <div class="relative">
            <button class="btn btn-ghost btn-sm" onClick={() => setShowColMenu(!showColMenu)}>
              Columns
            </button>
            {showColMenu && (
              <div class="ddb-col-menu">
                {allColumns.map(c => (
                  <label key={c} class="ddb-col-menu-item">
                    <input
                      type="checkbox"
                      checked={!hiddenCols.has(c)}
                      onChange={() => {
                        setHiddenCols(prev => {
                          const next = new Set(prev);
                          if (next.has(c)) next.delete(c); else next.add(c);
                          return next;
                        });
                      }}
                    />
                    <span>{c}</span>
                  </label>
                ))}
              </div>
            )}
          </div>
          <select
            class="select"
            style="height:32px;font-size:13px;width:auto"
            value={pageSize}
            onChange={(e) => onPageSizeChange(Number((e.target as HTMLSelectElement).value))}
          >
            <option value={25}>25 / page</option>
            <option value={50}>50 / page</option>
            <option value={100}>100 / page</option>
          </select>
        </div>
      </div>

      <div class="card">
        <div class="ddb-table-wrap">
          <table class="ddb-items-table">
            <thead>
              <tr>
                <th style="width:40px;text-align:center">
                  <input
                    type="checkbox"
                    checked={selected.size === sortedItems.length && sortedItems.length > 0}
                    onChange={toggleSelectAll}
                  />
                </th>
                {columns.map(c => (
                  <th
                    key={c}
                    class="sortable"
                    style={colWidths[c] ? `width:${colWidths[c]}px;min-width:${colWidths[c]}px` : 'min-width:120px'}
                    onClick={() => toggleSort(c)}
                  >
                    <div class="ddb-th-inner">
                      <span>{c}</span>
                      {keyAttrs.includes(c) && <span class="ddb-key-indicator">{tableDesc.KeySchema.find(k => k.AttributeName === c)?.KeyType === 'HASH' ? 'PK' : 'SK'}</span>}
                      {sortCol === c && <span class="ddb-sort-arrow">{sortDir === 'asc' ? '\u25B2' : '\u25BC'}</span>}
                      <div
                        class="ddb-resize-handle"
                        onMouseDown={(e: any) => handleResizeStart(e, c)}
                        onClick={(e) => e.stopPropagation()}
                      />
                    </div>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {sortedItems.length === 0 ? (
                <tr><td colSpan={columns.length + 1} class="empty-state">No items</td></tr>
              ) : sortedItems.map((item, idx) => (
                <>
                  <tr
                    key={idx}
                    class={`ddb-item-row ${selected.has(idx) ? 'selected' : ''} ${expandedRow === idx ? 'expanded' : ''}`}
                  >
                    <td style="width:40px;text-align:center" onClick={(e) => e.stopPropagation()}>
                      <input type="checkbox" checked={selected.has(idx)} onChange={() => toggleSelect(idx)} />
                    </td>
                    {columns.map(c => {
                      const val = item[c];
                      const display = extractValue(val);
                      const isEditing = editingCell?.row === idx && editingCell?.col === c;
                      return (
                        <td
                          key={c}
                          class="ddb-cell"
                          onClick={() => setExpandedRow(expandedRow === idx ? null : idx)}
                          onDblClick={(e) => { e.stopPropagation(); startCellEdit(idx, c, display); }}
                          style={colWidths[c] ? `max-width:${colWidths[c]}px` : 'max-width:250px'}
                        >
                          {isEditing ? (
                            <input
                              class="ddb-cell-edit"
                              value={editingValue}
                              onInput={(e) => setEditingValue((e.target as HTMLInputElement).value)}
                              onBlur={saveCellEdit}
                              onKeyDown={(e) => { if (e.key === 'Enter') saveCellEdit(); if (e.key === 'Escape') setEditingCell(null); }}
                              autoFocus
                              onClick={(e) => e.stopPropagation()}
                            />
                          ) : (
                            <div class="ddb-cell-content">
                              {val && typeBadge(val)}
                              <span class="ddb-cell-value truncate">{display}</span>
                            </div>
                          )}
                        </td>
                      );
                    })}
                  </tr>
                  {expandedRow === idx && (
                    <tr class="ddb-expanded-row">
                      <td colSpan={columns.length + 1}>
                        <div class="ddb-expanded-content">
                          <div class="flex items-center justify-between mb-4">
                            <span style="font-weight:600;font-size:14px">Item Detail</span>
                            <button class="btn btn-secondary btn-sm" onClick={() => onEditItem(item)}>Edit Item</button>
                          </div>
                          <JsonView data={item} />
                        </div>
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div class="pagination">
        <button onClick={onPrevPage} disabled={page === 0}>Previous</button>
        <span class="page-info">Page {page + 1}</span>
        <button onClick={onNextPage} disabled={!hasNext}>Next</button>
      </div>
    </div>
  );
}
