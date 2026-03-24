import { useState, useEffect, useCallback, useMemo } from 'preact/hooks';
import { GW_BASE } from '../api';
import { Modal } from '../components/Modal';
import { PlusIcon, RefreshIcon, TrashIcon, UploadIcon, SearchIcon } from '../components/Icons';

interface S3BrowserProps {
  showToast: (msg: string) => void;
}

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

async function s3Request(method: string, path: string, body?: any, extraHeaders?: Record<string, string>) {
  const res = await fetch(`${GW_BASE}${path}`, {
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

export function S3BrowserPage({ showToast }: S3BrowserProps) {
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

      // Get object counts for each bucket
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
      if (pfx) {
        params.set('prefix', pfx);
      }
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
      showToast('Failed to list objects');
    }
    setLoading(false);
  }, [showToast]);

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
      showToast(`Bucket "${newBucketName.trim()}" created`);
      setShowCreateBucket(false);
      setNewBucketName('');
      loadBuckets();
    } catch {
      showToast('Failed to create bucket');
    }
  }

  async function deleteBucket(name: string) {
    try {
      await s3Request('DELETE', `/${name}`);
      showToast(`Bucket "${name}" deleted`);
      setDeleteBucketConfirm(null);
      if (selectedBucket === name) {
        setSelectedBucket(null);
        setObjects([]);
      }
      loadBuckets();
    } catch {
      showToast('Failed to delete bucket (must be empty)');
      setDeleteBucketConfirm(null);
    }
  }

  async function deleteObject(key: string) {
    if (!selectedBucket) return;
    try {
      await s3Request('DELETE', `/${selectedBucket}/${key}`);
      showToast('Object deleted');
      setDeleteObjectConfirm(null);
      loadObjects(selectedBucket, prefix);
      loadBuckets();
    } catch {
      showToast('Failed to delete object');
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
      showToast('Failed to download object');
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
        showToast(`Uploaded "${file.name}"`);
        loadObjects(selectedBucket, prefix);
        loadBuckets();
      } catch {
        showToast('Upload failed');
      }
    };
    input.click();
  }

  const filteredBuckets = useMemo(() => {
    if (!search && !selectedBucket) return buckets;
    return buckets;
  }, [buckets, search, selectedBucket]);

  const [bucketSearch, setBucketSearch] = useState('');
  const displayBuckets = useMemo(() => {
    if (!bucketSearch) return filteredBuckets;
    const q = bucketSearch.toLowerCase();
    return filteredBuckets.filter(b => b.name.toLowerCase().includes(q));
  }, [filteredBuckets, bucketSearch]);

  const breadcrumbParts = prefix.split('/').filter(Boolean);
  const isFolder = (obj: S3Object) => obj.size === -1;
  const displayName = (obj: S3Object) => {
    const stripped = obj.key.replace(prefix, '');
    return stripped;
  };

  return (
    <div class="ddb-layout">
      {/* Bucket sidebar */}
      <div class="ddb-sidebar">
        <div class="ddb-sidebar-header">
          <div class="flex items-center justify-between mb-4">
            <span style="font-weight:700;font-size:15px">Buckets</span>
            <div class="flex gap-2">
              <button class="btn-icon btn-sm btn-ghost" title="Refresh" onClick={loadBuckets}
                style="border:1px solid var(--border-default);border-radius:var(--radius-md)">
                <RefreshIcon />
              </button>
              <button class="btn-icon btn-sm btn-ghost" title="Create Bucket" onClick={() => setShowCreateBucket(true)}
                style="border:1px solid var(--border-default);border-radius:var(--radius-md)">
                <PlusIcon />
              </button>
            </div>
          </div>
          <div style="position:relative">
            <SearchIcon style="position:absolute;left:8px;top:50%;transform:translateY(-50%);color:var(--text-tertiary)" />
            <input class="input w-full" placeholder="Filter buckets..." value={bucketSearch}
              onInput={(e) => setBucketSearch((e.target as HTMLInputElement).value)}
              style="padding-left:30px;font-size:13px" />
          </div>
        </div>
        <div class="ddb-sidebar-list">
          {displayBuckets.map(b => (
            <div key={b.name}
              class={`ddb-table-item ${selectedBucket === b.name ? 'active' : ''}`}
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
      <div class="ddb-main">
        {!selectedBucket ? (
          <div class="empty-state">
            <div style="font-size:48px;opacity:0.3">S3</div>
            <div style="margin-top:12px;font-size:16px;font-weight:500">Select a bucket to browse objects</div>
            <div style="margin-top:4px;font-size:13px;color:var(--text-tertiary)">Or create a new bucket to get started</div>
          </div>
        ) : (
          <div>
            <div class="ddb-header">
              <div>
                <h2 style="font-size:20px;font-weight:700;margin-bottom:4px">{selectedBucket}</h2>
                {/* Breadcrumb */}
                <div style="display:flex;gap:4px;align-items:center;font-size:13px;color:var(--text-secondary)">
                  <span style="cursor:pointer;color:var(--brand-blue)" onClick={() => navigateBreadcrumb(-1)}>/</span>
                  {breadcrumbParts.map((part, i) => (
                    <span key={i}>
                      <span style="color:var(--text-tertiary)">/</span>
                      <span style="cursor:pointer;color:var(--brand-blue)" onClick={() => navigateBreadcrumb(i)}>{part}</span>
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

            <div style="margin-bottom:12px;position:relative">
              <SearchIcon style="position:absolute;left:8px;top:50%;transform:translateY(-50%);color:var(--text-tertiary)" />
              <input class="input w-full" placeholder="Filter objects..." value={search}
                onInput={(e) => setSearch((e.target as HTMLInputElement).value)}
                style="padding-left:30px;font-size:13px" />
            </div>

            {loading ? (
              <div style="padding:32px;text-align:center;color:var(--text-tertiary)">Loading...</div>
            ) : (
              <table class="data-table" style="width:100%">
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
                  {objects
                    .filter(o => !search || displayName(o).toLowerCase().includes(search.toLowerCase()))
                    .map(obj => (
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
                          {!isFolder(obj) && (
                            <button class="btn btn-danger btn-sm" style="padding:2px 6px;font-size:11px"
                              onClick={() => setDeleteObjectConfirm(obj.key)}>
                              <TrashIcon />
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  {objects.length === 0 && (
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
        <Modal title="Create Bucket" size="sm" onClose={() => setShowCreateBucket(false)}
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
        <Modal title="Delete Bucket" size="sm" onClose={() => setDeleteBucketConfirm(null)}
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
        <Modal title="Delete Object" size="sm" onClose={() => setDeleteObjectConfirm(null)}
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
