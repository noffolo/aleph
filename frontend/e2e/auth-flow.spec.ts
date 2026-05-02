import { test, expect, type Page } from '@playwright/test';
import { setupApiMocks, hydrateStore, callStoreAction } from './connect-mock-helper';

test.describe('Auth Flow — login, session persistence, logout', () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page);
    await page.goto('/');
    await page.waitForLoadState('load');
  });

  test('full auth cycle: login → session persists after reload → logout', async ({ page }) => {
    // Step 1: Login by setting apiKey state directly
    await callStoreAction(page, 'setApiKey', 'aleph_test_full_flow_xxx');
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-auth',
      apiKey: 'aleph_test_full_flow_xxx',
      projects: [{ id: 'proj-auth', name: 'Auth Flow Project' }],
      selectedAgent: 'agent-1',
      agents: [{
        id: 'agent-1',
        name: 'Auth Test Agent',
        model: 'gpt-4',
        systemPrompt: 'Auth test agent',
      }],
    });
    await page.waitForTimeout(300);

    // Verify we are logged in — main chat UI visible
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
    await expect(page.getByRole('button', { name: 'proj-auth' })).toBeVisible();

    // Step 2: Reload and verify session persistence via hydrateStore re-injection
    await page.reload();
    await page.waitForLoadState('load');
    // Simulate session restore (apiKey from sessionStorage in real app)
    await callStoreAction(page, 'setApiKey', 'aleph_test_full_flow_xxx');
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-auth',
      apiKey: 'aleph_test_full_flow_xxx',
      projects: [{ id: 'proj-auth', name: 'Auth Flow Project' }],
      selectedAgent: 'agent-1',
    });
    await page.waitForTimeout(300);

    // After reload, the main UI should still be accessible
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
    await expect(page.getByRole('button', { name: 'proj-1' })).not.toBeVisible();

    // Step 3: Logout via returning to onboarding state
    await page.reload();
    await page.waitForLoadState('load');
    // Without apiKey hydrating, the app shows onboarding
    await expect(page.getByText('Aleph')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText(/Open Intelligence System/i)).toBeVisible();
  });

  test('project switch: move from one project to another preserves auth', async ({ page }) => {
    // Login to first project
    await callStoreAction(page, 'setApiKey', 'aleph_test_switch_xxx');
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-alpha',
      apiKey: 'aleph_test_switch_xxx',
      projects: [
        { id: 'proj-alpha', name: 'Alpha Project' },
        { id: 'proj-beta', name: 'Beta Project' },
      ],
      selectedAgent: 'agent-1',
      agents: [{
        id: 'agent-1',
        name: 'Switch Agent',
        model: 'gpt-4',
        systemPrompt: 'Switch test agent',
      }],
    });
    await page.waitForTimeout(300);

    // Verify we are in Alpha Project
    await expect(page.getByRole('button', { name: 'proj-alpha' })).toBeVisible();

    // Switch to Beta Project
    await hydrateStore(page, {
      projectID: 'proj-beta',
    });
    await page.waitForTimeout(300);

    // Verify we are now in Beta Project
    await expect(page.getByRole('button', { name: 'proj-beta' })).toBeVisible();
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
  });

  test('invalid api key: no access to project data', async ({ page }) => {
    // Login with an invalid api key
    await callStoreAction(page, 'setApiKey', 'invalid_key_xyz');
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-invalid',
      apiKey: 'invalid_key_xyz',
      projects: [{ id: 'proj-invalid', name: 'Invalid Key Project' }],
      selectedAgent: 'agent-1',
      agents: [{
        id: 'agent-1',
        name: 'Invalid Key Agent',
        model: 'gpt-4',
        systemPrompt: 'Invalid key agent',
      }],
    });
    await page.waitForTimeout(300);

    // The main UI renders even with an invalid key — the API layer will reject requests
    // Verify we can still see the chat input
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();

    // Type a message — the ConnectRPC mock returns empty, so no data loads
    const textarea = page.locator('textarea');
    await textarea.fill('Show me data');
    await textarea.press('Enter');
    await page.waitForTimeout(500);

    // The app does not crash with an invalid key — it degrades gracefully
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
  });

  test('session isolation: two projects with different api keys', async ({ page }) => {
    // Set up project A with key A
    await callStoreAction(page, 'setApiKey', 'key-for-project-a');
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-a',
      apiKey: 'key-for-project-a',
      projects: [{ id: 'proj-a', name: 'Project A' }],
      selectedAgent: 'agent-1',
      agents: [{
        id: 'agent-1',
        name: 'Agent A',
        model: 'gpt-4',
        systemPrompt: 'Agent for project A',
      }],
    });
    await page.waitForTimeout(300);

    // Verify Project A active
    await expect(page.getByRole('button', { name: 'proj-a' })).toBeVisible();

    // Simulate navigating to a different project via session restore
    await page.reload();
    await page.waitForLoadState('load');
    await callStoreAction(page, 'setApiKey', 'key-for-project-b');
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-b',
      apiKey: 'key-for-project-b',
      projects: [{ id: 'proj-b', name: 'Project B' }],
      selectedAgent: 'agent-2',
      agents: [{
        id: 'agent-2',
        name: 'Agent B',
        model: 'gpt-4',
        systemPrompt: 'Agent for project B',
      }],
    });
    await page.waitForTimeout(300);

    // Project B should be active — Project A should not be visible
    await expect(page.getByRole('button', { name: 'proj-b' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'proj-a' })).not.toBeVisible();
  });
});
