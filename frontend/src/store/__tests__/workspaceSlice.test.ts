import { describe, it, expect, vi } from 'vitest';
import { createWorkspaceSlice } from '../workspaceSlice';
import type { WorkspaceSlice } from '../workspaceSlice';

function makeSlice(): WorkspaceSlice {
  const set = vi.fn();
  const get = vi.fn<() => WorkspaceSlice>().mockReturnValue({} as WorkspaceSlice);
  return createWorkspaceSlice(set, get, {} as never);
}

describe('workspaceSlice', () => {
  it('has correct initial state', () => {
    const slice = makeSlice();
    expect(slice.sandboxInput).toBe('{}');
    expect(slice.predictions).toEqual([]);
    expect(slice.data).toBeNull();
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

  it('setSandboxInput is callable', () => {
    const slice = makeSlice();
    slice.setSandboxInput('{"key": "val"}');
  });

  it('setPredictions is callable', () => {
    const slice = makeSlice();
    slice.setPredictions([{ entityId: 'e1', probability: 0.8, predictedState: 'up', explanation: 'test' }]);
  });

  it('setData accepts and clears data', () => {
    const slice = makeSlice();
    slice.setData({ columns: ['a'], rows: [{ values: { a: 1 } }] });
    slice.setData(null);
  });

  it('setSelectedRow accepts a row', () => {
    const slice = makeSlice();
    slice.setSelectedRow({ values: { id: 'x' } });
    slice.setSelectedRow(null);
  });

  it('setAgents accepts agent list', () => {
    const slice = makeSlice();
    slice.setAgents([{ id: 'a1', name: 'Test', model: 'llama3', systemPrompt: 'You are helpful' }]);
  });

  it('setIngestionTasks accepts task list', () => {
    const slice = makeSlice();
    slice.setIngestionTasks([{ id: 't1', name: 'Import', sourceType: 'csv', status: 'running', progress: 50 }]);
  });

  it('setOntologyRaw accepts string', () => {
    const slice = makeSlice();
    slice.setOntologyRaw('entities:\n  - Person');
    slice.setOntologyRaw('');
  });

  it('setAvailableObjects accepts string list', () => {
    const slice = makeSlice();
    slice.setAvailableObjects(['users', 'orders']);
  });

  it('setScenarios accepts scenario list', () => {
    const slice = makeSlice();
    slice.setScenarios([{ id: 's1', name: 'Best', confidence: 0.9, signals: [], assumptions: [], trend: 'up', probability: 0.7 }]);
  });

  it('setSelectedScenarioIds accepts id list', () => {
    const slice = makeSlice();
    slice.setSelectedScenarioIds(['s1', 's2']);
  });

  it('setTaskLogs accepts string', () => {
    const slice = makeSlice();
    slice.setTaskLogs('Log output...');
  });

  it('setSkills accepts skill list', () => {
    const slice = makeSlice();
    slice.setSkills([{ id: 'sk1', name: 'Analyze', description: 'Analyzes data' }]);
  });

  it('setTools accepts tool list', () => {
    const slice = makeSlice();
    slice.setTools([{ id: 't1', name: 'Fetch', description: 'Fetches', code: 'print(1)' }]);
  });

  it('resetWorkspace is callable', () => {
    const slice = makeSlice();
    slice.resetWorkspace();
  });
});
