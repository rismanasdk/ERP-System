import { useCallback, useEffect, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import { useBranch } from '../contexts/BranchContext'
import { OrgSummaryCards } from '../components/organization/OrgSummaryCards'
import { OrganizationStructure } from '../components/organization/OrganizationStructure'
import { usersApi } from '../services/users'
import { readStoredAccessToken } from '../services/authSession'
import type { User } from '../types/auth'

export function OrganizationPage() {
  const { user } = useAuth()
  const { accessibleBranches, selectedBranch, isAllBranches, loading, error } = useBranch()

  const roles = user?.roles && user.roles.length > 0 ? user.roles : ['USER']
  const isSuperAdmin = Boolean(user?.roles?.includes('SUPER_ADMIN'))
  const contextLabel = isAllBranches ? 'All branches' : selectedBranch?.name ?? 'Selected branch'
  const userName = user?.name ?? user?.email ?? 'User'
  const [users, setUsers] = useState<User[]>(user ? [user] : [])
  const token = readStoredAccessToken() ?? undefined
  const canReadUsers = Boolean(user?.permissions?.includes('users.read'))

  const loadUsers = useCallback(async () => {
    if (!canReadUsers) {
      setUsers(user ? [user] : [])
      return
    }
    try {
      const result = await usersApi.list(token)
      setUsers(Array.isArray(result) && result.length > 0 ? result : (user ? [user] : []))
    } catch {
      setUsers(user ? [user] : [])
    }
  }, [canReadUsers, token, user])

  useEffect(() => {
    queueMicrotask(() => void loadUsers())
  }, [loadUsers])

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Organization</p>
        <h2 className="mt-3 text-3xl font-bold text-slate-900">Organization</h2>
        <div className="mt-3 flex flex-wrap items-center gap-3 text-sm text-slate-600">
          <span>{userName}</span>
          <span className="text-slate-300">•</span>
          <span>{contextLabel}</span>
        </div>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-red-800 shadow-sm">
          <p>{error}</p>
        </div>
      )}

      <OrgSummaryCards
        userName={userName}
        userEmail={user?.email}
        roles={roles}
        contextLabel={contextLabel}
        branchCount={accessibleBranches.length}
        loading={loading}
      />

      <OrganizationStructure
        title={contextLabel}
        scopeLabel={isAllBranches ? 'Global view' : 'Branch-scoped view'}
        isSuperAdmin={isSuperAdmin}
        userName={userName}
        rolesLabel={roles.join(', ')}
        branches={accessibleBranches}
        roles={roles}
        users={users}
      />
    </div>
  )
}

export default OrganizationPage