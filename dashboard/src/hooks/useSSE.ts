import { useState, useEffect } from 'preact/hooks';
import { ADMIN_BASE } from '../api';

export interface SSEEvent {
  type: string;
  data: any;
}

export interface SSEState {
  connected: boolean;
  events: SSEEvent[];
}

export function useSSE(): SSEState {
  const [connected, setConnected] = useState(false);
  const [events, setEvents] = useState<SSEEvent[]>([]);

  useEffect(() => {
    const es = new EventSource(`${ADMIN_BASE}/api/stream`);
    es.onopen = () => setConnected(true);
    es.onerror = () => setConnected(false);
    es.onmessage = (e) => {
      try {
        const event: SSEEvent = JSON.parse(e.data);
        setEvents(prev => [event, ...prev].slice(0, 500));
      } catch { /* ignore parse errors */ }
    };
    return () => es.close();
  }, []);

  return { connected, events };
}
