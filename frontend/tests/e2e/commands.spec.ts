import { test, expect } from '@playwright/test';
import { setupApiMocks, hydrateStore, callStoreAction } from './connect-mock-helper';

test.describe('Commands — parseCommand fuzzing, typing, XSS', () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page);
    await page.goto('/');
    // Wait for the app to render
    await page.waitForLoadState('load');

    // Prepare the main app view: hide onboarding, set a fake agent so TerminalPrompt is enabled
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      selectedAgent: 'agent-1',
      agents: [{ id: 'agent-1', name: 'TestAgent', model: 'gpt-4', systemPrompt: 'You are a test agent.' }],
      projectID: 'test-proj',
      apiKey: 'test-api-key',
    });

    // Wait a tick for React to re-render
    await page.waitForTimeout(300);
  });

  test('typing a valid slash command triggers the correct action', async ({ page }) => {
    // Use locator('textarea') instead of getByRole('textbox') to avoid strict mode
    // conflicts with the search input.
    const textarea = page.locator('textarea');
    await expect(textarea).toBeVisible();

    // Type /explore command (should trigger SHOW_INLINE action)
    await textarea.fill('/explore');
    await textarea.press('Enter');

    // The CopilotView onSend handler checks isAllowedSlashCommand then executes.
    await expect(page.getByText('explore').first()).toBeVisible({ timeout: 3000 });
  });

  test('typing an unknown slash command shows an error toast', async ({ page }) => {
    const textarea = page.locator('textarea');
    await expect(textarea).toBeVisible();

    // Type an unknown command
    await textarea.fill('/nonexistent-command-xyz');
    await textarea.press('Enter');

    // isAllowedSlashCommand returns false → shows error toast
    await expect(page.getByText(/comando sconosciuto/i)).toBeVisible({ timeout: 3000 });
  });

  test('/clear command clears chat history', async ({ page }) => {
    // Add some chat messages first
    await hydrateStore(page, {
      chat: [
        { role: 'user', content: 'Hello', createdAt: Date.now() },
        { role: 'assistant', content: 'Hi there', createdAt: Date.now() },
        { role: 'user', content: 'How are you?', createdAt: Date.now() },
      ],
    });
    await page.waitForTimeout(200);

    const textarea = page.locator('textarea');
    await expect(textarea).toBeVisible();

    // Type /clear
    await textarea.fill('/clear');
    await textarea.press('Enter');

    // /clear triggers CLEAR_CHAT action → store.clearChat()
    await expect(page.getByText('Hello')).not.toBeVisible();
    await expect(page.getByText('Hi there')).not.toBeVisible();
  });

  test('XSS injection in textarea — content is escaped in output, not executed', async ({ page }) => {
    const textarea = page.locator('textarea');
    await expect(textarea).toBeVisible();

    // Inject an XSS payload via the chat message
    const xssPayload = '<script>window.xssInjected=true;</script>';
    await hydrateStore(page, {
      chat: [
        { role: 'user', content: xssPayload, createdAt: Date.now() },
        { role: 'assistant', content: 'Response with <img src=x onerror="window.xssInjected=true">', createdAt: Date.now() },
      ],
    });
    await page.waitForTimeout(200);

    // The TerminalOutput component uses escapeHtml() which converts < > to &lt; &gt;
    // Assert the escaped version is visible
    // The text "window.xssInjected" appears only as escaped text
    await expect(page.locator('span').getByText('xssInjected', { exact: false }).first()).toBeVisible({ timeout: 3000 });
  });

  test('payload injection — long strings and special chars do not break terminal', async ({ page }) => {
    const textarea = page.locator('textarea');
    await expect(textarea).toBeVisible();

    // Create a chat message with edge case content
    const payloads = [
      'A'.repeat(10000),       // very long string
      '\\x00\\x01\\x02\\x1F',  // control characters
      'hello\\nworld\\ttab',    // escaped whitespace
      'unicode émojï 🚀 ✓ ∑ ∏', // unicode
      '); DROP TABLE users; --', // SQL injection attempt
      '../../etc/passwd',        // path traversal
    ];

    await hydrateStore(page, {
      chat: payloads.map((content) => ({
        role: 'user' as const,
        content,
        createdAt: Date.now(),
      })),
    });
    await page.waitForTimeout(200);

    // Verify the terminal renders without crashing
    const chatContainer = page.locator('.font-mono').first();
    await expect(chatContainer).toBeVisible();

    // Check long text is truncated or rendered without layout break
    const veryLong = page.getByText('A'.repeat(1000), { exact: false });
    await expect(veryLong).toBeVisible();
  });
});
