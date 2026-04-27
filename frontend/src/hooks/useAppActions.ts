import { useCallback, useRef } from 'react'
import { useStore } from '../store/useStore'
import { executeCommand, parseCommand, SLASH_COMMANDS } from '../components/terminal/slashCommands'
import {
  projectClient,
  queryClient,
  agentClient,
  ingestionClient,
  libraryClient,
  authClient,
  skillClient,
  toolClient,
  nlpClient,
  registryClient,
  sandboxClient,
  notificationClient,
} from '../api/factory'

export const handleError = (err: any, context: string) => {
  const store = useStore.getState()
  const msg = err?.message || `Errore in ${context}`
  store.setLastError(msg)
  store.setErrorToast(msg, 'error')
  setTimeout(() => useStore.getState().setLastError(null), 5000)
}

export function useAppActions() {
  const store = useStore()
  const errorTimerRef = useRef<any>(null)

  const handleError = useCallback((err: any, context: string) => {
    const msg = err?.message || `Errore in ${context}`
    store.setLastError(msg)
    if (errorTimerRef.current) clearTimeout(errorTimerRef.current)
    errorTimerRef.current = setTimeout(() => store.setLastError(null), 5000)
  }, [store])

  const loadProjectData = useCallback(() => {
    if (!store.projectID) return

    projectClient.getOntology({ projectId: store.projectID }).then((res: any) => {
      store.setOntologyRaw(res.alephDefinition || '')
      store.setAvailableObjects(res.objectNames || [])
      if (res.objectNames?.length > 0 && !store.selectedObject) {
        store.setSelectedObject(res.objectNames[0])
      }
    }).catch((e) => handleError(e, 'getOntology'))

    agentClient.listAgents({ projectId: store.projectID }).then((res: any) => {
      store.setAgents(res.agents || [])
    }).catch((e) => handleError(e, 'listAgents'))

    ingestionClient.listTasks({ projectId: store.projectID }).then((res: any) => {
      store.setIngestionTasks(res.tasks || [])
    }).catch((e) => handleError(e, 'listTasks'))

    libraryClient.listAssets({ projectId: store.projectID }).then((res: any) => {
      store.setAssets(res.assets || [])
    }).catch((e) => handleError(e, 'listAssets'))

    skillClient.listSkills({ projectId: store.projectID }).then((res: any) => {
      store.setSkills(res.skills || [])
    }).catch((e) => handleError(e, 'listSkills'))

    toolClient.listTools({ projectId: store.projectID }).then((res: any) => {
      store.setTools(res.tools || [])
    }).catch((e) => handleError(e, 'listTools'))

    agentClient.listModels({}).then((res: any) => {
      store.setOllamaHealthy(true)
      store.setOllamaModels(res.models || [])
    }).catch(() => {
      store.setOllamaHealthy(false)
      store.setOllamaModels([])
    })

    nlpClient.analyzeSentiment({ text: 'ping' }).then(() => {
      store.setNlpHealthy(true)
    }).catch(() => {
      store.setNlpHealthy(false)
    })

    authClient.listApiKeys({ projectId: store.projectID }).then((res: any) => {
      store.setApiKeys(res.keys || [])
    }).catch((e) => handleError(e, 'listApiKeys'))

    notificationClient.listChannels({ projectId: store.projectID }).then((res: any) => {
      store.setNotificationChannels(res.channels || [])
    }).catch((e) => handleError(e, 'listChannels'))

    registryClient.listComponents({}).then((res: any) => {
      store.setRegistryComponents(res.components || [])
    }).catch((e) => handleError(e, 'listComponents'))
  }, [store.projectID, handleError])

   const handleCommandResult = useCallback((result: ReturnType<typeof executeCommand>) => {
    if (!result.handled) return false
    
    switch (result.action) {
      case 'SHOW_INLINE':
        const slideOverTargets = ['explore', 'map', 'timeline', 'graph', 'explorer']
        const shouldUseSlideOver = result.target && slideOverTargets.includes(result.target)
        
        if (shouldUseSlideOver) {
          store.setSlideOverContent({
            type: result.target as any,
            title: result.target || 'View',
            data: result.args ? { text: result.args } : undefined,
          })
        } else {
          store.setInlineContent({
            type: result.target as any,
            title: result.target || 'View',
            data: result.args ? { text: result.args } : undefined,
          })
          store.setCurrentView('inline')
          store.setShowInlinePanel(true)
        }
        return true
      case 'CLEAR_CHAT':
        store.clearChat()
        return true
      case 'SWITCH_COPILOT':
        store.addChatMessage({ role: 'system', content: result.message || '', createdAt: Date.now() })
        return true
      case 'AGENT_COMMAND':
        store.addChatMessage({ role: 'system', content: `Model command: ${result.args || ''}`, createdAt: Date.now() })
        return true
      default:
        return false
    }
  }, [store])

  const onSend = useCallback(async () => {
    if (!store.input || store.isStreaming || !store.selectedAgent) return
    const userMsg = store.input
    const parsed = parseCommand(userMsg)
    const cmdResult = executeCommand(userMsg)
    
    if (cmdResult.handled) {
      const commandDef = SLASH_COMMANDS.find(c => c.name === parsed?.command)
      if (commandDef?.requiresConfirmation) {
        store.setChat([...store.chat, { 
          role: 'system', 
          content: `Confermi l'esecuzione di ${commandDef.name}?`, 
          requiresConfirmation: true,
          createdAt: Date.now() 
        }])
        store.setInput('')
        return
      }
      
      if (handleCommandResult(cmdResult)) {
        store.setInput('')
        store.addToHistory(userMsg)
        return
      }
    }
    store.addChatMessage({ role: 'user', content: userMsg, createdAt: Date.now() })
    store.setInput('')
    store.setIsStreaming(true)
    const ac = new AbortController()
    store.setStreamAbortController(ac)
    try {
      const stream = queryClient.chat({ projectId: store.projectID, agentId: store.selectedAgent, message: userMsg }, { signal: ac.signal })
      let fullContent = ''
      let fullToolCall = ''
      let requiresConfirmation = false
      store.addChatMessage({ role: 'assistant', content: '', toolCall: '', createdAt: Date.now() })
      for await (const res of stream) {
        fullContent += (res as any).token || ''
        fullToolCall += (res as any).toolCall || ''
        if ((res as any).requiresConfirmation) requiresConfirmation = true
        const currentChat = useStore.getState().chat
        store.setChat([...currentChat.slice(0, -1), { role: 'assistant', content: fullContent, toolCall: fullToolCall, requiresConfirmation, createdAt: Date.now() }])
      }
      if (requiresConfirmation) {
        store.setPendingConfirmation({ projectId: store.projectID, agentId: store.selectedAgent })
      }
    } catch (err: any) {
      const msg = err.name === 'AbortError' ? 'Richiesta annullata' : `Errore: ${err.message || 'impossibile contattare il backend'}`
      store.addChatMessage({ role: 'assistant', content: msg, toolCall: '', createdAt: Date.now() })
    } finally {
      store.setIsStreaming(false)
      store.setStreamAbortController(null)
    }
  }, [store, handleCommandResult])

  const onConfirmAction = useCallback(async (approved: boolean) => {
    const currentChat = useStore.getState().chat
    const lastMsg = currentChat[currentChat.length - 1]
    store.setChat([
      ...currentChat.slice(0, -1),
      { ...lastMsg, requiresConfirmation: false },
    ])
    const pc = store.pendingConfirmation
    try {
      await queryClient.confirmAction({ projectId: pc?.projectId || store.projectID, agentId: pc?.agentId || store.selectedAgent, approved })
      store.addChatMessage({ role: 'assistant', content: approved ? 'Azione approvata e eseguita.' : 'Azione rifiutata.', toolCall: '', createdAt: Date.now() })
    } catch (e: any) {
      const msg = e.message || 'Errore nella conferma dell\'azione.'
      store.addChatMessage({ role: 'assistant', content: msg, toolCall: '', createdAt: Date.now() })
      store.setErrorToast(msg, 'error')
    } finally {
      store.setPendingConfirmation(null)
    }
  }, [store])

  const onRunSkill = useCallback(async (skillId: string) => {
    let inputParams = {}
    try { inputParams = JSON.parse(useStore.getState().sandboxInput) } catch {}
    try {
      const res = await sandboxClient.runSkill({ skillId, inputParams, context: { projectId: store.projectID } }) as any
      store.setSandboxResult(res.result)
      store.setSlideOverContent({ type: 'sandbox', title: 'Risultato Esecuzione', data: res.result })
    } catch (e: any) {
      const msg = e.message || 'Errore durante l\'esecuzione della skill'
      store.setSandboxResult({ stderr: msg, exitCode: 1 })
      store.setSlideOverContent({ type: 'sandbox', title: 'Risultato Esecuzione', data: { stderr: msg, exitCode: 1 } })
      store.setErrorToast(msg, 'error')
    }
  }, [store.projectID])

  const onExecuteTool = useCallback(async (toolId: string) => {
    let inputParams = {}
    try { inputParams = JSON.parse(useStore.getState().sandboxInput) } catch {}
    try {
      const res = await sandboxClient.executeTool({ toolId, inputParams }) as any
      store.setSandboxResult(res.result)
      store.setSlideOverContent({ type: 'sandbox', title: 'Risultato Esecuzione', data: res.result })
    } catch (e: any) {
      const msg = e.message || 'Errore durante l\'esecuzione del tool'
      store.setSandboxResult({ stderr: msg, exitCode: 1 })
      store.setSlideOverContent({ type: 'sandbox', title: 'Risultato Esecuzione', data: { stderr: msg, exitCode: 1 } })
      store.setErrorToast(msg, 'error')
    }
  }, [store.projectID])

  const getAssetContent = useCallback(async (assetId: string): Promise<string> => {
    const res = await libraryClient.getAssetContent({ projectId: store.projectID, assetId }) as any
    return res.content || ''
  }, [store.projectID])

  return {
    handleError,
    loadProjectData,
    handleCommandResult,
    onSend,
    onConfirmAction,
    onRunSkill,
    onExecuteTool,
    getAssetContent,
  }
}