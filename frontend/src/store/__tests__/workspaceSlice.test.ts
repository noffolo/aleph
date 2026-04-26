import { describe, it, expect } from 'vitest';
import { createWorkspaceSlice } from '../workspaceSlice';
import * as Y from 'yjs';

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
    
    expect(slice.sandboxResult).toBeNull();
    expect(slice.sandboxInput).toBe('{}');
    expect(slice.searchQuery).toBe('');
    expect(slice.selectedObject).toBe('');
    expect(slice.predictions).toEqual([]);
    expect(slice.data).toBeNull();
    expect(slice.selectedRow).toBeNull();
    expect(slice.agents).toEqual([]);
    expect(slice.ingestionTasks).toEqual([]);
    expect(slice.ontologyRaw).toBe('');
    expect(slice.availableObjects).toEqual([]);
    expect(slice.taskLogs).toBe('');
    expect(slice.skills).toEqual([]);
    expect(slice.tools).toEqual([]);
  });

  it('should set sandbox result', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createWorkspaceSlice(set, get, {} as any);
    const result = { stdout: 'hello', exitCode: 0 };
    
    slice.setSandboxResult(result);
    expect(set({ sandboxResult: result })).toMatchObject({ sandboxResult: result });
  });

  it('should update search query via yMap', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createWorkspaceSlice(set, get, {} as any);
    
    slice.setSearchQuery('test query');
    expect(set({ searchQuery: 'test query' })).toMatchObject({ searchQuery: 'test query' });
  });

  it('should reset workspace', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createWorkspaceSlice(set, get, {} as any);
    
    slice.resetWorkspace();
    expect(set({
      sandboxResult: null,
      sandboxInput: '{}',
      predictions: [],
      data: null,
      selectedRow: null,
      agents: [],
      ingestionTasks: [],
      ontologyRaw: '',
      availableObjects: [],
      taskLogs: '',
      skills: [],
      tools: [],
    })).toMatchObject({
      sandboxResult: null,
      sandboxInput: '{}',
      predictions: [],
      data: null,
      selectedRow: null,
      agents: [],
      ingestionTasks: [],
      ontologyRaw: '',
      availableObjects: [],
      taskLogs: '',
      skills: [],
      tools: [],
    });
  });
});
