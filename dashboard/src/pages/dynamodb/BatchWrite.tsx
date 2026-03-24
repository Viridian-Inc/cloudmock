import { useState } from 'preact/hooks';
import { Modal } from '../../components/Modal';
import { ddbRequest } from '../../api';

interface BatchWriteProps {
  tableName: string;
  showToast: (msg: string) => void;
  onComplete: () => void;
}

export function BatchWrite({ tableName, showToast, onComplete }: BatchWriteProps) {
  const [showModal, setShowModal] = useState(false);
  const [jsonText, setJsonText] = useState('');
  const [writing, setWriting] = useState(false);
  const [progress, setProgress] = useState(0);
  const [total, setTotal] = useState(0);
  const [jsonError, setJsonError] = useState('');

  async function handleWrite() {
    let items: any[];
    try {
      items = JSON.parse(jsonText);
      if (!Array.isArray(items)) throw new Error('Expected a JSON array of items');
      if (items.length === 0) throw new Error('Array is empty');
      setJsonError('');
    } catch (e: any) {
      setJsonError(e.message || 'Invalid JSON');
      return;
    }

    setWriting(true);
    setTotal(items.length);
    setProgress(0);

    // BatchWriteItem accepts max 25 items at a time
    let successCount = 0;
    let failCount = 0;
    for (let i = 0; i < items.length; i += 25) {
      const batch = items.slice(i, i + 25);
      const requestItems = batch.map(item => ({ PutRequest: { Item: item } }));
      try {
        await ddbRequest('BatchWriteItem', {
          RequestItems: { [tableName]: requestItems },
        });
        successCount += batch.length;
      } catch {
        // Fall back to individual PutItem
        for (const item of batch) {
          try {
            await ddbRequest('PutItem', { TableName: tableName, Item: item });
            successCount++;
          } catch {
            failCount++;
          }
        }
      }
      setProgress(Math.min(i + 25, items.length));
    }

    setWriting(false);
    showToast(`Batch write: ${successCount} succeeded${failCount ? `, ${failCount} failed` : ''}`);
    setShowModal(false);
    setJsonText('');
    onComplete();
  }

  return (
    <>
      <button class="btn btn-ghost btn-sm" onClick={() => setShowModal(true)}>
        Batch Write
      </button>
      {showModal && (
        <Modal
          title="Batch Write Items"
          size="lg"
          onClose={() => { if (!writing) { setShowModal(false); setJsonText(''); setJsonError(''); } }}
          footer={
            <>
              <button class="btn btn-ghost btn-sm" onClick={() => { setShowModal(false); setJsonText(''); setJsonError(''); }} disabled={writing}>Cancel</button>
              <button class="btn btn-primary btn-sm" onClick={handleWrite} disabled={writing || !jsonText.trim()}>
                {writing ? `Writing ${progress}/${total}...` : 'Write Items'}
              </button>
            </>
          }
        >
          <div style="margin-bottom:8px">
            <div class="label">Paste JSON array of DynamoDB items (with typed attribute values)</div>
          </div>
          <textarea
            class="ddb-json-editor"
            style="min-height:250px"
            value={jsonText}
            onInput={(e) => { setJsonText((e.target as HTMLTextAreaElement).value); setJsonError(''); }}
            placeholder={'[\n  {\n    "pk": { "S": "user#1" },\n    "sk": { "S": "profile" },\n    "name": { "S": "Alice" }\n  }\n]'}
            spellcheck={false}
            disabled={writing}
          />
          {jsonError && (
            <div style="color:var(--error);font-size:13px;margin-top:8px">{jsonError}</div>
          )}
          {writing && (
            <div style="margin-top:12px">
              <div style="background:var(--bg-tertiary);border-radius:4px;height:8px;overflow:hidden">
                <div style={`background:var(--success);height:100%;width:${(progress / total) * 100}%;transition:width 0.2s`} />
              </div>
              <div style="font-size:12px;color:var(--text-secondary);margin-top:4px">{progress} / {total} items</div>
            </div>
          )}
        </Modal>
      )}
    </>
  );
}
