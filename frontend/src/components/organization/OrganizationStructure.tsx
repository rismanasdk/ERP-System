import { useState } from 'react'
import type { Branch, User } from '../../types/auth'
import { OrgRootNode } from './OrgRootNode'
import { BranchGrid } from './BranchGrid'
import { OrganizationHierarchy } from './OrganizationHierarchy'

type Props = {
  title: string
  scopeLabel: string
  isSuperAdmin: boolean
  userName: string
  rolesLabel: string
  branches: Branch[]
  roles: string[]
  users: User[]
}

export function OrganizationStructure({ title, scopeLabel, isSuperAdmin, userName, rolesLabel, branches, roles, users }: Props) {
  const [selectedBranch, setSelectedBranch] = useState<Branch | null>(null)

  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <div className="flex items-center justify-between gap-3 border-b border-slate-100 pb-4">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-slate-500">Structure</p>
          <h3 className="mt-1 text-xl font-semibold text-slate-900">{title}</h3>
        </div>
        <span className="inline-flex items-center rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-700">{scopeLabel}</span>
      </div>

      <div className="mt-6 space-y-4">
        <OrgRootNode isSuperAdmin={isSuperAdmin} userName={userName} rolesLabel={rolesLabel} />
        {selectedBranch ? (
          <OrganizationHierarchy
            branch={selectedBranch}
            users={users.filter((user) => user.branch_ids?.includes(selectedBranch.id))}
            onBack={() => setSelectedBranch(null)}
          />
        ) : (
          <BranchGrid branches={branches} roles={roles} onBranchSelect={setSelectedBranch} />
        )}
      </div>
    </div>
  )
}