import { useMemo } from 'react'
import { useAuth } from '../hooks/useAuth'
import { useBranch } from '../contexts/BranchContext'

export function OrganizationPage() {
  const { user } = useAuth()
  const { accessibleBranches, selectedBranch, isAllBranches, loading, error } = useBranch()

  const roles = useMemo(() => {
    if (!user?.roles || user.roles.length === 0) return ['USER']
    return user.roles
  }, [user?.roles])

  const contextLabel = isAllBranches ? 'All branches' : selectedBranch?.name ?? 'Selected branch'

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Organization</p>
        <h2 className="mt-3 text-3xl font-bold text-slate-900">Organization</h2>
        <div className="mt-3 flex flex-wrap items-center gap-3 text-sm text-slate-600">
          <span>{user?.name ?? user?.email ?? 'User'}</span>
          <span className="text-slate-300">•</span>
          <span>{contextLabel}</span>
        </div>
      </div>

      {error ? (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-red-800 shadow-sm">
          <p>{error}</p>
        </div>
      ) : null}

      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-xs font-medium uppercase tracking-[0.2em] text-slate-500">Current user</p>
          <p className="mt-3 text-lg font-semibold text-slate-900">{user?.name ?? user?.email ?? 'User'}</p>
          <p className="mt-1 text-sm text-slate-600">{user?.email ?? 'No email available'}</p>
        </div>

        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-xs font-medium uppercase tracking-[0.2em] text-slate-500">Role</p>
          <div className="mt-3 flex flex-wrap gap-2">
            {roles.map((role) => (
              <span key={role} className="inline-flex rounded-full bg-indigo-100 px-2.5 py-1 text-xs font-medium text-indigo-700">
                {role}
              </span>
            ))}
          </div>
        </div>

        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-xs font-medium uppercase tracking-[0.2em] text-slate-500">Context</p>
          <p className="mt-3 text-lg font-semibold text-slate-900">{contextLabel}</p>
          <p className="mt-1 text-sm text-slate-600">{loading ? 'Loading branch access...' : `${accessibleBranches.length} branch(es) available`}</p>
        </div>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex items-center justify-between gap-3 pb-4 border-b border-slate-100">
          <div>
            <p className="text-sm font-medium uppercase tracking-[0.2em] text-slate-500">Structure</p>
            <h3 className="mt-1 text-xl font-semibold text-slate-900">{isAllBranches ? 'All branches' : (selectedBranch?.name ?? 'Selected branch')}</h3>
          </div>
          <span className="inline-flex items-center rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-700">
            {isAllBranches ? 'Global view' : 'Branch-scoped view'}
          </span>
        </div>

        <div className="mt-6 space-y-4">
          <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Root</p>
                <p className="mt-1 text-lg font-semibold text-slate-900">{user?.roles?.includes('SUPER_ADMIN') ? 'SUPER ADMIN' : 'Organization'}</p>
              </div>
              <span className="rounded-full bg-indigo-100 px-2.5 py-1 text-xs font-medium text-indigo-700">
                {user?.roles?.includes('SUPER_ADMIN') ? 'Administrative access' : 'Standard access'}
              </span>
            </div>

            <div className="mt-4 rounded-xl border border-slate-200 bg-white p-4">
              <p className="text-sm font-medium text-slate-700">Current user</p>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <span className="text-base font-semibold text-slate-900">{user?.name ?? user?.email ?? 'User'}</span>
                <span className="text-slate-400">•</span>
                <span className="text-sm text-slate-600">{roles.join(', ')}</span>
              </div>
            </div>
          </div>

          {accessibleBranches.length === 0 ? (
            <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 p-8 text-center text-sm text-slate-500">
              No branch access is available for this user.
            </div>
          ) : (
            <div className="grid gap-4 lg:grid-cols-2">
              {accessibleBranches.map((branch) => (
                <div key={branch.id} className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Branch</p>
                      <h4 className="mt-1 text-lg font-semibold text-slate-900">{branch.name}</h4>
                    </div>
                    <span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${branch.is_active === false ? 'bg-slate-100 text-slate-700' : 'bg-emerald-100 text-emerald-700'}`}>
                      {branch.is_active === false ? 'Inactive' : 'Active'}
                    </span>
                  </div>

                  <div className="mt-4 rounded-lg border border-slate-200 bg-slate-50 p-3">
                    <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Assigned role(s)</p>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {roles.map((role) => (
                        <span key={`${branch.id}-${role}`} className="inline-flex rounded-full bg-white px-2.5 py-1 text-xs font-medium text-slate-700 border border-slate-200">
                          {role}
                        </span>
                      ))}
                    </div>
                  </div>

                  <div className="mt-4 text-sm text-slate-600">
                    <p className="font-medium text-slate-700">Branch access</p>
                    <p className="mt-1">{branch.code} • {branch.name}</p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
