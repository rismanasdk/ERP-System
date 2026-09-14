import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { useAuth } from '../hooks/useAuth'
import { rolesApi } from '../services/roles'
import { readStoredAccessToken } from '../services/authSession'
import { ApiError } from '../lib/api'
import { CreateIcon, DeleteIcon, EditIcon, SearchIcon, ViewIcon } from '../utils/iconsUtils'
import { useConfirm } from '../utils/confirmUtils'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../components/ui/dialog'
import type { Permission, Role } from '../types/auth'

type Payload = { name: string; description: string; permissions: string[] }
const groupPermissions = (items: Permission[]) => items.reduce<Record<string, Permission[]>>((groups, item) => { const group = item.name?.includes('.') ? item.name.split('.')[0] : 'Other'; (groups[group] ??= []).push(item); return groups }, {})

export function RolesPage() {
  const { user } = useAuth()
  const confirm = useConfirm()
  const token = readStoredAccessToken() ?? undefined
  const canRead = Boolean(user?.permissions?.includes('roles.read'))
  const canCreate = Boolean(user?.permissions?.includes('roles.create'))
  const canUpdate = Boolean(user?.permissions?.includes('roles.update'))
  const canDelete = Boolean(user?.permissions?.includes('roles.delete'))
  const [roles, setRoles] = useState<Role[]>([])
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [permissionError, setPermissionError] = useState<string | null>(null)
  const [editing, setEditing] = useState<Role | null>(null)
  const [creating, setCreating] = useState(false)
  const [viewing, setViewing] = useState<Role | null>(null)
  const [saving, setSaving] = useState(false)
  const load = useCallback(async () => { setLoading(true); setError(null); try { setRoles(await rolesApi.all(token)) } catch (err) { setError(err instanceof ApiError ? err.message : 'Unable to load roles') } finally { setLoading(false) } }, [token])
  useEffect(() => {
    if (!canRead) return
    const task = window.setTimeout(() => void load(), 0)
    return () => window.clearTimeout(task)
  }, [canRead, load])
  const loadPermissions = async () => { if (permissions.length || permissionError) return; try { setPermissions(await rolesApi.permissions(token)) } catch (err) { setPermissionError(err instanceof ApiError ? err.message : 'Unable to load permissions') } }
  const visible = useMemo(() => roles.filter((role) => (role.name ?? '').toLowerCase().includes(query.toLowerCase())), [roles, query])
  const save = async (payload: Payload) => { setSaving(true); try { if (editing?.id) await rolesApi.update(editing.id, payload, token); else await rolesApi.create(payload, token); setEditing(null); setCreating(false); await load() } catch (err) { setError(err instanceof ApiError ? err.message : 'Unable to save role') } finally { setSaving(false) } }
  const remove = async (role: Role) => { if (!role.id || !(await confirm({ title: 'Delete this role?', description: 'Roles assigned to users and SUPER_ADMIN cannot be deleted.', confirmLabel: 'Delete', variant: 'destructive' }))) return; setSaving(true); try { await rolesApi.remove(role.id, token); await load() } catch (err) { setError(err instanceof ApiError ? err.message : 'Unable to delete role') } finally { setSaving(false) } }
  if (!canRead) return <div className="rounded-2xl border border-amber-200 bg-amber-50 p-6 text-amber-800 shadow-sm">You do not have permission to view roles.</div>
  return <div className="space-y-6">
    <div className="flex items-center justify-between rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <div>
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">ROLES</p>
        <h2 className="mt-1 text-2xl font-bold text-slate-900">Roles</h2>
        <p className="mt-2 text-sm text-slate-600">Manage roles and permissions.</p>
      </div>
      {canCreate && <button type="button" onClick={() => { setCreating(true); setPermissionError(null); void loadPermissions() }} className="inline-flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white"><CreateIcon className="h-4 w-4" />Create</button>}
    </div>
    {error && <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-red-800">{error}</div>}
    <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
      <div className="flex gap-3">
        <div className="relative max-w-sm flex-1">
          <SearchIcon className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <input aria-label="Search roles" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search roles" className="w-full rounded-md border border-slate-200 py-2 pl-9 pr-3 text-sm" />
        </div>
        <button type="button" onClick={() => void load()} className="rounded-md border border-slate-300 px-4 py-2 text-sm">Refresh</button>
      </div>
      <div className="mt-4 overflow-auto">
        {loading ? <div className="p-6 text-sm text-slate-500">Loading roles...</div> : visible.length === 0 ? <div className="p-6 text-center text-sm text-slate-500">{query ? 'No roles match your search.' : 'No roles found.'}</div> : <table className="min-w-full">
          <thead><tr className="border-b border-slate-100 text-left text-xs uppercase text-slate-500"><th className="px-3 py-3">Name</th><th className="px-3 py-3">Description</th><th className="px-3 py-3">Users</th><th className="px-3 py-3">Permissions</th><th className="px-3 py-3 text-right">Actions</th></tr></thead>
          <tbody className="divide-y divide-slate-100">
            {visible.map((role) => { const protectedRole = role.name === 'SUPER_ADMIN'; return <tr key={role.id ?? role.name}>
              <td className="px-3 py-3 text-sm font-medium">{role.name}{protectedRole && <span className="ml-2 rounded-full bg-slate-900 px-2 py-0.5 text-xs text-white">Protected</span>}</td>
              <td className="px-3 py-3 text-sm">{role.description ?? '-'}</td>
              <td className="px-3 py-3 text-sm">{role.user_count ?? 0}</td>
              <td className="px-3 py-3 text-sm">{role.permissions?.length ?? 0}</td>
              <td className="px-3 py-3 text-right">
                <button type="button" onClick={() => setViewing(role)} className="mr-2 inline-flex items-center gap-1 rounded-md border px-2 py-1 text-xs"><ViewIcon className="h-3 w-3" />View</button>
                {canUpdate && <button type="button" onClick={() => { setEditing(role); setPermissionError(null); void loadPermissions() }} className="mr-2 inline-flex items-center gap-1 rounded-md border px-2 py-1 text-xs"><EditIcon className="h-3 w-3" />Edit</button>}
                {canDelete && !protectedRole && <button type="button" disabled={saving} onClick={() => void remove(role)} className="inline-flex items-center gap-1 rounded-md border border-red-200 px-2 py-1 text-xs text-red-600"><DeleteIcon className="h-3 w-3" />Delete</button>}
              </td>
            </tr> })}
          </tbody>
        </table>}
      </div>
    </div>
    <Dialog open={Boolean(viewing)} onOpenChange={(open) => !open && setViewing(null)}>
      <DialogContent className="max-h-[85vh] overflow-y-auto">
        <DialogHeader><DialogTitle>Role Details</DialogTitle><DialogDescription>Assigned users and permissions.</DialogDescription></DialogHeader>
        {viewing && <RoleDetails role={viewing} />}
      </DialogContent>
    </Dialog>
    <Dialog open={creating || Boolean(editing)} onOpenChange={(open) => { if (!open) { setCreating(false); setEditing(null) } }}>
      <DialogContent className="flex max-h-[85vh] max-w-2xl flex-col overflow-hidden">
        <DialogHeader><DialogTitle>{creating ? 'Create Role' : 'Edit Role'}</DialogTitle><DialogDescription>Set role details and permissions.</DialogDescription></DialogHeader>
        <RoleForm initial={editing ?? undefined} permissions={permissions} permissionError={permissionError} saving={saving} onSubmit={save} onCancel={() => { setCreating(false); setEditing(null) }} />
      </DialogContent>
    </Dialog>
  </div>
}

function RoleDetails({ role }: { role: Role }) {
  const groups = groupPermissions((role.permissions ?? []).map((name, id) => ({ id, name })))
  return <div className="space-y-3 text-sm">
    <p><span className="text-slate-500">Role:</span> {role.name}</p>
    <p><span className="text-slate-500">Description:</span> {role.description ?? '-'}</p>
    <p><span className="text-slate-500">Users:</span> {role.user_count ?? 0}</p>
    {Object.entries(groups).map(([group, items]) => <div key={group} className="rounded-lg border p-3">
      <p className="font-medium capitalize">{group}</p>
      <div className="mt-1 grid grid-cols-2 gap-x-4 gap-y-1 sm:grid-cols-3">
        {items.map((item) => <p key={item.name} className="text-slate-600">✓ {item.name}</p>)}
      </div>
    </div>)}
  </div>
}

function RoleForm({ initial, permissions, permissionError, saving, onSubmit, onCancel }: { initial?: Role; permissions: Permission[]; permissionError: string | null; saving: boolean; onSubmit: (payload: Payload) => Promise<void>; onCancel: () => void }) {
  const [name, setName] = useState(initial?.name ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [selected, setSelected] = useState(initial?.permissions ?? [])
  const [permQuery, setPermQuery] = useState('')
  const [formError, setFormError] = useState<string | null>(null)
  const groups = useMemo(() => {
    const q = permQuery.trim().toLowerCase()
    const filtered = q ? permissions.filter((p) => p.name?.toLowerCase().includes(q)) : permissions
    return groupPermissions(filtered)
  }, [permissions, permQuery])
  const toggle = (item: string) => setSelected((current) => current.includes(item) ? current.filter((value) => value !== item) : [...current, item])
  const submit = async (event: FormEvent) => { event.preventDefault(); if (!name.trim()) { setFormError('Role name is required.'); return }; await onSubmit({ name: name.trim(), description: description.trim(), permissions: selected }) }

  return <form onSubmit={(event) => void submit(event)} className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden">
    {/* fixed top section */}
    <div className="shrink-0 space-y-3">
      <input aria-label="Role Name" required value={name} onChange={(event) => setName(event.target.value)} placeholder="Role Name" className="w-full rounded-md border p-2 text-sm" />
      <textarea aria-label="Description" value={description} onChange={(event) => setDescription(event.target.value)} placeholder="Description" rows={2} className="w-full rounded-md border p-2 text-sm" />
      {permissions.length > 0 && (
        <div className="flex items-center justify-between gap-3">
          <div className="relative flex-1">
            <SearchIcon className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
            <input aria-label="Filter permissions" value={permQuery} onChange={(event) => setPermQuery(event.target.value)} placeholder="Filter permissions..." className="w-full rounded-md border border-slate-200 py-2 pl-9 pr-3 text-sm" />
          </div>
          <span className="shrink-0 text-xs text-slate-500">{selected.length} selected</span>
        </div>
      )}
      {permissionError && <p className="text-sm text-red-600">{permissionError}</p>}
    </div>

    {/* scrollable permissions section — this is the part that was overflowing the page */}
    <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
      {permissions.length === 0
        ? <p className="text-sm text-slate-500">Loading permissions...</p>
        : Object.keys(groups).length === 0
          ? <p className="text-sm text-slate-500">No permissions match your filter.</p>
          : Object.entries(groups).map(([group, items]) => <div key={group} className="rounded-lg border p-3">
              <div className="flex items-center justify-between">
                <b className="text-sm capitalize">{group}</b>
                <button type="button" onClick={() => items.forEach((item) => item.name && toggle(item.name))} className="text-xs text-indigo-600">Select all</button>
              </div>
              <div className="mt-2 grid grid-cols-2 gap-x-4 gap-y-2 sm:grid-cols-3">
                {items.map((item) => <label key={item.name} className="inline-flex items-center gap-2 text-sm">
                  <input type="checkbox" checked={Boolean(item.name && selected.includes(item.name))} onChange={() => item.name && toggle(item.name)} />
                  {item.name}
                </label>)}
              </div>
            </div>)}
    </div>

    {formError && <p className="shrink-0 text-sm text-red-600">{formError}</p>}

    {/* fixed bottom section */}
    <DialogFooter className="shrink-0">
      <button type="button" onClick={onCancel} className="rounded-md border px-3 py-2 text-sm">Cancel</button>
      <button type="submit" disabled={saving || permissions.length === 0} className="rounded-md bg-indigo-600 px-3 py-2 text-sm text-white">{saving ? 'Saving...' : 'Save'}</button>
    </DialogFooter>
  </form>
}

export default RolesPage