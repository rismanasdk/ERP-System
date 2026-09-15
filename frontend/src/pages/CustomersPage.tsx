import { useCallback, useEffect, useMemo, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import type { Customer, CustomerFilter } from '../types/customer'
import { customersApi, type CustomerUpdatePayload } from '../services/customers'
import { readStoredAccessToken } from '../services/authSession'
import { CustomerForm } from '../components/customers/CustomerForm'
import { ApiError } from '../lib/api'
import { CloseIcon, CreateIcon, DeleteIcon, EditIcon, SearchIcon, ViewIcon } from '../utils/iconsUtils'
import { useConfirm } from '../utils/confirmUtils'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../components/ui/dialog'

function formatNull(value?: string | null) {
  return value ?? '-'
}

export function CustomersPage() {
  const { user } = useAuth()
  const confirmDialog = useConfirm()
  const [customers, setCustomers] = useState<Customer[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState({ search: '', active: '' })
  const [editing, setEditing] = useState<Customer | null>(null)
  const [creating, setCreating] = useState(false)
  const [detailCustomer, setDetailCustomer] = useState<Customer | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const token = readStoredAccessToken() ?? undefined
  const rows = useMemo(() => customers, [customers])

  const canRead = user ? Boolean(user?.permissions?.includes('customers.read')) : true
  const canCreate = user?.permissions?.includes('customers.create')
  const canUpdate = user?.permissions?.includes('customers.update')
  const canDelete = user?.permissions?.includes('customers.delete')

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
      if (!canRead) {
        setIsLoading(false)
        setError(null)
        return
      }
      await load()
    }
    void run()
    return () => {
      active = false
    }
  }, [load, canRead])

  const handleView = useCallback(async (id: number) => {
    try {
      const detail = await customersApi.getById(id, token)
      setDetailCustomer(detail)
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    }
  }, [token])

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
      const updatePayload: CustomerUpdatePayload = { ...payload, expected_version: editing.version }
      await customersApi.update(editing.id, updatePayload, token)
      setEditing(null)
      await load()
    } catch (err) {
      const e = err as ApiError
      if (e instanceof ApiError && e.status === 409) {
        setEditing(null)
        setError('This customer was changed by someone else. The latest data has been loaded.')
        await load()
        return
      }
      setError(e.message)
      return
    } finally {
      setSubmitting(false)
    }
  }, [editing, load, token])

  const onDeactivate = useCallback(async (id: number) => {
    const ok = await confirmDialog({
      title: 'Deactivate this customer?',
      description: 'This action will mark the customer as inactive and soft-delete it from the active list.',
      confirmLabel: 'Deactivate',
      variant: 'destructive',
    })
    if (!ok) return

    setSubmitting(true)
    try {
      await customersApi.remove(id, token)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    } finally {
      setSubmitting(false)
    }
  }, [confirmDialog, load, token])

  if (!canRead) {
    return (
      <div className="rounded-2xl border border-amber-200 bg-amber-50 p-6 text-amber-800 shadow-sm">
        You do not have permission to view customers.
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">CUSTOMERS</p>
          <h2 className="mt-1 text-2xl font-bold text-slate-900">Customers</h2>
          <p className="mt-2 text-sm text-slate-600">Manage customer master data.</p>
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
          <div className="relative max-w-sm flex-1">
            <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
            <input
              value={filter.search}
              onChange={(e) => setFilter((current) => ({ ...current, search: e.target.value }))}
              placeholder="Search by code or name"
              className="w-full rounded-md border border-slate-200 pl-9 pr-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>

          <select
            value={filter.active}
            onChange={(e) => setFilter((current) => ({ ...current, active: e.target.value }))}
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
              {filter.search || filter.active !== '' ? 'No customers match the current filter.' : 'No customers found.'}
            </div>
          ) : (
            <table className="min-w-full table-auto">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                  <th className="px-3 py-3">Code</th>
                  <th className="px-3 py-3">Name</th>
                  <th className="px-3 py-3">Phone</th>
                  <th className="px-3 py-3">Email</th>
                  <th className="px-3 py-3">Tax ID</th>
                  <th className="px-3 py-3">Status</th>
                  <th className="px-3 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {rows.map((customer) => (
                  <tr key={customer.id} className="hover:bg-slate-50 transition-colors">
                    <td className="px-3 py-3 text-sm font-medium text-slate-900">{customer.code}</td>
                    <td className="px-3 py-3 text-sm text-slate-700">{customer.name}</td>
                    <td className="px-3 py-3 text-sm text-slate-500">{formatNull(customer.phone)}</td>
                    <td className="px-3 py-3 text-sm text-slate-500">{formatNull(customer.email)}</td>
                    <td className="px-3 py-3 text-sm text-slate-500">{formatNull(customer.tax_id)}</td>
                    <td className="px-3 py-3 text-sm">
                      <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${customer.is_active ? 'bg-green-100 text-green-800' : 'bg-slate-100 text-slate-700'}`}>
                        {customer.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="px-3 py-3 text-right">
                      <div className="inline-flex items-center gap-2">
                        {canRead ? (
                          <button
                            type="button"
                            onClick={() => void handleView(customer.id)}
                            className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors"
                          >
                            <ViewIcon className="h-3.5 w-3.5" />
                            View
                          </button>
                        ) : null}
                        {canUpdate ? (
                          <button
                            type="button"
                            onClick={() => setEditing(customer)}
                            className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors"
                          >
                            <EditIcon className="h-3.5 w-3.5" />
                            Edit
                          </button>
                        ) : null}
                        {canDelete && customer.is_active ? (
                          <button
                            type="button"
                            onClick={() => void onDeactivate(customer.id)}
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

      <Dialog open={Boolean(detailCustomer)} onOpenChange={(open) => !open && setDetailCustomer(null)}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>Customer Details</DialogTitle>
            <DialogDescription>View customer master data.</DialogDescription>
          </DialogHeader>

          {detailCustomer ? (
            <div className="space-y-3 text-sm">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <p className="text-slate-500">Code</p>
                  <p className="font-medium text-slate-900">{detailCustomer.code}</p>
                </div>
                <div>
                  <p className="text-slate-500">Status</p>
                  <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${detailCustomer.is_active ? 'bg-green-100 text-green-800' : 'bg-slate-100 text-slate-700'}`}>
                    {detailCustomer.is_active ? 'Active' : 'Inactive'}
                  </span>
                </div>
              </div>

              <div>
                <p className="text-slate-500">Name</p>
                <p className="font-medium text-slate-900">{detailCustomer.name}</p>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <p className="text-slate-500">Phone</p>
                  <p className="text-slate-900">{formatNull(detailCustomer.phone)}</p>
                </div>
                <div>
                  <p className="text-slate-500">Email</p>
                  <p className="text-slate-900">{formatNull(detailCustomer.email)}</p>
                </div>
              </div>

              <div>
                <p className="text-slate-500">Address</p>
                <p className="text-slate-900">{formatNull(detailCustomer.address)}</p>
              </div>

              <div>
                <p className="text-slate-500">Tax ID</p>
                <p className="text-slate-900">{formatNull(detailCustomer.tax_id)}</p>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <p className="text-slate-500">Created</p>
                  <p className="text-slate-900">{detailCustomer.created_at ? new Date(detailCustomer.created_at).toLocaleString('id-ID') : '-'}</p>
                </div>
                <div>
                  <p className="text-slate-500">Updated</p>
                  <p className="text-slate-900">{detailCustomer.updated_at ? new Date(detailCustomer.updated_at).toLocaleString('id-ID') : '-'}</p>
                </div>
              </div>
            </div>
          ) : null}

          <DialogFooter>
            <button
              type="button"
              onClick={() => setDetailCustomer(null)}
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
        <DialogContent className="max-w-2xl">
          <DialogHeader className="flex items-center justify-between">
            <div>
              <DialogTitle>{creating ? 'Create Customer' : 'Edit Customer'}</DialogTitle>
              <DialogDescription>{creating ? 'Create a new customer master record.' : 'Update customer details.'}</DialogDescription>
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
        </DialogContent>
      </Dialog>
    </div>
  )
}
