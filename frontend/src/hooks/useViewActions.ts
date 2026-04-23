import { useCallback } from 'react'
import { useStore } from '../store/useStore'
import {
  projectClient,
  queryClient,
  agentClient,
  ingestionClient,
  libraryClient,
  authClient,
  skillClient,
  toolClient,
  registryClient,
  sandboxClient,
  notificationClient,
} from '../api/factory'

export function useViewActions() {
  const store = useStore()

  const loadProjectData = useCallback(() => {
    if (!store.projectID) return
    projectClient.getOntology({ projectId: store.projectID })
      .then((res: any) => {
        store.setOntologyRaw(res.alephDefinition || '')
        store.setAvailableObjects(res.objectNames || [])
        if (res.objectNames?.length > 0 && !store.selectedObject) {
          store.setSelectedObject(res.objectNames[0])
        }
      })
      .catch(() => {})

    agentClient.listAgents({ projectId: store.projectID })
      .then((res: any) => store.setAgents(res.agents || []))
      .catch(() => {})

    ingestionClient.listTasks({ projectId: store.projectID })
      .then((res: any) => store.setIngestionTasks(res.tasks || []))
      .catch(() => {})

    libraryClient.listAssets({ projectId: store.projectID })
      .then((res: any) => store.setAssets(res.assets || []))
      .catch(() => {})

    skillClient.listSkills({ projectId: store.projectID })
      .then((res: any) => store.setSkills(res.skills || []))
      .catch(() => {})

    toolClient.listTools({ projectId: store.projectID })
      .then((res: any) => store.setTools(res.tools || []))
      .catch(() => {})

    agentClient.listModels({})
      .then((res: any) => {
        store.setOllamaHealthy(true)
        store.setOllamaModels(res.models || [])
      })
      .catch(() => {
        store.setOllamaHealthy(false)
        store.setOllamaModels([])
      })

    authClient.listApiKeys({ projectId: store.projectID })
      .then((res: any) => store.setApiKeys(res.keys || []))
      .catch(() => {})

    notificationClient.listChannels({ projectId: store.projectID })
      .then((res: any) => store.setNotificationChannels(res.channels || []))
      .catch(() => {})

    registryClient.listComponents({})
      .then((res: any) => store.setRegistryComponents(res.components || []))
      .catch(() => {})
  }, [store.projectID])

  const handleError = useCallback((err: any, context: string) => {
    const msg = err?.message || `Errore in ${context}`
    store.setLastError(msg)
    setTimeout(() => store.setLastError(null), 5000)
  }, [store])

  // EXPLORER
  const explorerActions = {
    setSelectedObject: store.setSelectedObject,
    setSearchQuery: store.setSearchQuery,
    setActiveView: store.setActiveView,
    onRowClick: store.setSelectedRow,
  }

  // DATA SOURCES
  const dataSourcesActions = {
    onAddSource: useCallback((config: { name: string; sourceType: string; configJson: string }) => {
      ingestionClient.createTask({ projectId: store.projectID, task: { name: config.name, sourceType: config.sourceType, configJson: config.configJson } })
        .then(() => loadProjectData())
        .catch((e: any) => handleError(e, 'createTask'))
    }, [store.projectID, loadProjectData, handleError]),
    onRunTask: useCallback((id: string) => {
      ingestionClient.runTask({ projectId: store.projectID, taskId: id })
        .then(() => {
          const poll = () => {
            ingestionClient.getProgress({ projectId: store.projectID, taskId: id })
              .then(() => {
                ingestionClient.listTasks({ projectId: store.projectID }).then((tasksRes: any) => {
                  store.setIngestionTasks(tasksRes.tasks || [])
                  const t = (tasksRes.tasks || []).find((x: any) => x.id === id)
                  if (t && t.status !== 'completed' && t.status !== 'failed') {
                    setTimeout(poll, 1500)
                  }
                })
              })
              .catch(() => setTimeout(poll, 2000))
          }
          setTimeout(poll, 1000)
        })
        .catch((e: any) => handleError(e, 'runTask'))
    }, [store.projectID, handleError]),
    onViewLogs: useCallback((id: string) => {
      ingestionClient.getTaskLogs({ projectId: store.projectID, taskId: id })
        .then((res: any) => store.setTaskLogs(res.logs || 'Nessun log'))
        .catch((e: any) => handleError(e, 'getTaskLogs'))
    }, [store.projectID, handleError]),
    onDeleteTask: useCallback((id: string) => {
      ingestionClient.deleteTask({ projectId: store.projectID, id })
        .then(() => loadProjectData())
        .catch((e: any) => handleError(e, 'deleteTask'))
    }, [store.projectID, loadProjectData, handleError]),
  }

  // AGENTS
  const agentsActions = {
    onCreateAgent: useCallback((name: string, model: string, systemPrompt: string, provider: string, apiKey: string, baseUrl: string) => {
      agentClient.createAgent({ projectId: store.projectID, agent: { name, model, systemPrompt, provider: provider || 'ollama', apiKey: apiKey || '', baseUrl: baseUrl || '' } })
        .then(() => loadProjectData())
        .catch((e: any) => handleError(e, 'createAgent'))
    }, [store.projectID, loadProjectData, handleError]),
    onDeleteAgent: useCallback((id: string) => {
      agentClient.deleteAgent({ projectId: store.projectID, id })
        .then(() => loadProjectData())
        .catch((e: any) => handleError(e, 'deleteAgent'))
    }, [store.projectID, loadProjectData, handleError]),
    onUpdateAgent: useCallback((agent: any) => {
      agentClient.updateAgent({ projectId: store.projectID, agent })
        .then(() => loadProjectData())
        .catch((e: any) => handleError(e, 'updateAgent'))
    }, [store.projectID, loadProjectData, handleError]),
  }

  // ONTLOGY
  const ontologyActions = {
    setOntologyRaw: store.setOntologyRaw,
    onEmerge: useCallback(() => {
      projectClient.emergeOntology({ projectId: store.projectID })
        .then((res: any) => { store.setOntologyRaw(res.alephDefinition || ''); loadProjectData() })
        .catch((e: any) => handleError(e, 'emergeOntology'))
    }, [store.projectID, loadProjectData, handleError]),
    onSave: useCallback(() => {
      projectClient.saveOntology({ projectId: store.projectID, alephDefinition: store.ontologyRaw })
        .then(() => loadProjectData())
        .catch((e: any) => handleError(e, 'saveOntology'))
    }, [store.projectID, store.ontologyRaw, loadProjectData, handleError]),
  }

  // SKILLS
  const skillsActions = {
    onCreateSkill: useCallback((name: string, description: string, toolIds: string[]) => {
      skillClient.createSkill({ projectId: store.projectID, skill: { name, description, toolIds } })
        .then(() => loadProjectData())
        .catch((e: any) => handleError(e, 'createSkill'))
    }, [store.projectID, loadProjectData, handleError]),
    onViewSkillDetail: useCallback((skill: any) => {
      store.setSlideOverContent({ type: 'skill', title: skill.name, data: skill })
    }, []),
    onDeleteSkill: useCallback((id: string) => {
      skillClient.deleteSkill({ projectId: store.projectID, id })
        .then(() => loadProjectData())
        .catch((e: any) => handleError(e, 'deleteSkill'))
    }, [store.projectID, loadProjectData, handleError]),
    onRunSkill: useCallback((id: string) => {
      const skill = useStore.getState().skills.find((s: any) => s.id === id)
      if (skill) useStore.getState().setSlideOverContent({ type: 'skill', title: skill.name, data: skill })
      store.setSandboxInput('{}')
    }, []),
  }

  // TOOLS
  const toolsActions = {
    onCreateTool: useCallback((name: string, description: string, code: string) => {
      toolClient.createTool({ projectId: store.projectID, tool: { name, description, code } })
        .then(() => loadProjectData())
        .catch((e: any) => handleError(e, 'createTool'))
    }, [store.projectID, loadProjectData, handleError]),
    onEditTool: useCallback((tool: any) => {
      store.setSlideOverContent({ type: 'tool', title: tool.name, data: tool })
    }, []),
    onDeleteTool: useCallback((id: string) => {
      toolClient.deleteTool({ projectId: store.projectID, id })
        .then(() => loadProjectData())
        .catch((e: any) => handleError(e, 'deleteTool'))
    }, [store.projectID, loadProjectData, handleError]),
    onExecuteTool: useCallback((id: string) => {
      const tool = useStore.getState().tools.find((t: any) => t.id === id)
      if (tool) useStore.getState().setSlideOverContent({ type: 'tool', title: tool.name, data: tool })
      store.setSandboxInput('{}')
    }, []),
  }

  // SETTINGS
  const settingsActions = {
    onCreateApiKey: useCallback((label: string) => {
      authClient.createApiKey({ projectId: store.projectID, label })
        .then(() => {
          authClient.listApiKeys({ projectId: store.projectID }).then((res: any) => store.setApiKeys(res.keys || []))
        })
        .catch((e: any) => handleError(e, 'createApiKey'))
    }, [store.projectID, handleError]),
    onDeleteApiKey: useCallback((id: string) => {
      authClient.deleteApiKey({ projectId: store.projectID, id })
        .then(() => {
          authClient.listApiKeys({ projectId: store.projectID }).then((res: any) => store.setApiKeys(res.keys || []))
        })
        .catch((e: any) => handleError(e, 'deleteApiKey'))
    }, [store.projectID, handleError]),
    onSendWebhook: useCallback((url: string, payloadJson: string, secret: string) => {
      notificationClient.sendWebhook({ url, payloadJson, secret })
        .then((res: any) => {
          if (res.success) {
            store.setLastError(null)
            alert('Webhook inviato con successo!')
          } else { handleError(new Error(res.error), 'sendWebhook') }
        })
        .catch((e: any) => handleError(e, 'sendWebhook'))
    }, [handleError]),
  }

  // COMPONENTS (REGISTRY)
  const componentsActions = {
    onUpdateComponentStatus: useCallback((id: string, status: string) => {
      registryClient.updateComponentStatus({ id, status })
        .then(() => {
          registryClient.listComponents({}).then((res: any) => store.setRegistryComponents(res.components || []))
        })
        .catch((e: any) => handleError(e, 'updateComponentStatus'))
    }, [handleError]),
    onRegisterComponent: useCallback((metadata: any) => {
      const { creationTimestamp, lastUpdatedTimestamp, ...rest } = metadata
      registryClient.registerComponent({ metadata: rest })
        .then(() => {
          registryClient.listComponents({}).then((res: any) => store.setRegistryComponents(res.components || []))
        })
        .catch((e: any) => handleError(e, 'registerComponent'))
    }, [handleError]),
    onGetComponent: useCallback(async (id: string) => {
      try {
        const res = await registryClient.getComponent({ id }) as any
        return res.metadata || null
      } catch (e: any) { handleError(e, 'getComponent'); return null }
    }, [handleError]),
  }

  // LIBRARY
  const libraryActions = {
    onViewAsset: useCallback((id: string) => {
      store.setSelectedAssetId(id)
      libraryClient.getAssetContent({ projectId: store.projectID, assetId: id })
        .then((res: any) => store.setSelectedAssetContent(res.content))
        .catch(() => store.setSelectedAssetContent('Errore nel caricamento'))
    }, [store.projectID]),
    onDeleteAsset: useCallback((id: string) => {
      libraryClient.deleteAsset({ projectId: store.projectID, id })
        .then(() => loadProjectData())
        .catch((e: any) => handleError(e, 'deleteAsset'))
    }, [store.projectID, loadProjectData, handleError]),
    selectedAssetContent: store.selectedAssetContent,
    setSelectedAssetContent: store.setSelectedAssetContent,
    selectedAssetName: store.assets.find((a: any) => a.id === store.selectedAssetId)?.name,
    onGetAssetContent: useCallback(async (assetId: string) => {
      const res = await libraryClient.getAssetContent({ projectId: store.projectID, assetId }) as any
      return res.content || ''
    }, [store.projectID]),
    onGeneratePdf: useCallback(async (assetId: string) => {
      const res = await libraryClient.generatePdf({ projectId: store.projectID, assetId }) as any
      return { pdfData: res.pdfData as Uint8Array, filename: res.filename as string }
    }, [store.projectID]),
    onUploadAsset: useCallback(async (filename: string, content: Uint8Array) => {
      await libraryClient.uploadAsset({ projectId: store.projectID, filename, content })
      loadProjectData()
    }, [store.projectID, loadProjectData]),
    selectedAssetId: store.selectedAssetId,
  }

  return {
    store,
    loadProjectData,
    handleError,
    explorerActions,
    dataSourcesActions,
    agentsActions,
    ontologyActions,
    skillsActions,
    toolsActions,
    settingsActions,
    componentsActions,
    libraryActions,
  }
}
