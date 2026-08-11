import { AlertTriangle, Check } from 'lucide-solid'
import { For, Show } from 'solid-js'
import type { Pendencia } from './pendencias'

/**
 * The "what's left to decide" banner pinned above the source sub-tabs. Each
 * bullet jumps to the tab + card that owns the choice; with nothing pending it
 * shows a compact success row, kept mounted so the list below never jumps.
 */
export function PendenciasCallout(props: {
  pendencias: Pendencia[]
  onJump: (pendencia: Pendencia) => void
}) {
  return (
    <Show
      when={props.pendencias.length > 0}
      fallback={
        <div class="flex items-center gap-2 rounded-sm border border-emerald-600/30 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-300">
          <Check aria-hidden="true" class="size-4 shrink-0" />
          Todas as escolhas feitas.
        </div>
      }
    >
      <div class="rounded-sm border border-amber-600/40 bg-amber-500/10 px-3 py-2">
        <p class="flex items-center gap-1.5 text-xs font-semibold text-amber-300">
          <AlertTriangle aria-hidden="true" class="size-3.5 shrink-0" /> Faltam escolhas
        </p>
        <ul class="mt-1 max-h-32 space-y-0.5 overflow-y-auto">
          <For each={props.pendencias}>
            {(pendencia) => (
              <li>
                <button
                  type="button"
                  onClick={() => props.onJump(pendencia)}
                  class="text-left text-xs text-amber-200 underline-offset-2 hover:underline"
                >
                  • {pendencia.label}
                </button>
              </li>
            )}
          </For>
        </ul>
      </div>
    </Show>
  )
}
