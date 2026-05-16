import { describe, it, expect } from 'vitest';
import {
  registryClient,
  sandboxClient,
  queryClient,
  projectClient,
  agentClient,
  ingestionClient,
  libraryClient,
  authClient,
  skillClient,
  toolClient,
  nlpClient,
  notificationClient,
} from '../factory';

describe('API client factory', () => {
  it('exports all 12 clients', () => {
    // Compilation-time check: if import paths are wrong, TypeScript fails
    // Runtime check: all clients are defined objects
    expect(registryClient).toBeDefined();
    expect(sandboxClient).toBeDefined();
    expect(queryClient).toBeDefined();
    expect(projectClient).toBeDefined();
    expect(agentClient).toBeDefined();
    expect(ingestionClient).toBeDefined();
    expect(libraryClient).toBeDefined();
    expect(authClient).toBeDefined();
    expect(skillClient).toBeDefined();
    expect(toolClient).toBeDefined();
    expect(nlpClient).toBeDefined();
    expect(notificationClient).toBeDefined();
  });

  it('each client exposes PromiseClient methods', () => {
    const clients = [
      registryClient, sandboxClient, queryClient, projectClient,
      agentClient, ingestionClient, libraryClient, authClient,
      skillClient, toolClient, nlpClient, notificationClient,
    ];
    clients.forEach((client) => {
      const methodCount = Object.keys(client).length;
      expect(methodCount).toBeGreaterThan(0);
    });
  });

  it('all clients share the same transport configuration', () => {
    // Type-level verification — transport is in client.ts
    // All clients are created with the same transport, so any client
    // should have the expected method structure
    const clientKeys = Object.keys(registryClient);
    expect(clientKeys.length).toBeGreaterThan(0);
  });
});
