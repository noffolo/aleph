import { describe, it, expect } from 'vitest';
import { createHealthSlice } from '../healthSlice';

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

describe('healthSlice', () => {
  it('should have correct initial state', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createHealthSlice(set, get, {} as any);
    
    expect(slice.ollamaHealthy).toBe(false);
    expect(slice.nlpHealthy).toBe(false);
    expect(slice.dataHealthStats).toEqual([]);
    expect(slice.lastError).toBeNull();
    expect(slice.ollamaModels).toEqual([]);
  });

  it('should update health statuses', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createHealthSlice(set, get, {} as any);
    
    slice.setOllamaHealthy(true);
    expect(set({ ollamaHealthy: true })).toMatchObject({ ollamaHealthy: true });
    
    slice.setNlpHealthy(true);
    expect(set({ nlpHealthy: true })).toMatchObject({ nlpHealthy: true });
  });

  it('should set data health stats', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createHealthSlice(set, get, {} as any);
    const stats = [{ columnName: 'test', min: '1', max: '10', count: 100, uniqueCount: 10, topValues: {} }];
    
    slice.setDataHealthStats(stats);
    expect(set({ dataHealthStats: stats })).toMatchObject({ dataHealthStats: stats });
  });

  it('should reset health', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createHealthSlice(set, get, {} as any);
    
    slice.resetHealth();
    expect(set({
      ollamaHealthy: false,
      nlpHealthy: false,
      dataHealthStats: [],
      lastError: null,
      ollamaModels: [],
    })).toMatchObject({
      ollamaHealthy: false,
      nlpHealthy: false,
      dataHealthStats: [],
      lastError: null,
      ollamaModels: [],
    });
  });
});
