import { useState } from 'react'
import type { Customer } from '../../types/customer'

type Props = {
  initial?: Partial<Customer>
  submitting?: boolean
  onSubmit: (payload: Partial<Customer>) => Promise<void>
  onCancel?: () => void
}

export function CustomerForm({ initial = {}, submitting = false, onSubmit, onCancel }: Props) {
  const [code, setCode] = useState(initial.code ?? '')
  const [name, setName] = useState(initial.name ?? '')
  const [phone, setPhone] = useState(initial.phone ?? '')
  const [email, setEmail] = useState(initial.email ?? '')
  const [address, setAddress] = useState(initial.address ?? '')
  const [taxId, setTaxId] = useState(initial.tax_id ?? '')
  const [isActive, setIsActive] = useState(initial.is_active ?? true)
  const [errors, setErrors] = useState<Record<string, string>>({})

  function validate() {
    const nextErrors: Record<string, string> = {}
    if (!code.trim()) nextErrors.code = 'Code is required'
    if (!name.trim()) nextErrors.name = 'Name is required'
    setErrors(nextErrors)
    return Object.keys(nextErrors).length === 0
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!validate()) return

    await onSubmit({
      code: code.trim(),
      name: name.trim(),
      phone: phone.trim() === '' ? undefined : phone.trim(),
      email: email.trim() === '' ? undefined : email.trim(),
      address: address.trim() === '' ? undefined : address.trim(),
      tax_id: taxId.trim() === '' ? undefined : taxId.trim(),
      is_active: isActive,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="customer-code" className="block text-sm font-medium text-slate-700">Code</label>
        <input
          id="customer-code"
          value={code}
          onChange={(e) => setCode(e.target.value)}
          className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
        />
        {errors.code ? <p className="mt-1 text-sm text-red-600">{errors.code}</p> : null}
      </div>

      <div>
        <label htmlFor="customer-name" className="block text-sm font-medium text-slate-700">Name</label>
        <input
          id="customer-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
        />
        {errors.name ? <p className="mt-1 text-sm text-red-600">{errors.name}</p> : null}
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div>
          <label htmlFor="customer-phone" className="block text-sm font-medium text-slate-700">Phone</label>
          <input
            id="customer-phone"
            value={phone ?? ''}
            onChange={(e) => setPhone(e.target.value)}
            className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>
        <div>
          <label htmlFor="customer-email" className="block text-sm font-medium text-slate-700">Email</label>
          <input
            id="customer-email"
            value={email ?? ''}
            onChange={(e) => setEmail(e.target.value)}
            className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>
      </div>

      <div>
        <label htmlFor="customer-address" className="block text-sm font-medium text-slate-700">Address</label>
        <input
          id="customer-address"
          value={address ?? ''}
          onChange={(e) => setAddress(e.target.value)}
          className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
        />
      </div>

      <div>
        <label htmlFor="customer-taxid" className="block text-sm font-medium text-slate-700">Tax ID</label>
        <input
          id="customer-taxid"
          value={taxId ?? ''}
          onChange={(e) => setTaxId(e.target.value)}
          className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
        />
      </div>

      <div className="flex items-center gap-4">
        <label htmlFor="customer-active" className="flex items-center gap-2 text-sm text-slate-700">
          <input id="customer-active" type="checkbox" checked={isActive} onChange={(e) => setIsActive(e.target.checked)} />
          <span>Active</span>
        </label>
      </div>

      <div className="flex items-center gap-3 pt-2">
        <button
          disabled={submitting}
          type="submit"
          className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50 transition-colors"
        >
          {submitting ? 'Saving...' : 'Save'}
        </button>
        {onCancel ? (
          <button
            type="button"
            onClick={onCancel}
            className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 transition-colors"
          >
            Cancel
          </button>
        ) : null}
      </div>
    </form>
  )
}
