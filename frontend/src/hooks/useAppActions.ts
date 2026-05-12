import { useCallback, useRef, useState } from 'react'
import { produce } from 'immer'
import { useStore } from '../store/useStore'
import type { PendingConfirmation, Agent, Asset, Skill, Tool, ApiKey, RegistryComponent } from '../store/types'
import type { AppState, SlideOverContent } from '../store/useStore'
import { VIEW_TO_SCENE } from '../store/sceneMapping'
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
import {
  ListAgentsResponse,
  ListToolsResponse,
  ListSkillsResponse,
  ListModelsResponse,
  ListAssetsResponse,
  GetAssetContentResponse,
  GetOntologyResponse,
  ListTasksResponse,
  ListApiKeysResponse,
} from '../api/proto/aleph/v1/query_pb'
import { ListChannelsResponse } from '../api/proto/aleph/v1/notification_pb'
import { ListComponentsResponse } from '../api/proto/aleph/v1/registry_pb'
import { ExecuteToolResponse, RunSkillResponse } from '../api/proto/aleph/v1/sandbox_pb'
import { AnalyzeSentimentResponse } from '../api/proto/aleph/nlp/v1/nlp_pb'

interface StreamChunk {
  token?: string
  toolCall?: string
  requiresConfirmation?: boolean
}

export const handleError = (err: unknown, context: string) => {
  const store = useStore.getState()
  const msg = err instanceof Error ? err.message : `Errore in ${context}`
  store.setLastError(msg)
  store.addToast({ message: msg, type: 'error', context })
  setTimeout(() => useStore.getState().setLastError(null), 5000)
}

export function useAppActions() {
  const projectID = useStore((s) => s.projectID)
  const errorTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const loadAbortRef = useRef<AbortController | null>(null)
  const streamAbortRef = useRef<AbortController | null>(null)
  const [pendingConfirmation, setPendingConfirmation] = useState<PendingConfirmation | null>(null)

  const handleError = useCallback((err: unknown, context: string) => {
    const store = useStore.getState()
    const msg = err instanceof Error ? err.message : `Errore in ${context}`
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

    projectClient.getOntology({ projectId: projectID }, opts).then((res: GetOntologyResponse) => {
      const current = useStore.getState()
      current.setOntologyRaw(res.alephDefinition || '')
      current.setAvailableObjects(res.objectNames || [])
      if (res.objectNames?.length > 0 && !current.selectedObject) {
        current.setSelectedObject(res.objectNames[0])
      }
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'getOntology') })

    agentClient.listAgents({ projectId: projectID }, opts).then((res: ListAgentsResponse) => {
      useStore.getState().setAgents((res.agents || []) as unknown as Agent[])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listAgents') })

    ingestionClient.listTasks({ projectId: projectID }, opts).then((res: ListTasksResponse) => {
      useStore.getState().setIngestionTasks(res.tasks || [])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listTasks') })

    libraryClient.listAssets({ projectId: projectID }, opts).then((res: ListAssetsResponse) => {
      useStore.getState().setAssets((res.assets || []) as unknown as Asset[])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listAssets') })

    skillClient.listSkills({ projectId: projectID }, opts).then((res: ListSkillsResponse) => {
      useStore.getState().setSkills((res.skills || []) as unknown as Skill[])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listSkills') })

    toolClient.listTools({ projectId: projectID }, opts).then((res: ListToolsResponse) => {
      useStore.getState().setTools((res.tools || []) as unknown as Tool[])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listTools') })

    agentClient.listModels({}, opts).then((res: ListModelsResponse) => {
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

    authClient.listApiKeys({ projectId: projectID }, opts).then((res: ListApiKeysResponse) => {
      useStore.getState().setApiKeys((res.keys || []) as unknown as ApiKey[])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listApiKeys') })

    notificationClient.listChannels({ projectId: projectID }, opts).then((res: ListChannelsResponse) => {
      useStore.getState().setNotificationChannels(res.channels || [])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listChannels') })

    registryClient.listComponents({}, opts).then((res: ListComponentsResponse) => {
      useStore.getState().setRegistryComponents((res.components || []) as unknown as RegistryComponent[])
    }).catch((e) => { if (e?.code !== 'CANCELLED') handleError(e, 'listComponents') })
  }, [projectID, handleError])

   const handleCommandResult = useCallback((result: ReturnType<typeof executeCommand>) => {
    const store = useStore.getState()
    if (!result.handled) return false

    switch (result.action) {
      case 'SHOW_INLINE':
        {
          const targetScene = VIEW_TO_SCENE[result.target as string] ?? null
          if (targetScene) store.setCurrentScene(targetScene)
          store.setSlideOverContent({
            type: result.target as SlideOverContent['type'],
            title: result.target ?? 'View',
            data: result.args ? { text: result.args } : undefined,
          })
        }
        return true
      case 'CLEAR_CHAT':
        store.clearMessages()
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

  const onSend = useCallback(async (message?: string) => {
    const store = useStore.getState()
    const userMsg = message?.trim()
    if (!userMsg || store.isStreaming || !store.selectedAgent) return
    const parsed = parseCommand(userMsg)
    const cmdResult = executeCommand(userMsg)

    if (cmdResult.handled) {
      const commandDef = SLASH_COMMANDS.find(c => c.name === parsed?.command)
      if (commandDef?.requiresConfirmation) {
        store.setMessages([...store.messages, {
          role: 'system',
          content: `Confermi l'esecuzione di ${commandDef.name}?`,
          requiresConfirmation: true,
          createdAt: Date.now()
        }])
        return
      }

      if (handleCommandResult(cmdResult)) {
        return
      }

      if (parsed) {
        store.addToast({ message: `Comando sconosciuto: ${parsed.command}`, type: 'error', context: 'command' })
        return
      }
    }

    if (parsed) {
      store.addToast({ message: `Comando sconosciuto: ${parsed.command}`, type: 'error', context: 'command' })
      return
    }

    store.addChatMessage({ role: 'user', content: userMsg, createdAt: Date.now() })
    store.setIsStreaming(true)
    const ac = new AbortController()
    streamAbortRef.current = ac
    try {
      const stream = queryClient.chat({ projectId: store.projectID, agentId: store.selectedAgent, message: userMsg }, { signal: ac.signal })
      let fullContent = ''
      let fullToolCall = ''
      let requiresConfirmation = false
      const msgIndex = useStore.getState().messages.length
      store.addChatMessage({ role: 'assistant', content: '', toolCall: '', createdAt: Date.now() })
       for await (const res of stream) {
         const chunk = res as unknown as StreamChunk
         fullContent += chunk.token || ''
         fullToolCall += chunk.toolCall || ''
         if (chunk.requiresConfirmation) requiresConfirmation = true
         
          useStore.setState(produce((state: AppState) => {
           const messages = state.messages
           if (messages[msgIndex] && messages[msgIndex].role === 'assistant') {
             messages[msgIndex] = { ...messages[msgIndex], content: fullContent, toolCall: fullToolCall, requiresConfirmation }
           } else {
             messages[msgIndex] = { role: 'assistant', content: fullContent, toolCall: fullToolCall, requiresConfirmation, createdAt: Date.now() }
           }
         }))
       }
      if (requiresConfirmation) {
        setPendingConfirmation({ projectId: store.projectID, agentId: store.selectedAgent })
      }
    } catch (err: unknown) {
      const msg = err instanceof DOMException && err.name === 'AbortError'
        ? 'Richiesta annullata'
        : `Errore: ${err instanceof Error ? err.message : 'impossibile contattare il backend'}`
      store.addChatMessage({ role: 'assistant', content: msg, toolCall: '', createdAt: Date.now() })
    } finally {
      store.setIsStreaming(false)
      streamAbortRef.current = null
    }
  }, [handleCommandResult])

  const onConfirmAction = useCallback(async (approved: boolean) => {
    const store = useStore.getState()
    const currentMessages = useStore.getState().messages
    const lastMsg = currentMessages[currentMessages.length - 1]
    store.setMessages([
      ...currentMessages.slice(0, -1),
      { ...lastMsg, requiresConfirmation: false },
    ])
    const pc = pendingConfirmation
    try {
      await queryClient.confirmAction({ projectId: pc?.projectId || store.projectID, agentId: pc?.agentId || store.selectedAgent, approved })
      store.addChatMessage({ role: 'assistant', content: approved ? 'Azione approvata e eseguita.' : 'Azione rifiutata.', toolCall: '', createdAt: Date.now() })
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Errore nella conferma dell\'azione.'
      store.addChatMessage({ role: 'assistant', content: msg, toolCall: '', createdAt: Date.now() })
      store.addToast({ message: msg, type: 'error', context: 'confirmAction' })
    } finally {
      setPendingConfirmation(null)
    }
  }, [pendingConfirmation])

  const onRunSkill = useCallback(async (skillId: string) => {
    const store = useStore.getState()
    let inputParams = {}
    try { inputParams = JSON.parse(useStore.getState().sandboxInput) } catch {}
    try {
      const res = await sandboxClient.runSkill({ skillId, inputParams, context: { projectId: store.projectID } })
      const r = (res as unknown as { result: { exitCode?: number; stdout?: string; stderr?: string } }).result
      store.setSlideOverContent({ type: 'sandbox', title: 'Risultato Esecuzione', data: r })
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Errore durante l\'esecuzione della skill'
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
      const r2 = (res as unknown as { result: { exitCode?: number; stdout?: string; stderr?: string } }).result
      store.setSlideOverContent({ type: 'sandbox', title: 'Risultato Esecuzione', data: r2 })
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Errore durante l\'esecuzione del tool'
      store.setSlideOverContent({ type: 'sandbox', title: 'Risultato Esecuzione', data: { stderr: msg, exitCode: 1 } })
      store.addToast({ message: msg, type: 'error', context: 'executeTool' })
    }
  }, [projectID])

  const getAssetContent = useCallback(async (assetId: string): Promise<string> => {
    const res = await libraryClient.getAssetContent({ projectId: useStore.getState().projectID, assetId })
    return (res as unknown as { content: string }).content || ''
  }, [projectID])

  const onCancelStream = useCallback(() => {
    streamAbortRef.current?.abort()
    streamAbortRef.current = null
    useStore.getState().setIsStreaming(false)
  }, [])

  return {
    handleError,
    loadProjectData,
    handleCommandResult,
    onSend,
    onConfirmAction,
    onRunSkill,
    onExecuteTool,
    getAssetContent,
    onCancelStream,
  }
}
