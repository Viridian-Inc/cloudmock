import { h } from 'preact'
import { useState, useEffect } from 'preact/hooks'
import { getViews, createView, deleteView } from '../api'

interface Props {
  currentFilters: Record<string, any>
  onLoadView: (filters: Record<string, any>) => void
}

export function SavedViewsPicker({ currentFilters, onLoadView }: Props) {
  const [views, setViews] = useState<any[]>([])
  const [showSave, setShowSave] = useState(false)
  const [name, setName] = useState('')

  useEffect(() => {
    getViews().then(setViews).catch(() => {})
  }, [])

  const handleSave = async () => {
    if (!name.trim()) return
    await createView({ name, filters: currentFilters })
    setName('')
    setShowSave(false)
    getViews().then(setViews)
  }

  const handleDelete = async (id: string) => {
    await deleteView(id)
    getViews().then(setViews)
  }

  return (
    <div class="saved-views">
      <div style="display: flex; gap: 8px; align-items: center; flex-wrap: wrap;">
        {views.map(v => (
          <div class="badge badge-pill" style="cursor: pointer; display: flex; align-items: center; gap: 4px;"
               onClick={() => onLoadView(v.filters)}>
            {v.name}
            <span style="cursor: pointer; opacity: 0.5; margin-left: 4px;"
                  onClick={(e) => { e.stopPropagation(); handleDelete(v.id) }}>x</span>
          </div>
        ))}
        {showSave ? (
          <div style="display: flex; gap: 4px; align-items: center;">
            <input class="filter-input" placeholder="View name" value={name}
                   onInput={(e) => setName((e.target as HTMLInputElement).value)}
                   style="width: 120px; font-size: 12px; padding: 4px 8px;" />
            <button class="btn btn-primary btn-sm" onClick={handleSave}>Save</button>
            <button class="btn btn-ghost btn-sm" onClick={() => setShowSave(false)}>Cancel</button>
          </div>
        ) : (
          <button class="btn btn-ghost btn-sm" onClick={() => setShowSave(true)}>Save View</button>
        )}
      </div>
    </div>
  )
}
