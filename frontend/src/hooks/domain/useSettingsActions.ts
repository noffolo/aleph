import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import { ApiKeySchema } from '../../schemas'
import { authClient, notificationClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'
import { z } from 'zod'

export function useSettingsActions() {
  const projectID = useStore(s => s.projectID)

  return {
    onCreateApiKey: useCallback((label: string) => {
      authClient.createApiKey({ projectId: projectID, label })
        .then(() => {
          authClient.listApiKeys({ projectId: projectID }).then((res) => useStore.getState().setApiKeys(fromProto(z.array(ApiKeySchema), res.keys || [])))
        })
        .catch((e: unknown) => handleError(e, 'createApiKey'))
    }, [projectID]),
    onDeleteApiKey: useCallback((id: string) => {
      authClient.deleteApiKey({ projectId: projectID, id })
        .then(() => {
          authClient.listApiKeys({ projectId: projectID }).then((res) => useStore.getState().setApiKeys(fromProto(z.array(ApiKeySchema), res.keys || [])))
        })
        .catch((e: unknown) => handleError(e, 'deleteApiKey'))
    }, [projectID]),
    onSendWebhook: useCallback((url: string, payloadJson: string, secret: string) => {
      notificationClient.sendWebhook({ url, payloadJson, secret })
        .then((res) => {
          const result = fromProto(z.object({ success: z.boolean(), error: z.optional(z.string()) }), res)
          if (result.success) {
            useStore.getState().setLastError(null)
            useStore.getState().addToast({ message: 'Webhook inviato con successo!', type: 'success', context: 'sendWebhook' })
          } else { handleError(new Error(result.error), 'sendWebhook') }
        })
        .catch((e: unknown) => handleError(e, 'sendWebhook'))
    }, []),
  }
}
