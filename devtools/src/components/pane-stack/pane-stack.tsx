import { useMemo } from 'preact/hooks';
import type { ComponentChildren } from 'preact';
import { usePaneStack, type Pane } from '../../lib/pane-stack';
import { ErrorBoundary } from '../error-boundary';
import './pane-stack.css';

/** Max peek panes rendered as full panels; older panes collapse to a breadcrumb rail. */
const MAX_VISIBLE_PEEK = 2;

interface PaneStackProps {
  /** The root view — always visible on the far left. */
  base: ComponentChildren;
  /** Renders a peek pane's view given its route. */
  renderPane: (pane: Pane) => ComponentChildren;
}

export function PaneStack({ base, renderPane }: PaneStackProps) {
  const stack = usePaneStack();
  const { panes, close, closeFrom } = stack;

  // Split panes into collapsed (leftmost, old) and visible (rightmost, live).
  const { collapsed, visible } = useMemo(() => {
    if (panes.length <= MAX_VISIBLE_PEEK) {
      return { collapsed: [] as Pane[], visible: panes };
    }
    const cut = panes.length - MAX_VISIBLE_PEEK;
    return { collapsed: panes.slice(0, cut), visible: panes.slice(cut) };
  }, [panes]);

  return (
    <div class="pane-stack">
      <div class="pane-stack-base">
        <ErrorBoundary>{base}</ErrorBoundary>
      </div>

      {collapsed.length > 0 && (
        <div class="pane-stack-rail" role="tablist" aria-label="Collapsed panes">
          {collapsed.map((p) => (
            <button
              key={p.id}
              class="pane-stack-rail-chip"
              onClick={() => closeFrom(p.id)}
              title={`${paneTitle(p)} — click to close this and later panes`}
            >
              <span class="pane-stack-rail-chip-view">{p.view}</span>
              <span class="pane-stack-rail-chip-label">{paneTitle(p)}</span>
            </button>
          ))}
        </div>
      )}

      {visible.map((p, idx) => (
        <div
          key={p.id}
          class={`pane-stack-peek pane-stack-peek-${idx === visible.length - 1 ? 'focused' : 'dim'}`}
        >
          <div class="pane-stack-peek-header">
            <div class="pane-stack-peek-title">
              <span class="pane-stack-peek-kind">{p.view}</span>
              <span class="pane-stack-peek-label">{paneTitle(p)}</span>
            </div>
            <div class="pane-stack-peek-actions">
              <button
                class="pane-stack-peek-btn"
                onClick={() => closeFrom(p.id)}
                title="Close this pane and any panes to the right"
                aria-label="Close from here"
              >
                {'\u2715'}
              </button>
            </div>
          </div>
          <div class="pane-stack-peek-body">
            <ErrorBoundary>{renderPane(p)}</ErrorBoundary>
          </div>
          {/* Clicking dim area focuses: closes every pane to the right of this one. */}
          {idx !== visible.length - 1 && (
            <button
              class="pane-stack-peek-focus-overlay"
              aria-label={`Focus ${paneTitle(p)}`}
              onClick={() => {
                const next = visible[idx + 1];
                if (next) close(next.id);
              }}
            />
          )}
        </div>
      ))}
    </div>
  );
}

function paneTitle(p: Pane): string {
  if (p.title) return p.title;
  if (p.segments.length > 0) return p.segments.join(' / ');
  const firstParam = Object.entries(p.params)[0];
  return firstParam ? `${firstParam[0]}=${firstParam[1]}` : p.view;
}
