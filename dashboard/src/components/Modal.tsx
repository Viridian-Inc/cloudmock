import { ComponentChildren } from 'preact';
import { XIcon } from './Icons';

interface ModalProps {
  title: string;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  onClose: () => void;
  footer?: ComponentChildren;
  children: ComponentChildren;
}

export function Modal({ title, size = 'md', onClose, footer, children }: ModalProps) {
  return (
    <div class="modal-backdrop" onClick={onClose}>
      <div class={`modal modal-${size}`} onClick={(e) => e.stopPropagation()}>
        <div class="modal-header">
          <h3>{title}</h3>
          <button class="btn-icon btn-sm btn-ghost" onClick={onClose}>
            <XIcon />
          </button>
        </div>
        <div class="modal-body">{children}</div>
        {footer && <div class="modal-footer">{footer}</div>}
      </div>
    </div>
  );
}
