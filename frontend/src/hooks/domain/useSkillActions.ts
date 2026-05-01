import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import type { Skill } from '../../store/types'
import { skillClient } from '../../api/factory'
import { handleError } from '../useAppActions'

export function useSkillActions(loadProjectData: () => void) {
  const projectID = useStore(s => s.projectID)

  return {
    onCreateSkill: useCallback((name: string, description: string, toolIds: string[]) => {
      skillClient.createSkill({ projectId: projectID, skill: { name, description, toolIds } })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'createSkill'))
    }, [projectID, loadProjectData]),
    onUpdateSkill: useCallback((skill: Skill) => {
      skillClient.updateSkill({ projectId: projectID, skill })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'updateSkill'))
    }, [projectID, loadProjectData]),
    onViewSkillDetail: useCallback((skill: Skill) => {
      useStore.getState().setSlideOverContent({ type: 'skill', title: skill.name, data: skill })
    }, []),
    onDeleteSkill: useCallback((id: string) => {
      skillClient.deleteSkill({ projectId: projectID, id })
        .then(() => loadProjectData())
        .catch((e: unknown) => handleError(e, 'deleteSkill'))
    }, [projectID, loadProjectData]),
    onRunSkill: useCallback((id: string) => {
      const skill = useStore.getState().skills.find((s: Skill) => s.id === id)
      if (skill) useStore.getState().setSlideOverContent({ type: 'skill', title: skill.name, data: skill })
      useStore.getState().setSandboxInput('{}')
    }, []),
  }
}
