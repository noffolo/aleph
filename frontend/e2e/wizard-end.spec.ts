import { test, expect } from '@playwright/test';
import { setupApiMocks } from './connect-mock-helper';

test('debug: complete wizard end to end', async ({ page }) => {
  await setupApiMocks(page);
  await page.goto('/');
  await page.waitForLoadState('load');
  await page.waitForTimeout(1000);

  const body1 = await page.locator('body').innerText();
  console.log('INITIAL BODY:', body1.substring(0, 200));

  await page.getByText(/Nuovo spazio di lavoro/i).click();
  await page.waitForTimeout(500);

  await page.getByPlaceholder('xyz').fill('WizardTest');
  await page.getByRole('button', { name: /Prosegui/i }).click();
  await page.waitForTimeout(3000);

  const body2 = await page.locator('body').innerText();
  console.log('AFTER STEP 1:', body2.substring(0, 300));

  const genBtn = page.getByRole('button', { name: /Genera API Key/i });
  if (await genBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
    await genBtn.click();
    await page.waitForTimeout(3000);
  }

  const body3 = await page.locator('body').innerText();
  console.log('AFTER STEP 2:', body3.substring(0, 300));

  const iniziaBtn = page.getByRole('button', { name: /Inizia/i });
  if (await iniziaBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
    await iniziaBtn.click();
    await page.waitForTimeout(3000);
  }

  const body4 = await page.locator('body').innerText();
  console.log('AFTER FINISH:', body4.substring(0, 300));

  await page.screenshot({ path: '/tmp/wizard-final.png', fullPage: true });
});
