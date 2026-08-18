import { useBranch } from '../../contexts/BranchContext'

export function BranchSelector() {
  const { accessibleBranches, selectedBranch, isAllBranches, loading, error, selectBranch } = useBranch()

  const options = isAllBranches ? [{ id: -1, name: 'All Branches', code: 'ALL' }, ...accessibleBranches] : accessibleBranches

  const value = isAllBranches ? 'all' : selectedBranch ? String(selectedBranch.id) : ''

  const handleChange = (nextValue: string) => {
    if (nextValue === 'all') {
      selectBranch(null)
      return
    }

    const branch = accessibleBranches.find((item) => String(item.id) === nextValue)
    if (branch) {
      selectBranch(branch)
    }
  }

  return (
    <div className="min-w-[190px]">
      <label className="sr-only" htmlFor="branch-selector">Branch selector</label>
      <select
        id="branch-selector"
        value={value}
        onChange={(event) => handleChange(event.target.value)}
        disabled={loading || options.length === 0}
        className="w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:cursor-not-allowed disabled:bg-slate-100"
      >
        {options.length === 0 ? <option value="">No branches available</option> : null}
        {options.length > 0 && isAllBranches ? <option value="all">All Branches</option> : null}
        {options
          .filter((branch) => branch.id !== -1 || !isAllBranches)
          .map((branch) => (
            <option key={branch.id} value={String(branch.id)}>
              {branch.name}
            </option>
          ))}
      </select>
      {error ? <p className="mt-1 text-xs text-red-600">{error}</p> : null}
    </div>
  )
}
