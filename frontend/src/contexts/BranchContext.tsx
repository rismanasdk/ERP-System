import { useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { branchesApi } from '../services/branches'
import { readStoredAccessToken } from '../services/authSession'
import type { Branch } from '../types/auth'
import { useAuth } from '../hooks/useAuth'
import { BranchContext, type BranchContextValue } from './branch-context'

const STORAGE_KEY = 'erp_selected_branch'

function readStoredSelectedBranch(): Branch | null {
  const raw = sessionStorage.getItem(STORAGE_KEY)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as Branch
    if (!parsed || !parsed.id || !parsed.name) return null
    return parsed
  } catch {
    return null
  }
}

function persistSelectedBranch(branch: Branch | null) {
  if (!branch) {
    sessionStorage.removeItem(STORAGE_KEY)
    return
  }
  sessionStorage.setItem(STORAGE_KEY, JSON.stringify(branch))
}

export function BranchProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [accessibleBranches, setAccessibleBranches] = useState<Branch[]>([])
  const [selectedBranch, setSelectedBranch] = useState<Branch | null>(readStoredSelectedBranch)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const isSuperAdmin = Boolean(user?.roles?.includes('SUPER_ADMIN'))
  const isAllBranches = !selectedBranch || (isSuperAdmin && selectedBranch.id === -1)

  const clearSelection = useCallback(() => {
    setSelectedBranch(null)
    persistSelectedBranch(null)
  }, [])

  const resetBranchState = useCallback(() => {
    setAccessibleBranches([])
    setSelectedBranch(null)
    setError(null)
    persistSelectedBranch(null)
  }, [])

  const applyDefaultSelection = useCallback((branches: Branch[]) => {
    if (!branches.length) {
      setSelectedBranch(null)
      persistSelectedBranch(null)
      return
    }

    const persisted = readStoredSelectedBranch()
    const persistedAllowed = persisted ? branches.some((branch) => branch.id === persisted.id) : false

    if (isSuperAdmin) {
      const nextSelection = persistedAllowed ? persisted : { id: -1, name: 'All Branches', code: 'ALL' }
      setSelectedBranch(nextSelection)
      persistSelectedBranch(nextSelection)
      return
    }

    const nextSelection = persistedAllowed ? persisted : branches[0]
    setSelectedBranch(nextSelection)
    persistSelectedBranch(nextSelection)
  }, [isSuperAdmin])

  const selectBranch = useCallback((branch: Branch | null) => {
    if (branch === null && isSuperAdmin) {
      setSelectedBranch({ id: -1, name: 'All Branches', code: 'ALL' })
      persistSelectedBranch({ id: -1, name: 'All Branches', code: 'ALL' })
      return
    }

    if (branch && !accessibleBranches.some((item) => item.id === branch.id)) {
      return
    }

    setSelectedBranch(branch)
    persistSelectedBranch(branch)
  }, [accessibleBranches, isSuperAdmin])

  const refreshBranches = useCallback(async () => {
    setLoading(true)
    setError(null)

    try {
      const token = readStoredAccessToken() ?? undefined
      const branches = await branchesApi.list(true, token)
      const safeBranches = Array.isArray(branches) ? branches : []
      setAccessibleBranches(safeBranches)

      if (safeBranches.length === 0) {
        setSelectedBranch(null)
        persistSelectedBranch(null)
        return
      }

      applyDefaultSelection(safeBranches)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to load branches.')
      setAccessibleBranches([])
      setSelectedBranch(null)
      persistSelectedBranch(null)
    } finally {
      setLoading(false)
    }
  }, [applyDefaultSelection])

  useEffect(() => {
    queueMicrotask(() => {
      if (!user) {
        resetBranchState()
        return
      }
      void refreshBranches()
    })
  }, [refreshBranches, resetBranchState, user])

  const value = useMemo<BranchContextValue>(() => ({
    accessibleBranches,
    selectedBranch,
    isAllBranches,
    loading,
    error,
    selectBranch,
    clearSelection,
    refreshBranches,
  }), [accessibleBranches, clearSelection, error, isAllBranches, loading, refreshBranches, selectBranch, selectedBranch])

  return <BranchContext.Provider value={value}>{children}</BranchContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useBranch() {
  const context = useContext(BranchContext)
  if (!context) {
    return {
      accessibleBranches: [],
      selectedBranch: null,
      isAllBranches: true,
      loading: false,
      error: null,
      selectBranch: () => undefined,
      clearSelection: () => undefined,
      refreshBranches: async () => undefined,
    }
  }
  return context
}
