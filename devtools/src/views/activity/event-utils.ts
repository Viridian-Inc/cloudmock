import type { RequestEvent } from '../../lib/types';

/** Generate a curl command string from a request event */
export function buildCurlCommand(event: RequestEvent): string {
  const parts: string[] = [`curl -X ${event.method} '${event.path}'`];

  const headers = event.request_headers ?? {};
  for (const [key, value] of Object.entries(headers)) {
    const escaped = value.replace(/'/g, "'\\''");
    parts.push(`-H '${key}: ${escaped}'`);
  }

  if (event.request_body != null && event.request_body !== '') {
    const body = typeof event.request_body === 'string'
      ? event.request_body
      : JSON.stringify(event.request_body);
    const escaped = body.replace(/'/g, "'\\''");
    parts.push(`-d '${escaped}'`);
  }

  return parts.join(' \\\n  ');
}

/** Compute byte size of a string body */
export function byteSize(body: unknown): number {
  if (body == null) return 0;
  const str = typeof body === 'string' ? body : JSON.stringify(body);
  return new TextEncoder().encode(str).length;
}

/** Format byte size for display */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

/** Try to detect if a string is XML */
export function isXml(str: string): boolean {
  const trimmed = str.trim();
  return trimmed.startsWith('<?xml') || (trimmed.startsWith('<') && trimmed.endsWith('>'));
}

/** Tokenize JSON string for syntax highlighting */
export interface JsonToken {
  type: 'key' | 'string' | 'number' | 'boolean' | 'null' | 'punctuation';
  text: string;
}

export function tokenizeJson(jsonStr: string): JsonToken[] {
  const tokens: JsonToken[] = [];
  const regex = /("(?:[^"\\]|\\.)*")(\s*:)?|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|(\btrue\b|\bfalse\b)|(\bnull\b)|([{}[\]:,\s])/g;
  let match: RegExpExecArray | null;

  while ((match = regex.exec(jsonStr)) !== null) {
    if (match[1]) {
      if (match[2]) {
        tokens.push({ type: 'key', text: match[1] });
        tokens.push({ type: 'punctuation', text: match[2] });
      } else {
        tokens.push({ type: 'string', text: match[1] });
      }
    } else if (match[3]) {
      tokens.push({ type: 'number', text: match[3] });
    } else if (match[4]) {
      tokens.push({ type: 'boolean', text: match[4] });
    } else if (match[5]) {
      tokens.push({ type: 'null', text: match[5] });
    } else if (match[6]) {
      tokens.push({ type: 'punctuation', text: match[6] });
    }
  }
  return tokens;
}
