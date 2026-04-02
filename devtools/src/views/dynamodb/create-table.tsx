import { useState } from 'preact/hooks';
import { ddbRequest } from './utils';
import { XIcon } from '../../components/icons';

interface CreateTableProps {
  onClose: () => void;
  onCreated: () => void;
  showToast: (msg: string) => void;
}

export function CreateTable({ onClose, onCreated, showToast }: CreateTableProps) {
  const [tableName, setTableName] = useState('');
  const [pkName, setPkName] = useState('id');
  const [pkType, setPkType] = useState('S');
  const [skName, setSkName] = useState('');
  const [skType, setSkType] = useState('S');
  const [billingMode, setBillingMode] = useState('PAY_PER_REQUEST');
  const [creating, setCreating] = useState(false);

  async function handleCreate() {
    if (!tableName) return;
    setCreating(true);
    try {
      const params: any = {
        TableName: tableName,
        KeySchema: [{ AttributeName: pkName, KeyType: 'HASH' }],
        AttributeDefinitions: [{ AttributeName: pkName, AttributeType: pkType }],
        BillingMode: billingMode,
      };
      if (skName) {
        params.KeySchema.push({ AttributeName: skName, KeyType: 'RANGE' });
        params.AttributeDefinitions.push({ AttributeName: skName, AttributeType: skType });
      }
      if (billingMode === 'PROVISIONED') {
        params.ProvisionedThroughput = { ReadCapacityUnits: 5, WriteCapacityUnits: 5 };
      }
      await ddbRequest('CreateTable', params);
      showToast('Table created');
      onCreated();
      onClose();
    } catch {
      showToast('Create table failed');
    } finally {
      setCreating(false);
    }
  }

  return (
    <div class="ddb-modal-backdrop" onClick={onClose}>
      <div class="ddb-modal" onClick={(e) => e.stopPropagation()}>
        <div class="ddb-modal-header">
          <h3>Create Table</h3>
          <button class="btn-icon" onClick={onClose}><XIcon /></button>
        </div>
        <div class="ddb-modal-body">
          <div class="label">Table Name</div>
          <input
            class="input w-full mb-4"
            value={tableName}
            onInput={(e) => setTableName((e.target as HTMLInputElement).value)}
            placeholder="my-table"
          />

          <div class="label">Partition Key</div>
          <div class="field-row mb-4">
            <input
              class="input"
              value={pkName}
              onInput={(e) => setPkName((e.target as HTMLInputElement).value)}
              placeholder="id"
            />
            <select class="select" value={pkType} onChange={(e) => setPkType((e.target as HTMLSelectElement).value)}>
              <option value="S">String (S)</option>
              <option value="N">Number (N)</option>
              <option value="B">Binary (B)</option>
            </select>
          </div>

          <div class="label">Sort Key (optional)</div>
          <div class="field-row mb-4">
            <input
              class="input"
              value={skName}
              onInput={(e) => setSkName((e.target as HTMLInputElement).value)}
              placeholder="sk"
            />
            <select class="select" value={skType} onChange={(e) => setSkType((e.target as HTMLSelectElement).value)}>
              <option value="S">String (S)</option>
              <option value="N">Number (N)</option>
              <option value="B">Binary (B)</option>
            </select>
          </div>

          <div class="label">Billing Mode</div>
          <div class="flex gap-3">
            <label class="flex items-center gap-2" style="cursor:pointer">
              <input
                type="radio"
                name="billing"
                checked={billingMode === 'PAY_PER_REQUEST'}
                onChange={() => setBillingMode('PAY_PER_REQUEST')}
              />
              <span class="text-sm">On-Demand</span>
            </label>
            <label class="flex items-center gap-2" style="cursor:pointer">
              <input
                type="radio"
                name="billing"
                checked={billingMode === 'PROVISIONED'}
                onChange={() => setBillingMode('PROVISIONED')}
              />
              <span class="text-sm">Provisioned</span>
            </label>
          </div>
        </div>
        <div class="ddb-modal-footer">
          <button class="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
          <button class="btn btn-primary btn-sm" onClick={handleCreate} disabled={!tableName || creating}>
            {creating ? 'Creating...' : 'Create Table'}
          </button>
        </div>
      </div>
    </div>
  );
}
