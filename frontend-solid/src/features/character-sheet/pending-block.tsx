import type { Character } from '@/shared/api/api'

/**
 * A block whose content has not been ported yet (ALE-73 is sliced one block per
 * issue). Says which issue brings it rather than rendering an empty panel that
 * reads as breakage — the rail, the layouts and the URL contract around it are
 * real and navigable today.
 *
 * Deleted as each block lands; nothing should still import this when ALE-89
 * closes.
 */
export function PendingBlock(props: { title: string; issue: string; character: Character }) {
  return (
    <div class="flex h-full min-h-0 flex-col items-center justify-center gap-3 rounded-sm border border-dashed border-grimorio-iron p-8 text-center">
      <p class="font-heading text-xl uppercase tracking-wide text-grimorio-gold/70">{props.title}</p>
      <p class="max-w-sm text-sm text-muted-foreground">
        Este bloco de {props.character.name} chega na {props.issue}.
      </p>
    </div>
  )
}
