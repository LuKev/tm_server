import { forwardRef, type ButtonHTMLAttributes, type HTMLAttributes, type ReactNode } from 'react'

// Forward the grid library's ref, style and event props unchanged.
export const GamePanel = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  function GamePanel({ className = '', ...props }, ref) {
    return <div {...props} ref={ref} className={`game-panel ${className}`} />
  },
)

export function ActionButton({ className = '', type = 'button', ...props }: ButtonHTMLAttributes<HTMLButtonElement>): React.ReactElement {
  return <button {...props} type={type} className={`game-action-button ${className}`} />
}

export function ResourceAmount({ label, value, icon }: { label: string; value: ReactNode; icon: ReactNode }): React.ReactElement {
  return <span className="game-resource" title={label}>
    <span className="game-resource-icon" aria-hidden="true">{icon}</span>
    <span><span className="sr-only">{label}: </span>{value}</span>
  </span>
}
