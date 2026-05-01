import { useCallback } from 'react'
import { useStore } from '../../store/useStore'
import type { RegistryComponent } from '../../store/types'
import { RegistryComponentSchema } from '../../schemas'
import { registryClient } from '../../api/factory'
import { handleError } from '../useAppActions'
import { fromProto } from '../../schemas/validate'
import { z } from 'zod'

export function useComponentActions() {
  return {
    onUpdateComponentStatus: useCallback((id: string, status: string) => {
      registryClient.updateComponentStatus({ id, status })
        .then(() => {
          registryClient.listComponents({}).then((res) => useStore.getState().setRegistryComponents(fromProto(z.array(RegistryComponentSchema), res.components || [])))
        })
        .catch((e: unknown) => handleError(e, 'updateComponentStatus'))
    }, []),
    onRegisterComponent: useCallback((metadata: RegistryComponent) => {
      const { creationTimestamp, lastUpdatedTimestamp, ...rest } = metadata
      registryClient.registerComponent({ metadata: rest })
        .then(() => {
          registryClient.listComponents({}).then((res) => useStore.getState().setRegistryComponents(fromProto(z.array(RegistryComponentSchema), res.components || [])))
        })
        .catch((e: unknown) => handleError(e, 'registerComponent'))
    }, []),
    onGetComponent: useCallback(async (id: string) => {
      try {
        const res = await registryClient.getComponent({ id })
        const result = fromProto(z.object({ metadata: RegistryComponentSchema.optional() }), res).metadata
        return (result ?? null) as import('../../store/types').RegistryComponent | null
      } catch (e: unknown) { handleError(e, 'getComponent'); return null }
    }, []),
  }
}
