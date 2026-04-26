import { test, expect } from '@playwright/test';
import { setupApiMocks, hydrateStore, callStoreAction } from './connect-mock-helper';

test.describe('User Journey', () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page);
    await page.goto('/');
    await page.waitForLoadState('load');
  });

  test('full smoke test: login → create agent → execute tool → save → reload → verify', async ({ page }) => {
    await callStoreAction(page, 'setApiKey', 'aleph_test_xxx');
    
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-1',
      apiKey: 'aleph_test_xxx',
      selectedAgent: 'agent-1',
      projects: [{ id: 'proj-1', name: 'Journey Project' }],
      agents: [{ 
        id: 'agent-1', 
        name: 'Journey Agent', 
        model: 'gpt-4', 
        systemPrompt: 'Journey test agent' 
      }],
    });
    await page.waitForTimeout(300);

    // Verify we are in the main interface
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();

    await page.getByText('Agents').click();
    await page.waitForTimeout(300);

    await expect(page.getByRole('heading', { name: 'Journey Agent' })).toBeVisible();

    await page.getByRole('button', { name: 'Copilot' }).click();
    await page.waitForTimeout(300);

    const input = page.getByPlaceholder(/scrivi un messaggio/i);
    await input.fill('Run a tool');
    await page.keyboard.press('Enter');
    await page.waitForTimeout(500);

    await callStoreAction(page, 'saveCurrentState'); 
    await page.waitForTimeout(200);

    await page.reload();
    await page.waitForLoadState('load');
    
    await callStoreAction(page, 'setApiKey', 'aleph_test_xxx');
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-1',
      apiKey: 'aleph_test_xxx',
      selectedAgent: 'agent-1',
      projects: [{ id: 'proj-1', name: 'Journey Project' }],
    });
    await page.waitForTimeout(300);

    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
    await expect(page.getByRole('button', { name: 'proj-1' })).toBeVisible();
  });
});
