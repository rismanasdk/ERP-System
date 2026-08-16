import { useState } from 'react'
import type { StockMovement } from '../../types/inventory'

type Props = {
  initial?: Partial<StockMovement>
  submitting?: boolean
  onSubmit: (payload: { movement_type: string; quantity_delta: number; reference_type?: string; reference_id?: number }) => Promise<void>
  onCancel?: () => void
}

export function InventoryAdjustForm({ initial = {}, submitting = false, onSubmit, onCancel }: Props) {
  const [movementType, setMovementType] = useState(initial.movement_type ?? 'IN')
  const [quantityDelta, setQuantityDelta] = useState(String(initial.quantity_delta ?? ''))
  const [referenceType, setReferenceType] = useState(initial.reference_type ?? '')
  const [referenceId, setReferenceId] = useState(initial.reference_id ? String(initial.reference_id) : '')
  const [errors, setErrors] = useState<Record<string, string>>({})

  function validate() {
    const e: Record<string, string> = {}
    const q = Number(quantityDelta)
    if (Number.isNaN(q) || q === 0) e.quantity_delta = 'Quantity must be non-zero'
    if (movementType === 'IN' && q <= 0) e.quantity_delta = 'IN movement requires positive quantity'
    if (movementType === 'OUT' && q >= 0) e.quantity_delta = 'OUT movement requires negative quantity'
    setErrors(e)
    return Object.keys(e).length === 0
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!validate()) return
    await onSubmit({
      movement_type: movementType,
      quantity_delta: Number(quantityDelta),
      reference_type: referenceType.trim() === '' ? undefined : referenceType.trim(),
      reference_id: referenceId.trim() === '' ? undefined : Number(referenceId.trim()),
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
            <label htmlFor="movement_type" className="block text-sm font-medium text-slate-700">Movement Type</label>
            <select id="movement_type" value={movementType} onChange={(e) => setMovementType(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm">
          <option value="IN">IN</option>
          <option value="OUT">OUT</option>
          <option value="ADJUSTMENT">ADJUSTMENT</option>
        </select>
      </div>

      <div>
        <label htmlFor="quantity_delta" className="block text-sm font-medium text-slate-700">Quantity Delta</label>
        <input id="quantity_delta" value={quantityDelta} onChange={(e) => setQuantityDelta(e.target.value)} type="number" className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
        {errors.quantity_delta ? <p className="mt-1 text-sm text-red-600">{errors.quantity_delta}</p> : null}
      </div>

      <div>
        <label htmlFor="reference_type" className="block text-sm font-medium text-slate-700">Reference Type (optional)</label>
        <input id="reference_type" value={referenceType} onChange={(e) => setReferenceType(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
      </div>

      <div>
        <label htmlFor="reference_id" className="block text-sm font-medium text-slate-700">Reference ID (optional)</label>
        <input id="reference_id" value={referenceId} onChange={(e) => setReferenceId(e.target.value)} type="number" className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
      </div>

      <div className="flex items-center gap-3">
        <button disabled={submitting} type="submit" className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50">{submitting ? 'Submitting...' : 'Submit'}</button>
        {onCancel ? (
          <button type="button" onClick={onCancel} className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700">Cancel</button>
        ) : null}
      </div>
    </form>
  )
}
