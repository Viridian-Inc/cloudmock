/** Environment configuration for multi-environment support. */

export interface Environment {
  id: string;
  name: string;
  endpoint: string;
  color: string;
  isProduction: boolean;
}

const STORAGE_KEY = 'neureaux-devtools:environments';
const ACTIVE_KEY = 'neureaux-devtools:active-environment';

/** Built-in environment presets. Users can customize endpoints. */
export const DEFAULT_ENVIRONMENTS: Environment[] = [
  {
    id: 'local',
    name: 'Local',
    endpoint: 'http://localhost:4599',
    color: '#029662', // green (matches --success)
    isProduction: false,
  },
  {
    id: 'dev',
    name: 'Dev',
    endpoint: '',
    color: '#7ccef2', // blue (matches --info)
    isProduction: false,
  },
  {
    id: 'staging',
    name: 'Staging',
    endpoint: '',
    color: '#FF9A4B', // yellow/orange (matches --warning)
    isProduction: false,
  },
  {
    id: 'production',
    name: 'Production',
    endpoint: '',
    color: '#FF4E5E', // red (matches --error)
    isProduction: true,
  },
];

/** Load environments from localStorage, falling back to defaults. */
export function loadEnvironments(): Environment[] {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored) as Environment[];
      if (Array.isArray(parsed) && parsed.length > 0) {
        return parsed;
      }
    }
  } catch {
    // localStorage unavailable or corrupt
  }
  return [...DEFAULT_ENVIRONMENTS];
}

/** Save environments to localStorage. */
export function saveEnvironments(envs: Environment[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(envs));
  } catch {
    // localStorage unavailable
  }
}

/** Load the active environment ID from localStorage. */
export function loadActiveEnvironmentId(): string {
  try {
    const stored = localStorage.getItem(ACTIVE_KEY);
    if (stored) return stored;
  } catch {
    // localStorage unavailable
  }
  return 'local';
}

/** Save the active environment ID to localStorage. */
export function saveActiveEnvironmentId(id: string): void {
  try {
    localStorage.setItem(ACTIVE_KEY, id);
  } catch {
    // localStorage unavailable
  }
}

/** Get the active environment object from a list of environments. */
export function getActiveEnvironment(
  envs: Environment[],
  activeId: string,
): Environment | undefined {
  return envs.find((e) => e.id === activeId) || envs[0];
}
