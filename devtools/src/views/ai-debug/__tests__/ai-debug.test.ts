import { describe, it, expect, vi, afterEach } from 'vitest';
import {
  parseInlineMarkdown,
  parseMarkdownLine,
  extractCodeBlocks,
  filterRecentErrors,
  formatRelativeTime,
} from '../helpers';
import type { RecentTrace } from '../helpers';

describe('parseInlineMarkdown', () => {
  it('returns plain text as single segment', () => {
    const result = parseInlineMarkdown('hello world');
    expect(result).toEqual([{ type: 'text', content: 'hello world' }]);
  });

  it('parses inline code', () => {
    const result = parseInlineMarkdown('use `console.log` here');
    expect(result).toEqual([
      { type: 'text', content: 'use ' },
      { type: 'code', content: 'console.log' },
      { type: 'text', content: ' here' },
    ]);
  });

  it('parses bold text', () => {
    const result = parseInlineMarkdown('this is **important** text');
    expect(result).toEqual([
      { type: 'text', content: 'this is ' },
      { type: 'bold', content: 'important' },
      { type: 'text', content: ' text' },
    ]);
  });

  it('parses mixed inline code and bold', () => {
    const result = parseInlineMarkdown('run `npm install` for **fast** setup');
    expect(result).toHaveLength(5);
    expect(result[1]).toEqual({ type: 'code', content: 'npm install' });
    expect(result[3]).toEqual({ type: 'bold', content: 'fast' });
  });

  it('handles empty string', () => {
    expect(parseInlineMarkdown('')).toEqual([]);
  });

  it('handles consecutive formatting', () => {
    const result = parseInlineMarkdown('`a``b`');
    expect(result).toHaveLength(2);
    expect(result[0]).toEqual({ type: 'code', content: 'a' });
    expect(result[1]).toEqual({ type: 'code', content: 'b' });
  });
});

describe('parseMarkdownLine', () => {
  it('parses h1 heading', () => {
    const result = parseMarkdownLine('# Title');
    expect(result).toEqual({ type: 'h1', content: 'Title' });
  });

  it('parses h2 heading', () => {
    const result = parseMarkdownLine('## Subtitle');
    expect(result).toEqual({ type: 'h2', content: 'Subtitle' });
  });

  it('parses h3 heading', () => {
    const result = parseMarkdownLine('### Section');
    expect(result).toEqual({ type: 'h3', content: 'Section' });
  });

  it('parses dash list item', () => {
    const result = parseMarkdownLine('- item one');
    expect(result).toEqual({ type: 'list-item', content: 'item one' });
  });

  it('parses asterisk list item', () => {
    const result = parseMarkdownLine('* item two');
    expect(result).toEqual({ type: 'list-item', content: 'item two' });
  });

  it('parses empty line as spacer', () => {
    expect(parseMarkdownLine('')).toEqual({ type: 'spacer', content: '' });
    expect(parseMarkdownLine('  ')).toEqual({ type: 'spacer', content: '' });
  });

  it('parses regular text as paragraph', () => {
    const result = parseMarkdownLine('Some regular text');
    expect(result).toEqual({ type: 'paragraph', content: 'Some regular text' });
  });
});

describe('extractCodeBlocks', () => {
  it('returns single text segment for no code blocks', () => {
    const result = extractCodeBlocks('just plain text');
    expect(result).toEqual([{ type: 'text', content: 'just plain text' }]);
  });

  it('extracts a fenced code block', () => {
    const input = 'before\n```js\nconsole.log("hi");\n```\nafter';
    const result = extractCodeBlocks(input);
    expect(result).toHaveLength(3);
    expect(result[0]).toEqual({ type: 'text', content: 'before\n' });
    expect(result[1]).toEqual({
      type: 'code-block',
      content: 'console.log("hi");\n',
      language: 'js',
    });
    expect(result[2]).toEqual({ type: 'text', content: '\nafter' });
  });

  it('handles code block without language', () => {
    const input = '```\ncode here\n```';
    const result = extractCodeBlocks(input);
    expect(result).toHaveLength(1);
    expect(result[0].type).toBe('code-block');
    expect(result[0].language).toBeUndefined();
  });

  it('handles multiple code blocks', () => {
    const input = 'text\n```py\nprint(1)\n```\nmiddle\n```go\nfmt.Println()\n```\nend';
    const result = extractCodeBlocks(input);
    const codeBlocks = result.filter((s) => s.type === 'code-block');
    expect(codeBlocks).toHaveLength(2);
    expect(codeBlocks[0].language).toBe('py');
    expect(codeBlocks[1].language).toBe('go');
  });
});

describe('filterRecentErrors', () => {
  const traces: RecentTrace[] = [
    { id: '1', service: 'api', action: 'GET /health', status_code: 200, timestamp: '2025-01-15T10:00:00Z' },
    { id: '2', service: 'api', action: 'POST /order', status_code: 500, timestamp: '2025-01-15T10:01:00Z' },
    { id: '3', service: 'api', action: 'GET /user', status_code: 404, timestamp: '2025-01-15T10:02:00Z' },
    { id: '4', service: 'api', action: 'PUT /item', status_code: 403, timestamp: '2025-01-15T10:03:00Z' },
    { id: '5', service: 'api', action: 'GET /list', status_code: 200, timestamp: '2025-01-15T10:04:00Z' },
    { id: '6', service: 'api', action: 'DELETE /old', status_code: 502, timestamp: '2025-01-15T10:05:00Z' },
  ];

  it('filters only traces with status_code >= 400', () => {
    const errors = filterRecentErrors(traces);
    expect(errors.every((t) => t.status_code >= 400)).toBe(true);
    expect(errors).toHaveLength(4);
  });

  it('sorts by most recent first', () => {
    const errors = filterRecentErrors(traces);
    expect(errors[0].id).toBe('6'); // 10:05
    expect(errors[1].id).toBe('4'); // 10:03
  });

  it('respects limit parameter', () => {
    const errors = filterRecentErrors(traces, 2);
    expect(errors).toHaveLength(2);
  });

  it('returns empty for all-successful traces', () => {
    const ok = [
      { id: '1', service: 'api', action: 'GET', status_code: 200, timestamp: '2025-01-15T10:00:00Z' },
    ];
    expect(filterRecentErrors(ok)).toHaveLength(0);
  });

  it('handles empty array', () => {
    expect(filterRecentErrors([])).toEqual([]);
  });
});

describe('formatRelativeTime', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('formats seconds ago', () => {
    vi.spyOn(Date, 'now').mockReturnValue(new Date('2025-01-15T10:00:30Z').getTime());
    expect(formatRelativeTime('2025-01-15T10:00:00Z')).toBe('30s ago');
  });

  it('formats minutes ago', () => {
    vi.spyOn(Date, 'now').mockReturnValue(new Date('2025-01-15T10:10:00Z').getTime());
    expect(formatRelativeTime('2025-01-15T10:00:00Z')).toBe('10m ago');
  });

  it('formats hours ago', () => {
    vi.spyOn(Date, 'now').mockReturnValue(new Date('2025-01-15T15:00:00Z').getTime());
    expect(formatRelativeTime('2025-01-15T10:00:00Z')).toBe('5h ago');
  });

  it('formats days ago', () => {
    vi.spyOn(Date, 'now').mockReturnValue(new Date('2025-01-18T10:00:00Z').getTime());
    expect(formatRelativeTime('2025-01-15T10:00:00Z')).toBe('3d ago');
  });
});
