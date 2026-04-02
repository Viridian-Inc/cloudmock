import { createContext } from 'preact';
import { useState, useEffect, useCallback, useContext } from 'preact/hooks';
import type { ComponentChildren } from 'preact';

export interface ConnectionState {
  adminUrl: string;
  gatewayUrl: string;
  connected: boolean;
  region: string;
  profile: string;
  iamMode: string;
  serviceCount: number;
  pid: number | null;
  lastHealthCheck: number | null;
}

interface ConnectionContextValue {
  state: ConnectionState;
  connect: (adminUrl: string, gatewayUrl: string) => void;
  disconnect: () => void;
  isFirstLaunch: boolean;
}

const DEFAULT_STATE: ConnectionState = {
  adminUrl: typeof window !== 'undefined' && (window.location.port === '1420' || window.location.port === '4501')
    ? `${window.location.protocol}//${window.location.hostname}:4599`
    : '',
  gatewayUrl: typeof window !== 'undefined' && (window.location.port === '1420' || window.location.port === '4501')
    ? `${window.location.protocol}//${window.location.hostname}:4566`
    : 'http://localhost:4566',
  connected: false,
  region: '',
  profile: '',
  iamMode: '',
  serviceCount: 0,
  pid: null,
  lastHealthCheck: null,
};

const ConnectionContext = createContext<ConnectionContextValue>({
  state: DEFAULT_STATE,
  connect: () => {},
  disconnect: () => {},
  isFirstLaunch: true,
});

export function useConnection() {
  return useContext(ConnectionContext);
}

export function ConnectionProvider({ children }: { children: ComponentChildren }) {
  const [state, setState] = useState<ConnectionState>(() => {
    const saved = localStorage.getItem('neureaux-devtools:connection');
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        return { ...DEFAULT_STATE, adminUrl: parsed.adminUrl, gatewayUrl: parsed.gatewayUrl };
      } catch (e) { console.warn('[Connection] Failed to parse saved connection:', e); }
    }
    return DEFAULT_STATE;
  });

  const [isFirstLaunch, setIsFirstLaunch] = useState(() => {
    return !localStorage.getItem('neureaux-devtools:connection');
  });

  // Poll health every 3 seconds
  useEffect(() => {
    if (!state.adminUrl) return;

    let cancelled = false;

    async function poll() {
      while (!cancelled) {
        try {
          const res = await fetch(`${state.adminUrl}/api/health`);
          if (res.ok) {
            const data = await res.json();
            if (!cancelled) {
              setState((prev) => ({
                ...prev,
                connected: data.status === 'ok' || data.status === 'healthy',
                region: data.region || prev.region || 'us-east-1',
                serviceCount: data.services ? Object.keys(data.services).length : prev.serviceCount,
                lastHealthCheck: Date.now(),
              }));
            }
          } else {
            if (!cancelled) setState((prev) => ({ ...prev, connected: false }));
          }
        } catch (e) {
          console.debug('[Connection] Health poll failed:', e);
          if (!cancelled) setState((prev) => ({ ...prev, connected: false }));
        }
        await new Promise((r) => setTimeout(r, 3000));
      }
    }

    poll();
    return () => { cancelled = true; };
  }, [state.adminUrl]);

  const connect = useCallback((adminUrl: string, gatewayUrl: string) => {
    localStorage.setItem('neureaux-devtools:connection', JSON.stringify({ adminUrl, gatewayUrl }));
    setState((prev) => ({ ...prev, adminUrl, gatewayUrl, connected: false }));
    setIsFirstLaunch(false);
  }, []);

  const disconnect = useCallback(() => {
    setState((prev) => ({ ...prev, connected: false }));
  }, []);

  return (
    <ConnectionContext.Provider value={{ state, connect, disconnect, isFirstLaunch }}>
      {children}
    </ConnectionContext.Provider>
  );
}
