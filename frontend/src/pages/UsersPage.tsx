import { useCallback, useEffect, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import { useBranch } from '../contexts/BranchContext'
import { usersApi } from '../services/users'
import { rolesApi } from '../services/roles'
import { readStoredAccessToken } from '../services/authSession'
import { ApiError } from '../lib/api'
import { EditIcon, DeleteIcon, CreateIcon, CloseIcon, SearchIcon, ViewIcon } from '../utils/iconsUtils'
import { useConfirm } from '../utils/confirmUtils'
import { Dialog, DialogContent } from '../components/ui/dialog'

type UserRow = {
  id: number
  name: string
  email: string
  roles: string[]
  permissions?: string[]
  branch_names?: string[]
  is_active?: boolean
}

type ApiUser = {
  id: number
  name?: string
  email: string
  roles?: string[]
  RoleNames?: string[]
  permissions?: string[]
  branch_ids?: number[]
  BranchIDs?: number[]
  is_active?: boolean
}

function RoleInput({ value, onChange, available, disabled, fetchError }: { value: string[]; onChange: (v: string[]) => void; available: string[]; disabled: boolean; fetchError?: string | null }) {
  if (disabled) {
    return (
      <div className="text-sm text-slate-500">
        {fetchError ? (
          <div className="text-sm text-red-600">{fetchError}</div>
        ) : (
          <div>No roles available from backend. Create roles in backend database or enable roles API.</div>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-2">
      <div className="grid grid-cols-2 gap-2 max-h-40 overflow-auto">
        {available.map((r) => (
          <label key={r} className="inline-flex items-center gap-2 text-sm">
            <input type="checkbox" checked={value.includes(r)} onChange={() => onChange(value.includes(r) ? value.filter((x) => x !== r) : [...value, r])} />
            <span>{r}</span>
          </label>
        ))}
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        {value.map((r) => (
          <span key={r} className="inline-flex items-center gap-2 rounded-full bg-indigo-100 px-2.5 py-1 text-xs font-medium text-indigo-700">{r}</span>
        ))}
      </div>
    </div>
  )
}

export function UsersPage() {
  const { user } = useAuth()
  const { accessibleBranches } = useBranch()
  const confirm = useConfirm()

  const [rows, setRows] = useState<UserRow[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<UserRow | null>(null)
  const [viewingFor, setViewingFor] = useState<UserRow | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const token = readStoredAccessToken() ?? undefined

  const load = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const res = await usersApi.list(token)
      const mapped: UserRow[] = (res as unknown as ApiUser[]).map((u) => {
        const branchIDs: number[] = Array.isArray(u.branch_ids) ? u.branch_ids : Array.isArray(u.BranchIDs) ? u.BranchIDs : []
        const names = branchIDs.map((id) => {
          const found = accessibleBranches.find((b) => b.id === id)
          return found ? found.name : String(id)
        })
        return {
          id: u.id,
          name: u.name ?? u.email,
          email: u.email,
          roles: u.roles ?? u.RoleNames ?? [],
          permissions: u.permissions,
          branch_names: names,
          is_active: u.is_active,
        } as UserRow
      })
      setRows(mapped)
    } catch (err) {
      const e = err as ApiError
      if (e instanceof ApiError) {
        if (e.status === 401) return setError('Session expired. Please sign in again.')
        if (e.status === 403) return setError('You do not have access to users.')
        return setError(e.message)
      }
      setError('Unable to load users')
    } finally {
      setIsLoading(false)
    }
  }, [token, accessibleBranches])

  useEffect(() => {
    let active = true
    const run = async () => {
      if (!active) return
      await load()
    }
    void run()
    return () => {
      active = false
    }
  }, [load])

  const onCreate = useCallback(async (payload: Partial<UserRow> & { password?: string; branch_ids?: number[] }) => {
    setSubmitting(true)
    try {
      await usersApi.create(payload, token)
      setCreating(false)
      await load()
    } catch (err) {
      const e = err as ApiError
      throw e
    } finally {
      setSubmitting(false)
    }
  }, [load, token])

  const onUpdate = useCallback(async (payload: Partial<UserRow> & { password?: string; branch_ids?: number[] }) => {
    if (!editing) return
    setSubmitting(true)
    try {
      await usersApi.update(editing.id, payload, token)
      setEditing(null)
      await load()
    } catch (err) {
      const e = err as ApiError
      throw e
    } finally {
      setSubmitting(false)
    }
  }, [editing, load, token])

  const onDelete = useCallback(async (id: number) => {
    const ok = await confirm({ title: 'Delete user?', description: 'This action may deactivate or remove the user. Proceed?', confirmLabel: 'Delete', variant: 'destructive' })
    if (!ok) return
    setSubmitting(true)
    try {
      await usersApi.remove(id, token)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    } finally {
      setSubmitting(false)
    }
  }, [confirm, load, token])

  const isSuperAdmin = Boolean(user?.roles?.includes('SUPER_ADMIN'))
  const canRead = isSuperAdmin || Boolean(user?.permissions?.includes('users.read'))
  const canCreate = isSuperAdmin || Boolean(user?.permissions?.includes('users.create'))
  const canUpdate = isSuperAdmin || Boolean(user?.permissions?.includes('users.update'))
  const canDelete = isSuperAdmin || Boolean(user?.permissions?.includes('users.delete'))

  if (!canRead) {
    return (
      <div className="rounded-2xl border border-amber-200 bg-amber-50 p-6 text-amber-800 shadow-sm">
        You do not have permission to view user management.
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Users</p>
          <h2 className="mt-1 text-2xl font-bold text-slate-900">User Management</h2>
          <p className="mt-2 text-sm text-slate-600">Manage user accounts, roles and branch access.</p>
        </div>
        <div className="flex items-center gap-3">
          {canCreate ? (
            <button onClick={() => setCreating(true)} className="inline-flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500 transition-colors">
              <CreateIcon className="h-4 w-4" />
              Create
            </button>
          ) : null}
        </div>
      </div>

      {error ? (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-red-800 shadow-sm">
          <p>{error}</p>
          <div className="mt-3">
            <button onClick={() => void load()} className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors">Retry</button>
          </div>
        </div>
      ) : null}

      <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="flex items-center gap-3">
          <div className="relative flex-1 max-w-sm">
            <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
            <input placeholder="Search users" className="w-full rounded-md border border-slate-200 pl-9 pr-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" />
          </div>
          <button onClick={() => void load()} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm hover:bg-slate-50 transition-colors">Refresh</button>
        </div>

        <div className="mt-4 overflow-auto">
          {isLoading ? (
            <div className="space-y-2">
              <div className="h-8 w-1/3 rounded bg-slate-200" />
              <div className="h-8 w-1/2 rounded bg-slate-200" />
            </div>
          ) : rows.length === 0 ? (
            <div className="p-6 text-center text-slate-500">No users found.</div>
          ) : (
            <table className="min-w-full table-auto">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                  <th className="px-3 py-3">Name</th>
                  <th className="px-3 py-3">Email</th>
                  <th className="px-3 py-3">Role(s)</th>
                  <th className="px-3 py-3">Branch access</th>
                  <th className="px-3 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {rows.map((r) => (
                  <tr key={r.id} className="hover:bg-slate-50 transition-colors">
                    <td className="px-3 py-3 text-sm font-medium text-slate-900">{r.name}</td>
                    <td className="px-3 py-3 text-sm text-slate-700">{r.email}</td>
                    <td className="px-3 py-3 text-sm text-slate-700">
                      <div className="flex flex-wrap gap-2">
                        {r.roles.map((role) => (
                          <span key={`${r.id}-${role}`} className="inline-flex rounded-full bg-indigo-100 px-2.5 py-1 text-xs font-medium text-indigo-700">{role}</span>
                        ))}
                      </div>
                    </td>
                    <td className="px-3 py-3 text-sm text-slate-700">
                      <div className="flex flex-wrap gap-2">
                        {r.roles.includes('SUPER_ADMIN') === true ? (
                          <span className="inline-flex rounded-full bg-black px-2.5 py-1 text-xs font-medium text-white">ALL BRANCHES</span>
                        ) : (r.branch_names ?? []).length === 0 ? (
                          <span className="text-slate-400">No branches</span>
                        ) : (
                          (r.branch_names ?? []).map((b) => (
                            <span key={`${r.id}-${b}`} className="inline-flex rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-medium text-emerald-700">{b}</span>
                          ))
                        )}
                      </div>
                    </td>
                    <td className="px-3 py-3 text-sm text-right">
                      <div className="inline-flex items-center gap-2">
                        {canRead ? (
                          <button onClick={() => setViewingFor(r)} className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors">
                            <ViewIcon className="h-3.5 w-3.5" />
                            View
                          </button>
                        ) : null}
                        {canUpdate ? (
                          <button onClick={() => setEditing(r)} className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors">
                            <EditIcon className="h-3.5 w-3.5" />
                            Edit
                          </button>
                        ) : null}
                        {canDelete ? (
                          <button onClick={() => void onDelete(r.id)} className="inline-flex items-center gap-1.5 rounded-md border border-red-200 bg-white px-2.5 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 transition-colors">
                            <DeleteIcon className="h-3.5 w-3.5" />
                            Delete
                          </button>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
          {viewingFor && (
            <Dialog open={Boolean(viewingFor)} onOpenChange={(open) => { if (!open) setViewingFor(null) }}>
              <DialogContent>
                <div className="mb-4">
                  <h3 className="text-lg font-semibold text-slate-900">User detail</h3>
                </div>
                <div className="space-y-3">
                  <div>
                    <div className="text-sm text-slate-500">Name</div>
                    <div className="text-sm font-medium">{viewingFor.name}</div>
                  </div>
                  <div>
                    <div className="text-sm text-slate-500">Email</div>
                    <div className="text-sm font-medium">{viewingFor.email}</div>
                  </div>
                  <div>
                    <div className="text-sm text-slate-500">Role(s)</div>
                    <div className="mt-1 flex flex-wrap gap-2">
                      {viewingFor.roles.length === 0 ? (
                        <span className="text-sm text-slate-400">No roles</span>
                      ) : (
                        viewingFor.roles.map((role) => (
                          <span key={role} className="inline-flex rounded-full bg-indigo-100 px-2.5 py-1 text-xs font-medium text-indigo-700">{role}</span>
                        ))
                      )}
                    </div>
                  </div>
                  <div>
                    <div className="text-sm text-slate-500">Branch access</div>
                    <div className="mt-1 flex flex-wrap gap-2">
                      {viewingFor.roles.includes('SUPER_ADMIN') ? (
                        <span className="inline-flex rounded-full bg-black px-2.5 py-1 text-xs font-medium text-white">ALL BRANCHES</span>
                      ) : (viewingFor.branch_names ?? []).length === 0 ? (
                        <span className="text-sm text-slate-400">No branches</span>
                      ) : (
                        (viewingFor.branch_names ?? []).map((b) => (
                          <span key={b} className="inline-flex rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-medium text-emerald-700">{b}</span>
                        ))
                      )}
                    </div>
                  </div>
                  {/* Status removed: backend does not provide user active status */}
                </div>
              </DialogContent>
            </Dialog>
          )}
      </div>

      {/* Create/Edit modal */}
      {(creating || editing) && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={() => { setCreating(false); setEditing(null) }} />
          <div className="relative w-full max-w-lg rounded-2xl border border-slate-200 bg-white p-6 shadow-2xl">
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-lg font-semibold text-slate-900">{creating ? 'Create User' : 'User'}</h3>
              <button onClick={() => { setCreating(false); setEditing(null) }} className="rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-colors"><CloseIcon className="h-5 w-5" /></button>
            </div>

            <UserForm
              key={editing ? `edit-${editing.id}` : 'create'}
              initial={editing ?? undefined}
              submitting={submitting}
              onSubmit={creating ? onCreate : onUpdate}
              onCancel={() => { setCreating(false); setEditing(null) }}
              branches={accessibleBranches}
            />
          </div>
        </div>
      )}
    </div>
  )
}

function UserForm({ initial, submitting, onSubmit, onCancel, branches }: { initial?: UserRow & { branch_ids?: number[] }; submitting: boolean; onSubmit: (p: { name: string; email: string; password?: string; roles: string[]; branch_ids?: number[] }) => Promise<void>; onCancel: () => void; branches: { id: number; name: string }[] }) {
  const [name, setName] = useState(initial?.name ?? '')
  const [email, setEmail] = useState(initial?.email ?? '')
  const [password, setPassword] = useState('')
  const [roles, setRoles] = useState<string[]>(initial?.roles ?? [])
  const [branchIds, setBranchIds] = useState<number[]>([])
  const [error, setError] = useState<string | null>(null)
  const [availableRoles, setAvailableRoles] = useState<string[]>([])
  const [rolesFetchError, setRolesFetchError] = useState<string | null>(null)
  const [rolesDisabled, setRolesDisabled] = useState(true)

  useEffect(() => {
    if (!initial) return
    // Prefer explicit branch_ids from API when available
    const idsFromAPI: number[] = initial.branch_ids ?? []
    if (idsFromAPI && idsFromAPI.length > 0) {
      queueMicrotask(() => setBranchIds(idsFromAPI))
      return
    }
    const selected = branches.filter((b) => (initial.branch_names ?? []).includes(b.name)).map((b) => b.id)
    queueMicrotask(() => setBranchIds(selected))
  }, [initial, branches])

  useEffect(() => {
    let active = true
    const run = async () => {
      try {
        const list = await rolesApi.list()
        if (!active) return
        if (Array.isArray(list) && list.length > 0) {
          setAvailableRoles(list)
          setRolesDisabled(false)
          setRolesFetchError(null)
        } else {
          setAvailableRoles([])
          setRolesDisabled(true)
          setRolesFetchError('No roles exposed by backend')
        }
      } catch (err: unknown) {
        setAvailableRoles([])
        setRolesDisabled(true)
        const msg = err instanceof Error ? err.message : String(err)
        setRolesFetchError(msg || 'Failed to load roles from backend')
      }
    }
    void run()
    return () => {
      active = false
    }
  }, [])

  const submit = async (e?: React.FormEvent) => {
    e?.preventDefault()
    setError(null)
    try {
      await onSubmit({ name, email, password: password || undefined, roles, branch_ids: branchIds })
    } catch (err) {
      const e = err as ApiError
      setError(e.message ?? 'Unable to save user')
    }
  }

  const toggleBranch = (id: number) => {
    setBranchIds((s) => (s.includes(id) ? s.filter((x) => x !== id) : [...s, id]))
  }

  return (
    <form onSubmit={submit}>
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-slate-700">Name</label>
          <input value={name} onChange={(e) => setName(e.target.value)} className="mt-1 block w-full rounded-md border p-2" />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700">Email</label>
          <input value={email} onChange={(e) => setEmail(e.target.value)} className="mt-1 block w-full rounded-md border p-2" />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700">Password {initial ? '(leave blank to keep)' : ''}</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} className="mt-1 block w-full rounded-md border p-2" />
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700">Roles</label>
          <div className="mt-1">
            <RoleInput value={roles} onChange={setRoles} available={availableRoles} disabled={rolesDisabled} fetchError={rolesFetchError} />
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700">Branch access</label>
          <div className="mt-2 grid grid-cols-2 gap-2 max-h-40 overflow-auto">
            {branches.map((b) => (
              <label key={b.id} className="inline-flex items-center gap-2 text-sm">
                <input type="checkbox" checked={branchIds.includes(b.id)} onChange={() => toggleBranch(b.id)} />
                <span>{b.name}</span>
              </label>
            ))}
          </div>
        </div>
        {error ? <div className="text-sm text-red-600">{error}</div> : null}
        <div className="flex justify-end gap-3">
          <button type="button" onClick={onCancel} className="rounded-md border px-3 py-2">Cancel</button>
          <button type="submit" disabled={submitting} className="rounded-md bg-indigo-600 px-3 py-2 text-white">{submitting ? 'Saving...' : 'Save'}</button>
        </div>
      </div>
    </form>
  )
}
