import { createContext } from 'preact';
import { useContext, useEffect, useRef } from 'preact/hooks';
import { DDBItem, TableDescription, FilterCondition } from './types';

// State for a single open tab
export interface TabState {
  tableName: string;
  items: DDBItem[];
  activeSubTab: 'items' | 'query' | 'scan' | 'patterns' | 'sql' | 'terminal' | 'info';
  page: number;
  pageSize: number;
  lastEvaluatedKeys: (Record<string, any> | null)[];
  tableDesc: TableDescription | null;
  loading: boolean;
  // Query state
  queryMode: 'query' | 'scan';
  queryPK: string;
  querySK: string;
  querySKOp: string;
  querySKValue2: string;
  queryIndex: string;
  queryScanForward: boolean;
  queryLimit: string;
  queryResults: DDBItem[] | null;
  queryFilters: FilterCondition[];
  queryScannedCount: number;
  // Scan state
  scanFilters: FilterCondition[];
  scanResults: DDBItem[] | null;
  // SQL state
  sqlQuery: string;
  sqlResults: DDBItem[] | null;
  // Terminal history
  terminalLines: Array<{ type: 'input' | 'output' | 'error'; text: string }>;
  terminalHistory: string[];
  // Scroll position
  scrollTop: number;
  // Initial hints for query tab (from AccessPatterns)
  queryInitialIndex?: string;
  queryInitialPk?: string;
}

// Global DynamoDB state
export interface DDBState {
  tabs: TabState[];
  activeTabIndex: number;
  tableSearch: string;
}

export type DDBAction =
  | { type: 'OPEN_TAB'; tableName: string }
  | { type: 'CLOSE_TAB'; index: number }
  | { type: 'SET_ACTIVE_TAB'; index: number }
  | { type: 'UPDATE_TAB'; index: number; patch: Partial<TabState> }
  | { type: 'SET_TABLE_SEARCH'; search: string }
  | { type: 'SET_ITEMS'; index: number; items: DDBItem[]; lastKey: Record<string, any> | null; page: number }
  | { type: 'SET_TABLE_DESC'; index: number; desc: TableDescription }
  | { type: 'SET_LOADING'; index: number; loading: boolean }
  | { type: 'SET_QUERY_RESULTS'; index: number; results: DDBItem[]; scannedCount: number }
  | { type: 'SET_SQL_RESULTS'; index: number; results: DDBItem[] }
  | { type: 'APPEND_TERMINAL_LINE'; index: number; line: { type: 'input' | 'output' | 'error'; text: string } }
  | { type: 'CLEAR_TERMINAL'; index: number }
  | { type: 'RESTORE'; state: DDBState };

function makeDefaultTerminalLines(tableName: string): Array<{ type: 'input' | 'output' | 'error'; text: string }> {
  return [
    { type: 'output', text: `DynamoDB REPL - Table: ${tableName}` },
    { type: 'output', text: `Commands: scan(table), query(table, {pk: val}), put(table, item), del(table, key), tables(), clear` },
    { type: 'output', text: `Type JavaScript expressions. Results are pretty-printed.\n` },
  ];
}

function makeTab(tableName: string): TabState {
  return {
    tableName,
    items: [],
    activeSubTab: 'items',
    page: 0,
    pageSize: 25,
    lastEvaluatedKeys: [null],
    tableDesc: null,
    loading: true,
    queryMode: 'query',
    queryPK: '',
    querySK: '',
    querySKOp: '=',
    querySKValue2: '',
    queryIndex: '',
    queryScanForward: true,
    queryLimit: '',
    queryResults: null,
    queryFilters: [],
    queryScannedCount: 0,
    scanFilters: [],
    scanResults: null,
    sqlQuery: `SELECT * FROM "${tableName}"`,
    sqlResults: null,
    terminalLines: makeDefaultTerminalLines(tableName),
    terminalHistory: [],
    scrollTop: 0,
  };
}

export function ddbReducer(state: DDBState, action: DDBAction): DDBState {
  switch (action.type) {
    case 'OPEN_TAB': {
      const existing = state.tabs.findIndex(t => t.tableName === action.tableName);
      if (existing >= 0) return { ...state, activeTabIndex: existing };
      const newTab = makeTab(action.tableName);
      return { ...state, tabs: [...state.tabs, newTab], activeTabIndex: state.tabs.length };
    }
    case 'CLOSE_TAB': {
      const tabs = state.tabs.filter((_, i) => i !== action.index);
      let activeTabIndex = state.activeTabIndex;
      if (tabs.length === 0) {
        activeTabIndex = -1;
      } else if (action.index < state.activeTabIndex) {
        activeTabIndex = state.activeTabIndex - 1;
      } else if (activeTabIndex >= tabs.length) {
        activeTabIndex = tabs.length - 1;
      }
      return { ...state, tabs, activeTabIndex };
    }
    case 'SET_ACTIVE_TAB':
      return { ...state, activeTabIndex: action.index };
    case 'UPDATE_TAB': {
      const tabs = state.tabs.map((t, i) => i === action.index ? { ...t, ...action.patch } : t);
      return { ...state, tabs };
    }
    case 'SET_TABLE_SEARCH':
      return { ...state, tableSearch: action.search };
    case 'SET_ITEMS': {
      const tabs = state.tabs.map((t, i) => {
        if (i !== action.index) return t;
        const lastKeys = [...t.lastEvaluatedKeys];
        lastKeys[action.page + 1] = action.lastKey;
        return { ...t, items: action.items, lastEvaluatedKeys: lastKeys, loading: false };
      });
      return { ...state, tabs };
    }
    case 'SET_TABLE_DESC': {
      const tabs = state.tabs.map((t, i) => i === action.index ? { ...t, tableDesc: action.desc } : t);
      return { ...state, tabs };
    }
    case 'SET_LOADING': {
      const tabs = state.tabs.map((t, i) => i === action.index ? { ...t, loading: action.loading } : t);
      return { ...state, tabs };
    }
    case 'SET_QUERY_RESULTS': {
      const tabs = state.tabs.map((t, i) => i === action.index ? { ...t, queryResults: action.results, queryScannedCount: action.scannedCount } : t);
      return { ...state, tabs };
    }
    case 'SET_SQL_RESULTS': {
      const tabs = state.tabs.map((t, i) => i === action.index ? { ...t, sqlResults: action.results } : t);
      return { ...state, tabs };
    }
    case 'APPEND_TERMINAL_LINE': {
      const tabs = state.tabs.map((t, i) => {
        if (i !== action.index) return t;
        return { ...t, terminalLines: [...t.terminalLines, action.line] };
      });
      return { ...state, tabs };
    }
    case 'CLEAR_TERMINAL': {
      const tabs = state.tabs.map((t, i) => i === action.index ? { ...t, terminalLines: [] } : t);
      return { ...state, tabs };
    }
    case 'RESTORE':
      return action.state;
    default:
      return state;
  }
}

export const initialState: DDBState = { tabs: [], activeTabIndex: -1, tableSearch: '' };

export const DDBContext = createContext<{ state: DDBState; dispatch: (a: DDBAction) => void }>({
  state: initialState,
  dispatch: () => {},
});

export function useDDB() {
  return useContext(DDBContext);
}

// ── localStorage persistence ──

const STORAGE_KEY = 'neureaux-ddb-state';

function loadPersistedState(): DDBState | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as DDBState;
  } catch {
    return null;
  }
}

export function useHydrateState(dispatch: (a: DDBAction) => void) {
  const hydrated = useRef(false);
  useEffect(() => {
    if (hydrated.current) return;
    hydrated.current = true;
    const saved = loadPersistedState();
    if (saved) dispatch({ type: 'RESTORE', state: saved });
  }, [dispatch]);
}

export function usePersistState(state: DDBState) {
  useEffect(() => {
    const timer = setTimeout(() => {
      try {
        const stripped = {
          ...state,
          tabs: state.tabs.map(tab => ({
            ...tab,
            items: undefined,
            queryResults: undefined,
            scanResults: undefined,
            sqlResults: undefined,
          })),
        };
        localStorage.setItem(STORAGE_KEY, JSON.stringify(stripped));
      } catch { /* ignore */ }
    }, 500);
    return () => clearTimeout(timer);
  }, [state]);
}
