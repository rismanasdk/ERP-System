import { useCallback, useEffect, useMemo, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import type { Branch } from '../types/auth'
import { branchesApi } from '../services/branches'
import { readStoredAccessToken } from '../services/authSession'
import { BranchForm } from '../components/branches/BranchForm'
import { ApiError } from '../lib/api'
import { CreateIcon, DeleteIcon, EditIcon, ViewIcon, CloseIcon } from '../utils/iconsUtils'
import { useConfirm } from '../utils/confirmUtils'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../components/ui/dialog'
import { formatDateTime } from '../utils/dateUtils'

export function BranchesPage() {
  const { user } = useAuth()
  const confirmDialog = useConfirm()
  const [branches, setBranches] = useState<Branch[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filterActive, setFilterActive] = useState('')
  const [editing, setEditing] = useState<Branch | null>(null)
  const [creating, setCreating] = useState(false)
  const [detailBranch, setDetailBranch] = useState<Branch | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const token = readStoredAccessToken() ?? undefined
  const rows = useMemo(() => branches, [branches])

  const isSuperAdmin = user?.roles?.includes('SUPER_ADMIN')
  const canRead = isSuperAdmin || user?.permissions?.includes('inventory.read')
  const canCreate = isSuperAdmin || user?.permissions?.includes('inventory.create')
  const canUpdate = isSuperAdmin || user?.permissions?.includes('inventory.adjust')

  const load = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const active = filterActive === '' ? undefined : filterActive === 'true'
      const res = await branchesApi.list(active, token)
      setBranches(res)
    } catch (err) {
      const e = err as ApiError
      if (e instanceof ApiError) {
        if (e.status === 401) return setError('Session expired. Please sign in again.')
        if (e.status === 403) return setError('You do not have access to branches.')
        return setError(e.message)
      }
      setError('Unable to load branches')
    } finally {
      setIsLoading(false)
    }
  }, [filterActive, token])

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

  const handleView = useCallback(async (id: number) => {
    try {
      const detail = await branchesApi.getById(id, token)
      setDetailBranch(detail)
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    }
  }, [token])

  const onCreate = useCallback(async (payload: Partial<Branch>) => {
    setSubmitting(true)
    try {
      await branchesApi.create(payload, token)
      setCreating(false)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
      return
    } finally {
      setSubmitting(false)
    }
  }, [load, token])

  const onUpdate = useCallback(async (payload: Partial<Branch>) => {
    if (!editing) return
    setSubmitting(true)
    try {
      await branchesApi.update(editing.id, payload, token)
      setEditing(null)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
      return
    } finally {
      setSubmitting(false)
    }
  }, [editing, load, token])

  const onDeactivate = useCallback(async (id: number) => {
    const branch = branches.find((item) => item.id === id)
    if (!branch) return

    const ok = await confirmDialog({
      title: 'Deactivate this branch?',
      description: 'This action will set the branch to inactive.',
      confirmLabel: 'Deactivate',
      variant: 'destructive',
    })
    if (!ok) return

    setSubmitting(true)
    try {
      await branchesApi.update(id, { name: branch.name, code: branch.code, is_active: false }, token)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    } finally {
      setSubmitting(false)
    }
  }, [branches, confirmDialog, load, token])

  if (!canRead) {
    return (
      <div className="rounded-2xl border border-amber-200 bg-amber-50 p-6 text-amber-800 shadow-sm">
        You do not have permission to view branches.
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">BRANCHES</p>
          <h2 className="mt-1 text-2xl font-bold text-slate-900">Branches</h2>
          <p className="mt-2 text-sm text-slate-600">Manage operational branches and access scope.</p>
        </div>
        <div className="flex items-center gap-3">
          {canCreate ? (
            <button
              type="button"
              onClick={() => setCreating(true)}
              className="inline-flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500 transition-colors"
            >
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
            <button type="button" onClick={() => void load()} className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors">
              Retry
            </button>
          </div>
        </div>
      ) : null}

      <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="flex flex-col gap-3 md:flex-row md:items-center">
          <select
            aria-label="Branch status filter"
            value={filterActive}
            onChange={(e) => setFilterActive(e.target.value)}
            className="rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          >
            <option value="">All</option>
            <option value="true">Active</option>
            <option value="false">Inactive</option>
          </select>

          <button
            type="button"
            onClick={() => void load()}
            className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm hover:bg-slate-50 transition-colors"
          >
            Apply
          </button>
        </div>

        <div className="mt-4 overflow-auto">
          {isLoading ? (
            <div className="space-y-2">
              <div className="h-8 w-1/3 rounded bg-slate-200" />
              <div className="h-8 w-1/2 rounded bg-slate-200" />
            </div>
          ) : rows.length === 0 ? (
            <div className="p-6 text-center text-sm text-slate-500">
              {filterActive ? 'No branches match the current filter.' : 'No branches found.'}
            </div>
          ) : (
            <table className="min-w-full table-auto">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                  <th className="px-3 py-3">Code</th>
                  <th className="px-3 py-3">Name</th>
                  <th className="px-3 py-3">Status</th>
                  <th className="px-3 py-3">Created</th>
                  <th className="px-3 py-3">Updated</th>
                  <th className="px-3 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {rows.map((branch) => (
                  <tr key={branch.id} className="hover:bg-slate-50 transition-colors">
                    <td className="px-3 py-3 text-sm font-medium text-slate-900">{branch.code}</td>
                    <td className="px-3 py-3 text-sm text-slate-700">{branch.name}</td>
                    <td className="px-3 py-3 text-sm">
                      <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${branch.is_active ? 'bg-green-100 text-green-800' : 'bg-slate-100 text-slate-700'}`}>
                        {branch.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="px-3 py-3 text-sm text-slate-500">{formatDateTime(branch.created_at)}</td>
                    <td className="px-3 py-3 text-sm text-slate-500">{formatDateTime(branch.updated_at)}</td>
                    <td className="px-3 py-3 text-right">
                      <div className="inline-flex items-center gap-2">
                        {canRead ? (
                          <button
                            type="button"
                            onClick={() => void handleView(branch.id)}
                            className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors"
                          >
                            <ViewIcon className="h-3.5 w-3.5" />
                            View
                          </button>
                        ) : null}
                        {canUpdate ? (
                          <button
                            type="button"
                            onClick={() => setEditing(branch)}
                            className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors"
                          >
                            <EditIcon className="h-3.5 w-3.5" />
                            Edit
                          </button>
                        ) : null}
                        {canUpdate && branch.is_active ? (
                          <button
                            type="button"
                            onClick={() => void onDeactivate(branch.id)}
                            className="inline-flex items-center gap-1.5 rounded-md border border-red-200 bg-white px-2.5 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 transition-colors"
                          >
                            <DeleteIcon className="h-3.5 w-3.5" />
                            Deactivate
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
      </div>

      <Dialog open={Boolean(detailBranch)} onOpenChange={(open) => !open && setDetailBranch(null)}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>Branch Details</DialogTitle>
            <DialogDescription>View branch operational details.</DialogDescription>
          </DialogHeader>

          {detailBranch ? (
            <div className="space-y-3 text-sm">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <p className="text-slate-500">Code</p>
                  <p className="font-medium text-slate-900">{detailBranch.code}</p>
                </div>
                <div>
                  <p className="text-slate-500">Status</p>
                  <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${detailBranch.is_active ? 'bg-green-100 text-green-800' : 'bg-slate-100 text-slate-700'}`}>
                    {detailBranch.is_active ? 'Active' : 'Inactive'}
                  </span>
                </div>
              </div>

              <div>
                <p className="text-slate-500">Name</p>
                <p className="font-medium text-slate-900">{detailBranch.name}</p>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <p className="text-slate-500">Created</p>
                  <p className="text-slate-900">{detailBranch.created_at ? new Date(detailBranch.created_at).toLocaleString('id-ID') : '-'}</p>
                </div>
                <div>
                  <p className="text-slate-500">Updated</p>
                  <p className="text-slate-900">{detailBranch.updated_at ? new Date(detailBranch.updated_at).toLocaleString('id-ID') : '-'}</p>
                </div>
              </div>
            </div>
          ) : null}

          <DialogFooter>
            <button
              type="button"
              onClick={() => setDetailBranch(null)}
              className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 transition-colors"
            >
              Close
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={creating || !!editing} onOpenChange={(open) => {
        if (!open) {
          setCreating(false)
          setEditing(null)
        }
      }}>
        <DialogContent className="max-w-xl">
          <DialogHeader className="flex items-center justify-between">
            <div>
              <DialogTitle>{creating ? 'Create Branch' : 'Edit Branch'}</DialogTitle>
              <DialogDescription>{creating ? 'Create a branch record.' : 'Update branch details.'}</DialogDescription>
            </div>
            <button
              type="button"
              onClick={() => {
                setCreating(false)
                setEditing(null)
              }}
              className="rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-colors"
            >
              <CloseIcon className="h-5 w-5" />
            </button>
          </DialogHeader>

          <BranchForm
            key={editing ? `edit-${editing.id}` : 'create'}
            initial={editing ?? undefined}
            submitting={submitting}
            onSubmit={creating ? onCreate : onUpdate}
            onCancel={() => {
              setCreating(false)
              setEditing(null)
            }}
          />
        </DialogContent>
      </Dialog>
    </div>
  )
}
