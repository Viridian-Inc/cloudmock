import { DDBAttributeValue, DDBItem, DDBType, FormAttribute } from './types';

/** Extract human-readable value from a DDB typed attribute. */
export function extractValue(val: DDBAttributeValue | undefined): string {
  if (!val) return '';
  if (val.S !== undefined) return val.S;
  if (val.N !== undefined) return val.N;
  if (val.BOOL !== undefined) return String(val.BOOL);
  if (val.NULL) return 'null';
  if (val.L) return `[${val.L.length} items]`;
  if (val.M) return `{${Object.keys(val.M).length} keys}`;
  if (val.SS) return val.SS.join(', ');
  if (val.NS) return val.NS.join(', ');
  if (val.B) return '(binary)';
  if (val.BS) return `(${val.BS.length} binaries)`;
  return JSON.stringify(val);
}

/** Get the DDB type key from an attribute value. */
export function getType(val: DDBAttributeValue | undefined): DDBType {
  if (!val) return 'S';
  if (val.S !== undefined) return 'S';
  if (val.N !== undefined) return 'N';
  if (val.BOOL !== undefined) return 'BOOL';
  if (val.NULL) return 'NULL';
  if (val.L) return 'L';
  if (val.M) return 'M';
  if (val.SS) return 'SS';
  if (val.NS) return 'NS';
  return 'S';
}

/** Type badge colors. */
export function typeBadgeColor(type: string): { bg: string; fg: string } {
  switch (type) {
    case 'S': return { bg: 'rgba(2,150,98,0.1)', fg: '#029662' };
    case 'N': return { bg: 'rgba(9,127,245,0.1)', fg: '#097FF5' };
    case 'BOOL': return { bg: 'rgba(167,139,250,0.15)', fg: '#7C3AED' };
    case 'NULL': return { bg: 'rgba(148,163,184,0.15)', fg: '#64748B' };
    case 'L': return { bg: 'rgba(255,154,75,0.12)', fg: '#D97706' };
    case 'M': return { bg: 'rgba(254,195,7,0.15)', fg: '#B8860B' };
    case 'SS': return { bg: 'rgba(2,150,98,0.15)', fg: '#029662' };
    case 'NS': return { bg: 'rgba(9,127,245,0.15)', fg: '#097FF5' };
    default: return { bg: 'rgba(148,163,184,0.1)', fg: '#64748B' };
  }
}

/** Collect all attribute names from a list of items. */
export function collectColumns(items: DDBItem[], keyAttrs: string[]): string[] {
  const cols = new Set<string>();
  items.forEach(item => Object.keys(item).forEach(k => cols.add(k)));
  const keyCols = keyAttrs.filter(k => cols.has(k));
  const rest = [...cols].filter(k => !keyAttrs.includes(k)).sort();
  return [...keyCols, ...rest];
}

/** Build a DDB AttributeValue from type + string value. */
export function buildAttributeValue(type: DDBType, value: string): DDBAttributeValue {
  switch (type) {
    case 'S': return { S: value };
    case 'N': return { N: value };
    case 'BOOL': return { BOOL: value === 'true' };
    case 'NULL': return { NULL: true };
    case 'L': try { return JSON.parse(value); } catch { return { L: [] }; }
    case 'M': try { return JSON.parse(value); } catch { return { M: {} }; }
    case 'SS': return { SS: value.split(',').map(s => s.trim()).filter(Boolean) };
    case 'NS': return { NS: value.split(',').map(s => s.trim()).filter(Boolean) };
    default: return { S: value };
  }
}

/** Convert a DDB item to form attributes. */
export function itemToFormAttrs(item: DDBItem): FormAttribute[] {
  return Object.entries(item).map(([key, val]) => ({
    key,
    type: getType(val),
    value: formatAttrForForm(val),
    listItems: val.L ? val.L.map((v, i) => ({ key: String(i), type: getType(v), value: formatAttrForForm(v) })) : undefined,
    mapItems: val.M ? Object.entries(val.M).map(([k, v]) => ({ key: k, type: getType(v), value: formatAttrForForm(v) })) : undefined,
  }));
}

function formatAttrForForm(val: DDBAttributeValue): string {
  if (val.S !== undefined) return val.S;
  if (val.N !== undefined) return val.N;
  if (val.BOOL !== undefined) return String(val.BOOL);
  if (val.NULL) return '';
  if (val.L) return JSON.stringify(val.L, null, 2);
  if (val.M) return JSON.stringify(val.M, null, 2);
  if (val.SS) return val.SS.join(', ');
  if (val.NS) return val.NS.join(', ');
  return JSON.stringify(val);
}

/** Convert form attributes back to a DDB item. */
export function formAttrsToItem(attrs: FormAttribute[]): DDBItem {
  const item: DDBItem = {};
  for (const attr of attrs) {
    if (!attr.key) continue;
    item[attr.key] = buildAttributeValue(attr.type, attr.value);
  }
  return item;
}

/** Format bytes to human-readable. */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

/** Format a Unix timestamp. */
export function formatDate(ts: number | undefined): string {
  if (!ts) return 'N/A';
  return new Date(ts * 1000).toLocaleString();
}

/** Generate items as JSON for export. */
export function exportAsJSON(items: DDBItem[]): string {
  return JSON.stringify(items, null, 2);
}

/** Generate items as CSV for export. */
export function exportAsCSV(items: DDBItem[]): string {
  if (items.length === 0) return '';
  const allKeys = new Set<string>();
  items.forEach(item => Object.keys(item).forEach(k => allKeys.add(k)));
  const headers = [...allKeys];
  const rows = items.map(item =>
    headers.map(h => {
      const v = extractValue(item[h]);
      // Escape CSV
      if (v.includes(',') || v.includes('"') || v.includes('\n')) {
        return `"${v.replace(/"/g, '""')}"`;
      }
      return v;
    }).join(',')
  );
  return [headers.join(','), ...rows].join('\n');
}

/** Download a string as a file. */
export function downloadFile(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
