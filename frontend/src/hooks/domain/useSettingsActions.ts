import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import { ApiKeySchema } from '../../schemas'
import { authClient, notificationClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'
import { z } from 'zod'

export function useSettingsActions() {
  const store = useStore()

  return {
    onCreateApiKey: useCallback((label: string) => {
      authClient.createApiKey({ projectId: store.projectID, label })
        .then(() => {
          authClient.listApiKeys({ projectId: store.projectID }).then((res) => store.setApiKeys(fromProto(z.array(ApiKeySchema), res.keys || [])))
        })
        .catch((e: unknown) => handleError(e, 'createApiKey'))
    }, [store.projectID]),
    onDeleteApiKey: useCallback((id: string) => {
      authClient.deleteApiKey({ projectId: store.projectID, id })
        .then(() => {
          authClient.listApiKeys({ projectId: store.projectID }).then((res) => store.setApiKeys(fromProto(z.array(ApiKeySchema), res.keys || [])))
        })
        .catch((e: unknown) => handleError(e, 'deleteApiKey'))
    }, [store.projectID]),
    onSendWebhook: useCallback((url: string, payloadJson: string, secret: string) => {
      notificationClient.sendWebhook({ url, payloadJson, secret })
        .then((res) => {
          const result = fromProto(z.object({ success: z.boolean(), error: z.optional(z.string()) }), res)
          if (result.success) {
            store.setLastError(null)
            store.addToast({ message: 'Webhook inviato con successo!', type: 'success', context: 'sendWebhook' })
          } else { handleError(new Error(result.error), 'sendWebhook') }
        })
        .catch((e: unknown) => handleError(e, 'sendWebhook'))
    }, []),
  }
}
