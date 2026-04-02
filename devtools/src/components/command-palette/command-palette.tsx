import { useState, useEffect, useRef, useMemo, useCallback } from 'preact/hooks';
import type { ViewId } from '../../app';
import './command-palette.css';

interface CommandItem {
  id: string;
  label: string;
  category: string;
  icon: string;
  action: () => void;
}

interface CommandPaletteProps {
  onNavigate: (view: ViewId) => void;
}

const NAV_COMMANDS: { id: ViewId; label: string; icon: string; category: string }[] = [
  // Observability
  { id: 'activity', label: 'Go to Activity', icon: '\u26A1', category: 'Navigate' },
  { id: 'topology', label: 'Go to Topology', icon: '\uD83D\uDDFA\uFE0F', category: 'Navigate' },
  { id: 'traces', label: 'Go to Traces', icon: '\uD83D\uDD0D', category: 'Navigate' },
  { id: 'metrics', label: 'Go to Metrics', icon: '\uD83D\uDCCA', category: 'Navigate' },
  { id: 'dashboards', label: 'Go to Dashboards', icon: '\uD83D\uDCCB', category: 'Navigate' },

  // AWS Browsers
  { id: 's3-browser', label: 'Go to S3 Browser', icon: '\uD83E\uDEA3', category: 'Navigate' },
  { id: 'dynamodb', label: 'Go to DynamoDB', icon: '\uD83D\uDDC4\uFE0F', category: 'Navigate' },
  { id: 'sqs-browser', label: 'Go to SQS Browser', icon: '\uD83D\uDCEC', category: 'Navigate' },
  { id: 'cognito', label: 'Go to Cognito', icon: '\uD83D\uDD10', category: 'Navigate' },
  { id: 'lambda', label: 'Go to Lambda Logs', icon: '\u03BB', category: 'Navigate' },
  { id: 'iam', label: 'Go to IAM Evaluator', icon: '\uD83D\uDEE1\uFE0F', category: 'Navigate' },
  { id: 'mail', label: 'Go to SES Mail', icon: '\u2709\uFE0F', category: 'Navigate' },

  // Operations
  { id: 'slos', label: 'Go to SLOs', icon: '\uD83C\uDFAF', category: 'Navigate' },
  { id: 'incidents', label: 'Go to Incidents', icon: '\uD83D\uDEA8', category: 'Navigate' },
  { id: 'monitors', label: 'Go to Monitors', icon: '\uD83D\uDD14', category: 'Navigate' },
  { id: 'chaos', label: 'Go to Chaos', icon: '\uD83E\uDDEA', category: 'Navigate' },
  { id: 'regressions', label: 'Go to Regressions', icon: '\uD83D\uDCC9', category: 'Navigate' },

  // Tools
  { id: 'ai-debug', label: 'Go to AI Debug', icon: '\uD83E\uDD16', category: 'Navigate' },
  { id: 'routing', label: 'Go to Routing', icon: '\uD83D\uDD00', category: 'Navigate' },
  { id: 'traffic', label: 'Go to Traffic', icon: '\uD83D\uDD04', category: 'Navigate' },
  { id: 'rum', label: 'Go to RUM', icon: '\uD83D\uDC64', category: 'Navigate' },
  { id: 'services', label: 'Go to Services', icon: '\u2601\uFE0F', category: 'Navigate' },
  { id: 'profiler', label: 'Go to Profiler', icon: '\uD83D\uDD25', category: 'Navigate' },

  // Settings
  { id: 'settings', label: 'Go to Settings', icon: '\u2699\uFE0F', category: 'Navigate' },
];

const QUICK_ACTIONS: { id: string; label: string; icon: string; event: string }[] = [
  { id: 'reset-all', label: 'Reset all data', icon: '\uD83D\uDDD1\uFE0F', event: 'neureaux:cmd-reset-all' },
  { id: 'toggle-chaos', label: 'Toggle chaos', icon: '\uD83E\uDDEA', event: 'neureaux:cmd-toggle-chaos' },
  { id: 'create-bucket', label: 'Create S3 bucket', icon: '\uD83E\uDEA3', event: 'neureaux:cmd-create-bucket' },
  { id: 'snap-live', label: 'Snap to live', icon: '\u25CF', event: 'neureaux:snap-live' },
];

export function CommandPalette({ onNavigate }: CommandPaletteProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Build all commands
  const allCommands = useMemo<CommandItem[]>(() => {
    const navItems: CommandItem[] = NAV_COMMANDS.map((cmd) => ({
      id: `nav-${cmd.id}`,
      label: cmd.label,
      category: cmd.category,
      icon: cmd.icon,
      action: () => {
        onNavigate(cmd.id);
        setOpen(false);
      },
    }));

    const quickItems: CommandItem[] = QUICK_ACTIONS.map((cmd) => ({
      id: `quick-${cmd.id}`,
      label: cmd.label,
      category: 'Quick Actions',
      icon: cmd.icon,
      action: () => {
        document.dispatchEvent(new CustomEvent(cmd.event));
        setOpen(false);
      },
    }));

    return [...navItems, ...quickItems];
  }, [onNavigate]);

  // Filter by query
  const filtered = useMemo(() => {
    if (!query.trim()) return allCommands;
    const lower = query.toLowerCase();
    return allCommands.filter(
      (cmd) =>
        cmd.label.toLowerCase().includes(lower) ||
        cmd.category.toLowerCase().includes(lower),
    );
  }, [allCommands, query]);

  // Clamp selection when filtered list changes
  useEffect(() => {
    setSelectedIndex(0);
  }, [filtered.length, query]);

  // Global Cmd+K / Ctrl+K listener
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const mod = e.metaKey || e.ctrlKey;
      if (mod && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setOpen((prev) => !prev);
        setQuery('');
        setSelectedIndex(0);
      }
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  // Focus input when opened
  useEffect(() => {
    if (open) {
      // Small delay to let the element render
      requestAnimationFrame(() => {
        inputRef.current?.focus();
      });
    }
  }, [open]);

  // Scroll selected item into view
  useEffect(() => {
    if (!listRef.current) return;
    const selectedEl = listRef.current.children[selectedIndex] as HTMLElement | undefined;
    if (selectedEl) {
      selectedEl.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedIndex]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        setOpen(false);
        return;
      }

      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedIndex((prev) => Math.min(filtered.length - 1, prev + 1));
        return;
      }

      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedIndex((prev) => Math.max(0, prev - 1));
        return;
      }

      if (e.key === 'Enter') {
        e.preventDefault();
        const item = filtered[selectedIndex];
        if (item) {
          item.action();
        }
        return;
      }
    },
    [filtered, selectedIndex],
  );

  if (!open) return null;

  return (
    <div class="cmd-palette-overlay" onClick={() => setOpen(false)}>
      <div class="cmd-palette" onClick={(e) => e.stopPropagation()}>
        <div class="cmd-palette-input-wrapper">
          <span class="cmd-palette-search-icon">{'\uD83D\uDD0D'}</span>
          <input
            ref={inputRef}
            class="cmd-palette-input"
            type="text"
            placeholder="Type a command..."
            value={query}
            onInput={(e) => setQuery((e.target as HTMLInputElement).value)}
            onKeyDown={handleKeyDown}
          />
          <kbd class="cmd-palette-kbd">ESC</kbd>
        </div>

        <div class="cmd-palette-list" ref={listRef}>
          {filtered.length === 0 && (
            <div class="cmd-palette-empty">No results found</div>
          )}
          {filtered.map((item, i) => (
            <button
              key={item.id}
              class={`cmd-palette-item ${i === selectedIndex ? 'selected' : ''}`}
              onClick={() => item.action()}
              onMouseEnter={() => setSelectedIndex(i)}
            >
              <span class="cmd-palette-item-icon">{item.icon}</span>
              <span class="cmd-palette-item-label">{item.label}</span>
              <span class="cmd-palette-item-category">{item.category}</span>
            </button>
          ))}
        </div>

        <div class="cmd-palette-footer">
          <span class="cmd-palette-hint">
            <kbd>{'\u2191'}</kbd>
            <kbd>{'\u2193'}</kbd> to navigate
          </span>
          <span class="cmd-palette-hint">
            <kbd>{'\u23CE'}</kbd> to select
          </span>
          <span class="cmd-palette-hint">
            <kbd>esc</kbd> to close
          </span>
        </div>
      </div>
    </div>
  );
}
