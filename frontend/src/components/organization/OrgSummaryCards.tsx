import { OrgBadgePill } from './OrgBadgePill'

type Props = {
  userName: string
  userEmail?: string
  roles: string[]
  contextLabel: string
  branchCount: number
  loading: boolean
}

export function OrgSummaryCards({ userName, userEmail, roles, contextLabel, branchCount, loading }: Props) {
  return (
    <div className="grid gap-4 md:grid-cols-3">
      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <p className="text-xs font-medium uppercase tracking-[0.2em] text-slate-500">Current user</p>
        <p className="mt-3 text-lg font-semibold text-slate-900">{userName}</p>
        <p className="mt-1 text-sm text-slate-600">{userEmail ?? 'No email available'}</p>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <p className="text-xs font-medium uppercase tracking-[0.2em] text-slate-500">Role</p>
        <div className="mt-3 flex flex-wrap gap-2">
          {roles.map((role) => <OrgBadgePill key={role} label={role} tone="indigo" />)}
        </div>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <p className="text-xs font-medium uppercase tracking-[0.2em] text-slate-500">Context</p>
        <p className="mt-3 text-lg font-semibold text-slate-900">{contextLabel}</p>
        <p className="mt-1 text-sm text-slate-600">{loading ? 'Loading branch access...' : `${branchCount} branch(es) available`}</p>
      </div>
    </div>
  )
}