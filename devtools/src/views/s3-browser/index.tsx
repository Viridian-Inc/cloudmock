import { useState, useEffect, useCallback, useMemo } from 'preact/hooks';
import { getAdminBase } from '../../lib/api';
import './s3-browser.css';

// ---------- Types ----------

interface S3Bucket {
  name: string;
  creationDate?: string;
}

interface S3Object {
  key: string;
  size: number;
  lastModified: string;
  etag: string;
}

// ---------- Helpers ----------

function gwBase(): string {
  return getAdminBase().replace(':4599', ':4566');
}

async function s3Request(method: string, path: string, body?: any, extraHeaders?: Record<string, string>) {
  const res = await fetch(`${gwBase()}${path}`, {
    method,
    headers: {
      'Authorization': 'AWS4-HMAC-SHA256 Credential=test/20260321/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=fake',
      ...extraHeaders,
    },
    body,
  });
  return res;
}

function parseXml(text: string): Document {
  return new DOMParser().parseFromString(text, 'text/xml');
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatDate(dateStr: string): string {
  try {
    return new Date(dateStr).toLocaleString();
  } catch {
    return dateStr;
  }
}

// ---------- Inline Icons ----------

function SearchIcon(props: any) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14" {...props}>
      <circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
  );
}

function RefreshIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <polyline points="23 4 23 10 17 10" /><polyline points="1 20 1 14 7 14" />
      <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
    </svg>
  );
}

function PlusIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <polyline points="3 6 5 6 21 6" /><path d="M19 6l-1.5 14a2 2 0 0 1-2 2h-7a2 2 0 0 1-2-2L5 6" />
      <path d="M10 11v6" /><path d="M14 11v6" /><path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
    </svg>
  );
}

function UploadIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="17 8 12 3 7 8" />
      <line x1="12" y1="3" x2="12" y2="15" />
    </svg>
  );
}

function XIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  );
}

function DownloadIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="7 10 12 15 17 10" />
      <line x1="12" y1="15" x2="12" y2="3" />
    </svg>
  );
}

// ---------- Modal ----------

interface ModalProps {
  title: string;
  onClose: () => void;
  footer?: preact.ComponentChildren;
  children: preact.ComponentChildren;
}

function Modal({ title, onClose, footer, children }: ModalProps) {
  return (
    <div class="s3-modal-backdrop" onClick={onClose}>
      <div class="s3-modal" onClick={(e) => e.stopPropagation()}>
        <div class="s3-modal-header">
          <h3>{title}</h3>
          <button class="btn-icon" onClick={onClose}><XIcon /></button>
        </div>
        <div class="s3-modal-body">{children}</div>
        {footer && <div class="s3-modal-footer">{footer}</div>}
      </div>
    </div>
  );
}

// ---------- Main Component ----------

export function S3BrowserView() {
  const [buckets, setBuckets] = useState<S3Bucket[]>([]);
  const [bucketCounts, setBucketCounts] = useState<Record<string, number>>({});
  const [selectedBucket, setSelectedBucket] = useState<string | null>(null);
  const [objects, setObjects] = useState<S3Object[]>([]);
  const [prefix, setPrefix] = useState('');
  const [search, setSearch] = useState('');
  const [showCreateBucket, setShowCreateBucket] = useState(false);
  const [newBucketName, setNewBucketName] = useState('');
  const [deleteBucketConfirm, setDeleteBucketConfirm] = useState<string | null>(null);
  const [deleteObjectConfirm, setDeleteObjectConfirm] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [bucketSearch, setBucketSearch] = useState('');

  // ---------- Data loading ----------

  const loadBuckets = useCallback(async () => {
    try {
      const res = await s3Request('GET', '/');
      const text = await res.text();
      const doc = parseXml(text);
      const bucketNodes = doc.querySelectorAll('Bucket');
      const list: S3Bucket[] = [];
      bucketNodes.forEach(node => {
        const name = node.querySelector('Name')?.textContent || '';
        const creationDate = node.querySelector('CreationDate')?.textContent || '';
        if (name) list.push({ name, creationDate });
      });
      setBuckets(list);

      const counts: Record<string, number> = {};
      for (const b of list) {
        try {
          const objRes = await s3Request('GET', `/${b.name}?list-type=2`);
          const objText = await objRes.text();
          const objDoc = parseXml(objText);
          const keyCount = objDoc.querySelector('KeyCount')?.textContent;
          counts[b.name] = keyCount ? parseInt(keyCount, 10) : 0;
        } catch {
          counts[b.name] = 0;
        }
      }
      setBucketCounts(counts);
    } catch {
      setBuckets([]);
    }
  }, []);

  useEffect(() => { loadBuckets(); }, [loadBuckets]);

  const loadObjects = useCallback(async (bucket: string, pfx: string) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ 'list-type': '2' });
      if (pfx) params.set('prefix', pfx);
      params.set('delimiter', '/');
      const res = await s3Request('GET', `/${bucket}?${params}`);
      const text = await res.text();
      const doc = parseXml(text);
      const items: S3Object[] = [];

      // Common prefixes (folders)
      doc.querySelectorAll('CommonPrefixes > Prefix').forEach(node => {
        const key = node.textContent || '';
        if (key) items.push({ key, size: -1, lastModified: '', etag: '' });
      });

      // Objects
      doc.querySelectorAll('Contents').forEach(node => {
        const key = node.querySelector('Key')?.textContent || '';
        const size = parseInt(node.querySelector('Size')?.textContent || '0', 10);
        const lastModified = node.querySelector('LastModified')?.textContent || '';
        const etag = node.querySelector('ETag')?.textContent || '';
        if (key && key !== pfx) items.push({ key, size, lastModified, etag });
      });

      setObjects(items);
    } catch {
      setObjects([]);
    }
    setLoading(false);
  }, []);

  // ---------- Actions ----------

  function selectBucket(name: string) {
    setSelectedBucket(name);
    setPrefix('');
    setSearch('');
    loadObjects(name, '');
  }

  function navigateToPrefix(pfx: string) {
    setPrefix(pfx);
    setSearch('');
    if (selectedBucket) loadObjects(selectedBucket, pfx);
  }

  function navigateBreadcrumb(index: number) {
    const parts = prefix.split('/').filter(Boolean);
    const newPrefix = index < 0 ? '' : parts.slice(0, index + 1).join('/') + '/';
    navigateToPrefix(newPrefix);
  }

  async function createBucket() {
    if (!newBucketName.trim()) return;
    try {
      await s3Request('PUT', `/${newBucketName.trim()}`);
      setShowCreateBucket(false);
      setNewBucketName('');
      loadBuckets();
    } catch {
      // silently fail
    }
  }

  async function deleteBucket(name: string) {
    try {
      await s3Request('DELETE', `/${name}`);
      setDeleteBucketConfirm(null);
      if (selectedBucket === name) {
        setSelectedBucket(null);
        setObjects([]);
      }
      loadBuckets();
    } catch {
      setDeleteBucketConfirm(null);
    }
  }

  async function deleteObject(key: string) {
    if (!selectedBucket) return;
    try {
      await s3Request('DELETE', `/${selectedBucket}/${key}`);
      setDeleteObjectConfirm(null);
      loadObjects(selectedBucket, prefix);
      loadBuckets();
    } catch {
      setDeleteObjectConfirm(null);
    }
  }

  async function downloadObject(key: string) {
    if (!selectedBucket) return;
    try {
      const res = await s3Request('GET', `/${selectedBucket}/${key}`);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = key.split('/').pop() || key;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      // silently fail
    }
  }

  async function uploadFile() {
    if (!selectedBucket) return;
    const input = document.createElement('input');
    input.type = 'file';
    input.onchange = async () => {
      const file = input.files?.[0];
      if (!file) return;
      const key = prefix + file.name;
      try {
        await s3Request('PUT', `/${selectedBucket}/${key}`, file, {
          'Content-Type': file.type || 'application/octet-stream',
        });
        loadObjects(selectedBucket, prefix);
        loadBuckets();
      } catch {
        // silently fail
      }
    };
    input.click();
  }

  // ---------- Derived state ----------

  const displayBuckets = useMemo(() => {
    if (!bucketSearch) return buckets;
    const q = bucketSearch.toLowerCase();
    return buckets.filter(b => b.name.toLowerCase().includes(q));
  }, [buckets, bucketSearch]);

  const breadcrumbParts = prefix.split('/').filter(Boolean);
  const isFolder = (obj: S3Object) => obj.size === -1;
  const displayName = (obj: S3Object) => obj.key.replace(prefix, '');

  const filteredObjects = useMemo(() => {
    if (!search) return objects;
    const q = search.toLowerCase();
    return objects.filter(o => displayName(o).toLowerCase().includes(q));
  }, [objects, search, prefix]);

  // ---------- Render ----------

  return (
    <div class="s3-layout">
      {/* Bucket sidebar */}
      <div class="s3-sidebar">
        <div class="s3-sidebar-header">
          <div class="flex items-center justify-between mb-4">
            <span style="font-weight:700;font-size:15px">Buckets</span>
            <div class="flex gap-2">
              <button class="btn-icon" title="Refresh" onClick={loadBuckets}
                style="border:1px solid var(--border-default);border-radius:var(--radius-md)">
                <RefreshIcon />
              </button>
              <button class="btn-icon" title="Create Bucket" onClick={() => setShowCreateBucket(true)}
                style="border:1px solid var(--border-default);border-radius:var(--radius-md)">
                <PlusIcon />
              </button>
            </div>
          </div>
          <div class="s3-search-wrap">
            <SearchIcon class="s3-search-icon" />
            <input class="input w-full s3-search-input" placeholder="Filter buckets..." value={bucketSearch}
              onInput={(e) => setBucketSearch((e.target as HTMLInputElement).value)} />
          </div>
        </div>
        <div class="s3-sidebar-list">
          {displayBuckets.map(b => (
            <div key={b.name}
              class={`s3-bucket-item ${selectedBucket === b.name ? 'active' : ''}`}
              onClick={() => selectBucket(b.name)}>
              <span class="name" title={b.name}>{b.name}</span>
              <span class="count">{bucketCounts[b.name] ?? '...'}</span>
            </div>
          ))}
          {displayBuckets.length === 0 && (
            <div style="padding:24px;text-align:center;font-size:13px;color:var(--text-tertiary)">
              No buckets found
            </div>
          )}
        </div>
      </div>

      {/* Object browser */}
      <div class="s3-main">
        {!selectedBucket ? (
          <div class="s3-empty-state">
            <div style="font-size:48px;opacity:0.3">S3</div>
            <div style="margin-top:12px;font-size:16px;font-weight:500">Select a bucket to browse objects</div>
            <div style="margin-top:4px;font-size:13px;color:var(--text-tertiary)">Or create a new bucket to get started</div>
          </div>
        ) : (
          <div>
            <div class="s3-header">
              <div>
                <h2 style="font-size:20px;font-weight:700;margin-bottom:4px">{selectedBucket}</h2>
                <div class="s3-breadcrumb">
                  <span class="s3-breadcrumb-link" onClick={() => navigateBreadcrumb(-1)}>/</span>
                  {breadcrumbParts.map((part, i) => (
                    <span key={i}>
                      <span class="s3-breadcrumb-sep">/</span>
                      <span class="s3-breadcrumb-link" onClick={() => navigateBreadcrumb(i)}>{part}</span>
                    </span>
                  ))}
                </div>
              </div>
              <div class="flex gap-2">
                <button class="btn btn-primary btn-sm" onClick={uploadFile}>
                  <UploadIcon /> Upload
                </button>
                <button class="btn btn-ghost btn-sm" onClick={() => loadObjects(selectedBucket, prefix)}>
                  <RefreshIcon /> Refresh
                </button>
                <button class="btn btn-danger btn-sm" onClick={() => setDeleteBucketConfirm(selectedBucket)}>
                  <TrashIcon />
                </button>
              </div>
            </div>

            <div class="s3-search-wrap">
              <SearchIcon class="s3-search-icon" />
              <input class="input w-full s3-search-input" placeholder="Filter objects..." value={search}
                onInput={(e) => setSearch((e.target as HTMLInputElement).value)} />
            </div>

            {loading ? (
              <div style="padding:32px;text-align:center;color:var(--text-tertiary)">Loading...</div>
            ) : (
              <table class="s3-table">
                <thead>
                  <tr>
                    <th>Key</th>
                    <th>Size</th>
                    <th>Last Modified</th>
                    <th>ETag</th>
                    <th style="width:80px">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredObjects.map(obj => (
                    <tr key={obj.key}>
                      <td>
                        {isFolder(obj) ? (
                          <span style="cursor:pointer;color:var(--brand-blue);font-weight:600"
                            onClick={() => navigateToPrefix(obj.key)}>
                            {displayName(obj)}
                          </span>
                        ) : (
                          <span style="cursor:pointer;color:var(--brand-blue)"
                            onClick={() => downloadObject(obj.key)}>
                            {displayName(obj)}
                          </span>
                        )}
                      </td>
                      <td style="font-family:var(--font-mono);font-size:12px">
                        {isFolder(obj) ? '--' : formatBytes(obj.size)}
                      </td>
                      <td style="font-size:12px;color:var(--text-secondary)">
                        {obj.lastModified ? formatDate(obj.lastModified) : '--'}
                      </td>
                      <td style="font-family:var(--font-mono);font-size:11px;color:var(--text-tertiary);max-width:120px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
                        {obj.etag || '--'}
                      </td>
                      <td>
                        <div class="flex gap-2">
                          {!isFolder(obj) && (
                            <>
                              <button class="btn-icon" title="Download" onClick={() => downloadObject(obj.key)}>
                                <DownloadIcon />
                              </button>
                              <button class="btn-icon" title="Delete" style="color:var(--error)"
                                onClick={() => setDeleteObjectConfirm(obj.key)}>
                                <TrashIcon />
                              </button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                  {filteredObjects.length === 0 && (
                    <tr>
                      <td colSpan={5} style="text-align:center;padding:24px;color:var(--text-tertiary)">
                        {prefix ? 'No objects in this prefix' : 'Bucket is empty'}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            )}
          </div>
        )}
      </div>

      {/* Create Bucket Modal */}
      {showCreateBucket && (
        <Modal title="Create Bucket" onClose={() => setShowCreateBucket(false)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setShowCreateBucket(false)}>Cancel</button>
              <button class="btn btn-primary btn-sm" onClick={createBucket}>Create</button>
            </>
          }>
          <div class="mb-4">
            <div class="label">Bucket Name</div>
            <input class="input w-full" placeholder="my-bucket" value={newBucketName}
              onInput={(e) => setNewBucketName((e.target as HTMLInputElement).value)}
              onKeyDown={(e) => { if (e.key === 'Enter') createBucket(); }} />
          </div>
        </Modal>
      )}

      {/* Delete Bucket Confirm */}
      {deleteBucketConfirm && (
        <Modal title="Delete Bucket" onClose={() => setDeleteBucketConfirm(null)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setDeleteBucketConfirm(null)}>Cancel</button>
              <button class="btn btn-danger btn-sm" onClick={() => deleteBucket(deleteBucketConfirm)}>Delete</button>
            </>
          }>
          <p>Are you sure you want to delete <strong>{deleteBucketConfirm}</strong>? The bucket must be empty.</p>
        </Modal>
      )}

      {/* Delete Object Confirm */}
      {deleteObjectConfirm && (
        <Modal title="Delete Object" onClose={() => setDeleteObjectConfirm(null)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setDeleteObjectConfirm(null)}>Cancel</button>
              <button class="btn btn-danger btn-sm" onClick={() => deleteObject(deleteObjectConfirm)}>Delete</button>
            </>
          }>
          <p>Are you sure you want to delete <strong>{deleteObjectConfirm.split('/').pop()}</strong>?</p>
        </Modal>
      )}
    </div>
  );
}
