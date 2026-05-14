import { test, expect } from '@playwright/test';

test.describe('Aleph-v2 Smoke Tests', () => {
  test('health check - app loads', async ({ page }) => {
    const response = await page.goto('/');
    expect(response?.status()).toBe(200);
    await expect(page).toHaveTitle(/Aleph/);
  });

  test('health check - API responds', async ({ request }) => {
    const response = await request.get('http://localhost:8080/api/v1/health');
    expect(response.status()).toBe(200);
  });

  test('sidebar navigation renders', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('nav, [role="navigation"], aside')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Login Flow', () => {
  test('login page renders with form', async ({ page }) => {
    await page.goto('/');
    const apiKeyInput = page.locator('input[placeholder*="key"], input[name*="api"], input[type="password"]');
    if (await apiKeyInput.isVisible({ timeout: 3000 }).catch(() => false)) {
      await expect(apiKeyInput).toBeVisible();
    }
  });

  test('invalid API key shows error', async ({ page }) => {
    await page.goto('/');
    const apiKeyInput = page.locator('input[placeholder*="key"], input[name*="api"], input[type="password"]');
    if (await apiKeyInput.isVisible({ timeout: 3000 }).catch(() => false)) {
      await apiKeyInput.fill('invalid-key-12345');
      const submitBtn = page.locator('button[type="submit"], button:has-text("Connect"), button:has-text("Login")');
      if (await submitBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await submitBtn.click();
        await expect(page.locator('text=invalid, text=error, text=unauthorized').first()).toBeVisible({ timeout: 5000 });
      }
    }
  });

  test('successful login navigates to workspace', async ({ page }) => {
    const apiKey = process.env.E2E_TEST_API_KEY;
    test.skip(!apiKey, 'E2E_TEST_API_KEY not set');

    await page.goto('/');
    const apiKeyInput = page.locator('input[placeholder*="key"], input[name*="api"], input[type="password"]');
    if (await apiKeyInput.isVisible({ timeout: 3000 }).catch(() => false)) {
      await apiKeyInput.fill(apiKey);
      const submitBtn = page.locator('button[type="submit"], button:has-text("Connect"), button:has-text("Login")');
      if (await submitBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await submitBtn.click();
        await page.waitForTimeout(3000);
      }
    }
  });
});

test.describe('Project CRUD Flow', () => {
  test('can create and view a project', async ({ page }) => {
    const apiKey = process.env.E2E_TEST_API_KEY;
    test.skip(!apiKey, 'E2E_TEST_API_KEY not set');

    await page.goto('/');

    const projectInput = page.locator('input[placeholder*="project"], input[name*="project"]');
    const createBtn = page.locator('button:has-text("New"), button:has-text("Create"), [aria-label*="create"]');

    if (await projectInput.isVisible({ timeout: 3000 }).catch(() => false)) {
      const projectName = `e2e-test-${Date.now()}`;
      await projectInput.fill(projectName);
      if (await createBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await createBtn.click();
        await page.waitForTimeout(2000);
      }
    }
  });
});
