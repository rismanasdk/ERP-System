import { useCallback, useEffect, useMemo, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import type { Supplier, SupplierFilter } from '../types/supplier'
import { suppliersApi } from '../services/suppliers'
import { readStoredAccessToken } from '../services/authSession'
import { SupplierForm } from '../components/suppliers/SupplierForm'
import { ApiError } from '../lib/api'
// Import icon sesuai utils yang disediakan
import {EditIcon,DeleteIcon,CreateIcon,CloseIcon,SearchIcon } from '../utils/iconsUtils'

export function SuppliersPage() {
  const { user } = useAuth()
  const [suppliers, setSuppliers] = useState<Supplier[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState({ search: '', active: '' })
  const [editing, setEditing] = useState<Supplier | null>(null)
  const [creating, setCreating] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const token = readStoredAccessToken() ?? undefined

  const load = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const f: SupplierFilter = {}
      if (filter.search) f.search = filter.search
      if (filter.active !== '') f.active = filter.active === 'true'
      const res = await suppliersApi.list(f, token)
      setSuppliers(res)
    } catch (err) {
      const e = err as ApiError
      if (e instanceof ApiError) {
        if (e.status === 401) return setError('Session expired. Please sign in again.')
        if (e.status === 403) return setError('You do not have access to suppliers.')
        return setError(e.message)
      }
      setError('Unable to load suppliers')
    } finally {
      setIsLoading(false)
    }
  }, [filter, token])

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

  const onCreate = useCallback(async (payload: Partial<Supplier>) => {
    setSubmitting(true)
    try {
      await suppliersApi.create(payload, token)
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

  const onUpdate = useCallback(async (payload: Partial<Supplier>) => {
    if (!editing) return
    setSubmitting(true)
    try {
      await suppliersApi.update(editing.id, payload, token)
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

  const onDelete = useCallback(async (id: number) => {
    if (!confirm('Are you sure you want to delete this supplier?')) return
    setSubmitting(true)
    try {
      await suppliersApi.softDelete(id, token)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    } finally {
      setSubmitting(false)
    }
  }, [load, token])

  const isSuperAdmin = user?.roles?.includes('SUPER_ADMIN')
  const canCreate = isSuperAdmin || user?.permissions?.includes('suppliers.create')
  const canUpdate = isSuperAdmin || user?.permissions?.includes('suppliers.update')
  const canDelete = isSuperAdmin || user?.permissions?.includes('suppliers.delete')

  const rows = useMemo(() => suppliers, [suppliers])

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Suppliers</p>
          <h2 className="mt-1 text-2xl font-bold text-slate-900">Suppliers</h2>
          <p className="mt-2 text-sm text-slate-600">Manage supplier master data.</p>
        </div>
        <div className="flex items-center gap-3">
          {canCreate ? (
            <button 
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
            <button onClick={() => void load()} className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors">Retry</button>
          </div>
        </div>
      ) : null}

      <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="flex items-center gap-3">
          <div className="relative flex-1 max-w-sm">
            <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
            <input 
              value={filter.search} 
              onChange={(e) => setFilter((s) => ({ ...s, search: e.target.value }))} 
              placeholder="Search by code or name" 
              className="w-full rounded-md border border-slate-200 pl-9 pr-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
            />
          </div>
          <select 
            value={filter.active} 
            onChange={(e) => setFilter((s) => ({ ...s, active: e.target.value }))} 
            className="rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          >
            <option value="">All</option>
            <option value="true">Active</option>
            <option value="false">Inactive</option>
          </select>
          <button 
            onClick={() => void load()} 
            className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm hover:bg-slate-50 transition-colors"
          >
            Apply
          </button>
        </div>

        <div className="mt-4 overflow-auto">
          {/* SKELETON LOADING DIPERTAHANKAN PERSIS SEPERTI ASLINYA */}
          {isLoading ? (
            <div className="space-y-2">
              <div className="h-8 w-1/3 rounded bg-slate-200" />
              <div className="h-8 w-1/2 rounded bg-slate-200" />
            </div>
          ) : rows.length === 0 ? (
            <div className="p-6 text-center text-slate-500">No suppliers found.</div>
          ) : (
            <table className="min-w-full table-auto">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                  <th className="px-3 py-3">Code</th>
                  <th className="px-3 py-3">Name</th>
                  <th className="px-3 py-3">Phone</th>
                  <th className="px-3 py-3">Email</th>
                  <th className="px-3 py-3">Active</th>
                  <th className="px-3 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {rows.map((s) => (
                  <tr key={s.id} className="hover:bg-slate-50 transition-colors">
                    <td className="px-3 py-3 text-sm font-medium text-slate-900">{s.code}</td>
                    <td className="px-3 py-3 text-sm text-slate-700">{s.name}</td>
                    <td className="px-3 py-3 text-sm text-slate-500">{s.phone ?? '-'}</td>
                    <td className="px-3 py-3 text-sm text-slate-500">{s.email ?? '-'}</td>
                    <td className="px-3 py-3 text-sm">
                      <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${s.is_active ? 'bg-green-100 text-green-800' : 'bg-slate-100 text-slate-800'}`}>
                        {s.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="px-3 py-3 text-sm text-right">
                      <div className="inline-flex items-center gap-2">
                        {canUpdate ? (
                          <button 
                            onClick={() => setEditing(s)} 
                            className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors"
                          >
                            <EditIcon className="h-3.5 w-3.5" />
                            Edit
                          </button>
                        ) : null}
                        {canDelete ? (
                          <button 
                            onClick={() => void onDelete(s.id)} 
                            className="inline-flex items-center gap-1.5 rounded-md border border-red-200 bg-white px-2.5 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 transition-colors"
                          >
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
      </div>

      {/* POP UP MODAL UNTUK CREATE & EDIT */}
      {(creating || editing) && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          {/* Backdrop / Overlay */}
          <div 
            className="absolute inset-0 bg-black/50 backdrop-blur-sm" 
            onClick={() => {
              setCreating(false)
              setEditing(null)
            }}
          />
          
          {/* Modal Content */}
          <div className="relative w-full max-w-lg rounded-2xl border border-slate-200 bg-white p-6 shadow-2xl">
            {/* Header Modal */}
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-lg font-semibold text-slate-900">
                {creating ? 'Create New Supplier' : 'Edit Supplier'}
              </h3>
              <button 
                onClick={() => {
                  setCreating(false)
                  setEditing(null)
                }}
                className="rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-colors"
              >
                <CloseIcon className="h-5 w-5" />
              </button>
            </div>
            
            {/* Form Modal */}
            <SupplierForm
              key={editing ? `edit-${editing.id}` : 'create'}
              initial={editing ?? undefined}
              submitting={submitting}
              onSubmit={creating ? onCreate : onUpdate}
              onCancel={() => {
                setCreating(false)
                setEditing(null)
              }}
            />
          </div>
        </div>
      )}
    </div>
  )
}