import { AlertTriangle, Check } from 'lucide-react'
import type { Pendencia } from './pendencias'

/**
 * The "what's left to decide" banner pinned above the source sub-tabs. Each
 * bullet jumps to the tab + card that owns the choice. When nothing is
 * pending it shows a compact success row (kept mounted to avoid layout jump).
 */
export function PendenciasCallout({
  pendencias,
  onJump,
}: {
  pendencias: Pendencia[]
  onJump: (pendencia: Pendencia) => void
}) {
  if (pendencias.length === 0) {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-emerald-600/30 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-700 dark:text-emerald-300">
        <Check className="size-4 shrink-0" />
        Todas as escolhas feitas.
      </div>
    )
  }
  return (
    <div className="rounded-lg border border-amber-600/40 bg-amber-500/10 px-3 py-2">
      <p className="flex items-center gap-1.5 text-xs font-semibold text-amber-700 dark:text-amber-300">
        <AlertTriangle className="size-3.5 shrink-0" /> Faltam escolhas
      </p>
      <ul className="mt-1 max-h-32 space-y-0.5 overflow-y-auto">
        {pendencias.map((p) => (
          <li key={`${p.cardId}:${p.label}`}>
            <button
              type="button"
              onClick={() => onJump(p)}
              className="text-left text-xs text-amber-800 underline-offset-2 hover:underline dark:text-amber-200"
            >
              • {p.label}
            </button>
          </li>
        ))}
      </ul>
    </div>
  )
}
