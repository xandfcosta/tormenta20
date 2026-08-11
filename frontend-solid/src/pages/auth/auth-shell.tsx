import type { JSX, ParentProps } from 'solid-js'
import { Show } from 'solid-js'

/**
 * Split-screen frame shared by /login and /register: a brand panel on the left
 * (lg+) and the form on the right; on phones the panel is hidden and its
 * wordmark collapses above the form.
 *
 * Pure layout — no routing or auth here.
 */
export function AuthShell(
  props: ParentProps<{ title: string; subtitle?: string; footer?: JSX.Element }>,
) {
  return (
    <div class="grid min-h-dvh lg:grid-cols-2">
      <AuthBrandPanel />
      <main class="flex flex-col overflow-y-auto px-6 py-10 sm:px-10">
        <div class="mx-auto flex w-full max-w-sm flex-1 flex-col justify-center gap-6">
          <p class="text-lg font-semibold tracking-tight lg:hidden">Tormenta 20</p>
          <div class="space-y-1.5">
            <h1 class="text-2xl font-semibold tracking-tight">{props.title}</h1>
            <Show when={props.subtitle}>
              {(subtitle) => <p class="text-sm text-muted-foreground">{subtitle()}</p>}
            </Show>
          </div>
          {props.children}
          <Show when={props.footer}>
            {(footer) => <p class="text-center text-sm text-muted-foreground">{footer()}</p>}
          </Show>
        </div>
      </main>
    </div>
  )
}

/** Left brand column — desktop only; phones show the wordmark inline instead. */
function AuthBrandPanel() {
  return (
    <aside class="hidden flex-col justify-between border-r border-border bg-muted p-10 lg:flex">
      <p class="text-lg font-semibold tracking-tight">Tormenta 20</p>
      <div class="space-y-3">
        <p class="text-3xl font-semibold leading-tight tracking-tight">Sua mesa, organizada.</p>
        <p class="max-w-sm text-muted-foreground">
          Fichas, campanhas e sessões ao vivo — tudo em um só lugar.
        </p>
      </div>
      <p class="text-xs text-muted-foreground">Gerenciador não-oficial de Tormenta 20.</p>
    </aside>
  )
}
