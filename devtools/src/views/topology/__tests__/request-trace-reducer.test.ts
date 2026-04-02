import { describe, it, expect } from 'vitest';
import {
  panelReducer,
  initialPanelState,
  type PanelState,
  type PanelAction,
} from '../request-trace-reducer';
import type { RequestFlow } from '../request-trace-utils';

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

function makeFlow(overrides: Partial<RequestFlow> = {}): RequestFlow {
  return {
    id: 'flow-1',
    requestId: 'flow-1',
    traceId: 'trace-1',
    method: 'POST',
    path: '/v1/orders',
    statusCode: 201,
    durationMs: 42,
    timestamp: '2025-06-01T12:00:00.000Z',
    inboundSource: 'BFF',
    inboundHeaders: undefined,
    outbound: [],
    responseSummary: '',
    ...overrides,
  };
}

function makeState(overrides: Partial<PanelState> = {}): PanelState {
  return { ...initialPanelState(), ...overrides };
}

/* ================================================================== */
/*  1. FETCH_START                                                     */
/* ================================================================== */

describe('FETCH_START', () => {
  it('sets phase to loading and clears error', () => {
    const state = makeState({ phase: 'error', error: 'old error' });
    const next = panelReducer(state, { type: 'FETCH_START' });
    expect(next.phase).toBe('loading');
    expect(next.error).toBeNull();
  });

  it('preserves existing flows during loading', () => {
    const state = makeState({ flows: [makeFlow({ id: 'f1' })], phase: 'loaded' });
    const next = panelReducer(state, { type: 'FETCH_START' });
    expect(next.flows.length).toBe(1);
  });
});

/* ================================================================== */
/*  2. FETCH_SUCCESS                                                   */
/* ================================================================== */

describe('FETCH_SUCCESS', () => {
  it('sets phase to loaded and stores flows', () => {
    const state = makeState({ phase: 'loading' });
    const flows = [makeFlow({ id: 'f1' }), makeFlow({ id: 'f2' })];
    const next = panelReducer(state, { type: 'FETCH_SUCCESS', flows });
    expect(next.phase).toBe('loaded');
    expect(next.flows.length).toBe(2);
  });

  it('auto-selects the first flow', () => {
    const state = makeState();
    const flows = [makeFlow({ id: 'f1' }), makeFlow({ id: 'f2' })];
    const next = panelReducer(state, { type: 'FETCH_SUCCESS', flows });
    expect(next.selectedFlowId).toBe('f1');
    expect(next.expandedIds.has('f1')).toBe(true);
  });

  it('does not auto-select when flows are empty', () => {
    const state = makeState();
    const next = panelReducer(state, { type: 'FETCH_SUCCESS', flows: [] });
    expect(next.selectedFlowId).toBeNull();
    expect(next.phase).toBe('loaded');
  });
});

/* ================================================================== */
/*  3. FETCH_ERROR                                                     */
/* ================================================================== */

describe('FETCH_ERROR', () => {
  it('sets phase to error and stores error message', () => {
    const state = makeState({ phase: 'loading' });
    const next = panelReducer(state, { type: 'FETCH_ERROR', error: 'Network failed' });
    expect(next.phase).toBe('error');
    expect(next.error).toBe('Network failed');
  });
});

/* ================================================================== */
/*  4. POLL_MERGE — the key fix for disappearing requests             */
/* ================================================================== */

describe('POLL_MERGE', () => {
  it('merges incoming flows without losing existing ones', () => {
    const state = makeState({
      phase: 'loaded',
      flows: [makeFlow({ id: 'f1', timestamp: '2025-06-01T12:00:00.000Z' })],
    });
    const incoming = [makeFlow({ id: 'f2', timestamp: '2025-06-01T12:01:00.000Z' })];
    const next = panelReducer(state, { type: 'POLL_MERGE', flows: incoming });
    expect(next.flows.length).toBe(2);
    expect(next.flows.map((f) => f.id).sort()).toEqual(['f1', 'f2']);
  });

  it('updates existing flows with fresher data', () => {
    const state = makeState({
      phase: 'loaded',
      flows: [makeFlow({ id: 'f1', statusCode: 200 })],
    });
    const incoming = [makeFlow({ id: 'f1', statusCode: 201 })];
    const next = panelReducer(state, { type: 'POLL_MERGE', flows: incoming });
    expect(next.flows.length).toBe(1);
    expect(next.flows[0].statusCode).toBe(201);
  });

  it('is idempotent with same data', () => {
    const flow = makeFlow({ id: 'f1' });
    const state = makeState({ phase: 'loaded', flows: [flow] });
    const next = panelReducer(state, { type: 'POLL_MERGE', flows: [flow] });
    expect(next.flows.length).toBe(1);
  });

  it('preserves selection when merging', () => {
    const state = makeState({
      phase: 'loaded',
      flows: [makeFlow({ id: 'f1' })],
      selectedFlowId: 'f1',
    });
    const incoming = [makeFlow({ id: 'f2' })];
    const next = panelReducer(state, { type: 'POLL_MERGE', flows: incoming });
    expect(next.selectedFlowId).toBe('f1');
  });

  it('does not change phase', () => {
    const state = makeState({ phase: 'loaded' });
    const next = panelReducer(state, { type: 'POLL_MERGE', flows: [] });
    expect(next.phase).toBe('loaded');
  });

  it('does not update timeRange (prevents flicker)', () => {
    const frozenRange = { start: 1000, end: 2000 };
    const state = makeState({ phase: 'loaded', timeRange: frozenRange });
    const incoming = [makeFlow({ id: 'f1' })];
    const next = panelReducer(state, { type: 'POLL_MERGE', flows: incoming });
    expect(next.timeRange).toBe(frozenRange); // same reference
  });
});

/* ================================================================== */
/*  5. SELECT_FLOW                                                     */
/* ================================================================== */

describe('SELECT_FLOW', () => {
  it('sets selectedFlowId and adds to expandedIds', () => {
    const state = makeState({ flows: [makeFlow({ id: 'f1' })] });
    const next = panelReducer(state, { type: 'SELECT_FLOW', id: 'f1' });
    expect(next.selectedFlowId).toBe('f1');
    expect(next.expandedIds.has('f1')).toBe(true);
  });

  it('preserves previously expanded items', () => {
    const state = makeState({ expandedIds: new Set(['f1']) });
    const next = panelReducer(state, { type: 'SELECT_FLOW', id: 'f2' });
    expect(next.expandedIds.has('f1')).toBe(true);
    expect(next.expandedIds.has('f2')).toBe(true);
  });
});

/* ================================================================== */
/*  6. TOGGLE_EXPAND                                                   */
/* ================================================================== */

describe('TOGGLE_EXPAND', () => {
  it('adds id to expandedIds when not present', () => {
    const state = makeState({ expandedIds: new Set() });
    const next = panelReducer(state, { type: 'TOGGLE_EXPAND', id: 'f1' });
    expect(next.expandedIds.has('f1')).toBe(true);
  });

  it('removes id from expandedIds when already present', () => {
    const state = makeState({ expandedIds: new Set(['f1']) });
    const next = panelReducer(state, { type: 'TOGGLE_EXPAND', id: 'f1' });
    expect(next.expandedIds.has('f1')).toBe(false);
  });
});

/* ================================================================== */
/*  7. SET_TIME_RANGE                                                  */
/* ================================================================== */

describe('SET_TIME_RANGE', () => {
  it('updates timeRange', () => {
    const state = makeState();
    const range = { start: 1000, end: 2000 };
    const next = panelReducer(state, { type: 'SET_TIME_RANGE', range });
    expect(next.timeRange).toEqual(range);
  });
});

/* ================================================================== */
/*  8. TOGGLE_OPTIONS_FILTER                                           */
/* ================================================================== */

describe('TOGGLE_OPTIONS_FILTER', () => {
  it('flips hideOptions from true to false', () => {
    const state = makeState({ hideOptions: true });
    const next = panelReducer(state, { type: 'TOGGLE_OPTIONS_FILTER' });
    expect(next.hideOptions).toBe(false);
  });

  it('flips hideOptions from false to true', () => {
    const state = makeState({ hideOptions: false });
    const next = panelReducer(state, { type: 'TOGGLE_OPTIONS_FILTER' });
    expect(next.hideOptions).toBe(true);
  });
});

/* ================================================================== */
/*  9. REPLAY lifecycle                                                */
/* ================================================================== */

describe('REPLAY actions', () => {
  it('REPLAY_START sets status for flow', () => {
    const state = makeState();
    const next = panelReducer(state, { type: 'REPLAY_START', flowId: 'f1' });
    expect(next.replayStatus['f1']).toBe('replaying...');
  });

  it('REPLAY_DONE sets result status', () => {
    const state = makeState({ replayStatus: { f1: 'replaying...' } });
    const next = panelReducer(state, { type: 'REPLAY_DONE', flowId: 'f1', status: '✓ 200 (12ms)' });
    expect(next.replayStatus['f1']).toBe('✓ 200 (12ms)');
  });

  it('REPLAY_CLEAR removes status for flow', () => {
    const state = makeState({ replayStatus: { f1: '✓ 200', f2: 'replaying...' } });
    const next = panelReducer(state, { type: 'REPLAY_CLEAR', flowId: 'f1' });
    expect(next.replayStatus['f1']).toBeUndefined();
    expect(next.replayStatus['f2']).toBe('replaying...');
  });
});

/* ================================================================== */
/*  10. Reducer returns same reference for unknown action              */
/* ================================================================== */

describe('unknown action', () => {
  it('returns state unchanged', () => {
    const state = makeState();
    const next = panelReducer(state, { type: 'UNKNOWN' } as any);
    expect(next).toBe(state);
  });
});
