import { useState, useRef, useEffect } from 'preact/hooks';
import { DDBItem } from './types';
import { exportAsJSON, exportAsCSV, downloadFile } from './utils';

interface ExportMenuProps {
  items: DDBItem[];
  selectedItems: DDBItem[];
  tableName: string;
  showToast: (msg: string) => void;
}

export function ExportMenu({ items, selectedItems, tableName, showToast }: ExportMenuProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    if (open) {
      document.addEventListener('mousedown', handleClick);
      return () => document.removeEventListener('mousedown', handleClick);
    }
  }, [open]);

  const target = selectedItems.length > 0 ? selectedItems : items;
  const label = selectedItems.length > 0 ? `${selectedItems.length} selected` : 'all items';

  function handleExportJSON() {
    const content = exportAsJSON(target);
    downloadFile(content, `${tableName}.json`, 'application/json');
    showToast(`Exported ${target.length} items as JSON`);
    setOpen(false);
  }

  function handleExportCSV() {
    const content = exportAsCSV(target);
    downloadFile(content, `${tableName}.csv`, 'text/csv');
    showToast(`Exported ${target.length} items as CSV`);
    setOpen(false);
  }

  return (
    <div class="relative" ref={ref}>
      <button class="btn btn-ghost btn-sm" onClick={() => setOpen(!open)}>
        Export
      </button>
      {open && (
        <div class="ddb-export-menu">
          <div class="ddb-export-header">Export {label}</div>
          <div class="ddb-context-item" onClick={handleExportJSON}>
            Export as JSON
          </div>
          <div class="ddb-context-item" onClick={handleExportCSV}>
            Export as CSV
          </div>
        </div>
      )}
    </div>
  );
}
