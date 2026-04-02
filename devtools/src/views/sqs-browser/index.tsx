import { useState, useEffect, useCallback, useMemo, useRef } from 'preact/hooks';
import { getAdminBase } from '../../lib/api';
import './sqs-browser.css';

// ---------- Types ----------

interface SQSQueue {
  url: string;
  name: string;
  messageCount: number;
}

interface SQSMessage {
  MessageId: string;
  Body: string;
  ReceiptHandle: string;
  MD5OfBody?: string;
  Attributes?: Record<string, string>;
}

// ---------- Helpers ----------

function gwBase(): string {
  return getAdminBase().replace(':4599', ':4566');
}

async function sqsRequest(action: string, body: any) {
  const res = await fetch(gwBase(), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-amz-json-1.0',
      'X-Amz-Target': `AmazonSQS.${action}`,
      'Authorization': 'AWS4-HMAC-SHA256 Credential=test/20260321/us-east-1/sqs/aws4_request, SignedHeaders=host, Signature=fake',
    },
    body: JSON.stringify(body),
  });
  return res.json();
}

function queueNameFromUrl(url: string): string {
  const parts = url.split('/');
  return parts[parts.length - 1] || url;
}

function prettyPrintBody(body: string): string {
  try {
    return JSON.stringify(JSON.parse(body), null, 2);
  } catch {
    return body;
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

function PlayIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
      <polygon points="5 3 19 12 5 21 5 3" />
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

// ---------- Modal ----------

interface ModalProps {
  title: string;
  onClose: () => void;
  footer?: preact.ComponentChildren;
  children: preact.ComponentChildren;
}

function Modal({ title, onClose, footer, children }: ModalProps) {
  return (
    <div class="sqs-modal-backdrop" onClick={onClose}>
      <div class="sqs-modal" onClick={(e) => e.stopPropagation()}>
        <div class="sqs-modal-header">
          <h3>{title}</h3>
          <button class="btn-icon" onClick={onClose}><XIcon /></button>
        </div>
        <div class="sqs-modal-body">{children}</div>
        {footer && <div class="sqs-modal-footer">{footer}</div>}
      </div>
    </div>
  );
}

// ---------- Main Component ----------

export function SQSBrowserView() {
  const [queues, setQueues] = useState<SQSQueue[]>([]);
  const [selectedQueue, setSelectedQueue] = useState<SQSQueue | null>(null);
  const [messages, setMessages] = useState<SQSMessage[]>([]);
  const [queueSearch, setQueueSearch] = useState('');
  const [showCreateQueue, setShowCreateQueue] = useState(false);
  const [newQueueName, setNewQueueName] = useState('');
  const [showSendMessage, setShowSendMessage] = useState(false);
  const [sendBody, setSendBody] = useState('');
  const [purgeConfirm, setPurgeConfirm] = useState(false);
  const [deleteQueueConfirm, setDeleteQueueConfirm] = useState<string | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [loading, setLoading] = useState(false);
  const autoRefreshRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // ---------- Data loading ----------

  const loadQueues = useCallback(async () => {
    try {
      const res = await sqsRequest('ListQueues', {});
      const urls: string[] = res.QueueUrls || [];
      const queueList: SQSQueue[] = [];
      for (const url of urls) {
        try {
          const attrs = await sqsRequest('GetQueueAttributes', {
            QueueUrl: url,
            AttributeNames: ['ApproximateNumberOfMessages'],
          });
          queueList.push({
            url,
            name: queueNameFromUrl(url),
            messageCount: parseInt(attrs.Attributes?.ApproximateNumberOfMessages || '0', 10),
          });
        } catch {
          queueList.push({ url, name: queueNameFromUrl(url), messageCount: 0 });
        }
      }
      setQueues(queueList);
    } catch {
      setQueues([]);
    }
  }, []);

  useEffect(() => { loadQueues(); }, [loadQueues]);

  // Auto-refresh
  useEffect(() => {
    if (autoRefresh && selectedQueue) {
      autoRefreshRef.current = setInterval(() => {
        receiveMessages(selectedQueue);
      }, 3000);
      return () => { if (autoRefreshRef.current) clearInterval(autoRefreshRef.current); };
    } else {
      if (autoRefreshRef.current) clearInterval(autoRefreshRef.current);
    }
  }, [autoRefresh, selectedQueue]);

  async function receiveMessages(queue: SQSQueue) {
    setLoading(true);
    try {
      const res = await sqsRequest('ReceiveMessage', {
        QueueUrl: queue.url,
        MaxNumberOfMessages: 10,
        WaitTimeSeconds: 0,
        AttributeNames: ['All'],
        VisibilityTimeout: 0,
      });
      const msgs: SQSMessage[] = res.Messages || [];
      setMessages(prev => {
        const existingIds = new Set(prev.map(m => m.MessageId));
        const newMsgs = msgs.filter(m => !existingIds.has(m.MessageId));
        return [...newMsgs, ...prev];
      });
    } catch {
      // silently fail
    }
    setLoading(false);
  }

  // ---------- Actions ----------

  function selectQueue(queue: SQSQueue) {
    setSelectedQueue(queue);
    setMessages([]);
    setAutoRefresh(false);
  }

  async function createQueue() {
    if (!newQueueName.trim()) return;
    try {
      await sqsRequest('CreateQueue', { QueueName: newQueueName.trim() });
      setShowCreateQueue(false);
      setNewQueueName('');
      loadQueues();
    } catch {
      // silently fail
    }
  }

  async function deleteQueue(url: string) {
    try {
      await sqsRequest('DeleteQueue', { QueueUrl: url });
      setDeleteQueueConfirm(null);
      if (selectedQueue?.url === url) {
        setSelectedQueue(null);
        setMessages([]);
      }
      loadQueues();
    } catch {
      setDeleteQueueConfirm(null);
    }
  }

  async function sendMessage() {
    if (!selectedQueue || !sendBody.trim()) return;
    try {
      await sqsRequest('SendMessage', {
        QueueUrl: selectedQueue.url,
        MessageBody: sendBody,
      });
      setShowSendMessage(false);
      setSendBody('');
      loadQueues();
    } catch {
      // silently fail
    }
  }

  async function purgeQueue() {
    if (!selectedQueue) return;
    try {
      await sqsRequest('PurgeQueue', { QueueUrl: selectedQueue.url });
      setPurgeConfirm(false);
      setMessages([]);
      loadQueues();
    } catch {
      setPurgeConfirm(false);
    }
  }

  async function deleteMessage(msg: SQSMessage) {
    if (!selectedQueue) return;
    try {
      await sqsRequest('DeleteMessage', {
        QueueUrl: selectedQueue.url,
        ReceiptHandle: msg.ReceiptHandle,
      });
      setMessages(prev => prev.filter(m => m.MessageId !== msg.MessageId));
      loadQueues();
    } catch {
      // silently fail
    }
  }

  // ---------- Derived state ----------

  const displayQueues = useMemo(() => {
    if (!queueSearch) return queues;
    const q = queueSearch.toLowerCase();
    return queues.filter(queue => queue.name.toLowerCase().includes(q));
  }, [queues, queueSearch]);

  // ---------- Render ----------

  return (
    <div class="sqs-layout">
      {/* Queue sidebar */}
      <div class="sqs-sidebar">
        <div class="sqs-sidebar-header">
          <div class="flex items-center justify-between mb-4">
            <span style="font-weight:700;font-size:15px">Queues</span>
            <div class="flex gap-2">
              <button class="btn-icon" title="Refresh" onClick={loadQueues}
                style="border:1px solid var(--border-default);border-radius:var(--radius-md)">
                <RefreshIcon />
              </button>
              <button class="btn-icon" title="Create Queue" onClick={() => setShowCreateQueue(true)}
                style="border:1px solid var(--border-default);border-radius:var(--radius-md)">
                <PlusIcon />
              </button>
            </div>
          </div>
          <div class="search-wrap">
            <SearchIcon class="search-icon" />
            <input class="input w-full search-input" placeholder="Filter queues..." value={queueSearch}
              onInput={(e) => setQueueSearch((e.target as HTMLInputElement).value)} />
          </div>
        </div>
        <div class="sqs-sidebar-list">
          {displayQueues.map(q => (
            <div key={q.url}
              class={`sqs-queue-item ${selectedQueue?.url === q.url ? 'active' : ''}`}
              onClick={() => selectQueue(q)}>
              <span class="name" title={q.name}>{q.name}</span>
              <span class="count">{q.messageCount}</span>
            </div>
          ))}
          {displayQueues.length === 0 && (
            <div style="padding:24px;text-align:center;font-size:13px;color:var(--text-tertiary)">
              No queues found
            </div>
          )}
        </div>
      </div>

      {/* Message inspector */}
      <div class="sqs-main">
        {!selectedQueue ? (
          <div class="sqs-empty-state">
            <div style="font-size:48px;opacity:0.3">SQS</div>
            <div style="margin-top:12px;font-size:16px;font-weight:500">Select a queue to inspect messages</div>
            <div style="margin-top:4px;font-size:13px;color:var(--text-tertiary)">Or create a new queue to get started</div>
          </div>
        ) : (
          <div>
            <div class="sqs-header">
              <div>
                <h2 style="font-size:20px;font-weight:700;margin-bottom:4px">{selectedQueue.name}</h2>
                <div style="font-size:13px;color:var(--text-secondary);font-family:var(--font-mono)">{selectedQueue.url}</div>
              </div>
              <div class="flex gap-2" style="align-items:center">
                <button class="btn btn-primary btn-sm" onClick={() => receiveMessages(selectedQueue)}>
                  <PlayIcon /> Receive
                </button>
                <button class="btn btn-ghost btn-sm" onClick={() => setShowSendMessage(true)}>
                  <PlusIcon /> Send
                </button>
                <label class="sqs-auto-refresh">
                  <input type="checkbox" checked={autoRefresh}
                    onChange={(e) => setAutoRefresh((e.target as HTMLInputElement).checked)} />
                  Auto-refresh
                </label>
                <button class="btn btn-danger btn-sm" onClick={() => setPurgeConfirm(true)} title="Purge Queue">
                  Purge
                </button>
                <button class="btn btn-danger btn-sm" onClick={() => setDeleteQueueConfirm(selectedQueue.url)}>
                  <TrashIcon />
                </button>
              </div>
            </div>

            {loading && messages.length === 0 && (
              <div style="padding:32px;text-align:center;color:var(--text-tertiary)">Polling...</div>
            )}

            <div style="display:flex;flex-direction:column;gap:8px">
              {messages.map(msg => (
                <div key={msg.MessageId} class="sqs-message-card">
                  <div class="sqs-message-header">
                    <div>
                      <span style="font-weight:600;font-size:13px">ID: </span>
                      <span style="font-family:var(--font-mono);font-size:12px;color:var(--text-secondary)">{msg.MessageId}</span>
                    </div>
                    <div class="flex gap-2" style="align-items:center">
                      {msg.Attributes?.SentTimestamp && (
                        <span style="font-size:11px;color:var(--text-tertiary)">
                          {new Date(parseInt(msg.Attributes.SentTimestamp, 10)).toLocaleString()}
                        </span>
                      )}
                      <button class="btn-icon" style="color:var(--error)" onClick={() => deleteMessage(msg)} title="Delete">
                        <TrashIcon />
                      </button>
                    </div>
                  </div>
                  <div class="sqs-message-body">
                    <pre>{prettyPrintBody(msg.Body)}</pre>
                  </div>
                </div>
              ))}
              {messages.length === 0 && !loading && (
                <div style="text-align:center;padding:48px;color:var(--text-tertiary);font-size:13px">
                  Click "Receive" to poll messages from the queue
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Create Queue Modal */}
      {showCreateQueue && (
        <Modal title="Create Queue" onClose={() => setShowCreateQueue(false)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setShowCreateQueue(false)}>Cancel</button>
              <button class="btn btn-primary btn-sm" onClick={createQueue}>Create</button>
            </>
          }>
          <div class="mb-4">
            <div class="label">Queue Name</div>
            <input class="input w-full" placeholder="my-queue" value={newQueueName}
              onInput={(e) => setNewQueueName((e.target as HTMLInputElement).value)}
              onKeyDown={(e) => { if (e.key === 'Enter') createQueue(); }} />
          </div>
        </Modal>
      )}

      {/* Send Message Modal */}
      {showSendMessage && (
        <Modal title="Send Message" onClose={() => setShowSendMessage(false)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setShowSendMessage(false)}>Cancel</button>
              <button class="btn btn-primary btn-sm" onClick={sendMessage}>Send</button>
            </>
          }>
          <div class="mb-4">
            <div class="label">Message Body</div>
            <textarea class="input w-full" rows={10} placeholder='{"key": "value"}' value={sendBody}
              onInput={(e) => setSendBody((e.target as HTMLTextAreaElement).value)}
              style="font-family:var(--font-mono);font-size:13px;resize:vertical" />
          </div>
        </Modal>
      )}

      {/* Purge Confirm */}
      {purgeConfirm && (
        <Modal title="Purge Queue" onClose={() => setPurgeConfirm(false)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setPurgeConfirm(false)}>Cancel</button>
              <button class="btn btn-danger btn-sm" onClick={purgeQueue}>Purge</button>
            </>
          }>
          <p>Are you sure you want to purge all messages from <strong>{selectedQueue?.name}</strong>?</p>
        </Modal>
      )}

      {/* Delete Queue Confirm */}
      {deleteQueueConfirm && (
        <Modal title="Delete Queue" onClose={() => setDeleteQueueConfirm(null)}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => setDeleteQueueConfirm(null)}>Cancel</button>
              <button class="btn btn-danger btn-sm" onClick={() => deleteQueue(deleteQueueConfirm)}>Delete</button>
            </>
          }>
          <p>Are you sure you want to delete <strong>{queueNameFromUrl(deleteQueueConfirm)}</strong>?</p>
        </Modal>
      )}
    </div>
  );
}
