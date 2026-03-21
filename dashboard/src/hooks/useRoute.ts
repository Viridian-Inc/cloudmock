import { useState, useEffect } from 'preact/hooks';

export function useRoute() {
  const [hash, setHash] = useState(location.hash || '#/');
  useEffect(() => {
    const handler = () => setHash(location.hash || '#/');
    window.addEventListener('hashchange', handler);
    return () => window.removeEventListener('hashchange', handler);
  }, []);

  const path = hash.replace('#', '') || '/';
  const segments = path.split('/').filter(Boolean);
  return { path, segments, activePath: '/' + (segments[0] || '') };
}
