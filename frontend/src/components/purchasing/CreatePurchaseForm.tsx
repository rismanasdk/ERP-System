import { useCallback, useMemo, useState } from 'react'
import type { CreatePurchaseInput, CreatePurchaseItemInput } from '../../types/purchase'
import { suppliersApi } from '../../services/suppliers'
import type { Supplier } from '../../types/supplier'
import { PurchaseItemEditor } from './PurchaseItemEditor'

type Props = {
  submitting?: boolean
  onSubmit: (payload: CreatePurchaseInput) => Promise<void>
  onCancel?: () => void
  defaultBranchId?: number
}

export function CreatePurchaseForm({ submitting = false, onSubmit, onCancel, defaultBranchId }: Props) {
  const [branchId, setBranchId] = useState(String(defaultBranchId ?? ''))
  const [supplierId, setSupplierId] = useState('')
  const [notes, setNotes] = useState('')
  const [items, setItems] = useState<CreatePurchaseItemInput[]>([])
  const [suppliers, setSuppliers] = useState<Supplier[]>([])
  const [errors, setErrors] = useState<Record<string, string>>({})

  const loadSuppliers = useCallback(async () => {
    try {
      const res = await suppliersApi.list({ active: true }, undefined)
      setSuppliers(res)
    } catch {
      setSuppliers([])
    }
  }, [])

  const addItem = useCallback(() => {
    setItems((s) => [...s, { product_id: 0, quantity: 1, unit_cost: 0 }])
  }, [])

  const updateItem = useCallback((idx: number, v: CreatePurchaseItemInput) => {
    setItems((s) => s.map((it, i) => (i === idx ? v : it)))
  }, [])

  const removeItem = useCallback((idx: number) => {
    setItems((s) => s.filter((_, i) => i !== idx))
  }, [])

  const total = useMemo(() => items.reduce((acc, it) => acc + it.quantity * it.unit_cost, 0), [items])

  function validate() {
    const e: Record<string, string> = {}
    const bid = Number(branchId)
    const sid = Number(supplierId)
    if (!Number.isFinite(bid) || bid <= 0) e.branch_id = 'Branch ID is required'
    if (!Number.isFinite(sid) || sid <= 0) e.supplier_id = 'Supplier is required'
    if (items.length === 0) e.items = 'Add at least one item'
    setErrors(e)
    return Object.keys(e).length === 0
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!validate()) return
    await onSubmit({ branch_id: Number(branchId), supplier_id: Number(supplierId), notes: notes.trim() === '' ? undefined : notes.trim(), items })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <div>
          <label className="block text-sm font-medium text-slate-700">Branch ID</label>
          <input value={branchId} onChange={(e) => setBranchId(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
          {errors.branch_id ? <p className="mt-1 text-sm text-red-600">{errors.branch_id}</p> : null}
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700">Supplier</label>
          <select value={supplierId} onChange={(e) => setSupplierId(e.target.value)} onFocus={() => void loadSuppliers()} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm">
            <option value="">-- Select supplier --</option>
            {suppliers.map((s: Supplier) => (
              <option key={s.id} value={s.id}>{s.code} - {s.name}</option>
            ))}
          </select>
          {errors.supplier_id ? <p className="mt-1 text-sm text-red-600">{errors.supplier_id}</p> : null}
        </div>
        <div>
          <label className="block text-sm font-medium text-slate-700">Notes</label>
          <input value={notes} onChange={(e) => setNotes(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
        </div>
      </div>

      <div className="space-y-3">
        {items.map((it, idx) => (
          <div key={idx} className="rounded-md border border-slate-200 p-3">
            <PurchaseItemEditor value={it} onChange={(v) => updateItem(idx, v)} onRemove={() => removeItem(idx)} existingProductIds={items.map((x) => x.product_id)} />
          </div>
        ))}
      </div>

      <div className="flex items-center gap-3">
        <button type="button" onClick={addItem} className="rounded-md bg-white border px-3 py-2">Add item</button>
        <div className="ml-auto text-lg font-semibold">Total: {new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(total)}</div>
      </div>

      <div className="flex items-center gap-3">
        <button disabled={submitting} type="submit" className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50">{submitting ? 'Creating...' : 'Create Purchase'}</button>
        {onCancel ? <button type="button" onClick={onCancel} className="rounded-md border border-slate-300 bg-white px-3 py-2">Cancel</button> : null}
      </div>
    </form>
  )
}
