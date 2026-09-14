/* eslint-disable react-refresh/only-export-components */
import { createContext, useCallback, useContext, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { Toast as ToastPrimitive } from 'radix-ui'
import { AlertTriangle, CheckCircle2, Info, X } from 'lucide-react'
import { cn } from '../../lib/utils'

type ToastVariant = 'default' | 'success' | 'destructive'

type ToastInput = {
  title: string
  description?: string
  variant?: ToastVariant
  duration?: number
}

type ToastItem = ToastInput & {
  id: string
}

type ToastContextValue = {
  toast: (toast: ToastInput) => void
}

const ToastContext = createContext<ToastContextValue>({
  toast: () => undefined,
})

function createToastId() {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }

  return `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([])

  const toast = useCallback((toastInput: ToastInput) => {
    const id = createToastId()
    setToasts((current) => [...current, { id, variant: 'default', duration: 5000, ...toastInput }])
  }, [])

  const removeToast = useCallback((id: string) => {
    setToasts((current) => current.filter((item) => item.id !== id))
  }, [])

  const value = useMemo(() => ({ toast }), [toast])

  return (
    <ToastContext.Provider value={value}>
      <ToastPrimitive.Provider swipeDirection="right">
        {children}
        <Toaster toasts={toasts} onRemove={removeToast} />
      </ToastPrimitive.Provider>
    </ToastContext.Provider>
  )
}

export function useToast() {
  return useContext(ToastContext)
}

function Toaster({ toasts, onRemove }: { toasts: ToastItem[]; onRemove: (id: string) => void }) {
  return (
    <>
      {toasts.map((toast) => (
        <AppToast key={toast.id} toast={toast} onRemove={() => onRemove(toast.id)} />
      ))}
      <ToastPrimitive.Viewport className="fixed right-4 top-4 z-[100] flex w-[calc(100vw-2rem)] max-w-sm flex-col gap-3 outline-none sm:right-6 sm:top-6 sm:w-full" />
    </>
  )
}

function AppToast({ toast, onRemove }: { toast: ToastItem; onRemove: () => void }) {
  const Icon = toast.variant === 'success' ? CheckCircle2 : toast.variant === 'destructive' ? AlertTriangle : Info

  return (
    <ToastPrimitive.Root
      duration={toast.duration}
      onOpenChange={(open) => {
        if (!open) onRemove()
      }}
      className={cn(
        'grid grid-cols-[auto_1fr_auto] items-start gap-3 rounded-lg border bg-white p-4 text-slate-900 shadow-lg',
        'data-[state=open]:animate-in data-[state=open]:slide-in-from-top-2 data-[state=closed]:animate-out data-[state=closed]:fade-out-80 data-[state=closed]:slide-out-to-right-full',
        toast.variant === 'destructive' && 'border-red-200 bg-red-50 text-red-950',
        toast.variant === 'success' && 'border-emerald-200 bg-emerald-50 text-emerald-950',
        toast.variant === 'default' && 'border-slate-200',
      )}
    >
      <Icon
        className={cn(
          'mt-0.5 h-5 w-5',
          toast.variant === 'destructive' && 'text-red-600',
          toast.variant === 'success' && 'text-emerald-600',
          toast.variant === 'default' && 'text-slate-500',
        )}
        aria-hidden="true"
      />
      <div className="min-w-0">
        <ToastPrimitive.Title className="text-sm font-semibold leading-5">{toast.title}</ToastPrimitive.Title>
        {toast.description ? (
          <ToastPrimitive.Description className="mt-1 text-sm leading-5 text-slate-600">
            {toast.description}
          </ToastPrimitive.Description>
        ) : null}
      </div>
      <ToastPrimitive.Close className="rounded-md p-1 text-slate-500 transition-colors hover:bg-black/5 hover:text-slate-900 focus:outline-none focus:ring-2 focus:ring-slate-400">
        <X className="h-4 w-4" aria-hidden="true" />
        <span className="sr-only">Close</span>
      </ToastPrimitive.Close>
    </ToastPrimitive.Root>
  )
}
