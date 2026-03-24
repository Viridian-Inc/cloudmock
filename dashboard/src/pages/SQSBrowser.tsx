import { useState, useEffect, useCallback, useMemo, useRef } from 'preact/hooks';
import { GW_BASE } from '../api';
import { Modal } from '../components/Modal';
import { PlusIcon, RefreshIcon, TrashIcon, SearchIcon, PlayIcon } from '../components/Icons';

interface SQSBrowserProps {
  showToast: (msg: string) => void;
}

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

async function sqsRequest(action: string, body: any) {
  const res = await fetch(GW_BASE, {
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

export function SQSBrowserPage({ showToast }: SQSBrowserProps) {
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
        // Merge, avoiding duplicates by MessageId
        const existingIds = new Set(prev.map(m => m.MessageId));
        const newMsgs = msgs.filter(m => !existingIds.has(m.MessageId));
        return [...newMsgs, ...prev];
      });
    } catch {
      showToast('Failed to receive messages');
    }
    setLoading(false);
  }

  function selectQueue(queue: SQSQueue) {
    setSelectedQueue(queue);
    setMessages([]);
    setAutoRefresh(false);
  }

  async function createQueue() {
    if (!newQueueName.trim()) return;
    try {
      await sqsRequest('CreateQueue', { QueueName: newQueueName.trim() });
      showToast(`Queue "${newQueueName.trim()}" created`);
      setShowCreateQueue(false);
      setNewQueueName('');
      loadQueues();
    } catch {
      showToast('Failed to create queue');
    }
  }

  async function deleteQueue(url: string) {
    try {
      await sqsRequest('DeleteQueue', { QueueUrl: url });
      showToast('Queue deleted');
      setDeleteQueueConfirm(null);
      if (selectedQueue?.url === url) {
        setSelectedQueue(null);
        setMessages([]);
      }
      loadQueues();
    } catch {
      showToast('Failed to delete queue');
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
      showToast('Message sent');
      setShowSendMessage(false);
      setSendBody('');
      loadQueues();
    } catch {
      showToast('Failed to send message');
    }
  }

  async function purgeQueue() {
    if (!selectedQueue) return;
    try {
      await sqsRequest('PurgeQueue', { QueueUrl: selectedQueue.url });
      showToast('Queue purged');
      setPurgeConfirm(false);
      setMessages([]);
      loadQueues();
    } catch {
      showToast('Failed to purge queue');
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
      showToast('Message deleted');
      loadQueues();
    } catch {
      showToast('Failed to delete message');
    }
  }

  const displayQueues = useMemo(() => {
    if (!queueSearch) return queues;
    const q = queueSearch.toLowerCase();
    return queues.filter(queue => queue.name.toLowerCase().includes(q));
  }, [queues, queueSearch]);

  return (
    <div class="ddb-layout">
      {/* Queue sidebar */}
      <div class="ddb-sidebar">
        <div class="ddb-sidebar-header">
          <div class="flex items-center justify-between mb-4">
            <span style="font-weight:700;font-size:15px">Queues</span>
            <div class="flex gap-2">
              <button class="btn-icon btn-sm btn-ghost" title="Refresh" onClick={loadQueues}
                style="border:1px solid var(--border-default);border-radius:var(--radius-md)">
                <RefreshIcon />
              </button>
              <button class="btn-icon btn-sm btn-ghost" title="Create Queue" onClick={() => setShowCreateQueue(true)}
                style="border:1px solid var(--border-default);border-radius:var(--radius-md)">
                <PlusIcon />
              </button>
            </div>
          </div>
          <div style="position:relative">
            <SearchIcon style="position:absolute;left:8px;top:50%;transform:translateY(-50%);color:var(--text-tertiary)" />
            <input class="input w-full" placeholder="Filter queues..." value={queueSearch}
              onInput={(e) => setQueueSearch((e.target as HTMLInputElement).value)}
              style="padding-left:30px;font-size:13px" />
          </div>
        </div>
        <div class="ddb-sidebar-list">
          {displayQueues.map(q => (
            <div key={q.url}
              class={`ddb-table-item ${selectedQueue?.url === q.url ? 'active' : ''}`}
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
      <div class="ddb-main">
        {!selectedQueue ? (
          <div class="empty-state">
            <div style="font-size:48px;opacity:0.3">SQS</div>
            <div style="margin-top:12px;font-size:16px;font-weight:500">Select a queue to inspect messages</div>
            <div style="margin-top:4px;font-size:13px;color:var(--text-tertiary)">Or create a new queue to get started</div>
          </div>
        ) : (
          <div>
            <div class="ddb-header">
              <div>
                <h2 style="font-size:20px;font-weight:700;margin-bottom:4px">{selectedQueue.name}</h2>
                <div style="font-size:13px;color:var(--text-secondary);font-family:var(--font-mono)">{selectedQueue.url}</div>
              </div>
              <div class="flex gap-2">
                <button class="btn btn-primary btn-sm" onClick={() => receiveMessages(selectedQueue)}>
                  <PlayIcon /> Receive
                </button>
                <button class="btn btn-ghost btn-sm" onClick={() => setShowSendMessage(true)}>
                  <PlusIcon /> Send
                </button>
                <label style="display:flex;align-items:center;gap:6px;font-size:13px;cursor:pointer;padding:0 8px">
                  <input type="checkbox" checked={autoRefresh}
                    onChange={(e) => setAutoRefresh((e.target as HTMLInputElement).checked)} />
                  Auto-refresh
                </label>
                <button class="btn btn-danger btn-sm" onClick={() => setPurgeConfirm(true)}
                  title="Purge Queue">
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
                <div key={msg.MessageId} class="card" style="margin-bottom:0">
                  <div class="card-header" style="padding:10px 16px;display:flex;justify-content:space-between;align-items:center">
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
                      <button class="btn btn-danger btn-sm" style="padding:2px 6px;font-size:11px"
                        onClick={() => deleteMessage(msg)}>
                        <TrashIcon />
                      </button>
                    </div>
                  </div>
                  <div class="card-body" style="padding:10px 16px">
                    <pre style="font-family:var(--font-mono);font-size:12px;white-space:pre-wrap;word-break:break-all;margin:0;max-height:200px;overflow-y:auto">
                      {prettyPrintBody(msg.Body)}
                    </pre>
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
        <Modal title="Create Queue" size="sm" onClose={() => setShowCreateQueue(false)}
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
        <Modal title="Send Message" size="md" onClose={() => setShowSendMessage(false)}
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
        <Modal title="Purge Queue" size="sm" onClose={() => setPurgeConfirm(false)}
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
        <Modal title="Delete Queue" size="sm" onClose={() => setDeleteQueueConfirm(null)}
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
