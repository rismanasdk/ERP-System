import type { Branch } from '../../types/auth'
import { OrgBadgePill } from './OrgBadgePill'

export function BranchCard({ branch, roles, onClick }: { branch: Branch; roles: string[]; onClick: () => void }) {
  const isActive = branch.is_active !== false
  return (
    <button type="button" onClick={onClick} className="w-full rounded-xl border border-slate-200 bg-white p-4 text-left shadow-sm transition hover:border-indigo-300 hover:shadow-md focus:outline-none focus:ring-2 focus:ring-indigo-500">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Branch</p>
          <h4 className="mt-1 text-lg font-semibold text-slate-900">{branch.name}</h4>
        </div>
        <OrgBadgePill label={isActive ? 'Active' : 'Inactive'} tone={isActive ? 'emerald' : 'slate'} />
      </div>

      <div className="mt-4 rounded-lg border border-slate-200 bg-slate-50 p-3">
        <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Assigned role(s)</p>
        <div className="mt-2 flex flex-wrap gap-2">
          {roles.map((role) => (
            <span key={`${branch.id}-${role}`} className="inline-flex rounded-full border border-slate-200 bg-white px-2.5 py-1 text-xs font-medium text-slate-700">
              {role}
            </span>
          ))}
        </div>
      </div>

      <div className="mt-4 text-sm text-slate-600">
        <p className="font-medium text-slate-700">Branch access</p>
        <p className="mt-1">{branch.code} • {branch.name}</p>
      </div>
    </button>
  )
}