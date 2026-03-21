import { useState, useRef, useEffect, useCallback } from 'preact/hooks';
import { ddbRequest } from '../../api';
import { DDBItem } from './types';

interface TerminalProps {
  tableName: string;
  showToast: (msg: string) => void;
}

interface TerminalLine {
  type: 'input' | 'output' | 'error';
  text: string;
}

export function Terminal({ tableName, showToast }: TerminalProps) {
  const [lines, setLines] = useState<TerminalLine[]>([
    { type: 'output', text: `CloudMock DynamoDB REPL - Table: ${tableName}` },
    { type: 'output', text: `Commands: scan(table), query(table, {pk: val}), put(table, item), del(table, key), tables(), clear` },
    { type: 'output', text: `Type JavaScript expressions. Results are pretty-printed.\n` },
  ]);
  const [input, setInput] = useState('');
  const [history, setHistory] = useState<string[]>([]);
  const [historyIdx, setHistoryIdx] = useState(-1);
  const [running, setRunning] = useState(false);
  const termRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (termRef.current) {
      termRef.current.scrollTop = termRef.current.scrollHeight;
    }
  }, [lines]);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const addLine = useCallback((type: TerminalLine['type'], text: string) => {
    setLines(prev => [...prev, { type, text }]);
  }, []);

  // Helper functions available in the REPL
  async function execScan(table: string, opts?: any): Promise<DDBItem[]> {
    const params: any = { TableName: table, ...(opts || {}) };
    const r = await ddbRequest('Scan', params);
    return r.Items || [];
  }

  async function execQuery(table: string, keyConditions: Record<string, any>, opts?: any): Promise<DDBItem[]> {
    const exprNames: Record<string, string> = {};
    const exprValues: Record<string, any> = {};
    const keyParts: string[] = [];
    let i = 0;
    for (const [attr, val] of Object.entries(keyConditions)) {
      const nk = `#k${i}`;
      const vk = `:v${i}`;
      exprNames[nk] = attr;
      // Auto-detect type
      if (typeof val === 'number') {
        exprValues[vk] = { N: String(val) };
      } else if (typeof val === 'object' && val !== null) {
        exprValues[vk] = val; // Already typed
      } else {
        exprValues[vk] = { S: String(val) };
      }
      keyParts.push(`${nk} = ${vk}`);
      i++;
    }
    const params: any = {
      TableName: table,
      KeyConditionExpression: keyParts.join(' AND '),
      ExpressionAttributeNames: exprNames,
      ExpressionAttributeValues: exprValues,
      ...(opts || {}),
    };
    const r = await ddbRequest('Query', params);
    return r.Items || [];
  }

  async function execPut(table: string, item: DDBItem): Promise<string> {
    await ddbRequest('PutItem', { TableName: table, Item: item });
    return 'Item written successfully.';
  }

  async function execDel(table: string, key: DDBItem): Promise<string> {
    await ddbRequest('DeleteItem', { TableName: table, Key: key });
    return 'Item deleted.';
  }

  async function execTables(): Promise<string[]> {
    const r = await ddbRequest('ListTables', {});
    return r.TableNames || [];
  }

  async function executeCommand(cmd: string) {
    setRunning(true);
    addLine('input', cmd);

    try {
      if (cmd.trim() === 'clear') {
        setLines([]);
        setRunning(false);
        return;
      }

      if (cmd.trim() === 'help') {
        addLine('output', [
          'Available commands:',
          '  scan("table")                    - Scan entire table',
          '  query("table", {pk: "val"})      - Query by partition key',
          '  put("table", {pk: {S: "v"}, ..}) - PutItem',
          '  del("table", {pk: {S: "v"}})     - DeleteItem',
          '  tables()                         - List all tables',
          '  clear                            - Clear terminal',
          '',
          'Results can be chained with .filter(), .map(), etc.',
        ].join('\n'));
        setRunning(false);
        return;
      }

      // Build a scope with the helper functions
      const scan = execScan;
      const query = execQuery;
      const put = execPut;
      const del = execDel;
      const tables = execTables;

      // Use AsyncFunction to allow await in expressions
      const AsyncFunction = Object.getPrototypeOf(async function(){}).constructor;
      const fn = new AsyncFunction('scan', 'query', 'put', 'del', 'tables', `return (${cmd})`);
      let result = await fn(scan, query, put, del, tables);

      // If the result has a .filter or .map, it's already applied
      if (result === undefined) {
        addLine('output', 'undefined');
      } else {
        const formatted = typeof result === 'string' ? result : JSON.stringify(result, null, 2);
        addLine('output', formatted);
      }
    } catch (e: any) {
      addLine('error', `Error: ${e.message || String(e)}`);
    } finally {
      setRunning(false);
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !running && input.trim()) {
      e.preventDefault();
      const cmd = input.trim();
      setHistory(prev => [cmd, ...prev].slice(0, 50));
      setHistoryIdx(-1);
      setInput('');
      executeCommand(cmd);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (history.length > 0) {
        const newIdx = Math.min(historyIdx + 1, history.length - 1);
        setHistoryIdx(newIdx);
        setInput(history[newIdx]);
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (historyIdx > 0) {
        const newIdx = historyIdx - 1;
        setHistoryIdx(newIdx);
        setInput(history[newIdx]);
      } else {
        setHistoryIdx(-1);
        setInput('');
      }
    }
  }

  return (
    <div class="ddb-terminal" onClick={() => inputRef.current?.focus()}>
      <div class="ddb-terminal-output" ref={termRef}>
        {lines.map((line, i) => (
          <div key={i} class={`ddb-terminal-line ddb-terminal-${line.type}`}>
            {line.type === 'input' && <span class="ddb-terminal-prompt">&gt; </span>}
            <span>{line.text}</span>
          </div>
        ))}
      </div>
      <div class="ddb-terminal-input-row">
        <span class="ddb-terminal-prompt">&gt; </span>
        <input
          ref={inputRef}
          class="ddb-terminal-input"
          value={input}
          onInput={(e) => setInput((e.target as HTMLInputElement).value)}
          onKeyDown={handleKeyDown}
          placeholder={running ? 'Running...' : 'Enter command...'}
          disabled={running}
          spellcheck={false}
          autocomplete="off"
        />
      </div>
    </div>
  );
}
