import type { Branch, User } from '../../types/auth'

type Props = {
  branch: Branch
  users: User[]
  onBack: () => void
}

const roleRank = (role: string) => {
  if (role === 'SUPER_ADMIN' || role === 'ADMIN') return 0
  if (role === 'MANAGER') return 1
  return 2
}

function displayRole(user: User) {
  return user.roles?.[0] ?? 'USER'
}

export function OrganizationHierarchy({ branch, users, onBack }: Props) {
  const groups = [0, 1, 2].map((rank) => ({
    rank,
    users: users.filter((user) => roleRank(displayRole(user)) === rank),
  })).filter((group) => group.users.length > 0)

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-xs font-medium uppercase tracking-[0.2em] text-indigo-600">Branch hierarchy</p>
          <h4 className="mt-1 text-2xl font-semibold text-slate-900">{branch.name}</h4>
          <p className="mt-1 text-sm text-slate-500">{branch.code}</p>
        </div>
        <button type="button" onClick={onBack} className="rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50">Back to branches</button>
      </div>

      {groups.length === 0 ? (
        <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 p-8 text-center text-sm text-slate-500">No users are assigned to this branch.</div>
      ) : (
        <div className="relative overflow-x-auto rounded-xl border border-slate-200 bg-slate-50 p-6">
          <div className="absolute bottom-10 left-1/2 top-10 hidden w-px -translate-x-1/2 bg-indigo-200 md:block" />
          <div className="relative space-y-8">
            {groups.map((group, index) => (
              <div key={group.rank} className="relative">
                {index > 0 ? <div className="absolute -top-8 left-1/2 hidden h-8 w-px -translate-x-1/2 bg-indigo-200 md:block" /> : null}
                <div className="mb-3 text-center text-xs font-medium uppercase tracking-[0.18em] text-slate-500">{group.rank === 0 ? 'Branch administration' : group.rank === 1 ? 'Management' : 'Team members'}</div>
                <div className="flex min-w-max justify-center gap-4">
                  {group.users.map((user) => (
                    <div key={user.id} className="relative w-56 rounded-xl border border-slate-200 bg-white p-4 text-center shadow-sm">
                      <div className="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-indigo-100 text-sm font-semibold text-indigo-700">{(user.name ?? user.email ?? 'U').slice(0, 1).toUpperCase()}</div>
                      <p className="mt-3 truncate text-sm font-semibold text-slate-900">{user.name ?? user.email}</p>
                      <p className="mt-1 text-xs font-medium text-indigo-600">{displayRole(user)}</p>
                      <p className="mt-2 truncate text-xs text-slate-500">{user.email}</p>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}