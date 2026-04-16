// Lightweight hash-based router for cloudmock devtools.
//
// URL shape:  #/<view>[/<segment>[/<segment>...]][?param=value&...]
// Examples:
//   #/                             -> { view: '', segments: [], params: {} }
//   #/topology                     -> { view: 'topology', segments: [] }
//   #/settings/routing             -> { view: 'settings', segments: ['routing'] }
//   #/traces/abc123?tab=waterfall  -> { view: 'traces', segments: ['abc123'], params: { tab: 'waterfall' } }
//
// One source of truth (the URL) drives view selection + per-view state. The
// browser back/forward buttons work, links are shareable, and bookmarking
// a specific tab of a specific resource does the right thing.

import { createContext } from 'preact';
import { useContext, useEffect, useState, useCallback, useMemo } from 'preact/hooks';
import type { ComponentChildren } from 'preact';

export interface Route {
  view: string;
  segments: string[];
  params: Record<string, string>;
  raw: string;
}

export interface Navigation {
  push: (path: string | PathPatch) => void;
  replace: (path: string | PathPatch) => void;
  setParams: (patch: Record<string, string | null | undefined>, opts?: { replace?: boolean }) => void;
}

export type PathPatch = {
  view?: string;
  segments?: string[];
  params?: Record<string, string | null | undefined>;
};

const RouterContext = createContext<(Route & Navigation) | null>(null);

// Parses "#/view/seg1/seg2?k=v&k2=v2" → Route.
function parseHash(hash: string): Route {
  const raw = hash.startsWith('#') ? hash.slice(1) : hash;
  const [path, qs = ''] = raw.split('?');
  const trimmed = path.replace(/^\/+/, '').replace(/\/+$/, '');
  const parts = trimmed === '' ? [] : trimmed.split('/');
  const [view = '', ...segments] = parts;

  const params: Record<string, string> = {};
  if (qs) {
    const usp = new URLSearchParams(qs);
    usp.forEach((v, k) => {
      params[k] = v;
    });
  }

  return {
    view: decodeURIComponent(view),
    segments: segments.map(decodeURIComponent),
    params,
    raw,
  };
}

// Back-compat: legacy hash formats that aren't path-style.
//   #service=dynamodb  -> { view: 'activity', params: { service: 'dynamodb' } }
//   #trace=abc123      -> { view: 'traces', segments: ['abc123'] }
// These were hard-coded in App.tsx and view code before the router existed;
// we normalise them so the router reports the clean shape, and older code
// paths continue to work.
function normaliseLegacyHash(hash: string): string | null {
  const raw = hash.startsWith('#') ? hash.slice(1) : hash;
  if (raw.startsWith('/') || raw === '') return null;

  const usp = new URLSearchParams(raw);
  if (usp.has('service')) {
    const svc = usp.get('service') || '';
    usp.delete('service');
    const rest = usp.toString();
    return `/activity?service=${encodeURIComponent(svc)}${rest ? '&' + rest : ''}`;
  }
  if (usp.has('trace')) {
    const id = usp.get('trace') || '';
    usp.delete('trace');
    const rest = usp.toString();
    return `/traces/${encodeURIComponent(id)}${rest ? '?' + rest : ''}`;
  }
  return null;
}

function serialisePatch(current: Route, patch: PathPatch): string {
  const view = patch.view ?? current.view;
  const segments = patch.segments ?? current.segments;
  const mergedParams: Record<string, string> = { ...current.params };
  if (patch.params) {
    for (const [k, v] of Object.entries(patch.params)) {
      if (v == null || v === '') delete mergedParams[k];
      else mergedParams[k] = v;
    }
  }
  const segStr = [view, ...segments].filter(Boolean).map(encodeURIComponent).join('/');
  const qs = new URLSearchParams(mergedParams).toString();
  return `/${segStr}${qs ? '?' + qs : ''}`;
}

function pathToString(path: string | PathPatch, current: Route): string {
  if (typeof path === 'string') {
    const p = path.startsWith('/') ? path : '/' + path;
    return p;
  }
  return serialisePatch(current, path);
}

function applyHash(path: string, replace: boolean): void {
  const target = '#' + path;
  if (replace) {
    const url = window.location.pathname + window.location.search + target;
    window.history.replaceState(null, '', url);
    // replaceState doesn't fire hashchange; do it manually so subscribers update.
    window.dispatchEvent(new HashChangeEvent('hashchange'));
  } else {
    window.location.hash = path;
  }
}

export function RouterProvider({ children }: { children: ComponentChildren }) {
  const [route, setRoute] = useState<Route>(() => {
    const normalised = normaliseLegacyHash(window.location.hash);
    if (normalised) {
      window.history.replaceState(null, '', window.location.pathname + window.location.search + '#' + normalised);
    }
    return parseHash(window.location.hash);
  });

  useEffect(() => {
    const onChange = () => {
      const normalised = normaliseLegacyHash(window.location.hash);
      if (normalised) {
        window.history.replaceState(null, '', window.location.pathname + window.location.search + '#' + normalised);
      }
      setRoute(parseHash(window.location.hash));
    };
    window.addEventListener('hashchange', onChange);
    return () => window.removeEventListener('hashchange', onChange);
  }, []);

  const push = useCallback(
    (path: string | PathPatch) => applyHash(pathToString(path, route), false),
    [route],
  );
  const replace = useCallback(
    (path: string | PathPatch) => applyHash(pathToString(path, route), true),
    [route],
  );
  const setParams = useCallback(
    (patch: Record<string, string | null | undefined>, opts?: { replace?: boolean }) => {
      const path = serialisePatch(route, { params: patch });
      applyHash(path, opts?.replace ?? true);
    },
    [route],
  );

  const value = useMemo<Route & Navigation>(
    () => ({ ...route, push, replace, setParams }),
    [route, push, replace, setParams],
  );

  return <RouterContext.Provider value={value}>{children}</RouterContext.Provider>;
}

export function useRouter(): Route & Navigation {
  const ctx = useContext(RouterContext);
  if (!ctx) throw new Error('useRouter must be used inside RouterProvider');
  return ctx;
}

// Convenience hook: read a single query param with an optional default, and
// get a setter that writes it back to the URL via replaceState (so the back
// button isn't cluttered with tab-flipping history).
export function useRouteParam(
  key: string,
  defaultValue = '',
): [string, (v: string | null) => void] {
  const router = useRouter();
  const value = router.params[key] ?? defaultValue;
  const setValue = useCallback(
    (v: string | null) => router.setParams({ [key]: v }, { replace: true }),
    [router, key],
  );
  return [value, setValue];
}

// Convenience hook: the first path segment after the view, treated as a
// primary "selection" id (e.g. trace id, service id, settings tab).
export function useRouteSelection(): [string | null, (id: string | null) => void] {
  const router = useRouter();
  const selection = router.segments[0] ?? null;
  const setSelection = useCallback(
    (id: string | null) => {
      router.push({ segments: id ? [id] : [] });
    },
    [router],
  );
  return [selection, setSelection];
}

// Build a href="" for an anchor or button. Use this in nav UI so the browser
// treats the link normally (middle-click, cmd-click) and we don't have to
// prevent defaults.
export function hrefFor(patch: PathPatch, base?: Route): string {
  if (base) return '#' + serialisePatch(base, patch);
  // Standalone — caller didn't pass a base, build from blanks.
  return '#' + serialisePatch({ view: '', segments: [], params: {}, raw: '' }, patch);
}
