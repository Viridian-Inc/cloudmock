export interface DomainConfig {
  domains: DomainGroup[];
  userFacingOverrides?: Record<string, boolean>;
  routing?: Record<string, Record<string, string>>;
}

export interface DomainGroup {
  name: string;
  services: string[];
}

let _domainConfig: DomainConfig | null = null;

/**
 * Load domain config from service-domains.json.
 * Tries Tauri file read first, falls back to fetch.
 */
export async function loadDomainConfig(): Promise<DomainConfig> {
  if (_domainConfig) return _domainConfig;

  try {
    // Try fetching from the dev server (Vite serves static files from root)
    const res = await fetch('/service-domains.json');
    if (res.ok) {
      _domainConfig = await res.json();
      return _domainConfig!;
    }
  } catch (e) { console.warn('[Domains] Failed to fetch service-domains.json:', e); }

  // Fallback: empty config
  _domainConfig = { domains: [] };
  return _domainConfig;
}

/**
 * Get the domain group a service belongs to.
 */
export function getServiceDomain(
  serviceName: string,
  config: DomainConfig,
): string | undefined {
  for (const domain of config.domains) {
    if (domain.services.includes(serviceName)) {
      return domain.name;
    }
  }
  return undefined;
}

/**
 * Check if a service has a manual user-facing override.
 */
export function getUserFacingOverride(
  serviceName: string,
  config: DomainConfig,
): boolean | undefined {
  return config.userFacingOverrides?.[serviceName];
}

/**
 * Get routing config for a service.
 */
export function getRoutingConfig(
  serviceName: string,
  config: DomainConfig,
): Record<string, string> | undefined {
  return config.routing?.[serviceName];
}
