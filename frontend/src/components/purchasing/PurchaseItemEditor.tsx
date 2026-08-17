import { useCallback, useEffect, useMemo, useState } from 'react'
import type { Product } from '../../types/product'
import { productsApi } from '../../services/products'
import { readStoredAccessToken } from '../../services/authSession'

type Props = {
  value?: { product_id?: number; quantity?: number; unit_cost?: number }
  onChange: (v: { product_id: number; quantity: number; unit_cost: number }) => void
  onRemove?: () => void
  existingProductIds?: number[]
}

export function PurchaseItemEditor({ value = {}, onChange, onRemove, existingProductIds = [] }: Props) {
  const [productId, setProductId] = useState(value.product_id ?? 0)
  const [quantity, setQuantity] = useState(String(value.quantity ?? '1'))
  const [unitCost, setUnitCost] = useState(String(value.unit_cost ?? '0'))
  const [products, setProducts] = useState<Product[]>([])
  const token = readStoredAccessToken() ?? undefined

  const loadProducts = useCallback(async (q?: string) => {
    try {
      const res = await productsApi.list({ search: q }, token)
      setProducts(Array.isArray(res) ? res : [])
    } catch {
      setProducts([])
    }
  }, [token])

  useEffect(() => {
    const id = window.setTimeout(() => {
      void loadProducts()
    }, 0)

    return () => window.clearTimeout(id)
  }, [loadProducts])

  const subtotal = useMemo(() => {
    const q = Number(quantity)
    const u = Number(unitCost)
    if (Number.isNaN(q) || Number.isNaN(u)) return 0
    return q * u
  }, [quantity, unitCost])

  const handleProductChange = useCallback((v: number) => {
    setProductId(v)
    onChange({ product_id: v, quantity: Number(quantity), unit_cost: Number(unitCost) })
  }, [onChange, quantity, unitCost])

  const handleQuantityChange = useCallback((v: string) => {
    setQuantity(v)
    onChange({ product_id: productId, quantity: Number(v), unit_cost: Number(unitCost) })
  }, [onChange, productId, unitCost])

  const handleUnitCostChange = useCallback((v: string) => {
    setUnitCost(v)
    onChange({ product_id: productId, quantity: Number(quantity), unit_cost: Number(v) })
  }, [onChange, productId, quantity])

  return (
    <div className="grid grid-cols-1 gap-3 md:grid-cols-6 items-end">
      <div className="md:col-span-2">
        <label htmlFor={`product-${productId || 'new'}`} className="block text-sm font-medium text-slate-700">Product</label>
        <select id={`product-${productId || 'new'}`} value={productId} onChange={(e) => handleProductChange(Number(e.target.value))} onFocus={() => void loadProducts()} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm">
          <option value={0}>-- Select product --</option>
          {(products ?? []).map((p) => (
            <option key={p.id} value={p.id} disabled={existingProductIds.includes(p.id)}>
              {p.name}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label htmlFor={`quantity-${productId || 'new'}`} className="block text-sm font-medium text-slate-700">Quantity</label>
        <input id={`quantity-${productId || 'new'}`} value={quantity} onChange={(e) => handleQuantityChange(e.target.value)} type="number" min={1} className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
      </div>

      <div>
        <label htmlFor={`unitcost-${productId || 'new'}`} className="block text-sm font-medium text-slate-700">Unit Cost</label>
        <input id={`unitcost-${productId || 'new'}`} value={unitCost} onChange={(e) => handleUnitCostChange(e.target.value)} type="number" min={0} step="0.01" className="mt-1 block w-full rounded-md border-slate-200 shadow-sm" />
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-700">Subtotal</label>
        <div className="mt-1 text-slate-700">{new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(subtotal)}</div>
      </div>

      <div className="flex items-center gap-2 md:col-span-2 justify-end">
        {onRemove ? <button type="button" onClick={onRemove} className="rounded-md border border-red-300 bg-white px-3 py-1 text-sm text-red-700">Remove</button> : null}
      </div>
    </div>
  )
}
