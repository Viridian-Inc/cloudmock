const SESSION_KEY = '__cloudmock_rum_sid';
const SESSION_TIMEOUT_MS = 30 * 60 * 1000; // 30 minutes

let currentSessionId: string | null = null;
let lastActivity = 0;

function generateId(): string {
  const bytes = new Uint8Array(16);
  if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
    crypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < 16; i++) bytes[i] = (Math.random() * 256) | 0;
  }
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
}

export function getSessionId(): string {
  const now = Date.now();

  // If we have a current session and it hasn't expired, reuse it.
  if (currentSessionId && now - lastActivity < SESSION_TIMEOUT_MS) {
    lastActivity = now;
    return currentSessionId;
  }

  // Try restoring from sessionStorage (survives soft navigations).
  try {
    const stored = sessionStorage.getItem(SESSION_KEY);
    if (stored) {
      const { id, ts } = JSON.parse(stored);
      if (now - ts < SESSION_TIMEOUT_MS) {
        currentSessionId = id;
        lastActivity = now;
        persistSession();
        return id;
      }
    }
  } catch {
    // sessionStorage unavailable (e.g. SSR, privacy mode)
  }

  // Create a new session.
  currentSessionId = generateId();
  lastActivity = now;
  persistSession();
  return currentSessionId;
}

function persistSession(): void {
  try {
    sessionStorage.setItem(
      SESSION_KEY,
      JSON.stringify({ id: currentSessionId, ts: lastActivity })
    );
  } catch {
    // ignore
  }
}

/**
 * Returns true if this session should be sampled.
 * Decision is sticky per session — once decided, it doesn't change.
 */
const sampledSessions = new Map<string, boolean>();

export function shouldSample(sampleRate: number): boolean {
  if (sampleRate >= 1) return true;
  const id = getSessionId();
  const existing = sampledSessions.get(id);
  if (existing !== undefined) return existing;

  const accepted = Math.random() < sampleRate;
  sampledSessions.set(id, accepted);
  return accepted;
}
