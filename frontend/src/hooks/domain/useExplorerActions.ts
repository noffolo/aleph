import { useStore } from '../../store/useStore'

export function useExplorerActions() {
  const setSelectedObject = useStore(s => s.setSelectedObject)
  const setSearchQuery = useStore(s => s.setSearchQuery)
  const setActiveView = useStore(s => s.setActiveView)
  const setSelectedRow = useStore(s => s.setSelectedRow)

  return {
    setSelectedObject,
    setSearchQuery,
    setActiveView,
    onRowClick: setSelectedRow,
  }
}
