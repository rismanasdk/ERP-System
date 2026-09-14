import type { Branch } from '../../types/auth'
import { BranchCard } from './BranchCard'

export function BranchGrid({ branches, roles, onBranchSelect }: { branches: Branch[]; roles: string[]; onBranchSelect: (branch: Branch) => void }) {
  if (branches.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 p-8 text-center text-sm text-slate-500">
        No branch access is available for this user.
      </div>
    )
  }

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {branches.map((branch) => <BranchCard key={branch.id} branch={branch} roles={roles} onClick={() => onBranchSelect(branch)} />)}
    </div>
  )
}