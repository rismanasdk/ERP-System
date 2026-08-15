type PlaceholderPageProps = {
  title: string
}

export function PlaceholderPage({ title }: PlaceholderPageProps) {
  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <p className="text-sm font-medium uppercase tracking-[0.2em] text-indigo-600">Module</p>
      <h2 className="mt-3 text-3xl font-bold text-slate-900">{title}</h2>
      <p className="mt-2 text-slate-600">This page is planned for the next phase of ERP frontend work.</p>
    </div>
  )
}
