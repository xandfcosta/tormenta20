import type { ReactNode } from 'react'

/**
 * A journal entry on the tome page: an illuminated Cinzel heading in gold with a
 * small eyebrow and an optional primary action, over the section body. Shared by
 * the Visão geral panels, the Membros roster and the Sessões log so every
 * section reads as one hand-written book rather than stacked shadcn cards.
 */
export function TomeSection({
  eyebrow,
  title,
  action,
  children,
}: {
  eyebrow: string
  title: string
  action?: ReactNode
  children: ReactNode
}) {
  return (
    <section className="space-y-4">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div className="space-y-1">
          <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
            {eyebrow}
          </p>
          <h2 className="font-heading text-xl uppercase tracking-wide text-grimorio-gold sm:text-2xl">
            {title}
          </h2>
        </div>
        {action}
      </header>
      {children}
    </section>
  )
}
