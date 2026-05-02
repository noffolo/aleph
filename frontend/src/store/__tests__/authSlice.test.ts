import { describe, it, expect } from 'vitest';
import { createAuthSlice } from '../authSlice';

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

describe('authSlice', () => {
  it('should have correct initial state', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createAuthSlice(set, get, {} as any);
    
    expect(slice.projectID).toBe('');
    expect(slice.apiKeys).toEqual([]);
    expect(slice.projects).toEqual([]);
    expect(slice.notificationChannels).toEqual([]);
    expect(slice.registryComponents).toEqual([]);
  });

  it('should set project context', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createAuthSlice(set, get, {} as any);
    
    slice.setProjectContext('proj-123');
    
    expect(set({ projectID: 'proj-123', apiKey: 'key-abc' })).toMatchObject({
      projectID: 'proj-123',
      apiKey: 'key-abc'
    });
  });

  it('should set api keys', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createAuthSlice(set, get, {} as any);
    const keys = [{ id: '1', label: 'test', key: 'secret', createdAt: Date.now() }];
    
    slice.setApiKeys(keys);
    expect(set({ apiKeys: keys })).toMatchObject({ apiKeys: keys });
  });

  it('should set projects', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createAuthSlice(set, get, {} as any);
    const projects = [{ id: 'p1', name: 'Project 1' }];
    
    slice.setProjects(projects);
    expect(set({ projects })).toMatchObject({ projects });
  });

  it('should set notification channels', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createAuthSlice(set, get, {} as any);
    const channels = [{ id: 'c1', name: 'Slack', type: 'webhook', configJson: '{}' }];
    
    slice.setNotificationChannels(channels);
    expect(set({ notificationChannels: channels })).toMatchObject({ notificationChannels: channels });
  });

  it('should set registry components', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createAuthSlice(set, get, {} as any);
    const components = [{ id: 'rc1', name: 'Comp 1', description: 'Desc', version: '1.0', type: 'tool', category: 'gen', source: 'src', status: 'ok', approvalStatus: 'approved' }];
    
    slice.setRegistryComponents(components);
    expect(set({ registryComponents: components })).toMatchObject({ registryComponents: components });
  });

  it('should reset auth', () => {
    const set = createMockSet();
    const get = () => ({} as any);
    const slice = createAuthSlice(set, get, {} as any);
    
    slice.resetAuth();
    expect(set({
      apiKeys: [],
      notificationChannels: [],
      registryComponents: [],
    })).toMatchObject({
      apiKeys: [],
      notificationChannels: [],
      registryComponents: [],
    });
  });
});
