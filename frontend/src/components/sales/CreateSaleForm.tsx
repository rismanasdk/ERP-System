import { useCallback, useEffect, useMemo, useState } from 'react'
import type { Branch } from '../../types/auth'
import type { CreateSaleInput, CreateSaleItemInput } from '../../types/sale'
import { branchesApi } from '../../services/branches'
import { readStoredAccessToken } from '../../services/authSession'
import { SaleItemEditor } from './SaleItemEditor'

type Props = {
  submitting?: boolean
  onSubmit: (payload: CreateSaleInput) => Promise<void>
  onCancel?: () => void
  defaultBranchId?: number
}

export function CreateSaleForm({ submitting = false, onSubmit, onCancel, defaultBranchId }: Props) {
  const [branchId, setBranchId] = useState(String(defaultBranchId ?? ''))
  const [notes, setNotes] = useState('')
  const [items, setItems] = useState<CreateSaleItemInput[]>([])
  const [branches, setBranches] = useState<Branch[]>([])
  const [errors, setErrors] = useState<Record<string, string>>({})
  const token = readStoredAccessToken() ?? undefined

  const loadBranches = useCallback(async () => {
    try {
      const res = await branchesApi.list(true, token)
      setBranches(Array.isArray(res) ? res : [])
    } catch {
      setBranches([])
    }
  }, [token])

  useEffect(() => {
    const id = window.setTimeout(() => {
      void loadBranches()
    }, 0)

    return () => window.clearTimeout(id)
  }, [loadBranches])

  const addItem = useCallback(() => {
    setItems((s) => [...s, { product_id: 0, quantity: 1, unit_price: 0 }])
  }, [])

  const updateItem = useCallback((idx: number, v: CreateSaleItemInput) => {
    setItems((s) => s.map((it, i) => (i === idx ? v : it)))
  }, [])

  const removeItem = useCallback((idx: number) => {
    setItems((s) => s.filter((_, i) => i !== idx))
  }, [])

  const total = useMemo(() => items.reduce((acc, it) => acc + it.quantity * it.unit_price, 0), [items])

  function validate() {
    const e: Record<string, string> = {}
    const bid = Number(branchId)
    if (!Number.isFinite(bid) || bid <= 0) e.branch_id = 'Branch is required'
    if (items.length === 0) e.items = 'Add at least one item'
    setErrors(e)
    return Object.keys(e).length === 0
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!validate()) return
    await onSubmit({ branch_id: Number(branchId), notes: notes.trim() === '' ? undefined : notes.trim(), items })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <div>
          <label htmlFor="sale-branch" className="block text-sm font-medium text-slate-700">Branch</label>
          <select id="sale-branch" value={branchId} onChange={(e) => setBranchId(e.target.value)} onFocus={() => void loadBranches()} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm">
            <option value="">-- Select branch --</option>
            {(branches ?? []).map((b: Branch) => (
              <option key={b.id} value={b.id}>{b.name}</option>
            ))}
          </select>
          {errors.branch_id ? <p className="mt-1 text-sm text-red-600">{errors.branch_id}</p> : null}
        </div>
        <div className="md:col-span-2">
          <label htmlFor="sale-notes" className="block text-sm font-medium text-slate-700">Notes</label>
          <input id="sale-notes" value={notes} onChange={(e) => setNotes(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
        </div>
      </div>

      <div className="space-y-3">
        {items.map((it, idx) => (
          <div key={idx} className="rounded-md border border-slate-200 p-3">
            <SaleItemEditor value={it} onChange={(v) => updateItem(idx, v)} onRemove={() => removeItem(idx)} existingProductIds={items.map((x) => x.product_id)} />
          </div>
        ))}
      </div>

      <div className="flex items-center gap-3">
        <button type="button" onClick={addItem} className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700">Add item</button>
        <div className="ml-auto text-lg font-semibold">Total: {new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(total)}</div>
      </div>

      <div className="flex items-center gap-3">
        <button disabled={submitting} type="submit" className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50">{submitting ? 'Creating...' : 'Create Sale'}</button>
        {onCancel ? <button type="button" onClick={onCancel} className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm">Cancel</button> : null}
      </div>
    </form>
  )
}
