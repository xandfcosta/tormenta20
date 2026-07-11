import type { ReactNode } from 'react'
import { accentStrong, surface } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'

/**
 * Shell used by every ability sub-section (race, origin, class). Kept
 * in its own module to avoid a cycle: `abilities-panel` imports the
 * three subtree modules, and each subtree module imports this shell.
 */
export function AbilitiesSection({
  title,
  children,
}: {
  title: string
  children: ReactNode
}) {
  return (
    <section className={cn('rounded-lg border p-3', surface)}>
      <h3 className={cn('font-serif text-sm font-bold', accentStrong)}>{title}</h3>
      <div className="mt-2">{children}</div>
    </section>
  )
}
