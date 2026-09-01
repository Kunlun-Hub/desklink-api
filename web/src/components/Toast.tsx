import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { CircleCheck, CircleX, Info, X } from 'lucide-react'

type ToastKind = 'success' | 'error' | 'info'
interface ToastItem { id: number; message: string; kind: ToastKind }
interface ToastValue { show: (message: string, kind?: ToastKind) => void }

const ToastContext = createContext<ToastValue | null>(null)

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([])
  const show = useCallback((message: string, kind: ToastKind = 'info') => {
    const id = Date.now() + Math.random()
    setItems((current) => [...current, { id, message, kind }])
    window.setTimeout(() => setItems((current) => current.filter((item) => item.id !== id)), 3600)
  }, [])
  const value = useMemo(() => ({ show }), [show])

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast toast-end z-[100]">
        {items.map((item) => {
          const Icon = item.kind === 'success' ? CircleCheck : item.kind === 'error' ? CircleX : Info
          return (
            <div key={item.id} className={`alert py-3 shadow-lg ${item.kind === 'success' ? 'alert-success' : item.kind === 'error' ? 'alert-error' : 'alert-info'}`}>
              <Icon size={18} />
              <span className="text-sm">{item.message}</span>
              <button className="btn btn-ghost btn-xs" onClick={() => setItems((current) => current.filter((entry) => entry.id !== item.id))} aria-label="关闭"><X size={15} /></button>
            </div>
          )
        })}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast() {
  const value = useContext(ToastContext)
  if (!value) throw new Error('useToast must be used inside ToastProvider')
  return value
}
