import type { ReactNode } from 'react'

/**
 * AuthShell — split-screen frame shared by /login and /register.
 *
 * Desktop (lg+): two columns — a brand panel on the left, the form on the
 * right. Phone: the brand panel is hidden and its wordmark collapses into a
 * compact header above the form, which scrolls if it overflows. Rendered in
 * the root's `bare` shell (no app nav), so it owns the whole viewport.
 *
 * Pure layout: pages pass their heading + form in; no routing/auth here.
 */
export function AuthShell({
  title,
  subtitle,
  children,
  footer,
}: {
  title: string
  subtitle?: string
  children: ReactNode
  footer?: ReactNode
}) {
  return (
    <div className="grid min-h-0 flex-1 lg:grid-cols-2">
      <AuthBrandPanel />
      <main className="flex flex-col overflow-y-auto px-6 py-10 sm:px-10">
        <div className="mx-auto flex w-full max-w-sm flex-1 flex-col justify-center gap-6">
          <p className="text-lg font-semibold tracking-tight lg:hidden">
            Tormenta 20
          </p>
          <div className="space-y-1.5">
            <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
            {subtitle && (
              <p className="text-sm text-muted-foreground">{subtitle}</p>
            )}
          </div>
          {children}
          {footer && (
            <p className="text-center text-sm text-muted-foreground">{footer}</p>
          )}
        </div>
      </main>
    </div>
  )
}

/** Left brand column — desktop only; phone shows the wordmark inline instead. */
function AuthBrandPanel() {
  return (
    <aside className="hidden flex-col justify-between border-r border-border bg-muted p-10 lg:flex">
      <p className="text-lg font-semibold tracking-tight">Tormenta 20</p>
      <div className="space-y-3">
        <p className="text-3xl font-semibold leading-tight tracking-tight">
          Sua mesa, organizada.
        </p>
        <p className="max-w-sm text-muted-foreground">
          Fichas, campanhas e sessões ao vivo — tudo em um só lugar.
        </p>
      </div>
      <p className="text-xs text-muted-foreground">
        Gerenciador não-oficial de Tormenta 20.
      </p>
    </aside>
  )
}
