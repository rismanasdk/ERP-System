import { createContext } from 'react'
import type { Branch } from '../types/auth'

export type BranchContextValue = {
  accessibleBranches: Branch[]
  selectedBranch: Branch | null
  isAllBranches: boolean
  loading: boolean
  error: string | null
  selectBranch: (branch: Branch | null) => void
  clearSelection: () => void
  refreshBranches: () => Promise<void>
}

export const BranchContext = createContext<BranchContextValue | undefined>(undefined)
