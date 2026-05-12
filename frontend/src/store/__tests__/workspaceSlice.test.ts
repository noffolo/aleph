import { describe, it, expect } from 'vitest';
import { createWorkspaceSlice } from '../workspaceSlice';

const createMockSet = () => {
  let state: any = {};
  return (update: any) => {
    if (typeof update === 'function') {
      state = update(state);
    } else {
      state = { ...state, ...update };
    }
    return state;
  };
};

describe('workspaceSlice', () => {
  it('should have correct initial state', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createWorkspaceSlice(set, get, {} as any);
    
    expect(slice.sandboxInput).toBe('{}');
    expect(slice.predictions).toEqual([]);
    expect(slice.data).toBeNull();
    expect(slice.selectedRow).toBeNull();
    expect(slice.agents).toEqual([]);
    expect(slice.ingestionTasks).toEqual([]);
    expect(slice.ontologyRaw).toBe('');
    expect(slice.availableObjects).toEqual([]);
    expect(slice.scenarios).toEqual([]);
    expect(slice.selectedScenarioIds).toEqual([]);
    expect(slice.taskLogs).toBe('');
    expect(slice.skills).toEqual([]);
    expect(slice.tools).toEqual([]);
  });

  it('should reset workspace', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createWorkspaceSlice(set, get, {} as any);
    
    slice.resetWorkspace();
    expect(set({
      sandboxInput: '{}',
      predictions: [],
      data: null,
      selectedRow: null,
      agents: [],
      ingestionTasks: [],
      ontologyRaw: '',
      availableObjects: [],
      scenarios: [],
      selectedScenarioIds: [],
      taskLogs: '',
      skills: [],
      tools: [],
    })).toMatchObject({
      sandboxInput: '{}',
      predictions: [],
      data: null,
      selectedRow: null,
      agents: [],
      ingestionTasks: [],
      ontologyRaw: '',
      availableObjects: [],
      scenarios: [],
      selectedScenarioIds: [],
      taskLogs: '',
      skills: [],
      tools: [],
    });
  });
});
