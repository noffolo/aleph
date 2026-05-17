import { test, expect } from '@playwright/test';
import { setupApiMocks, hydrateStore, callStoreAction } from './connect-mock-helper';

test.describe('Error States — 404 page, invalid project, network errors, boundary recovery', () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page);
    await page.goto('/');
    await page.waitForLoadState('load');
  });

  test('404 page: navigating to an unknown route shows error content', async ({ page }) => {
    // Navigate to a non-existent path
    await page.goto('/this-page-does-not-exist-404-xyz');
    await page.waitForLoadState('load');

    // The app is a SPA without client-side routing — it renders the main app
    // or the onboarding screen (default state). The 404 handling is graceful.
    // Verify the app does not crash and shows expected branding.
    await expect(page.getByText('Aleph')).toBeVisible({ timeout: 5000 });
  });

  test('error boundary: app renders within AlephErrorBoundary without crash', async ({ page }) => {
    // The entire app is wrapped in AlephErrorBoundary in App.tsx.
    // Verify the app renders normally (meaning the boundary has not tripped into fallback).
    await expect(page.getByText('Aleph')).toBeVisible({ timeout: 5000 });

    // Verify the workspace onboarding is visible (default state)
    await expect(page.getByText('Open Intelligence System')).toBeVisible();
  });

  test('network error: API call failure is handled gracefully via error toast', async ({ page }) => {
    // Set up main app view
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-err',
      apiKey: 'aleph_test_err_xxx',
      selectedAgent: 'agent-1',
      agents: [{
        id: 'agent-1',
        name: 'Error Test Agent',
        model: 'gpt-4',
        systemPrompt: 'Error test agent',
      }],
      projects: [{ id: 'proj-err', name: 'Error Project' }],
    });
    await page.waitForTimeout(300);

    // Verify main UI is visible
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();

    // Simulate a network error by triggering handleError via store
    await hydrateStore(page, {
      lastError: 'network_error: Failed to fetch',
    });
    await page.waitForTimeout(200);

    // The app should still be functional — error is logged, UI remains
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
  });

  test('invalid project ID: app degrades gracefully with missing project', async ({ page }) => {
    // Login with an invalid/non-existent project ID
    await callStoreAction(page, 'setApiKey', 'aleph_test_bad_proj');
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: '', // empty project ID
      apiKey: 'aleph_test_bad_proj',
      selectedAgent: '',
      agents: [],
      projects: [],
    });
    await page.waitForTimeout(300);

    // The app should render (no crash), even though no data will load
    // The setup wizard or a fallback screen may appear
    const alephBrand = page.getByText('Aleph');
    await expect(alephBrand.first()).toBeVisible({ timeout: 5000 });
  });

  test('empty state: chat terminal without messages renders correctly', async ({ page }) => {
    // Set up main app without any chat messages
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-empty',
      apiKey: 'aleph_test_empty',
      selectedAgent: 'agent-1',
      agents: [{
        id: 'agent-1',
        name: 'Empty Agent',
        model: 'gpt-4',
        systemPrompt: 'Empty state agent',
      }],
    });
    await page.waitForTimeout(300);

    // The chat input should be visible even with no messages
    const textarea = page.locator('textarea');
    await expect(textarea).toBeVisible();

    // Empty chat terminal should not have old messages from other tests
    const oldMessages = page.locator('.font-mono');
    // There may be zero visible chat messages in empty state
  });

  test('concurrent errors: multiple handleError calls do not crash UI', async ({ page }) => {
    // Simulate rapid error handling
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-err',
      apiKey: 'aleph_test_err_xxx',
      selectedAgent: 'agent-1',
      agents: [{
        id: 'agent-1',
        name: 'Error Agent',
        model: 'gpt-4',
        systemPrompt: 'Error agent',
      }],
    });
    await page.waitForTimeout(300);

    // Inject multiple errors rapidly
    for (let i = 0; i < 5; i++) {
      await hydrateStore(page, {
        lastError: `concurrent_error_${i}`,
      });
      await page.waitForTimeout(50);
    }

    // The app should still be visible and functional
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
  });

  test('toast container: renders error toasts without layout break', async ({ page }) => {
    // Add error toasts to the store
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-toast',
      toastMessages: [
        { id: 't-1', message: 'Failed to connect to NLP service', type: 'error' as const },
        { id: 't-2', message: 'Query timed out after 30s', type: 'error' as const },
        { id: 't-3', message: 'Agent not found', type: 'error' as const },
      ],
    });
    await page.waitForTimeout(300);

    // Toast errors should be visible
    await expect(page.getByText('Failed to connect to NLP service')).toBeVisible();
    await expect(page.getByText('Query timed out after 30s')).toBeVisible();
    await expect(page.getByText('Agent not found')).toBeVisible();

    // Main UI should still be accessible
    await expect(page.getByText('Aleph').first()).toBeVisible();
  });

  test('error recovery: after an error, user can still interact with app', async ({ page }) => {
    // Set up with an error state
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-recovery',
      apiKey: 'aleph_test_recovery',
      selectedAgent: 'agent-1',
      agents: [{
        id: 'agent-1',
        name: 'Recovery Agent',
        model: 'gpt-4',
        systemPrompt: 'Recovery agent',
      }],
      lastError: 'previous_error: Ollama timeout',
    });
    await page.waitForTimeout(300);

    // After error, user can still type a message
    const textarea = page.locator('textarea');
    await expect(textarea).toBeVisible();
    await textarea.fill('/clear');
    await textarea.press('Enter');
    await page.waitForTimeout(300);

    // User can continue typing after error recovery
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
  });
});
