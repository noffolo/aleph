import { useStore } from '../store/useStore'

export function reportError(context: string, error: unknown) {
    if (import.meta.env.DEV) {
        console.error(`[${context}]`, error)
    }
    useStore.getState().addToast({ 
        type: 'error', 
        context: context, 
        message: error instanceof Error ? error.message : String(error) 
    })
}
