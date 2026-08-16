import { useState } from 'react'

type Props = {
  submitting?: boolean
  onSubmit: (payload: { product_id: number; branch_id: number; quantity: number }) => Promise<void>
  onCancel?: () => void
}

export function InventoryCreateForm({ submitting = false, onSubmit, onCancel }: Props) {
  const [productId, setProductId] = useState('')
  const [branchId, setBranchId] = useState('')
  const [quantity, setQuantity] = useState('0')
  const [errors, setErrors] = useState<Record<string, string>>({})

  function validate() {
    const e: Record<string, string> = {}
    const pid = Number(productId)
    const bid = Number(branchId)
    const q = Number(quantity)
    if (!Number.isFinite(pid) || pid <= 0) e.product_id = 'Product ID must be a positive integer'
    if (!Number.isFinite(bid) || bid <= 0) e.branch_id = 'Branch ID must be a positive integer'
    if (!Number.isFinite(q) || q < 0) e.quantity = 'Quantity must be >= 0'
    setErrors(e)
    return Object.keys(e).length === 0
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!validate()) return
    await onSubmit({ product_id: Number(productId), branch_id: Number(branchId), quantity: Number(quantity) })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-slate-700">Product ID</label>
        <input value={productId} onChange={(e) => setProductId(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
        {errors.product_id ? <p className="mt-1 text-sm text-red-600">{errors.product_id}</p> : null}
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-700">Branch ID</label>
        <input value={branchId} onChange={(e) => setBranchId(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
        {errors.branch_id ? <p className="mt-1 text-sm text-red-600">{errors.branch_id}</p> : null}
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-700">Quantity</label>
        <input value={quantity} onChange={(e) => setQuantity(e.target.value)} type="number" className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
        {errors.quantity ? <p className="mt-1 text-sm text-red-600">{errors.quantity}</p> : null}
      </div>

      <div className="flex items-center gap-3">
        <button disabled={submitting} type="submit" className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50">{submitting ? 'Creating...' : 'Create'}</button>
        {onCancel ? (
          <button type="button" onClick={onCancel} className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700">Cancel</button>
        ) : null}
      </div>
    </form>
  )
}
