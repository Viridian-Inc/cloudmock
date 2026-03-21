import { useState, useRef } from 'preact/hooks';
import { Modal } from '../../components/Modal';
import { ddbRequest } from '../../api';
import { UploadIcon } from '../../components/Icons';
import { DDBItem } from './types';
import { buildAttributeValue } from './utils';

interface ImportMenuProps {
  tableName: string;
  showToast: (msg: string) => void;
  onComplete: () => void;
}

function parseCSV(text: string): Record<string, string>[] {
  const lines = text.split('\n').filter(l => l.trim());
  if (lines.length < 2) return [];
  const headers = lines[0].split(',').map(h => h.trim().replace(/^"|"$/g, ''));
  const rows: Record<string, string>[] = [];
  for (let i = 1; i < lines.length; i++) {
    const vals: string[] = [];
    let current = '';
    let inQuotes = false;
    for (const ch of lines[i]) {
      if (ch === '"') { inQuotes = !inQuotes; continue; }
      if (ch === ',' && !inQuotes) { vals.push(current.trim()); current = ''; continue; }
      current += ch;
    }
    vals.push(current.trim());
    const row: Record<string, string> = {};
    headers.forEach((h, j) => { if (vals[j] !== undefined) row[h] = vals[j]; });
    rows.push(row);
  }
  return rows;
}

function csvRowToDDBItem(row: Record<string, string>): DDBItem {
  const item: DDBItem = {};
  for (const [key, val] of Object.entries(row)) {
    // Auto-detect type: number vs string
    if (val === 'true' || val === 'false') {
      item[key] = { BOOL: val === 'true' };
    } else if (val === '' || val === 'null') {
      item[key] = { NULL: true };
    } else if (!isNaN(Number(val)) && val !== '') {
      item[key] = { N: val };
    } else {
      item[key] = { S: val };
    }
  }
  return item;
}

export function ImportMenu({ tableName, showToast, onComplete }: ImportMenuProps) {
  const [showModal, setShowModal] = useState(false);
  const [importing, setImporting] = useState(false);
  const [progress, setProgress] = useState(0);
  const [total, setTotal] = useState(0);
  const [errors, setErrors] = useState(0);
  const [preview, setPreview] = useState<DDBItem[] | null>(null);
  const [fileName, setFileName] = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  function handleFileSelect(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;
    setFileName(file.name);
    const reader = new FileReader();
    reader.onload = () => {
      try {
        const text = reader.result as string;
        let items: DDBItem[];
        if (file.name.endsWith('.csv')) {
          const rows = parseCSV(text);
          items = rows.map(csvRowToDDBItem);
        } else {
          // JSON
          const parsed = JSON.parse(text);
          if (!Array.isArray(parsed)) throw new Error('Expected JSON array');
          items = parsed;
        }
        if (items.length === 0) throw new Error('No items found');
        setPreview(items);
      } catch (err: any) {
        showToast('Import error: ' + (err.message || 'Invalid file'));
        setPreview(null);
      }
    };
    reader.readAsText(file);
  }

  async function doImport() {
    if (!preview) return;
    setImporting(true);
    setTotal(preview.length);
    setProgress(0);
    setErrors(0);
    let errCount = 0;
    for (let i = 0; i < preview.length; i++) {
      try {
        await ddbRequest('PutItem', { TableName: tableName, Item: preview[i] });
      } catch {
        errCount++;
      }
      setProgress(i + 1);
      setErrors(errCount);
    }
    setImporting(false);
    showToast(`Imported ${preview.length - errCount} items${errCount ? `, ${errCount} failed` : ''}`);
    setShowModal(false);
    setPreview(null);
    setFileName('');
    onComplete();
  }

  return (
    <>
      <button class="btn btn-ghost btn-sm" onClick={() => setShowModal(true)}>
        <UploadIcon /> Import
      </button>
      {showModal && (
        <Modal
          title="Import Items"
          size="md"
          onClose={() => { if (!importing) { setShowModal(false); setPreview(null); setFileName(''); } }}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => { setShowModal(false); setPreview(null); setFileName(''); }} disabled={importing}>Cancel</button>
              <button class="btn btn-primary btn-sm" onClick={doImport} disabled={importing || !preview}>
                {importing ? `Importing ${progress}/${total}...` : `Import${preview ? ` (${preview.length} items)` : ''}`}
              </button>
            </>
          }
        >
          <div style="margin-bottom:16px">
            <input
              ref={fileRef}
              type="file"
              accept=".json,.csv"
              style="display:none"
              onChange={handleFileSelect}
            />
            <button class="btn btn-ghost" onClick={() => fileRef.current?.click()}>
              <UploadIcon /> Choose File (JSON or CSV)
            </button>
            {fileName && <span style="margin-left:12px;font-size:13px;color:var(--n500)">{fileName}</span>}
          </div>

          {importing && (
            <div style="margin-bottom:16px">
              <div style="background:var(--n200);border-radius:4px;height:8px;overflow:hidden">
                <div style={`background:var(--primary-green);height:100%;width:${(progress / total) * 100}%;transition:width 0.2s`} />
              </div>
              <div style="font-size:12px;color:var(--n500);margin-top:4px">{progress} / {total} items{errors > 0 ? ` (${errors} errors)` : ''}</div>
            </div>
          )}

          {preview && !importing && (
            <div>
              <div style="font-size:13px;color:var(--n600);margin-bottom:8px">
                Preview: {preview.length} items from <strong>{fileName}</strong>
              </div>
              <div style="max-height:200px;overflow:auto;background:var(--n900);color:var(--n300);padding:12px;border-radius:var(--radius-md);font-family:var(--font-mono);font-size:12px">
                {JSON.stringify(preview.slice(0, 3), null, 2)}
                {preview.length > 3 && `\n... and ${preview.length - 3} more`}
              </div>
            </div>
          )}
        </Modal>
      )}
    </>
  );
}
