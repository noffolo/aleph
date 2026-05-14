import { describe, it, expect } from 'vitest'
import { VIEW_TO_SCENE, EXPLORE_VIEWS, AGENT_VIEWS, SYSTEM_VIEWS, VIEW_LABELS } from '../sceneMapping'

describe('sceneMapping', () => {
  describe('VIEW_TO_SCENE', () => {
    it('maps explore views to explore scene', () => {
      expect(VIEW_TO_SCENE.explore).toBe('explore')
      expect(VIEW_TO_SCENE.library).toBe('explore')
      expect(VIEW_TO_SCENE.ontology).toBe('explore')
      expect(VIEW_TO_SCENE.data).toBe('explore')
    })

    it('maps agent views to agents scene', () => {
      expect(VIEW_TO_SCENE.agent).toBe('agents')
      expect(VIEW_TO_SCENE.skill).toBe('agents')
      expect(VIEW_TO_SCENE.tool).toBe('agents')
      expect(VIEW_TO_SCENE.component).toBe('agents')
    })

    it('maps system views to system scene', () => {
      expect(VIEW_TO_SCENE.health).toBe('system')
      expect(VIEW_TO_SCENE.settings).toBe('system')
      expect(VIEW_TO_SCENE.predict).toBe('system')
    })

    it('maps dashboard to terminal scene', () => {
      expect(VIEW_TO_SCENE.dashboard).toBe('terminal')
    })
  })

  describe('VIEW_LISTS', () => {
    it('EXPLORE_VIEWS contains 4 views', () => {
      expect(EXPLORE_VIEWS).toHaveLength(4)
      expect(EXPLORE_VIEWS).toContain('explore')
      expect(EXPLORE_VIEWS).toContain('library')
    })

    it('AGENT_VIEWS contains 4 views', () => {
      expect(AGENT_VIEWS).toHaveLength(4)
      expect(AGENT_VIEWS).toContain('agent')
      expect(AGENT_VIEWS).toContain('tool')
    })

    it('SYSTEM_VIEWS contains 3 views', () => {
      expect(SYSTEM_VIEWS).toHaveLength(3)
      expect(SYSTEM_VIEWS).toContain('health')
    })
  })

  describe('VIEW_LABELS', () => {
    it('has labels for all mapped views', () => {
      const allViews = new Set([...Object.keys(VIEW_TO_SCENE)])
      for (const view of allViews) {
        expect(VIEW_LABELS[view]).toBeDefined()
      }
    })

    it('returns correct labels', () => {
      expect(VIEW_LABELS.explore).toBe('Explorer')
      expect(VIEW_LABELS.agent).toBe('Agents')
      expect(VIEW_LABELS.settings).toBe('Settings')
      expect(VIEW_LABELS.dashboard).toBe('Dashboard')
    })
  })
})
