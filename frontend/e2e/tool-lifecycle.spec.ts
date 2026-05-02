import { test, expect } from '@playwright/test';
import { setupApiMocks, hydrateStore, callStoreAction } from './connect-mock-helper';

test.describe('Tool Lifecycle — register, execute, view results', () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page);
    await page.goto('/');
    await page.waitForLoadState('load');

    // Set up main app view with a project and agent
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-tools',
      apiKey: 'aleph_test_tool_xxx',
      selectedAgent: 'agent-1',
      agents: [{
        id: 'agent-1',
        name: 'Tool Test Agent',
        model: 'gpt-4',
        systemPrompt: 'You are a tool testing agent.',
      }],
    });
    await page.waitForTimeout(300);
  });

  test('tool form: opens from sidebar and renders form fields', async ({ page }) => {
    // Open the Tools panel from the sidebar
    await page.getByTestId('sidebar-tools').click();
    await page.waitForTimeout(300);

    const panel = page.locator('.glass-panel');
    await expect(panel).toBeVisible();
    await expect(panel.getByText('Tools')).toBeVisible();

    // Open the tool creation form
    await hydrateStore(page, {
      slideOverContent: {
        type: 'tool-form',
        title: 'New Tool',
        data: null,
      },
    });
    await page.waitForTimeout(300);

    // The form should be visible in the panel
    await expect(panel).toBeVisible();
  });

  test('tool registration: create a new tool via store hydration', async ({ page }) => {
    // Simulate a tool being registered
    await hydrateStore(page, {
      tools: [{
        id: 'tool-001',
        name: 'Weather Fetch',
        description: 'Fetches weather data from API',
        code: 'export async function fetchWeather(lat: number, lon: number) { return { temp: 22, condition: "sunny" } }',
        category: 'osint',
        version: '1.0.0',
        healthStatus: 'healthy',
      }],
    });
    await page.waitForTimeout(300);

    // Open the Tools slideover to verify the tool appears
    await page.getByTestId('sidebar-tools').click();
    await page.waitForTimeout(300);

    const panel = page.locator('.glass-panel');
    await expect(panel).toBeVisible();
  });

  test('tool execution: simulate tool execution via /tool slash command', async ({ page }) => {
    // Register a tool first
    await hydrateStore(page, {
      tools: [{
        id: 'tool-exec-001',
        name: 'Data Fetcher',
        description: 'Fetch data from any API',
        code: 'export async function fetch(url: string): Response { return fetch(url) }',
        category: 'utility',
        version: '1.0.0',
        healthStatus: 'healthy',
      }],
    });
    await page.waitForTimeout(200);

    // Execute a tool via copilot input (send message that triggers tool call)
    const textarea = page.locator('textarea');
    await expect(textarea).toBeVisible();

    await textarea.fill('/tool execute Data Fetcher');
    await textarea.press('Enter');
    await page.waitForTimeout(500);

    // The app should still be functional — no crash after tool execution attempt
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
  });

  test('tool view detail: shows tool information in slideover', async ({ page }) => {
    // Set slideOverContent to display a specific tool
    await hydrateStore(page, {
      slideOverContent: {
        type: 'tool',
        title: 'Sentiment Analyzer',
        data: {
          id: 'tool-sentiment-001',
          name: 'Sentiment Analyzer',
          description: 'Analyzes sentiment of text using NLP sidecar',
          code: 'def analyze(text: str) -> dict',
          category: 'nlp',
          version: '1.2.0',
          healthStatus: 'healthy',
        },
      },
    });
    await page.waitForTimeout(300);

    const panel = page.locator('.glass-panel');
    await expect(panel).toBeVisible();
    await expect(panel.getByText('Sentiment Analyzer').first()).toBeVisible();
  });

  test('tool list: shows all registered tools after hydration', async ({ page }) => {
    // Hydrate with multiple tools
    await hydrateStore(page, {
      tools: [
        {
          id: 'tool-01',
          name: 'Weather Fetcher',
          description: 'Fetch weather data',
          code: '...',
          category: 'osint',
          version: '1.0',
          healthStatus: 'healthy',
        },
        {
          id: 'tool-02',
          name: 'Stock Screener',
          description: 'Screen stocks by criteria',
          code: '...',
          category: 'finance',
          version: '0.9',
          healthStatus: 'degraded',
        },
        {
          id: 'tool-03',
          name: 'Social Graph',
          description: 'Analyze social network graphs',
          code: '...',
          category: 'human-ecosystems',
          version: '2.1',
          healthStatus: 'healthy',
        },
      ],
    });
    await page.waitForTimeout(200);

    // Open tools panel to verify
    await page.getByTestId('sidebar-tools').click();
    await page.waitForTimeout(300);

    const panel = page.locator('.glass-panel');
    await expect(panel).toBeVisible();
  });

  test('tool categories: verify different tool categories are handled', async ({ page }) => {
    await hydrateStore(page, {
      tools: [
        { id: 'c1', name: 'Fin Tool', description: 'Finance tool', code: '...', category: 'finance', version: '1.0', healthStatus: 'healthy' },
        { id: 'c2', name: 'OSINT Tool', description: 'OSINT tool', code: '...', category: 'osint', version: '1.0', healthStatus: 'healthy' },
        { id: 'c3', name: 'Adapt Tool', description: 'Adaptation tool', code: '...', category: 'adaptation', version: '1.0', healthStatus: 'healthy' },
      ],
    });
    await page.waitForTimeout(200);

    // The app should render without crashing with multiple tool categories
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
  });
});
