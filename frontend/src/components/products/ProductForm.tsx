import { useState } from 'react'
import type { ProductCategory } from '../../types/category'
import type { Product } from '../../types/product'

type Props = { initial?: Partial<Product>; submitting?: boolean; onSubmit: (payload: Partial<Product>) => Promise<void>; onCancel?: () => void; categories?: ProductCategory[] }

export function ProductForm({ initial = {}, submitting = false, onSubmit, onCancel, categories = [] }: Props) {
  const [sku, setSku] = useState(initial.sku ?? '')
  const [name, setName] = useState(initial.name ?? '')
  const [description, setDescription] = useState(initial.description ?? '')
  const [barcode, setBarcode] = useState(initial.barcode ?? '')
  const [categoryId, setCategoryId] = useState(String(initial.category_id ?? ''))
  const [category, setCategory] = useState(initial.category ?? '')
  const [unit, setUnit] = useState(initial.unit ?? '')
  const [purchasePrice, setPurchasePrice] = useState(String(initial.purchase_price ?? '0'))
  const [sellingPrice, setSellingPrice] = useState(String(initial.selling_price ?? '0'))
  const [minimumStock, setMinimumStock] = useState(String(initial.minimum_stock ?? '0'))
  const [isActive, setIsActive] = useState(initial.is_active ?? true)
  const [errors, setErrors] = useState<Record<string, string>>({})

  const validate = () => {
    const next: Record<string, string> = {}
    if (!sku.trim()) next.sku = 'SKU is required'
    if (!name.trim()) next.name = 'Name is required'
    if (categories.length > 0 && !categoryId) next.category_id = 'Category is required'
    if (Number.isNaN(Number(purchasePrice)) || Number(purchasePrice) < 0) next.purchase_price = 'Purchase price must be >= 0'
    if (Number.isNaN(Number(sellingPrice)) || Number(sellingPrice) < 0) next.selling_price = 'Selling price must be >= 0'
    if (Number.isNaN(Number(minimumStock)) || Number(minimumStock) < 0) next.minimum_stock = 'Minimum stock must be >= 0'
    setErrors(next)
    return Object.keys(next).length === 0
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    if (!validate()) return
    await onSubmit({ sku: sku.trim(), name: name.trim(), description: description.trim() || undefined, barcode: barcode.trim() || undefined, category: category || undefined, category_id: categoryId ? Number(categoryId) : undefined, unit: unit.trim() || undefined, purchase_price: Number(purchasePrice), selling_price: Number(sellingPrice), minimum_stock: Number(minimumStock), is_active: isActive })
  }

  return <form onSubmit={(event) => void submit(event)} className="space-y-4">
    <div><label htmlFor="product-sku" className="block text-sm font-medium text-slate-700">SKU</label><input id="product-sku" value={sku} onChange={(event) => setSku(event.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />{errors.sku && <p className="mt-1 text-sm text-red-600">{errors.sku}</p>}</div>
    <div><label htmlFor="product-name" className="block text-sm font-medium text-slate-700">Name</label><input id="product-name" value={name} onChange={(event) => setName(event.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />{errors.name && <p className="mt-1 text-sm text-red-600">{errors.name}</p>}</div>
    <div><label htmlFor="product-description" className="block text-sm font-medium text-slate-700">Description</label><textarea id="product-description" value={description ?? ''} onChange={(event) => setDescription(event.target.value)} rows={2} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" /></div>
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2"><div><label htmlFor="product-barcode" className="block text-sm font-medium text-slate-700">Barcode</label><input id="product-barcode" value={barcode ?? ''} onChange={(event) => setBarcode(event.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" /></div><div><label htmlFor="product-category" className="block text-sm font-medium text-slate-700">Category</label><select id="product-category" value={categoryId} onChange={(event) => { setCategoryId(event.target.value); setCategory(categories.find((item) => String(item.id) === event.target.value)?.name ?? '') }} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm"><option value="">Select category</option>{categories.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select>{errors.category_id && <p className="mt-1 text-sm text-red-600">{errors.category_id}</p>}</div></div>
    <div className="grid grid-cols-1 gap-4 md:grid-cols-4"><div><label htmlFor="product-unit" className="block text-sm font-medium text-slate-700">Unit</label><input id="product-unit" value={unit ?? ''} onChange={(event) => setUnit(event.target.value)} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" /></div><div><label htmlFor="product-purchase-price" className="block text-sm font-medium text-slate-700">Cost Price</label><input id="product-purchase-price" value={purchasePrice} onChange={(event) => setPurchasePrice(event.target.value)} type="number" min="0" step="0.01" className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />{errors.purchase_price && <p className="mt-1 text-sm text-red-600">{errors.purchase_price}</p>}</div><div><label htmlFor="product-selling-price" className="block text-sm font-medium text-slate-700">Selling Price</label><input id="product-selling-price" value={sellingPrice} onChange={(event) => setSellingPrice(event.target.value)} type="number" min="0" step="0.01" className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />{errors.selling_price && <p className="mt-1 text-sm text-red-600">{errors.selling_price}</p>}</div><div><label htmlFor="product-minimum-stock" className="block text-sm font-medium text-slate-700">Minimum Stock</label><input id="product-minimum-stock" value={minimumStock} onChange={(event) => setMinimumStock(event.target.value)} type="number" min="0" step="1" className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />{errors.minimum_stock && <p className="mt-1 text-sm text-red-600">{errors.minimum_stock}</p>}</div></div>
    <label htmlFor="product-active" className="flex items-center gap-2 text-sm"><input id="product-active" type="checkbox" checked={isActive} onChange={(event) => setIsActive(event.target.checked)} />Active</label>
    <div className="flex items-center gap-3"><button disabled={submitting} type="submit" className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white disabled:opacity-50">{submitting ? 'Saving...' : 'Save'}</button>{onCancel && <button type="button" onClick={onCancel} className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm">Cancel</button>}</div>
  </form>
}
