import { useCallback, useRef } from 'react'
import { produce } from 'immer'
import { useStore } from '../store/useStore'
import type { SandboxResult } from '../store/types'
import type { InlineContent, SlideOverContent } from '../store/useStore'
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

interface StreamChunk {
  token?: string
  toolCall?: string
  requiresConfirmation?: boolean
}

export const handleError = (err: any, context: string) => {
  const store = useStore.getState()
  const msg = err?.message || `Errore in ${context}`
  store.setLastError(msg)
  store.addToast({ message: msg, type: 'error', context })
  setTimeout(() => useStore.getState().setLastError(null), 5000)
}

export function useAppActions() {
  const projectID = useStore((s) => s.projectID)
  const errorTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const loadAbortRef = useRef<AbortController | null>(null)

  const handleError = useCallback((err: any, context: string) => {
    const store = useStore.getState()
    const msg = err?.message || `Errore in ${context}`
    store.setLastError(msg)
    if (errorTimerRef.current) clearTimeout(errorTimerRef.current)
    errorTimerRef.current = setTimeout(() => store.setLastError(null), 5000)
  }, [])

  const loadProjectData = useCallback(() => {
    const store = useStore.getState()
    if (!projectID) return

    loadAbortRef.current?.abort()
    const ac = new AbortController()
    loadAbortRef.current = ac
    const opts = { signal: ac.signal }

    projectClient.getOntology({ projectId: projectID }, opts).then((res: any) => {
      const current = useStore.getState()
      current.setOntologyRaw(res.alephDefinition || '')
      current.setAvailableObjects(res.objectNames || [])
      if (res.objectNames?.length > 0 && !current.selectedObject) {
        current.setSelectedObject(res.objectNames[0])
      }
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'getOntology') })

    agentClient.listAgents({ projectId: projectID }, opts).then((res: any) => {
      useStore.getState().setAgents(res.agents || [])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listAgents') })

    ingestionClient.listTasks({ projectId: projectID }, opts).then((res: any) => {
      useStore.getState().setIngestionTasks(res.tasks || [])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listTasks') })

    libraryClient.listAssets({ projectId: projectID }, opts).then((res: any) => {
      useStore.getState().setAssets(res.assets || [])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listAssets') })

    skillClient.listSkills({ projectId: projectID }, opts).then((res: any) => {
      useStore.getState().setSkills(res.skills || [])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listSkills') })

    toolClient.listTools({ projectId: projectID }, opts).then((res: any) => {
      useStore.getState().setTools(res.tools || [])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listTools') })

    agentClient.listModels({}, opts).then((res: any) => {
      const current = useStore.getState()
      current.setOllamaHealthy(true)
      current.setOllamaModels(res.models || [])
    }).catch(() => {
      const current = useStore.getState()
      current.setOllamaHealthy(false)
      current.setOllamaModels([])
    })

    nlpClient.analyzeSentiment({ text: 'ping' }, opts).then(() => {
      useStore.getState().setNlpHealthy(true)
    }).catch(() => {
      useStore.getState().setNlpHealthy(false)
    })

    authClient.listApiKeys({ projectId: projectID }, opts).then((res: any) => {
      useStore.getState().setApiKeys(res.keys || [])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listApiKeys') })

    notificationClient.listChannels({ projectId: projectID }, opts).then((res: any) => {
      useStore.getState().setNotificationChannels(res.channels || [])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listChannels') })

    registryClient.listComponents({}, opts).then((res: any) => {
      useStore.getState().setRegistryComponents(res.components || [])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listComponents') })
  }, [projectID, handleError])

   const handleCommandResult = useCallback((result: ReturnType<typeof executeCommand>) => {
    const store = useStore.getState()
    if (!result.handled) return false

    switch (result.action) {
      case 'SHOW_INLINE':
        const slideOverTargets = ['explore', 'map', 'timeline', 'graph', 'explorer']
        const shouldUseSlideOver = result.target && slideOverTargets.includes(result.target)

        if (shouldUseSlideOver) {
          store.setSlideOverContent({
            type: result.target as unknown as SlideOverContent['type'],
            title: result.target || 'View',
            data: result.args ? { text: result.args } : undefined,
          })
        } else {
          store.setInlineContent({
            type: result.target as unknown as InlineContent['type'],
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
  }, [])

  const onSend = useCallback(async () => {
    const store = useStore.getState()
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

    if (parsed) {
      store.addToast({ message: `Comando sconosciuto: ${parsed.command}`, type: 'error', context: 'command' })
      store.addToHistory(userMsg)
      store.setInput('')
      return
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
      const msgIndex = useStore.getState().chat.length
      store.addChatMessage({ role: 'assistant', content: '', toolCall: '', createdAt: Date.now() })
       for await (const res of stream) {
         const chunk = res as unknown as StreamChunk
         fullContent += chunk.token || ''
         fullToolCall += chunk.toolCall || ''
         if (chunk.requiresConfirmation) requiresConfirmation = true
         
         useStore.setState(produce((state: any) => {
           const chat = state.chat
           if (chat[msgIndex] && chat[msgIndex].role === 'assistant') {
             chat[msgIndex] = { ...chat[msgIndex], content: fullContent, toolCall: fullToolCall, requiresConfirmation }
           } else {
             chat[msgIndex] = { role: 'assistant', content: fullContent, toolCall: fullToolCall, requiresConfirmation, createdAt: Date.now() }
           }
         }))
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
  }, [handleCommandResult])

  const onConfirmAction = useCallback(async (approved: boolean) => {
    const store = useStore.getState()
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
      store.addToast({ message: msg, type: 'error', context: 'confirmAction' })
    } finally {
      store.setPendingConfirmation(null)
    }
  }, [])

  const onRunSkill = useCallback(async (skillId: string) => {
    const store = useStore.getState()
    let inputParams = {}
    try { inputParams = JSON.parse(useStore.getState().sandboxInput) } catch {}
    try {
      const res = await sandboxClient.runSkill({ skillId, inputParams, context: { projectId: store.projectID } })
      const r = (res as unknown as { result: SandboxResult }).result
      store.setSandboxResult(r)
      store.setSlideOverContent({ type: 'sandbox', title: 'Risultato Esecuzione', data: r })
    } catch (e: any) {
      const msg = e.message || 'Errore durante l\'esecuzione della skill'
      store.setSandboxResult({ stderr: msg, exitCode: 1 })
      store.setSlideOverContent({ type: 'sandbox', title: 'Risultato Esecuzione', data: { stderr: msg, exitCode: 1 } })
      store.addToast({ message: msg, type: 'error', context: 'runSkill' })
    }
  }, [projectID])

  const onExecuteTool = useCallback(async (toolId: string) => {
    const store = useStore.getState()
    let inputParams = {}
    try { inputParams = JSON.parse(useStore.getState().sandboxInput) } catch {}
    try {
      const res = await sandboxClient.executeTool({ toolId, inputParams })
      const r2 = (res as unknown as { result: SandboxResult }).result
      store.setSandboxResult(r2)
      store.setSlideOverContent({ type: 'sandbox', title: 'Risultato Esecuzione', data: r2 })
    } catch (e: any) {
      const msg = e.message || 'Errore durante l\'esecuzione del tool'
      store.setSandboxResult({ stderr: msg, exitCode: 1 })
      store.setSlideOverContent({ type: 'sandbox', title: 'Risultato Esecuzione', data: { stderr: msg, exitCode: 1 } })
      store.addToast({ message: msg, type: 'error', context: 'executeTool' })
    }
  }, [projectID])

  const getAssetContent = useCallback(async (assetId: string): Promise<string> => {
    const res = await libraryClient.getAssetContent({ projectId: useStore.getState().projectID, assetId })
    return (res as unknown as { content: string }).content || ''
  }, [projectID])

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
