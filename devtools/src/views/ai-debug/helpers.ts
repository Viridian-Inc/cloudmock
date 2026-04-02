/**
 * Pure functions extracted from AIDebugView for testing.
 * Markdown parsing and recent error suggestion filtering.
 */

export interface RecentTrace {
  id: string;
  trace_id?: string;
  service: string;
  action: string;
  status_code: number;
  timestamp: string;
  path?: string;
}

/**
 * Format inline markdown: backtick code and bold.
 * Returns an array of text segments with type annotations.
 */
export interface InlineSegment {
  type: 'text' | 'code' | 'bold';
  content: string;
}

export function parseInlineMarkdown(text: string): InlineSegment[] {
  const segments: InlineSegment[] = [];
  const regex = /(`[^`]+`|\*\*[^*]+\*\*)/g;
  let last = 0;
  let match: RegExpExecArray | null;

  while ((match = regex.exec(text)) !== null) {
    if (match.index > last) {
      segments.push({ type: 'text', content: text.slice(last, match.index) });
    }
    const token = match[0];
    if (token.startsWith('`')) {
      segments.push({ type: 'code', content: token.slice(1, -1) });
    } else {
      segments.push({ type: 'bold', content: token.slice(2, -2) });
    }
    last = match.index + token.length;
  }

  if (last < text.length) {
    segments.push({ type: 'text', content: text.slice(last) });
  }

  return segments;
}

/**
 * Parse a line of markdown into its block-level type.
 */
export type BlockType = 'h1' | 'h2' | 'h3' | 'list-item' | 'spacer' | 'paragraph';

export interface BlockLine {
  type: BlockType;
  content: string;
}

export function parseMarkdownLine(line: string): BlockLine {
  if (line.startsWith('### ')) return { type: 'h3', content: line.slice(4) };
  if (line.startsWith('## ')) return { type: 'h2', content: line.slice(3) };
  if (line.startsWith('# ')) return { type: 'h1', content: line.slice(2) };
  if (line.startsWith('- ') || line.startsWith('* ')) return { type: 'list-item', content: line.slice(2) };
  if (line.trim() === '') return { type: 'spacer', content: '' };
  return { type: 'paragraph', content: line };
}

/**
 * Extract fenced code blocks from markdown text.
 * Returns an array of segments, each either a code block or plain text.
 */
export interface MarkdownSegment {
  type: 'text' | 'code-block';
  content: string;
  language?: string;
}

export function extractCodeBlocks(text: string): MarkdownSegment[] {
  const segments: MarkdownSegment[] = [];
  const codeBlockRegex = /```(\w*)\n([\s\S]*?)```/g;
  let lastIndex = 0;
  let match;

  while ((match = codeBlockRegex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      segments.push({ type: 'text', content: text.slice(lastIndex, match.index) });
    }
    segments.push({ type: 'code-block', content: match[2], language: match[1] || undefined });
    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < text.length) {
    segments.push({ type: 'text', content: text.slice(lastIndex) });
  }

  return segments;
}

/**
 * Filter and sort traces to get recent errors (status_code >= 400),
 * sorted by most recent first, limited to `limit` entries.
 */
export function filterRecentErrors(traces: RecentTrace[], limit: number = 5): RecentTrace[] {
  return traces
    .filter((t) => t.status_code >= 400)
    .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
    .slice(0, limit);
}

export function formatRelativeTime(timestamp: string): string {
  const now = Date.now();
  const then = new Date(timestamp).getTime();
  const diffMs = now - then;
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  return `${Math.floor(diffHr / 24)}d ago`;
}
