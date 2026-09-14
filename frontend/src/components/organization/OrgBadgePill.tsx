import type { OrgBadgeTone } from '../../types/organization'

const toneClasses: Record<OrgBadgeTone, string> = {
  indigo: 'bg-indigo-100 text-indigo-700',
  emerald: 'bg-emerald-100 text-emerald-700',
  slate: 'bg-slate-100 text-slate-700',
}

export function OrgBadgePill({ label, tone = 'slate' }: { label: string; tone?: OrgBadgeTone }) {
  return <span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${toneClasses[tone]}`}>{label}</span>
}