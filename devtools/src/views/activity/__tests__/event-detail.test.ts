import { describe, it, expect } from 'vitest';
import { buildCurlCommand, byteSize, formatBytes, isXml, tokenizeJson } from '../event-utils';
import type { RequestEvent } from '../../../lib/types';

function makeEvent(overrides: Partial<RequestEvent> = {}): RequestEvent {
  return {
    id: 'evt-1',
    service: 'bff-service',
    action: 'GetUser',
    method: 'GET',
    path: '/api/users/123',
    status_code: 200,
    latency_ms: 12,
    timestamp: '2025-01-01T00:00:00Z',
    ...overrides,
  };
}

describe('buildCurlCommand', () => {
  it('generates basic GET curl', () => {
    const event = makeEvent();
    const curl = buildCurlCommand(event);
    expect(curl).toContain("curl -X GET '/api/users/123'");
  });

  it('includes request headers', () => {
    const event = makeEvent({
      request_headers: {
        'Authorization': 'Bearer tok123',
        'Content-Type': 'application/json',
      },
    });
    const curl = buildCurlCommand(event);
    expect(curl).toContain("-H 'Authorization: Bearer tok123'");
    expect(curl).toContain("-H 'Content-Type: application/json'");
  });

  it('includes request body for POST', () => {
    const event = makeEvent({
      method: 'POST',
      path: '/api/users',
      request_body: '{"name":"Alice"}',
    });
    const curl = buildCurlCommand(event);
    expect(curl).toContain("curl -X POST '/api/users'");
    expect(curl).toContain("-d '{\"name\":\"Alice\"}'");
  });

  it('escapes single quotes in header values', () => {
    const event = makeEvent({
      request_headers: { 'X-Custom': "it's a value" },
    });
    const curl = buildCurlCommand(event);
    expect(curl).toContain("it'\\''s a value");
  });

  it('omits -d flag when body is empty string', () => {
    const event = makeEvent({ request_body: '' });
    const curl = buildCurlCommand(event);
    expect(curl).not.toContain('-d');
  });

  it('omits -d flag when body is null', () => {
    const event = makeEvent({ request_body: undefined });
    const curl = buildCurlCommand(event);
    expect(curl).not.toContain('-d');
  });

  it('joins parts with backslash-newline continuation', () => {
    const event = makeEvent({
      request_headers: { 'Accept': 'application/json' },
    });
    const curl = buildCurlCommand(event);
    expect(curl).toContain(' \\\n  ');
  });
});

describe('byteSize', () => {
  it('returns 0 for null', () => {
    expect(byteSize(null)).toBe(0);
  });

  it('returns 0 for undefined', () => {
    expect(byteSize(undefined)).toBe(0);
  });

  it('returns correct byte count for ASCII', () => {
    expect(byteSize('hello')).toBe(5);
  });

  it('returns correct byte count for multi-byte characters', () => {
    // UTF-8: each character may be > 1 byte
    expect(byteSize('\u00e9')).toBe(2); // e-acute is 2 bytes in UTF-8
  });

  it('handles object by JSON-stringifying', () => {
    const obj = { key: 'value' };
    const expected = new TextEncoder().encode(JSON.stringify(obj)).length;
    expect(byteSize(obj)).toBe(expected);
  });
});

describe('formatBytes', () => {
  it('formats 0 as "0 B"', () => {
    expect(formatBytes(0)).toBe('0 B');
  });

  it('formats bytes', () => {
    expect(formatBytes(512)).toBe('512 B');
  });

  it('formats kilobytes', () => {
    expect(formatBytes(2048)).toBe('2.0 KB');
  });

  it('formats megabytes', () => {
    expect(formatBytes(1048576)).toBe('1.00 MB');
  });

  it('formats fractional kilobytes', () => {
    expect(formatBytes(1536)).toBe('1.5 KB');
  });
});

describe('isXml', () => {
  it('detects XML declaration', () => {
    expect(isXml('<?xml version="1.0"?>')).toBe(true);
  });

  it('detects element wrapper', () => {
    expect(isXml('<root><child/></root>')).toBe(true);
  });

  it('handles whitespace around XML', () => {
    expect(isXml('  <?xml version="1.0"?>  ')).toBe(true);
  });

  it('returns false for JSON', () => {
    expect(isXml('{"key": "value"}')).toBe(false);
  });

  it('returns false for plain text', () => {
    expect(isXml('hello world')).toBe(false);
  });
});

describe('tokenizeJson', () => {
  it('tokenizes keys and string values', () => {
    const tokens = tokenizeJson('{"name": "Alice"}');
    const types = tokens.map((t) => t.type);
    expect(types).toContain('key');
    expect(types).toContain('string');
  });

  it('tokenizes numbers', () => {
    const tokens = tokenizeJson('{"age": 30}');
    const numTokens = tokens.filter((t) => t.type === 'number');
    expect(numTokens.length).toBe(1);
    expect(numTokens[0].text).toBe('30');
  });

  it('tokenizes booleans', () => {
    const tokens = tokenizeJson('{"active": true}');
    const boolTokens = tokens.filter((t) => t.type === 'boolean');
    expect(boolTokens.length).toBe(1);
    expect(boolTokens[0].text).toBe('true');
  });

  it('tokenizes null', () => {
    const tokens = tokenizeJson('{"data": null}');
    const nullTokens = tokens.filter((t) => t.type === 'null');
    expect(nullTokens.length).toBe(1);
  });

  it('tokenizes punctuation', () => {
    const tokens = tokenizeJson('{"a": [1, 2]}');
    const punctTokens = tokens.filter((t) => t.type === 'punctuation');
    // Should include {, }, [, ], commas, colons, spaces
    expect(punctTokens.length).toBeGreaterThan(0);
  });
});
