import { useState, useRef, useEffect, useContext } from 'preact/hooks';
import { ddbRequest } from './utils';
import { DDBItem } from './types';
import { DDBContext } from './store';

interface TerminalProps {
  tableName: string;
  showToast: (msg: string) => void;
  tabIndex: number;
}

interface TerminalLine {
  type: 'input' | 'output' | 'error';
  text: string;
}

export function Terminal({ tableName, showToast, tabIndex }: TerminalProps) {
  const { state, dispatch } = useContext(DDBContext);
  const tab = state.tabs[tabIndex];

  const lines: TerminalLine[] = tab?.terminalLines ?? [
    { type: 'output', text: `DynamoDB REPL - Table: ${tableName}` },
    { type: 'output', text: `Commands: scan(table), query(table, {pk: val}), put(table, item), del(table, key), tables(), clear` },
    { type: 'output', text: `Type JavaScript expressions. Results are pretty-printed.\n` },
  ];
  const cmdHistory: string[] = tab?.terminalHistory ?? [];

  const [input, setInput] = useState('');
  const [historyIdx, setHistoryIdx] = useState(-1);
  const [running, setRunning] = useState(false);
  const termRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (termRef.current) termRef.current.scrollTop = termRef.current.scrollHeight;
  }, [lines]);

  useEffect(() => { inputRef.current?.focus(); }, []);

  function addLine(type: TerminalLine['type'], text: string) {
    dispatch({ type: 'APPEND_TERMINAL_LINE', index: tabIndex, line: { type, text } });
  }

  async function execScan(table: string, opts?: any): Promise<DDBItem[]> {
    const r = await ddbRequest('Scan', { TableName: table, ...(opts || {}) });
    return r.Items || [];
  }

  async function execQuery(table: string, keyConditions: Record<string, any>, opts?: any): Promise<DDBItem[]> {
    const exprNames: Record<string, string> = {};
    const exprValues: Record<string, any> = {};
    const keyParts: string[] = [];
    let i = 0;
    for (const [attr, val] of Object.entries(keyConditions)) {
      const nk = `#k${i}`, vk = `:v${i}`;
      exprNames[nk] = attr;
      exprValues[vk] = typeof val === 'number' ? { N: String(val) } : (typeof val === 'object' && val !== null ? val : { S: String(val) });
      keyParts.push(`${nk} = ${vk}`);
      i++;
    }
    const r = await ddbRequest('Query', {
      TableName: table,
      KeyConditionExpression: keyParts.join(' AND '),
      ExpressionAttributeNames: exprNames,
      ExpressionAttributeValues: exprValues,
      ...(opts || {}),
    });
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
        dispatch({ type: 'CLEAR_TERMINAL', index: tabIndex });
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

      const scan = execScan;
      const query = execQuery;
      const put = execPut;
      const del = execDel;
      const tables = execTables;

      const AsyncFunction = Object.getPrototypeOf(async function(){}).constructor;
      const fn = new AsyncFunction('scan', 'query', 'put', 'del', 'tables', `return (${cmd})`);
      const result = await fn(scan, query, put, del, tables);

      if (result === undefined) {
        addLine('output', 'undefined');
      } else {
        addLine('output', typeof result === 'string' ? result : JSON.stringify(result, null, 2));
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
      dispatch({
        type: 'UPDATE_TAB',
        index: tabIndex,
        patch: { terminalHistory: [cmd, ...cmdHistory].slice(0, 50) },
      });
      setHistoryIdx(-1);
      setInput('');
      executeCommand(cmd);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (cmdHistory.length > 0) {
        const newIdx = Math.min(historyIdx + 1, cmdHistory.length - 1);
        setHistoryIdx(newIdx);
        setInput(cmdHistory[newIdx]);
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (historyIdx > 0) {
        const newIdx = historyIdx - 1;
        setHistoryIdx(newIdx);
        setInput(cmdHistory[newIdx]);
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
