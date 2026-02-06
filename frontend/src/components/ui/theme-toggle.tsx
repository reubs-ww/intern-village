import { Sun, Moon } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ThemeToggleProps {
  checked: boolean
  onCheckedChange: () => void
  className?: string
}

export function ThemeToggle({ checked, onCheckedChange, className }: ThemeToggleProps) {
  return (
    <button
      role="switch"
      aria-checked={checked}
      aria-label="Toggle dark mode"
      onClick={onCheckedChange}
      className={cn(
        'relative inline-flex h-7 w-12 shrink-0 cursor-pointer items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
        checked ? 'bg-slate-700' : 'bg-amber-400',
        className
      )}
    >
      {/* Track icons */}
      <Sun className="absolute left-1.5 h-3.5 w-3.5 text-amber-100" />
      <Moon className="absolute right-1.5 h-3.5 w-3.5 text-slate-400" />

      {/* Thumb */}
      <span
        className={cn(
          'pointer-events-none flex h-5 w-5 items-center justify-center rounded-full bg-white shadow-lg ring-0 transition-transform',
          checked ? 'translate-x-6' : 'translate-x-1'
        )}
      >
        {checked ? (
          <Moon className="h-3 w-3 text-slate-700" />
        ) : (
          <Sun className="h-3 w-3 text-amber-500" />
        )}
      </span>
    </button>
  )
}
