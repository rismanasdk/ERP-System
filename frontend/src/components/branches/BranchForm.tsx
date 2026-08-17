import { useState } from 'react'
import type { Branch } from '../../types/auth'

type Props = {
  initial?: Partial<Branch>
  submitting?: boolean
  onSubmit: (payload: Partial<Branch>) => Promise<void>
  onCancel?: () => void
}

export function BranchForm({ initial = {}, submitting = false, onSubmit, onCancel }: Props) {
  const [name, setName] = useState(initial.name ?? '')
  const [code, setCode] = useState(initial.code ?? '')
  const [isActive, setIsActive] = useState(initial.is_active ?? true)
  const [errors, setErrors] = useState<Record<string, string>>({})

  function validate() {
    const nextErrors: Record<string, string> = {}
    if (!name.trim()) nextErrors.name = 'Name is required'
    if (!code.trim()) nextErrors.code = 'Code is required'
    setErrors(nextErrors)
    return Object.keys(nextErrors).length === 0
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!validate()) return

    await onSubmit({
      name: name.trim(),
      code: code.trim(),
      is_active: isActive,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="branch-name" className="block text-sm font-medium text-slate-700">Name</label>
        <input
          id="branch-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
        />
        {errors.name ? <p className="mt-1 text-sm text-red-600">{errors.name}</p> : null}
      </div>

      <div>
        <label htmlFor="branch-code" className="block text-sm font-medium text-slate-700">Code</label>
        <input
          id="branch-code"
          value={code}
          onChange={(e) => setCode(e.target.value)}
          className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
        />
        {errors.code ? <p className="mt-1 text-sm text-red-600">{errors.code}</p> : null}
      </div>

      <div className="flex items-center gap-4">
        <label htmlFor="branch-active" className="flex items-center gap-2 text-sm text-slate-700">
          <input id="branch-active" type="checkbox" checked={isActive} onChange={(e) => setIsActive(e.target.checked)} />
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
