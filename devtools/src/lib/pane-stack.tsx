// Pane stack — hierarchical drill-down panes stacked horizontally alongside
// the root view. A pane holds a view + its selection, just like a route, but
// lives *in addition* to the root view instead of replacing it.
//
// URL shape:   #/<root>?...&panes=<b64-json>
// where panes decodes to: [{ v: 'traces', s: ['abc123'], p: { tab: 'waterfall' }, t?: 'Trace abc123' }, ...]
//
// Base64url-encoded JSON is unusual in URLs but trivial to round-trip and
// survives any selection/param we want to pack in. Devtools users can still
// decode with `JSON.parse(atob(value))` if they want to inspect a link.
//
// Panes are pushed via usePaneStack().push() or by dispatching a
// 'neureaux:peek' CustomEvent — the latter exists so any view can drill down
// without taking the hook as a dependency.

import { createContext } from 'preact';
import { useContext, useEffect, useCallback, useMemo, useRef } from 'preact/hooks';
import type { ComponentChildren } from 'preact';
import { useRouter, RouterContext, type Route, type Navigation, type PathPatch } from './router';

export interface Pane {
  /** Stable ID — pushing a pane with the same id replaces instead of stacking. */
  id: string;
  view: string;
  segments: string[];
  params: Record<string, string>;
  /** Optional human-readable title for the pane header. */
  title?: string;
}

export interface PaneStack {
  panes: Pane[];
  push: (pane: Omit<Pane, 'id'> & { id?: string }) => void;
  /**
   * Replace a pane in-place by id without truncating panes to its right.
   * Used by the pane-scoped router so a view syncing its own selection/params
   * to the URL doesn't collapse drill-downs the user has opened below it.
   */
  update: (pane: Pane) => void;
  close: (id: string) => void;
  /** Close this pane and every pane to the right of it. */
  closeFrom: (id: string) => void;
  clear: () => void;
}

const PaneStackContext = createContext<PaneStack | null>(null);

interface EncodedPane {
  v: string;
  s: string[];
  p: Record<string, string>;
  t?: string;
  i: string;
}

function encodePanes(panes: Pane[]): string {
  if (panes.length === 0) return '';
  const compact: EncodedPane[] = panes.map((p) => ({
    v: p.view,
    s: p.segments,
    p: p.params,
    t: p.title,
    i: p.id,
  }));
  // base64url: no +/= characters so it survives URL encoding unescaped.
  const json = JSON.stringify(compact);
  const b64 = typeof btoa === 'function' ? btoa(unescape(encodeURIComponent(json))) : '';
  return b64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function decodePanes(value: string): Pane[] {
  if (!value) return [];
  try {
    const b64 = value.replace(/-/g, '+').replace(/_/g, '/');
    const padded = b64 + '==='.slice((b64.length + 3) % 4);
    const json = decodeURIComponent(escape(atob(padded)));
    const raw = JSON.parse(json) as EncodedPane[];
    if (!Array.isArray(raw)) return [];
    return raw
      .filter((p) => p && typeof p.v === 'string' && typeof p.i === 'string')
      .map((p) => ({
        id: p.i,
        view: p.v,
        segments: Array.isArray(p.s) ? p.s : [],
        params: p.p && typeof p.p === 'object' ? p.p : {},
        title: typeof p.t === 'string' ? p.t : undefined,
      }));
  } catch {
    return [];
  }
}

function paneId(view: string, segments: string[]): string {
  return segments.length > 0 ? `${view}:${segments.join('/')}` : view;
}

function segmentsEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return false;
  return true;
}

function paramsEqual(a: Record<string, string>, b: Record<string, string>): boolean {
  const ak = Object.keys(a);
  const bk = Object.keys(b);
  if (ak.length !== bk.length) return false;
  for (const k of ak) if (a[k] !== b[k]) return false;
  return true;
}

interface PeekEventDetail {
  view: string;
  segments?: string[];
  params?: Record<string, string>;
  title?: string;
  id?: string;
}

export function PaneStackProvider({ children }: { children: ComponentChildren }) {
  const router = useRouter();
  const encoded = router.params.panes ?? '';
  // Decode lazily per value change — cheap and keeps identity stable when the URL is untouched.
  const panes = useMemo(() => decodePanes(encoded), [encoded]);

  // Mutations round-trip through the URL so back/forward work.
  const writePanes = useCallback(
    (next: Pane[]) => {
      const value = encodePanes(next);
      router.setParams({ panes: value || null }, { replace: false });
    },
    [router],
  );

  const push = useCallback(
    (p: Omit<Pane, 'id'> & { id?: string }) => {
      const id = p.id ?? paneId(p.view, p.segments ?? []);
      const next: Pane = {
        id,
        view: p.view,
        segments: p.segments ?? [],
        params: p.params ?? {},
        title: p.title,
      };
      // Replace an existing pane with the same id (e.g. re-opening the same trace),
      // truncating everything to its right — pushing re-focuses the drill-down chain.
      const existing = panes.findIndex((x) => x.id === id);
      if (existing >= 0) {
        writePanes([...panes.slice(0, existing), next]);
      } else {
        writePanes([...panes, next]);
      }
    },
    [panes, writePanes],
  );

  const update = useCallback(
    (next: Pane) => {
      const idx = panes.findIndex((p) => p.id === next.id);
      if (idx < 0) {
        // Pane has already been closed (e.g. user hit Esc while an async effect
        // was still in flight). Don't resurrect it.
        return;
      }
      // Identity short-circuit: skip the URL write if nothing actually changed.
      const curr = panes[idx];
      if (
        curr.view === next.view &&
        curr.title === next.title &&
        segmentsEqual(curr.segments, next.segments) &&
        paramsEqual(curr.params, next.params)
      ) {
        return;
      }
      const copy = panes.slice();
      copy[idx] = next;
      writePanes(copy);
    },
    [panes, writePanes],
  );

  const close = useCallback(
    (id: string) => {
      writePanes(panes.filter((p) => p.id !== id));
    },
    [panes, writePanes],
  );

  const closeFrom = useCallback(
    (id: string) => {
      const idx = panes.findIndex((p) => p.id === id);
      if (idx < 0) return;
      writePanes(panes.slice(0, idx));
    },
    [panes, writePanes],
  );

  const clear = useCallback(() => writePanes([]), [writePanes]);

  const pushRef = useRef(push);
  pushRef.current = push;

  // Global peek event — any view can request a drill-down without the hook.
  useEffect(() => {
    const handler = (e: Event) => {
      const d = (e as CustomEvent<PeekEventDetail>).detail;
      if (!d || typeof d.view !== 'string') return;
      pushRef.current({
        view: d.view,
        segments: d.segments ?? [],
        params: d.params ?? {},
        title: d.title,
        id: d.id,
      });
    };
    document.addEventListener('neureaux:peek', handler);
    return () => document.removeEventListener('neureaux:peek', handler);
  }, []);

  // Esc closes the rightmost pane when any pane is open. Does not fire if the user
  // is typing in an input or a view has explicitly captured the key (stopPropagation).
  useEffect(() => {
    if (panes.length === 0) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      const t = e.target as HTMLElement | null;
      if (t && ['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName)) return;
      if (t && t.isContentEditable) return;
      close(panes[panes.length - 1].id);
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [panes, close]);

  const value = useMemo<PaneStack>(
    () => ({ panes, push, update, close, closeFrom, clear }),
    [panes, push, update, close, closeFrom, clear],
  );

  return <PaneStackContext.Provider value={value}>{children}</PaneStackContext.Provider>;
}

export function usePaneStack(): PaneStack {
  const ctx = useContext(PaneStackContext);
  if (!ctx) throw new Error('usePaneStack must be used inside PaneStackProvider');
  return ctx;
}

/** Fire-and-forget — push a pane from code that doesn't have access to the hook. */
export function peek(detail: PeekEventDetail): void {
  document.dispatchEvent(new CustomEvent('neureaux:peek', { detail }));
}

/**
 * Scopes useRouter() to a specific pane. Inside this scope, views read their
 * route from the pane (view/segments/params) and write back into the same
 * pane instead of the top-level URL route. Pushing a completely different
 * view replaces the pane — views that "navigate" still work, they just do it
 * inside their own pane.
 */
export function PaneRouterScope({
  pane,
  children,
}: {
  pane: Pane;
  children: ComponentChildren;
}) {
  const stack = usePaneStack();
  const topLevel = useRouter();

  const value = useMemo<Route & Navigation>(() => {
    const route: Route = {
      view: pane.view,
      segments: pane.segments,
      params: pane.params,
      raw: serialisePaneRoute(pane),
    };

    // Any mutation rewrites this pane in place without disturbing panes to the
    // right. Views freely call router.push/setParams to sync selection or
    // filters to the URL (e.g. activity's service filter effect); that must
    // not collapse a deeper pane the user has opened below.
    const writePane = (patch: PathPatch, replace: boolean) => {
      void replace;
      const nextView = patch.view ?? pane.view;
      const nextSegments = patch.segments ?? pane.segments;
      const nextParams: Record<string, string> = { ...pane.params };
      if (patch.params) {
        for (const [k, v] of Object.entries(patch.params)) {
          if (v == null || v === '') delete nextParams[k];
          else nextParams[k] = v;
        }
      }
      stack.update({
        id: pane.id,
        view: nextView,
        segments: nextSegments,
        params: nextParams,
        title: pane.title,
      });
    };

    const push: Navigation['push'] = (path) => writePane(coerce(path, route), false);
    const replace: Navigation['replace'] = (path) => writePane(coerce(path, route), true);
    const setParams: Navigation['setParams'] = (patch, opts) => {
      writePane({ params: patch }, opts?.replace ?? true);
    };

    return { ...route, push, replace, setParams };
    // topLevel is included so scoped panes re-render when the outer URL changes
    // (e.g. a peek event that rewrites the stack).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pane, stack, topLevel.raw]);

  return <RouterContext.Provider value={value}>{children}</RouterContext.Provider>;
}

function serialisePaneRoute(p: Pane): string {
  const path = [p.view, ...p.segments].filter(Boolean).map(encodeURIComponent).join('/');
  const qs = new URLSearchParams(p.params).toString();
  return `/${path}${qs ? '?' + qs : ''}`;
}

function coerce(path: string | PathPatch, current: Route): PathPatch {
  if (typeof path !== 'string') return path;
  // Parse a plain path like "/traces/abc123?tab=waterfall" into a patch.
  const raw = path.startsWith('/') ? path.slice(1) : path;
  const [pathPart, qs = ''] = raw.split('?');
  const parts = pathPart.split('/').filter(Boolean);
  const [view, ...segments] = parts;
  const params: Record<string, string> = {};
  if (qs) new URLSearchParams(qs).forEach((v, k) => (params[k] = v));
  return {
    view: view ?? current.view,
    segments,
    params,
  };
}
