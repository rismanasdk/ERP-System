import { useState, useCallback, createContext, useContext, type ReactNode } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from '../components/ui/dialog'

type ConfirmOptions = {
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'default' | 'destructive'
}

type ConfirmContextType = (options: ConfirmOptions) => Promise<boolean>

const ConfirmContext = createContext<ConfirmContextType | null>(null)

export function ConfirmDialogProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<{ options: ConfirmOptions; resolve: (v: boolean) => void } | null>(null)

  const confirm = useCallback((options: ConfirmOptions) => {
    return new Promise<boolean>((resolve) => {
      setState({ options, resolve })
    })
  }, [])

  const handleClose = (result: boolean) => {
    state?.resolve(result)
    setState(null)
  }

  return (
    <ConfirmContext.Provider value={confirm}>
      {children}
      <Dialog open={!!state} onOpenChange={(open) => !open && handleClose(false)}>
        {state && (
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{state.options.title}</DialogTitle>
              {state.options.description && <DialogDescription>{state.options.description}</DialogDescription>}
            </DialogHeader>
            <DialogFooter>
              <DialogClose asChild>
                <button
                  onClick={() => handleClose(false)}
                  className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm"
                >
                  {state.options.cancelLabel ?? 'Cancel'}
                </button>
              </DialogClose>
              <button
                onClick={() => handleClose(true)}
                className={
                  state.options.variant === 'destructive'
                    ? 'rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-500'
                    : 'rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500'
                }
              >
                {state.options.confirmLabel ?? 'Confirm'}
              </button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </ConfirmContext.Provider>
  )
}

export function useConfirm() {
  const ctx = useContext(ConfirmContext)
  if (!ctx) throw new Error('useConfirm must be used within ConfirmDialogProvider')
  return ctx
}