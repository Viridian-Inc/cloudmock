import { useState, useEffect, useCallback, useRef } from 'preact/hooks';
import { getAdminBase } from '../lib/api';
import type { SSEEvent } from '../lib/types';

const MAX_BUFFER_SIZE = 500;

interface UseSSEResult {
  connected: boolean;
  events: SSEEvent[];
  clear: () => void;
}

/**
 * SSE hook — same pattern as the working cloudmock dashboard.
 * Connects to {adminBase}/api/stream via EventSource.
 */
export function useSSE(): UseSSEResult {
  const [connected, setConnected] = useState(false);
  const [events, setEvents] = useState<SSEEvent[]>([]);
  const eventsRef = useRef<SSEEvent[]>([]);

  const clear = useCallback(() => {
    eventsRef.current = [];
    setEvents([]);
  }, []);

  useEffect(() => {
    const adminBase = getAdminBase();
    const es = new EventSource(`${adminBase}/api/stream`);

    es.onopen = () => setConnected(true);
    es.onerror = () => setConnected(false);
    es.onmessage = (e) => {
      try {
        const event: SSEEvent = {
          type: 'message',
          data: JSON.parse(e.data),
          timestamp: Date.now(),
        };
        const next = [event, ...eventsRef.current].slice(0, MAX_BUFFER_SIZE);
        eventsRef.current = next;
        setEvents(next);
      } catch (e) { console.warn('[SSE] Parse error:', e); }
    };

    return () => es.close();
  }, []);

  return { connected, events, clear };
}
