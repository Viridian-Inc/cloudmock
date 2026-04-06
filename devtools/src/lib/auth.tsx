import { createContext } from 'preact';
import { useState, useEffect, useCallback, useContext } from 'preact/hooks';
import type { ComponentChildren } from 'preact';

/** SaaS configuration returned by GET /api/saas/config */
export interface SaaSConfig {
  saas_enabled: boolean;
  auth_enabled: boolean;
  clerk_publishable_key?: string;
  clerk_domain?: string;
}

export interface AuthState {
  /** Whether the connected instance has SaaS mode enabled */
  saasEnabled: boolean;
  /** Clerk publishable key (only set in hosted mode) */
  clerkPublishableKey: string | null;
  /** Clerk domain for JWKS */
  clerkDomain: string | null;
  /** Current Bearer token (from Clerk sign-in or API key) */
  token: string | null;
  /** User email from Clerk claims */
  email: string | null;
  /** Organization ID from Clerk claims */
  orgId: string | null;
  /** Organization slug (used for endpoint subdomain) */
  orgSlug: string | null;
  /** Whether auth is loading */
  loading: boolean;
}

interface AuthContextValue {
  auth: AuthState;
  setToken: (token: string, claims?: { email?: string; org_id?: string; org_slug?: string }) => void;
  clearToken: () => void;
  fetchSaaSConfig: (adminUrl: string) => Promise<SaaSConfig | null>;
}

const DEFAULT_AUTH: AuthState = {
  saasEnabled: false,
  clerkPublishableKey: null,
  clerkDomain: null,
  token: null,
  email: null,
  orgId: null,
  orgSlug: null,
  loading: false,
};

const AuthContext = createContext<AuthContextValue>({
  auth: DEFAULT_AUTH,
  setToken: () => {},
  clearToken: () => {},
  fetchSaaSConfig: async () => null,
});

export function useAuth() {
  return useContext(AuthContext);
}

const TOKEN_KEY = 'cloudmock:auth-token';
const CLAIMS_KEY = 'cloudmock:auth-claims';

export function AuthProvider({ children }: { children: ComponentChildren }) {
  const [auth, setAuth] = useState<AuthState>(() => {
    // Restore token from localStorage on mount
    const savedToken = localStorage.getItem(TOKEN_KEY);
    const savedClaims = localStorage.getItem(CLAIMS_KEY);
    let claims: any = {};
    if (savedClaims) {
      try { claims = JSON.parse(savedClaims); } catch { /* ignore */ }
    }
    return {
      ...DEFAULT_AUTH,
      token: savedToken,
      email: claims.email || null,
      orgId: claims.org_id || null,
      orgSlug: claims.org_slug || null,
    };
  });

  const setToken = useCallback((token: string, claims?: { email?: string; org_id?: string; org_slug?: string }) => {
    localStorage.setItem(TOKEN_KEY, token);
    if (claims) {
      localStorage.setItem(CLAIMS_KEY, JSON.stringify(claims));
    }
    setAuth((prev) => ({
      ...prev,
      token,
      email: claims?.email || prev.email,
      orgId: claims?.org_id || prev.orgId,
      orgSlug: claims?.org_slug || prev.orgSlug,
    }));
  }, []);

  const clearToken = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(CLAIMS_KEY);
    setAuth((prev) => ({
      ...prev,
      token: null,
      email: null,
      orgId: null,
      orgSlug: null,
    }));
  }, []);

  const fetchSaaSConfig = useCallback(async (adminUrl: string): Promise<SaaSConfig | null> => {
    setAuth((prev) => ({ ...prev, loading: true }));
    try {
      const res = await fetch(`${adminUrl}/api/saas/config`);
      if (!res.ok) {
        // Not a SaaS instance — that's fine (local mode)
        setAuth((prev) => ({ ...prev, saasEnabled: false, loading: false }));
        return null;
      }
      const config: SaaSConfig = await res.json();
      setAuth((prev) => ({
        ...prev,
        saasEnabled: config.saas_enabled,
        clerkPublishableKey: config.clerk_publishable_key || null,
        clerkDomain: config.clerk_domain || null,
        loading: false,
      }));
      return config;
    } catch {
      // Connection failed — not a SaaS concern
      setAuth((prev) => ({ ...prev, saasEnabled: false, loading: false }));
      return null;
    }
  }, []);

  return (
    <AuthContext.Provider value={{ auth, setToken, clearToken, fetchSaaSConfig }}>
      {children}
    </AuthContext.Provider>
  );
}
