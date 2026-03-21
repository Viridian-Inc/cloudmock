import { ComponentChildren } from 'preact';
import { XIcon } from './Icons';

interface DrawerProps {
  title: string;
  onClose: () => void;
  actions?: ComponentChildren;
  children: ComponentChildren;
}

export function Drawer({ title, onClose, actions, children }: DrawerProps) {
  return (
    <div class="drawer-backdrop" onClick={onClose}>
      <div class="drawer" onClick={(e) => e.stopPropagation()}>
        <div class="drawer-header">
          <h3 style="font-size:16px;font-weight:700">{title}</h3>
          <div class="flex gap-2">
            {actions}
            <button class="btn-icon btn-sm btn-ghost" onClick={onClose}>
              <XIcon />
            </button>
          </div>
        </div>
        <div class="drawer-body">{children}</div>
      </div>
    </div>
  );
}
