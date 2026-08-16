import { useCallback, useEffect, useMemo, useState } from 'react'
import { useAuth } from '../hooks/useAuth'
import type { Product, ProductFilter } from '../types/product'
import { productsApi } from '../services/products'
import { readStoredAccessToken } from '../services/authSession'
import { ProductForm } from '../components/products/ProductForm'
import { ApiError } from '../lib/api'
import { EditIcon, DeleteIcon, CreateIcon, CloseIcon, SearchIcon } from '../utils/iconsUtils'
import { useConfirm } from '../utils/confirmUtils'
import { usePagination, PaginationControl } from '../utils/paginationUtils'

function money(v: number) {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(v)
}

export function ProductsPage() {
  const { user } = useAuth()
  const confirmDialog = useConfirm()
  const [products, setProducts] = useState<Product[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState({ search: '', active: '' })
  const [editing, setEditing] = useState<Product | null>(null)
  const [creating, setCreating] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const token = readStoredAccessToken() ?? undefined

  const rows = useMemo(() => products, [products])

  // Pagination: 10 item per halaman, slicing dilakukan di client dari `rows`
  const { page, totalPages, pageItems, goToPage, resetPage } = usePagination(rows, 10)

  const load = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const f: ProductFilter = {}
      if (filter.search) f.search = filter.search
      if (filter.active !== '') f.active = filter.active === 'true'
      const res = await productsApi.list(f, token)
      setProducts(res)
      resetPage() // balik ke halaman 1 tiap kali hasil filter berubah
    } catch (err) {
      const e = err as ApiError
      if (e instanceof ApiError) {
        if (e.status === 401) return setError('Session expired. Please sign in again.')
        if (e.status === 403) return setError('You do not have access to products.')
        return setError(e.message)
      }
      setError('Unable to load products')
    } finally {
      setIsLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
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

  const onCreate = useCallback(async (payload: Partial<Product>) => {
    setSubmitting(true)
    try {
      await productsApi.create(payload, token)
      setCreating(false)
      await load()
    } catch (err) {
      const e = err as ApiError
      throw e
    } finally {
      setSubmitting(false)
    }
  }, [load, token])

  const onUpdate = useCallback(async (payload: Partial<Product>) => {
    if (!editing) return
    setSubmitting(true)
    try {
      await productsApi.update(editing.id, payload, token)
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
    const ok = await confirmDialog({
      title: 'Delete this Product?',
      description: 'This action cannot be undone.',
      confirmLabel: 'Delete',
      variant: 'destructive',
    })
    if (!ok) return

    setSubmitting(true)
    try {
      await productsApi.softDelete(id, token)
      await load()
    } catch (err) {
      const e = err as ApiError
      setError(e.message)
    } finally {
      setSubmitting(false)
    }
  }, [confirmDialog, load, token])

  const isSuperAdmin = user?.roles?.includes('SUPER_ADMIN')
  const canCreate = isSuperAdmin || user?.permissions?.includes('products.create')
  const canUpdate = isSuperAdmin || user?.permissions?.includes('products.update')
  const canDelete = isSuperAdmin || user?.permissions?.includes('products.delete')

  return (
    <div className="space-y-6">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm flex items-center justify-between">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Products</p>
          <h2 className="mt-1 text-2xl font-bold text-slate-900">Products</h2>
          <p className="mt-2 text-sm text-slate-600">Manage product master data.</p>
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
              placeholder="Search by SKU or name" 
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
            <div className="p-6 text-center text-slate-500">No products found.</div>
          ) : (
            <table className="min-w-full table-auto">
              <thead>
                <tr className="border-b border-slate-100 text-left text-xs font-medium uppercase tracking-wider text-slate-500">
                  <th className="px-3 py-3">SKU</th>
                  <th className="px-3 py-3">Name</th>
                  <th className="px-3 py-3">Category</th>
                  <th className="px-3 py-3">Unit</th>
                  <th className="px-3 py-3">Selling</th>
                  <th className="px-3 py-3">Active</th>
                  <th className="px-3 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {pageItems.map((p) => (
                  <tr key={p.id} className="hover:bg-slate-50 transition-colors">
                    <td className="px-3 py-3 text-sm font-medium text-slate-900">{p.sku}</td>
                    <td className="px-3 py-3 text-sm text-slate-700">{p.name}</td>
                    <td className="px-3 py-3 text-sm text-slate-500">{p.category ?? '-'}</td>
                    <td className="px-3 py-3 text-sm text-slate-500">{p.unit ?? '-'}</td>
                    <td className="px-3 py-3 text-sm text-slate-500">{money(p.selling_price)}</td>
                    <td className="px-3 py-3 text-sm">
                      <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${p.is_active ? 'bg-green-100 text-green-800' : 'bg-slate-100 text-slate-800'}`}>
                        {p.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="px-3 py-3 text-sm text-right">
                      <div className="inline-flex items-center gap-2">
                        {canUpdate ? (
                          <button 
                            onClick={() => setEditing(p)} 
                            className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors"
                          >
                            <EditIcon className="h-3.5 w-3.5" />
                            Edit
                          </button>
                        ) : null}
                        {canDelete ? (
                          <button 
                            onClick={() => void onDelete(p.id)} 
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

        {/* Pagination control - otomatis hidden kalau totalPages <= 1 */}
        {!isLoading && rows.length > 0 ? (
          <div className="mt-4 flex justify-center">
            <PaginationControl currentPage={page} totalPages={totalPages} onPageChange={goToPage} />
          </div>
        ) : null}
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
                {creating ? 'Create New Product' : 'Edit Product'}
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
            <ProductForm
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