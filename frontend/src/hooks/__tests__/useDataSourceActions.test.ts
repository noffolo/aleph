import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useDataSourceActions } from '../domain/useDataSourceActions';
import { useStore } from '@/store/useStore';
import { ingestionClient } from '@/api/factory';

vi.mock('@/store/useStore', () => ({
  useStore: Object.assign(vi.fn(), { getState: vi.fn() }),
}));

vi.mock('@/api/factory', () => ({
  ingestionClient: {
    createTask: vi.fn(),
    runTask: vi.fn(),
    getProgress: vi.fn(),
    listTasks: vi.fn(),
    getTaskLogs: vi.fn(),
    deleteTask: vi.fn(),
  },
}));

describe('useDataSourceActions', () => {
  const mockStore: Record<string, unknown> = {
    projectID: 'test-project',
    ingestionTasks: [],
    setIngestionTasks: vi.fn(),
    setTaskLogs: vi.fn(),
    setLastError: vi.fn(),
    addToast: vi.fn(),
  };

  const mockLoadProjectData = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    vi.spyOn(console, 'error').mockImplementation(() => {});
    (useStore as any).mockImplementation((selector?: any) => {
      if (typeof selector === 'function') return selector(mockStore);
      return mockStore;
    });
    (useStore.getState as any).mockReturnValue(mockStore);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('onAddSource creates a task and reloads project data', async () => {
    (ingestionClient as any).createTask.mockResolvedValue({});

    const { result } = renderHook(() => useDataSourceActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onAddSource({ name: 'CSV Import', sourceType: 'csv', configJson: '{}' });
    });

    expect(ingestionClient.createTask).toHaveBeenCalledWith({
      projectId: 'test-project',
      task: { name: 'CSV Import', sourceType: 'csv', configJson: '{}' },
    });
    expect(mockLoadProjectData).toHaveBeenCalled();
  });

  it('onAddSource handles createTask error via handleError', async () => {
    (ingestionClient as any).createTask.mockRejectedValue(new Error('Network error'));

    const { result } = renderHook(() => useDataSourceActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onAddSource({ name: 'Bad', sourceType: 'broken', configJson: '{}' });
    });

    expect(mockStore.addToast).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'error', context: 'createTask' }),
    );
  });

  it('onRunTask triggers ingestion, starts polling, and stops on completed status', async () => {
    (ingestionClient as any).runTask.mockResolvedValue({});
    (ingestionClient as any).getProgress.mockResolvedValue({});
    (ingestionClient as any).listTasks.mockResolvedValue({
      tasks: [{ id: 'task-1', name: 'My Task', sourceType: 'csv', status: 'completed', progress: 100 }],
    });

    const { result } = renderHook(() => useDataSourceActions(mockLoadProjectData));

    await act(async () => {
      result.current.onRunTask('task-1');
    });

    expect(ingestionClient.runTask).toHaveBeenCalledWith({
      projectId: 'test-project',
      taskId: 'task-1',
    });

    await act(async () => {
      vi.advanceTimersByTime(1100);
    });

    expect(ingestionClient.getProgress).toHaveBeenCalled();
    expect(ingestionClient.listTasks).toHaveBeenCalled();
    expect(mockStore.setIngestionTasks).toHaveBeenCalled();
  });

  it('onRunTask handles error gracefully', async () => {
    (ingestionClient as any).runTask.mockRejectedValue(new Error('Run failed'));

    const { result } = renderHook(() => useDataSourceActions(mockLoadProjectData));

    await act(async () => {
      result.current.onRunTask('task-1');
    });

    expect(mockStore.addToast).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'error', context: 'runTask' }),
    );
  });

  it('onViewLogs fetches and stores task logs', async () => {
    (ingestionClient as any).getTaskLogs.mockResolvedValue({ logs: 'INFO: Starting...' });

    const { result } = renderHook(() => useDataSourceActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onViewLogs('task-1');
    });

    expect(ingestionClient.getTaskLogs).toHaveBeenCalledWith({
      projectId: 'test-project',
      taskId: 'task-1',
    });
    expect(mockStore.setTaskLogs).toHaveBeenCalledWith('INFO: Starting...');
  });

  it('onViewLogs defaults to "Nessun log" when logs field is missing', async () => {
    (ingestionClient as any).getTaskLogs.mockResolvedValue({});

    const { result } = renderHook(() => useDataSourceActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onViewLogs('task-1');
    });

    expect(mockStore.setTaskLogs).toHaveBeenCalledWith('Nessun log');
  });

  it('onDeleteTask removes task and reloads', async () => {
    (ingestionClient as any).deleteTask.mockResolvedValue({});

    const { result } = renderHook(() => useDataSourceActions(mockLoadProjectData));

    await act(async () => {
      await result.current.onDeleteTask('task-1');
    });

    expect(ingestionClient.deleteTask).toHaveBeenCalledWith({
      projectId: 'test-project',
      id: 'task-1',
    });
    expect(mockLoadProjectData).toHaveBeenCalled();
  });
});
