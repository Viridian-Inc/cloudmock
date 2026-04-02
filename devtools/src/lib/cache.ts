const CACHE_PREFIX = 'neureaux:cache:';

export function cacheSet(key: string, data: any, ttlMs: number = 5 * 60 * 1000): void {
  try {
    localStorage.setItem(CACHE_PREFIX + key, JSON.stringify({
      data,
      expiresAt: Date.now() + ttlMs,
      cachedAt: Date.now(),
    }));
  } catch (e) { console.debug('[Cache] Failed to write:', key, e); }
}

export function cacheGet<T>(key: string): { data: T; cachedAt: number; stale: boolean } | null {
  try {
    const raw = localStorage.getItem(CACHE_PREFIX + key);
    if (!raw) return null;
    const { data, expiresAt, cachedAt } = JSON.parse(raw);
    return { data, cachedAt, stale: Date.now() > expiresAt };
  } catch (e) { console.debug('[Cache] Failed to read:', key, e); return null; }
}
