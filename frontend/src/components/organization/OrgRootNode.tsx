import { OrgBadgePill } from './OrgBadgePill'

type Props = { isSuperAdmin: boolean; userName: string; rolesLabel: string }

export function OrgRootNode({ isSuperAdmin, userName, rolesLabel }: Props) {
  return (
    <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-xs uppercase tracking-[0.2em] text-slate-500">Root</p>
          <p className="mt-1 text-lg font-semibold text-slate-900">{isSuperAdmin ? 'SUPER ADMIN' : 'Organization'}</p>
        </div>
        <OrgBadgePill label={isSuperAdmin ? 'Administrative access' : 'Standard access'} tone="indigo" />
      </div>

      <div className="mt-4 rounded-xl border border-slate-200 bg-white p-4">
        <p className="text-sm font-medium text-slate-700">Current user</p>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <span className="text-base font-semibold text-slate-900">{userName}</span>
          <span className="text-slate-400">•</span>
          <span className="text-sm text-slate-600">{rolesLabel}</span>
        </div>
      </div>
    </div>
  )
}