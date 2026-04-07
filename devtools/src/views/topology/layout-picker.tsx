import './layout-picker.css';

export type LayoutMode = 'layered' | 'tree' | 'force';

interface LayoutPickerProps {
  value: LayoutMode;
  onChange: (mode: LayoutMode) => void;
  treeAvailable: boolean;
}

export function LayoutPicker({ value, onChange, treeAvailable }: LayoutPickerProps) {
  return (
    <div class="layout-picker">
      <button
        class={`layout-picker-btn ${value === 'layered' ? 'active' : ''}`}
        onClick={() => onChange('layered')}
      >
        Layered
      </button>
      <button
        class={`layout-picker-btn ${value === 'tree' ? 'active' : ''}`}
        onClick={() => onChange('tree')}
        disabled={!treeAvailable}
        title={treeAvailable ? 'IaC dependency tree' : 'No IaC project configured'}
      >
        Tree
      </button>
      <button
        class={`layout-picker-btn ${value === 'force' ? 'active' : ''}`}
        onClick={() => onChange('force')}
      >
        Force
      </button>
    </div>
  );
}
