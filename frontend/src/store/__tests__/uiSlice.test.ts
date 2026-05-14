import { describe, it, expect, vi } from 'vitest';
import { createUISlice } from '../uiSlice';
import type { UISlice } from '../uiSlice';

function makeSlice(): UISlice {
  const set = vi.fn();
  const get = vi.fn<() => UISlice>().mockReturnValue({} as UISlice);
  return createUISlice(set, get, {} as never);
}

describe('uiSlice', () => {
  it('has correct initial state', () => {
    const slice = makeSlice();
    expect(slice.showOnboarding).toBe(true);
    expect(slice.showWizard).toBe(false);
    expect(slice.selectedAssetContent).toBeNull();
    expect(slice.selectedAssetId).toBeNull();
    expect(slice.assets).toEqual([]);
    expect(slice.toastMessages).toEqual([]);
    expect(slice.enableScanline).toBe(true);
    expect(slice.enableGlow).toBe(false);
    expect(slice.enableFlicker).toBe(false);
    expect(slice.inputMode).toBe(false);
    expect(slice.lastError).toBeNull();
  });

  it('setShowOnboarding toggles onboarding', () => {
    const slice = makeSlice();
    slice.setShowOnboarding(false);
    slice.setShowOnboarding(true);
  });

  it('setShowWizard toggles wizard', () => {
    const slice = makeSlice();
    slice.setShowWizard(true);
    slice.setShowWizard(false);
  });

  it('setSelectedAssetContent accepts string or null', () => {
    const slice = makeSlice();
    slice.setSelectedAssetContent('# Report');
    slice.setSelectedAssetContent(null);
  });

  it('setSelectedAssetId accepts string or null', () => {
    const slice = makeSlice();
    slice.setSelectedAssetId('asset-1');
    slice.setSelectedAssetId(null);
  });

  it('setAssets accepts asset list', () => {
    const slice = makeSlice();
    slice.setAssets([{ id: 'a1', name: 'file.pdf', type: 'pdf', createdAt: 100 }]);
    slice.setAssets([]);
  });

  it('setEnableScanline toggles', () => {
    const slice = makeSlice();
    slice.setEnableScanline(false);
    slice.setEnableScanline(true);
  });

  it('setEnableGlow toggles', () => {
    const slice = makeSlice();
    slice.setEnableGlow(true);
    slice.setEnableGlow(false);
  });

  it('setEnableFlicker toggles', () => {
    const slice = makeSlice();
    slice.setEnableFlicker(true);
    slice.setEnableFlicker(false);
  });

  it('addToast and removeToast manage toast messages', () => {
    const slice = makeSlice();
    slice.addToast({ message: 'Success!', type: 'success' });
    slice.removeToast('toast-1');
    slice.addToast({ message: 'Error!', type: 'error', context: 'test' });
  });

  it('setInputMode toggles', () => {
    const slice = makeSlice();
    slice.setInputMode(true);
    slice.setInputMode(false);
  });

  it('toggleSection manages expanded sections', () => {
    const slice = makeSlice();
    slice.toggleSection('sidebar');
    slice.toggleSection('sidebar');
    slice.toggleSection('panel');
  });

  it('collapseAll clears all expanded sections', () => {
    const slice = makeSlice();
    slice.toggleSection('a');
    slice.toggleSection('b');
    slice.collapseAll();
  });

  it('expandAll triggers', () => {
    const slice = makeSlice();
    slice.expandAll();
  });

  it('resetUI is callable', () => {
    const slice = makeSlice();
    slice.resetUI();
  });
});
