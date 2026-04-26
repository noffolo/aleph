import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import type { Skill } from '../../store/types'
import { SkillSchema } from '../../schemas'
import { skillClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'

export function useSkillActions(loadProjectData: () => void) {
  const store = useStore()

  return {
    onCreateSkill: useCallback((name: string, description: string, toolIds: string[]) => {
      skillClient.createSkill({ projectId: store.projectID, skill: { name, description, toolIds } })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'createSkill'))
    }, [store.projectID, loadProjectData]),
    onViewSkillDetail: useCallback((skill: Skill) => {
      store.setSlideOverContent({ type: 'skill', title: skill.name, data: skill })
    }, []),
    onDeleteSkill: useCallback((id: string) => {
      skillClient.deleteSkill({ projectId: store.projectID, id })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'deleteSkill'))
    }, [store.projectID, loadProjectData]),
    onRunSkill: useCallback((id: string) => {
      const skill = useStore.getState().skills.find((s: Skill) => s.id === id)
      if (skill) useStore.getState().setSlideOverContent({ type: 'skill', title: skill.name, data: skill })
      store.setSandboxInput('{}')
    }, []),
  }
}
