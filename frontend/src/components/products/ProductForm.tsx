import { useState } from 'react'
import type { Product } from '../../types/product'

type Props = {
  initial?: Partial<Product>
  submitting?: boolean
  onSubmit: (payload: Partial<Product>) => Promise<void>
  onCancel?: () => void
}

export function ProductForm({ initial = {}, submitting = false, onSubmit, onCancel }: Props) {
  const [sku, setSku] = useState(initial.sku ?? '')
  const [name, setName] = useState(initial.name ?? '')
  const [barcode, setBarcode] = useState(initial.barcode ?? '')
  const [category, setCategory] = useState(initial.category ?? '')
  const [unit, setUnit] = useState(initial.unit ?? '')
  const [purchasePrice, setPurchasePrice] = useState(String(initial.purchase_price ?? '0'))
  const [sellingPrice, setSellingPrice] = useState(String(initial.selling_price ?? '0'))
  const [isActive, setIsActive] = useState(initial.is_active ?? true)
  const [errors, setErrors] = useState<Record<string, string>>({})

  // Note: parent components should provide a `key` when switching `initial`
  // to ensure the form remounts and state initializes from `initial`.

  function validate() {
    const e: Record<string, string> = {}
    if (!sku.trim()) e.sku = 'SKU is required'
    if (!name.trim()) e.name = 'Name is required'
    const pp = Number(purchasePrice)
    if (Number.isNaN(pp) || pp < 0) e.purchase_price = 'Purchase price must be >= 0'
    const sp = Number(sellingPrice)
    if (Number.isNaN(sp) || sp < 0) e.selling_price = 'Selling price must be >= 0'
    setErrors(e)
    return Object.keys(e).length === 0
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!validate()) return
    await onSubmit({
      sku: sku.trim(),
      name: name.trim(),
      barcode: barcode.trim() === '' ? undefined : barcode.trim(),
      category: category.trim() === '' ? undefined : category.trim(),
      unit: unit.trim() === '' ? undefined : unit.trim(),
      purchase_price: Number(purchasePrice),
      selling_price: Number(sellingPrice),
      is_active: isActive,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="product-sku" className="block text-sm font-medium text-slate-700">SKU</label>
        <input id="product-sku" value={sku} onChange={(e) => setSku(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
        {errors.sku ? <p className="mt-1 text-sm text-red-600">{errors.sku}</p> : null}
      </div>

      <div>
        <label htmlFor="product-name" className="block text-sm font-medium text-slate-700">Name</label>
        <input id="product-name" value={name} onChange={(e) => setName(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
        {errors.name ? <p className="mt-1 text-sm text-red-600">{errors.name}</p> : null}
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label htmlFor="product-barcode" className="block text-sm font-medium text-slate-700">Barcode</label>
          <input id="product-barcode" value={barcode ?? ''} onChange={(e) => setBarcode(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
        </div>
        <div>
          <label htmlFor="product-category" className="block text-sm font-medium text-slate-700">Category</label>
          <input id="product-category" value={category ?? ''} onChange={(e) => setCategory(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <div>
          <label htmlFor="product-unit" className="block text-sm font-medium text-slate-700">Unit</label>
          <input id="product-unit" value={unit ?? ''} onChange={(e) => setUnit(e.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
        </div>
        <div>
          <label htmlFor="product-purchase-price" className="block text-sm font-medium text-slate-700">Purchase Price</label>
          <input id="product-purchase-price" value={purchasePrice} onChange={(e) => setPurchasePrice(e.target.value)} type="number" step="0.01" className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
          {errors.purchase_price ? <p className="mt-1 text-sm text-red-600">{errors.purchase_price}</p> : null}
        </div>
        <div>
          <label htmlFor="product-selling-price" className="block text-sm font-medium text-slate-700">Selling Price</label>
          <input id="product-selling-price" value={sellingPrice} onChange={(e) => setSellingPrice(e.target.value)} type="number" step="0.01" className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
          {errors.selling_price ? <p className="mt-1 text-sm text-red-600">{errors.selling_price}</p> : null}
        </div>
      </div>

      <div className="flex items-center gap-4">
        <label htmlFor="product-active" className="flex items-center gap-2 text-sm">
          <input id="product-active" type="checkbox" checked={isActive} onChange={(e) => setIsActive(e.target.checked)} />
          <span>Active</span>
        </label>
      </div>

      <div className="flex items-center gap-3">
        <button disabled={submitting} type="submit" className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50">{submitting ? 'Saving...' : 'Save'}</button>
        {onCancel ? (
          <button type="button" onClick={onCancel} className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700">Cancel</button>
        ) : null}
      </div>
    </form>
  )
}
