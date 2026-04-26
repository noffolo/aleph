import { useStore } from '../../store/useStore'

export function useExplorerActions() {
  const store = useStore()

  return {
    setSelectedObject: store.setSelectedObject,
    setSearchQuery: store.setSearchQuery,
    setActiveView: store.setActiveView,
    onRowClick: store.setSelectedRow,
  }
}
