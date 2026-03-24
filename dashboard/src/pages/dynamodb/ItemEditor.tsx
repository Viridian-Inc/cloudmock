import { useState, useEffect } from 'preact/hooks';
import { Modal } from '../../components/Modal';
import { DDBItem, DDBType, FormAttribute, TableDescription } from './types';
import { itemToFormAttrs, formAttrsToItem, buildAttributeValue, getType } from './utils';
import { syntaxHighlight } from '../../components/JsonView';
import { PlusIcon, TrashIcon, CopyIcon } from '../../components/Icons';

interface ItemEditorProps {
  item: DDBItem | null; // null = create new
  tableDesc: TableDescription;
  onSave: (item: DDBItem) => void;
  onDelete?: (item: DDBItem) => void;
  onDuplicate?: (item: DDBItem) => void;
  onClose: () => void;
}

const DDB_TYPES: DDBType[] = ['S', 'N', 'BOOL', 'NULL', 'L', 'M', 'SS', 'NS'];

export function ItemEditor({ item, tableDesc, onSave, onDelete, onDuplicate, onClose }: ItemEditorProps) {
  const [viewMode, setViewMode] = useState<'json' | 'form'>('form');
  const [jsonText, setJsonText] = useState('');
  const [formAttrs, setFormAttrs] = useState<FormAttribute[]>([]);
  const [jsonError, setJsonError] = useState('');

  const isNew = !item;
  const keyAttrs = tableDesc.KeySchema.map(k => k.AttributeName);

  useEffect(() => {
    if (item) {
      setJsonText(JSON.stringify(item, null, 2));
      setFormAttrs(itemToFormAttrs(item));
    } else {
      // New item: pre-populate keys
      const template: DDBItem = {};
      tableDesc.KeySchema.forEach(k => {
        const attrDef = tableDesc.AttributeDefinitions.find(a => a.AttributeName === k.AttributeName);
        const type = attrDef ? attrDef.AttributeType : 'S';
        template[k.AttributeName] = { [type]: '' } as any;
      });
      setJsonText(JSON.stringify(template, null, 2));
      setFormAttrs(itemToFormAttrs(template));
    }
  }, [item, tableDesc]);

  function handleSave() {
    if (viewMode === 'json') {
      try {
        const parsed = JSON.parse(jsonText);
        setJsonError('');
        onSave(parsed);
      } catch (e: any) {
        setJsonError('Invalid JSON: ' + e.message);
      }
    } else {
      const built = formAttrsToItem(formAttrs);
      onSave(built);
    }
  }

  function addAttribute() {
    setFormAttrs(prev => [...prev, { key: '', type: 'S', value: '' }]);
  }

  function updateAttr(idx: number, patch: Partial<FormAttribute>) {
    setFormAttrs(prev => prev.map((a, i) => i === idx ? { ...a, ...patch } : a));
  }

  function removeAttr(idx: number) {
    const attr = formAttrs[idx];
    if (keyAttrs.includes(attr.key)) return; // can't remove key attrs
    setFormAttrs(prev => prev.filter((_, i) => i !== idx));
  }

  function handleDuplicate() {
    if (!item || !onDuplicate) return;
    // Create a copy — user must change key values
    try {
      const parsed = viewMode === 'json' ? JSON.parse(jsonText) : formAttrsToItem(formAttrs);
      onDuplicate(parsed);
    } catch {
      // fallback
      onDuplicate(item);
    }
  }

  // Sync between views on mode switch
  function switchToJson() {
    if (viewMode === 'form') {
      const built = formAttrsToItem(formAttrs);
      setJsonText(JSON.stringify(built, null, 2));
    }
    setViewMode('json');
  }

  function switchToForm() {
    if (viewMode === 'json') {
      try {
        const parsed = JSON.parse(jsonText);
        setFormAttrs(itemToFormAttrs(parsed));
        setJsonError('');
      } catch (e: any) {
        setJsonError('Cannot switch: invalid JSON');
        return;
      }
    }
    setViewMode('form');
  }

  return (
    <Modal
      title={isNew ? 'Create Item' : 'Edit Item'}
      size="lg"
      onClose={onClose}
      footer={
        <>
          {!isNew && onDelete && (
            <button class="btn btn-danger btn-sm" onClick={() => onDelete(item!)} style="margin-right:auto">Delete</button>
          )}
          {!isNew && onDuplicate && (
            <button class="btn btn-ghost btn-sm" onClick={handleDuplicate}>
              <CopyIcon /> Duplicate
            </button>
          )}
          <button class="btn btn-ghost btn-sm" onClick={onClose}>Cancel</button>
          <button class="btn btn-primary btn-sm" onClick={handleSave}>{isNew ? 'Create' : 'Save'}</button>
        </>
      }
    >
      {/* View mode toggle */}
      <div class="flex gap-2 mb-4">
        <button
          class={`btn btn-sm ${viewMode === 'form' ? 'btn-secondary' : 'btn-ghost'}`}
          onClick={switchToForm}
        >Form</button>
        <button
          class={`btn btn-sm ${viewMode === 'json' ? 'btn-secondary' : 'btn-ghost'}`}
          onClick={switchToJson}
        >JSON</button>
      </div>

      {viewMode === 'json' ? (
        <div>
          <textarea
            class="ddb-json-editor"
            value={jsonText}
            onInput={(e) => { setJsonText((e.target as HTMLTextAreaElement).value); setJsonError(''); }}
            spellcheck={false}
          />
          {jsonError && (
            <div style="color:var(--error);font-size:13px;margin-top:8px">{jsonError}</div>
          )}
        </div>
      ) : (
        <div class="ddb-form-editor">
          {formAttrs.map((attr, idx) => {
            const isKey = keyAttrs.includes(attr.key);
            return (
              <div key={idx} class="ddb-form-attr-row">
                <input
                  class="input ddb-form-key"
                  placeholder="Attribute name"
                  value={attr.key}
                  onInput={(e) => updateAttr(idx, { key: (e.target as HTMLInputElement).value })}
                  disabled={isKey}
                  style={isKey ? 'background:var(--bg-secondary);font-weight:600' : ''}
                />
                <select
                  class="select ddb-form-type"
                  value={attr.type}
                  onChange={(e) => updateAttr(idx, { type: (e.target as HTMLSelectElement).value as DDBType })}
                >
                  {DDB_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                </select>
                {attr.type === 'BOOL' ? (
                  <select
                    class="select ddb-form-value"
                    value={attr.value}
                    onChange={(e) => updateAttr(idx, { value: (e.target as HTMLSelectElement).value })}
                  >
                    <option value="true">true</option>
                    <option value="false">false</option>
                  </select>
                ) : attr.type === 'NULL' ? (
                  <input class="input ddb-form-value" disabled value="null" />
                ) : attr.type === 'L' || attr.type === 'M' ? (
                  <textarea
                    class="ddb-form-value-area"
                    value={attr.value}
                    onInput={(e) => updateAttr(idx, { value: (e.target as HTMLTextAreaElement).value })}
                    placeholder={attr.type === 'L' ? '[{"S":"val"},...]' : '{"key":{"S":"val"}}'}
                    rows={3}
                  />
                ) : (
                  <input
                    class="input ddb-form-value"
                    placeholder={attr.type === 'SS' || attr.type === 'NS' ? 'comma separated' : 'Value'}
                    value={attr.value}
                    onInput={(e) => updateAttr(idx, { value: (e.target as HTMLInputElement).value })}
                  />
                )}
                <button
                  class="btn-icon btn-sm btn-ghost"
                  onClick={() => removeAttr(idx)}
                  disabled={isKey}
                  title={isKey ? 'Cannot remove key attribute' : 'Remove'}
                  style={isKey ? 'opacity:0.3' : ''}
                >
                  <TrashIcon />
                </button>
              </div>
            );
          })}
          <button class="btn btn-ghost btn-sm mt-4" onClick={addAttribute}>
            <PlusIcon /> Add Attribute
          </button>
        </div>
      )}
    </Modal>
  );
}
