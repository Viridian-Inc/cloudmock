import { useState, useMemo, useRef, useEffect } from 'preact/hooks';
import { PlusIcon, RefreshIcon } from '../../components/icons';
import { copyToClipboard } from './utils';

interface TableListProps {
  tables: string[];
  tableCounts: Record<string, number>;
  selectedTable: string | null;
  onSelect: (name: string) => void;
  onCreateTable: () => void;
  onRefresh: () => void;
  onDeleteTable: (name: string) => void;
  onDescribeTable: (name: string) => void;
  onTruncateTable: (name: string) => void;
  showToast: (msg: string) => void;
}

export function TableList({
  tables, tableCounts, selectedTable, onSelect,
  onCreateTable, onRefresh, onDeleteTable, onDescribeTable, onTruncateTable, showToast,
}: TableListProps) {
  const [search, setSearch] = useState('');
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; table: string } | null>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    if (!search) return tables;
    const q = search.toLowerCase();
    return tables.filter(t => t.toLowerCase().includes(q));
  }, [tables, search]);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setContextMenu(null);
      }
    }
    if (contextMenu) {
      document.addEventListener('mousedown', handleClick);
      return () => document.removeEventListener('mousedown', handleClick);
    }
  }, [contextMenu]);

  function handleContextMenu(e: MouseEvent, table: string) {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY, table });
  }

  return (
    <div class="ddb-sidebar">
      <div class="ddb-sidebar-header">
        <div class="flex items-center justify-between mb-4">
          <span style="font-weight:700;font-size:15px">Tables</span>
          <div class="flex gap-2">
            <button
              class="btn-icon btn-sm btn-ghost"
              title="Refresh"
              onClick={onRefresh}
              style="border:1px solid var(--border-default);border-radius:var(--radius-md)"
            >
              <RefreshIcon />
            </button>
            <button class="btn btn-primary btn-sm" onClick={onCreateTable}>
              <PlusIcon /> New
            </button>
          </div>
        </div>
        <input
          class="input w-full"
          placeholder="Filter tables..."
          value={search}
          onInput={(e) => setSearch((e.target as HTMLInputElement).value)}
          style="height:32px;font-size:13px"
        />
      </div>
      <div class="ddb-sidebar-list">
        {filtered.length === 0 ? (
          <div style="padding:24px;text-align:center;color:var(--text-tertiary);font-size:13px">No tables found</div>
        ) : filtered.map(t => (
          <div
            key={t}
            class={`ddb-table-item ${selectedTable === t ? 'active' : ''}`}
            onClick={() => onSelect(t)}
            onContextMenu={(e: any) => handleContextMenu(e, t)}
          >
            <span class="name">{t}</span>
            {tableCounts[t] !== undefined && (
              <span class="count">{tableCounts[t]}</span>
            )}
          </div>
        ))}
      </div>

      {contextMenu && (
        <div
          ref={menuRef}
          class="ddb-context-menu"
          style={`position:fixed;top:${contextMenu.y}px;left:${contextMenu.x}px;z-index:2000`}
        >
          <div class="ddb-context-item" onClick={() => { onDescribeTable(contextMenu.table); setContextMenu(null); }}>
            Describe Table
          </div>
          <div class="ddb-context-item" onClick={() => {
            const arn = `arn:aws:dynamodb:us-east-1:000000000000:table/${contextMenu.table}`;
            copyToClipboard(arn);
            showToast('ARN copied');
            setContextMenu(null);
          }}>
            Copy ARN
          </div>
          <div class="ddb-context-sep" />
          <div class="ddb-context-item ddb-context-warning" onClick={() => { onTruncateTable(contextMenu.table); setContextMenu(null); }}>
            Truncate Table
          </div>
          <div class="ddb-context-item ddb-context-danger" onClick={() => { onDeleteTable(contextMenu.table); setContextMenu(null); }}>
            Delete Table
          </div>
        </div>
      )}
    </div>
  );
}
