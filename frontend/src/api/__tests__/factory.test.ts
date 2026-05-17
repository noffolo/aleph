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
    const clients: Record<string, object>[] = [
      registryClient, sandboxClient, queryClient, projectClient,
      agentClient, ingestionClient, libraryClient, authClient,
      skillClient, toolClient, nlpClient, notificationClient,
    ];
    clients.forEach((client, i) => {
      const methodCount = Object.keys(client).length;
      expect(methodCount).toBeGreaterThan(0);
    });
  });

  it('distinct services produce distinct method signatures', () => {
    const pairs: [Record<string, object>, Record<string, object>][] = [
      [registryClient, sandboxClient],
      [queryClient, projectClient],
      [agentClient, ingestionClient],
      [libraryClient, authClient],
      [skillClient, toolClient],
      [nlpClient, notificationClient],
    ];
    pairs.forEach(([a, b]) => {
      const keysA = Object.keys(a).sort();
      const keysB = Object.keys(b).sort();
      expect(keysA).not.toEqual(keysB);
    });
  });
});
