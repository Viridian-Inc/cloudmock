import { useState, useEffect, useRef, useMemo } from 'preact/hooks';
import { api } from '../api';

interface Command {
  label: string;
  desc: string;
  action: () => void;
}

interface CommandPaletteProps {
  services: any[];
  onClose: () => void;
}

export function CommandPalette({ services, onClose }: CommandPaletteProps) {
  const [query, setQuery] = useState('');
  const [activeIdx, setActiveIdx] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (inputRef.current) inputRef.current.focus();
  }, []);

  const commands = useMemo<Command[]>(() => {
    const items: Command[] = [
      { label: 'Services', desc: 'View all services', action: () => { location.hash = '/'; onClose(); } },
      { label: 'Requests', desc: 'Request log', action: () => { location.hash = '/requests'; onClose(); } },
      { label: 'DynamoDB', desc: 'Table browser', action: () => { location.hash = '/dynamodb'; onClose(); } },
      { label: 'Resources', desc: 'Resource explorer', action: () => { location.hash = '/resources'; onClose(); } },
      { label: 'Lambda Logs', desc: 'Function logs', action: () => { location.hash = '/lambda'; onClose(); } },
      { label: 'IAM Debugger', desc: 'Policy evaluation', action: () => { location.hash = '/iam'; onClose(); } },
      { label: 'Mail', desc: 'SES emails', action: () => { location.hash = '/mail'; onClose(); } },
      { label: 'Topology', desc: 'Service map', action: () => { location.hash = '/topology'; onClose(); } },
      {
        label: 'Reset All Services', desc: 'Clear all state', action: () => {
          api('/api/reset', { method: 'POST' }).then(() => { onClose(); location.reload(); });
        },
      },
    ];
    (services || []).forEach((svc: any) => {
      items.push({
        label: svc.name,
        desc: 'Service',
        action: () => { location.hash = `/resources?service=${svc.name}`; onClose(); },
      });
    });
    if (!query) return items;
    const q = query.toLowerCase();
    return items.filter(i => i.label.toLowerCase().includes(q) || i.desc.toLowerCase().includes(q));
  }, [query, services, onClose]);

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') { e.preventDefault(); setActiveIdx(i => Math.min(i + 1, commands.length - 1)); }
    if (e.key === 'ArrowUp') { e.preventDefault(); setActiveIdx(i => Math.max(i - 1, 0)); }
    if (e.key === 'Enter' && commands[activeIdx]) { commands[activeIdx].action(); }
    if (e.key === 'Escape') { onClose(); }
  }

  return (
    <div class="palette-backdrop" onClick={onClose}>
      <div class="palette" onClick={(e) => e.stopPropagation()}>
        <input
          class="palette-input"
          ref={inputRef}
          placeholder="Search commands, services..."
          value={query}
          onInput={(e) => { setQuery((e.target as HTMLInputElement).value); setActiveIdx(0); }}
          onKeyDown={handleKeyDown}
        />
        <div class="palette-results">
          {commands.map((cmd, i) => (
            <div
              class={`palette-item ${i === activeIdx ? 'active' : ''}`}
              onClick={cmd.action}
              onMouseEnter={() => setActiveIdx(i)}
            >
              <span class="label">{cmd.label}</span>
              <span class="desc">{cmd.desc}</span>
            </div>
          ))}
          {commands.length === 0 && (
            <div style="padding:24px;text-align:center;color:var(--n400);font-size:14px">No results</div>
          )}
        </div>
      </div>
    </div>
  );
}
