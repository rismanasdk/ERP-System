import { useCallback, useEffect, useMemo, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import type { Customer, CustomerFilter } from '../types/customer'
import { customersApi } from '../services/customers'
import { readStoredAccessToken } from '../services/authSession'
import { CustomerForm } from '../components/customers/CustomerForm'
import { ApiError } from '../lib/api'

export function CustomersPage() {
  const { user } = useAuth()
  const [customers, setCustomers] = useState<Customer[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState({ search: '', active: '' })
  const [editing, setEditing] = useState<Customer | null>(null)
  const [creating, setCreating] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const token = readStoredAccessToken() ?? undefined

  const load = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const f: CustomerFilter = {}
      if (filter.search) f.search = filter.search
      if (filter.active !== '') f.active = filter.active === 'true'
      const res = await customersApi.list(f, token)
      setCustomers(res)
    } catch (err) {
      const e = err as ApiError
      if (e instanceof ApiError) {
        if (e.status === 401) return setError('Session expired. Please sign in again.')
        if (e.status === 403) return setError('You do not have access to customers.')
        return setError(e.message)
      }
      setError('Unable to load customers')
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

  const onCreate = useCallback(async (payload: Partial<Customer>) => {
    setSubmitting(true)
    try {
      await customersApi.create(payload, token)
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

  const onUpdate = useCallback(async (payload: Partial<Customer>) => {
    if (!editing) return
    setSubmitting(true)
    try {
      await customersApi.update(editing.id, payload, token)
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
    if (!confirm('Are you sure you want to delete this customer?')) return
    setSubmitting(true)
    try {
      await customersApi.softDelete(id, token)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    } finally {
      setSubmitting(false)
    }
  }, [load, token])

  const canCreate = user?.permissions?.includes('customers.create')
  const canUpdate = user?.permissions?.includes('customers.update')
  const canDelete = user?.permissions?.includes('customers.delete')

  const rows = useMemo(() => customers, [customers])

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Customers</p>
          <h2 className="mt-1 text-2xl font-bold text-slate-900">Customers</h2>
          <p className="mt-2 text-sm text-slate-600">Manage customer master data.</p>
        </div>
        <div className="flex items-center gap-3">
          {canCreate ? (
            <button onClick={() => setCreating(true)} className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500">Create</button>
          ) : null}
        </div>
      </div>

      {error ? (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-red-800 shadow-sm">
          <p>{error}</p>
          <div className="mt-3">
            <button onClick={() => void load()} className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white">Retry</button>
          </div>
        </div>
      ) : null}

      <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="flex items-center gap-3">
          <input value={filter.search} onChange={(e) => setFilter((s) => ({ ...s, search: e.target.value }))} placeholder="Search by code or name" className="rounded-md border border-slate-200 px-3 py-2" />
          <select value={filter.active} onChange={(e) => setFilter((s) => ({ ...s, active: e.target.value }))} className="rounded-md border border-slate-200 px-3 py-2">
            <option value="">All</option>
            <option value="true">Active</option>
            <option value="false">Inactive</option>
          </select>
          <button onClick={() => void load()} className="rounded-md border border-slate-300 bg-white px-3 py-2">Apply</button>
        </div>

        <div className="mt-4 overflow-auto">
          {isLoading ? (
            <div className="space-y-2">
              <div className="h-8 w-1/3 rounded bg-slate-200" />
              <div className="h-8 w-1/2 rounded bg-slate-200" />
            </div>
          ) : rows.length === 0 ? (
            <div className="p-6 text-slate-600">No customers found.</div>
          ) : (
            <table className="min-w-full table-auto">
              <thead>
                <tr className="text-left text-sm text-slate-500">
                  <th className="px-3 py-2">Code</th>
                  <th className="px-3 py-2">Name</th>
                  <th className="px-3 py-2">Phone</th>
                  <th className="px-3 py-2">Email</th>
                  <th className="px-3 py-2">Active</th>
                  <th className="px-3 py-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((c) => (
                  <tr key={c.id} className="border-t">
                    <td className="px-3 py-2 align-top">{c.code}</td>
                    <td className="px-3 py-2 align-top">{c.name}</td>
                    <td className="px-3 py-2 align-top">{c.phone ?? '-'}</td>
                    <td className="px-3 py-2 align-top">{c.email ?? '-'}</td>
                    <td className="px-3 py-2 align-top">{c.is_active ? 'Yes' : 'No'}</td>
                    <td className="px-3 py-2 align-top">
                      <div className="flex gap-2">
                        {canUpdate ? (
                          <button onClick={() => setEditing(c)} className="rounded-md border border-slate-300 bg-white px-2 py-1 text-sm">Edit</button>
                        ) : null}
                        {canDelete ? (
                          <button onClick={() => void onDelete(c.id)} className="rounded-md border border-red-300 bg-white px-2 py-1 text-sm text-red-700">Delete</button>
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

      {(creating || editing) && (
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h3 className="text-lg font-semibold">{creating ? 'Create customer' : 'Edit customer'}</h3>
          <div className="mt-4">
            <CustomerForm
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
