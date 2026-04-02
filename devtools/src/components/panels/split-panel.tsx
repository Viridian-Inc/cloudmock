import { useState, useRef, useCallback } from 'preact/hooks';
import type { ComponentChildren } from 'preact';
import './split-panel.css';

interface SplitPanelProps {
  left: ComponentChildren;
  right: ComponentChildren;
  initialSplit?: number;
  direction?: 'horizontal' | 'vertical';
  minSize?: number;
  id?: string; // for localStorage persistence
}

function storageKey(id: string): string {
  return `neureaux:split-panel:${id}`;
}

function loadSplit(id: string | undefined, fallback: number): number {
  if (!id) return fallback;
  try {
    const stored = localStorage.getItem(storageKey(id));
    if (stored !== null) {
      const val = parseFloat(stored);
      if (!isNaN(val) && val > 0 && val < 1) return val;
    }
  } catch { /* localStorage unavailable */ }
  return fallback;
}

function saveSplit(id: string | undefined, value: number): void {
  if (!id) return;
  try {
    localStorage.setItem(storageKey(id), String(value));
  } catch { /* localStorage unavailable */ }
}

export function SplitPanel({
  left,
  right,
  initialSplit = 0.5,
  direction = 'horizontal',
  minSize = 100,
  id,
}: SplitPanelProps) {
  // Accept both 0-1 fractions and 1-100 percentages
  const normalizedInitial = initialSplit > 1 ? initialSplit / 100 : initialSplit;
  const [split, setSplit] = useState(() => loadSplit(id, normalizedInitial));
  const [dragging, setDragging] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const onPointerDown = useCallback(
    (e: PointerEvent) => {
      e.preventDefault();
      setDragging(true);

      // Listen at window level so we catch events even if cursor leaves component/viewport
      const handleMove = (me: PointerEvent) => {
        if (!containerRef.current) return;

        const rect = containerRef.current.getBoundingClientRect();
        const isHorizontal = direction === 'horizontal';
        const total = isHorizontal ? rect.width : rect.height;
        const pos = isHorizontal ? me.clientX - rect.left : me.clientY - rect.top;

        const minFraction = minSize / total;
        const maxFraction = 1 - minFraction;
        const fraction = Math.min(maxFraction, Math.max(minFraction, pos / total));

        setSplit(fraction);
      };

      const handleUp = () => {
        setDragging(false);
        // Persist final ratio to localStorage
        if (containerRef.current) {
          const rect = containerRef.current.getBoundingClientRect();
          // Read the current split from the first pane's rendered size
          const firstPane = containerRef.current.querySelector('.split-panel-pane') as HTMLElement | null;
          if (firstPane) {
            const isH = direction === 'horizontal';
            const total = isH ? rect.width : rect.height;
            const paneSize = isH ? firstPane.offsetWidth : firstPane.offsetHeight;
            saveSplit(id, paneSize / total);
          }
        }
        window.removeEventListener('pointermove', handleMove);
        window.removeEventListener('pointerup', handleUp);
      };

      window.addEventListener('pointermove', handleMove);
      window.addEventListener('pointerup', handleUp);
    },
    [direction, minSize, id],
  );

  const isHorizontal = direction === 'horizontal';
  const firstStyle = isHorizontal
    ? { width: `${split * 100}%`, height: '100%' }
    : { height: `${split * 100}%`, width: '100%' };
  const secondStyle = isHorizontal
    ? { width: `${(1 - split) * 100}%`, height: '100%' }
    : { height: `${(1 - split) * 100}%`, width: '100%' };

  return (
    <div
      ref={containerRef}
      class={`split-panel ${direction}`}
      style={dragging ? { userSelect: 'none' } : undefined}
    >
      <div class="split-panel-pane" style={firstStyle}>
        {left}
      </div>
      <div
        class={`split-panel-divider${dragging ? ' dragging' : ''}`}
        onPointerDown={onPointerDown}
      />
      <div class="split-panel-pane" style={secondStyle}>
        {right}
      </div>
    </div>
  );
}
