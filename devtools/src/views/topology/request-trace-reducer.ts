import { mergeFlows, computeTimeRange, type RequestFlow } from './request-trace-utils';

/* ------------------------------------------------------------------ */
/*  State                                                              */
/* ------------------------------------------------------------------ */

export interface PanelState {
  phase: 'loading' | 'loaded' | 'error';
  flows: RequestFlow[];
  selectedFlowId: string | null;
  expandedIds: Set<string>;
  timeRange: { start: number; end: number };
  hideOptions: boolean;
  error: string | null;
  replayStatus: Record<string, string>;
  /** Timestamp (epoch ms) of the last successful data fetch/poll */
  lastFetchedAt: number | null;
}

const TIME_WINDOW_MS = 5 * 60 * 1000;

export function initialPanelState(): PanelState {
  return {
    phase: 'loading',
    flows: [],
    selectedFlowId: null,
    expandedIds: new Set(),
    timeRange: computeTimeRange([], TIME_WINDOW_MS),
    hideOptions: true,
    error: null,
    replayStatus: {},
    lastFetchedAt: null,
  };
}

/* ------------------------------------------------------------------ */
/*  Actions                                                            */
/* ------------------------------------------------------------------ */

export type PanelAction =
  | { type: 'FETCH_START' }
  | { type: 'FETCH_SUCCESS'; flows: RequestFlow[] }
  | { type: 'FETCH_ERROR'; error: string }
  | { type: 'POLL_MERGE'; flows: RequestFlow[] }
  | { type: 'SELECT_FLOW'; id: string }
  | { type: 'TOGGLE_EXPAND'; id: string }
  | { type: 'SET_TIME_RANGE'; range: { start: number; end: number } }
  | { type: 'TOGGLE_OPTIONS_FILTER' }
  | { type: 'REPLAY_START'; flowId: string }
  | { type: 'REPLAY_DONE'; flowId: string; status: string }
  | { type: 'REPLAY_CLEAR'; flowId: string };

/* ------------------------------------------------------------------ */
/*  Reducer                                                            */
/* ------------------------------------------------------------------ */

export function panelReducer(state: PanelState, action: PanelAction): PanelState {
  switch (action.type) {
    case 'FETCH_START':
      return { ...state, phase: 'loading', error: null };

    case 'FETCH_SUCCESS': {
      const selectedFlowId = action.flows.length > 0 ? action.flows[0].id : null;
      const expandedIds = selectedFlowId
        ? new Set([...state.expandedIds, selectedFlowId])
        : state.expandedIds;
      return {
        ...state,
        phase: 'loaded',
        flows: action.flows,
        selectedFlowId,
        expandedIds,
        timeRange: computeTimeRange(action.flows, TIME_WINDOW_MS),
        error: null,
        lastFetchedAt: Date.now(),
      };
    }

    case 'FETCH_ERROR':
      return { ...state, phase: 'error', error: action.error };

    case 'POLL_MERGE': {
      const merged = mergeFlows(state.flows, action.flows);
      // Do NOT update timeRange on polls — shifting the window every 5s
      // causes flows at the boundary to flicker in/out.
      // Time range is only set on initial load or user scrub.
      return { ...state, flows: merged, lastFetchedAt: Date.now() };
    }

    case 'SELECT_FLOW':
      return {
        ...state,
        selectedFlowId: action.id,
        expandedIds: new Set([...state.expandedIds, action.id]),
      };

    case 'TOGGLE_EXPAND': {
      const next = new Set(state.expandedIds);
      if (next.has(action.id)) next.delete(action.id);
      else next.add(action.id);
      return { ...state, expandedIds: next };
    }

    case 'SET_TIME_RANGE':
      return { ...state, timeRange: action.range };

    case 'TOGGLE_OPTIONS_FILTER':
      return { ...state, hideOptions: !state.hideOptions };

    case 'REPLAY_START':
      return {
        ...state,
        replayStatus: { ...state.replayStatus, [action.flowId]: 'replaying...' },
      };

    case 'REPLAY_DONE':
      return {
        ...state,
        replayStatus: { ...state.replayStatus, [action.flowId]: action.status },
      };

    case 'REPLAY_CLEAR': {
      const { [action.flowId]: _, ...rest } = state.replayStatus;
      return { ...state, replayStatus: rest };
    }

    default:
      return state;
  }
}
