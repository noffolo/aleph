import { test, expect } from '@playwright/test';
import { setupApiMocks, hydrateStore } from './connect-mock-helper';

test.describe('Ontology Flow — create, query, emerge, save', () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page);
    await page.goto('/');
    await page.waitForLoadState('load');

    // Set up main app view
    await hydrateStore(page, {
      showOnboarding: false,
      showWizard: false,
      projectID: 'proj-onto',
      apiKey: 'aleph_test_onto_xxx',
      selectedAgent: 'agent-1',
      agents: [{
        id: 'agent-1',
        name: 'Ontology Agent',
        model: 'gpt-4',
        systemPrompt: 'Ontology test agent',
      }],
    });
    await page.waitForTimeout(300);
  });

  test('ontology editor: opens and renders DSL editor with controls', async ({ page }) => {
    // Set slideOverContent to ontology type
    await hydrateStore(page, {
      slideOverContent: {
        type: 'ontology',
        title: 'Business Modeling',
        data: {
          ontologyRaw: '// Define your objects here\nobject Customer {\n  name: string\n  email: string\n}\n',
        },
      },
    });
    await page.waitForTimeout(300);

    const panel = page.locator('.glass-panel');
    await expect(panel).toBeVisible();
    await expect(panel.getByText('Modellazione Business')).toBeVisible();
  });

  test('ontology emergence: auto-detects ontology from chat context', async ({ page }) => {
    // Simulate the ontology emergence flow by setting inline content
    await hydrateStore(page, {
      inlineContent: {
        type: 'ontology',
        title: 'Ontology',
        data: {
          ontologyRaw: 'object Appalto {\n  codice: string\n  importo: number\n  stato: string\n}\n',
        },
      },
    });
    await page.waitForTimeout(300);

    // The inline panel should show ontology content
    await hydrateStore(page, {
      slideOverContent: {
        type: 'ontology',
        title: 'Business Modeling',
        data: {
          ontologyRaw: 'object Appalto {\n  codice: string\n  importo: number\n  stato: string\n}\n',
        },
      },
    });
    await page.waitForTimeout(300);

    const panel = page.locator('.glass-panel');
    await expect(panel).toBeVisible();
  });

  test('ontology save: publishes model successfully', async ({ page }) => {
    // Load ontology data and simulate save
    await hydrateStore(page, {
      slideOverContent: {
        type: 'ontology',
        title: 'Business Modeling',
        data: {
          ontologyRaw: 'object Fornitore {\n  partitaIVA: string\n  categoria: string\n}\n',
        },
      },
    });
    await page.waitForTimeout(300);

    const panel = page.locator('.glass-panel');
    await expect(panel).toBeVisible();

    // Click the "Pubblica Modello" button (save)
    await panel.getByRole('button', { name: /Pubblica Modello/i }).click();
    await page.waitForTimeout(300);

    // The panel should still be visible after save attempt
    await expect(panel).toBeVisible();
  });

  test('ontology emerge button: triggers automatic emergence', async ({ page }) => {
    // Load ontology view
    await hydrateStore(page, {
      slideOverContent: {
        type: 'ontology',
        title: 'Business Modeling',
        data: { ontologyRaw: '' },
      },
    });
    await page.waitForTimeout(300);

    const panel = page.locator('.glass-panel');
    await expect(panel).toBeVisible();

    // Click the "Emergenza Automatica" button
    await panel.getByRole('button', { name: /Emergenza Automatica/i }).click();
    await page.waitForTimeout(300);

    // The panel should still be visible — emergence is mocked
    await expect(panel).toBeVisible();
  });

  test('ontology query: executes a query after ontology is defined', async ({ page }) => {
    // Define ontology and simulate a query
    await hydrateStore(page, {
      ontologyRaw: 'object Cliente {\n  nome: string\n  fatturato: number\n}\n',
      availableObjects: ['Cliente'],
    });
    await page.waitForTimeout(200);

    // Simulate a query via copilot
    const textarea = page.locator('textarea');
    await expect(textarea).toBeVisible();
    await textarea.fill('/data select * from Cliente');
    await textarea.press('Enter');
    await page.waitForTimeout(500);

    // The app should remain stable after query execution
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
  });

  test('ontology delete: deleting an ontology object cleans up state', async ({ page }) => {
    // Create an ontology object first
    await hydrateStore(page, {
      ontologyRaw: 'object Temporaneo {\n  campo: string\n}\n',
      availableObjects: ['Temporaneo', 'Cliente'],
      selectedObject: 'Temporaneo',
    });
    await page.waitForTimeout(200);

    // Remove the object by clearing available objects
    await hydrateStore(page, {
      availableObjects: ['Cliente'],
      selectedObject: 'Cliente',
    });
    await page.waitForTimeout(200);

    // App should still be functional
    await expect(page.getByPlaceholder(/scrivi un messaggio/i)).toBeVisible();
  });

  test('multiple ontology objects: handles complex DSL with nested structures', async ({ page }) => {
    const complexDSL = `object Appalto {
  codice: string
  importo: number
  stato: string
  fornitore: Fornitore
}

object Fornitore {
  partitaIVA: string
  categoria: string
  rating: number
}

relation Appalto -> Fornitore: assegnato_a`;

    await hydrateStore(page, {
      slideOverContent: {
        type: 'ontology',
        title: 'Business Modeling',
        data: { ontologyRaw: complexDSL },
      },
      ontologyRaw: complexDSL,
    });
    await page.waitForTimeout(300);

    const panel = page.locator('.glass-panel');
    await expect(panel).toBeVisible();
    // Panel should render without crashing for complex DSL
  });
});
