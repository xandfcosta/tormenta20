import { Sword } from 'lucide-solid'
import { For, Show, createMemo } from 'solid-js'
import { weaponCardsFor } from '@/entities/character/weapon-cards'
import type { WeaponCard } from '@/shared/lib/computed-sheet-v2'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/shared/ui/dialog'
import type { StatsProps } from './combat-stats'
import { signed } from './signed'
import { StatRowList } from './stat-box'
import { critLabel, weaponCardRows } from './stat-rows'

/**
 * The wielded weapons as ready-to-roll formulas — "Machado · +10 · 1d12+7 ·
 * 19-20/x3" — so landing a hit never costs a block switch to find the damage
 * dice. The numbers come from the Go engine; the dialog shows where each came
 * from.
 */
export function WeaponFormulaCards(props: StatsProps) {
  const cards = createMemo(() => weaponCardsFor(props.character, props.activeConditionals))

  return (
    <Show
      when={cards().length > 0}
      fallback={
        <p class="self-center text-center text-xs italic text-muted-foreground">
          Nenhuma arma empunhada.
        </p>
      }
    >
      <div
        class="grid gap-2"
        style={{ 'grid-template-columns': `repeat(${cards().length}, 1fr)` }}
      >
        <For each={cards()}>{(card) => <WeaponCardTile card={card} />}</For>
      </div>
    </Show>
  )
}

function WeaponCardTile(props: { card: WeaponCard }) {
  const bonus = () => props.card.damageBonus
  const damageLabel = () => `${props.card.damage}${bonus() !== 0 ? signed(bonus()) : ''}`
  const rows = createMemo(() => weaponCardRows(props.card))

  return (
    <Dialog>
      <DialogTrigger
        as="button"
        type="button"
        aria-label={`Detalhamento de ${props.card.name}`}
        title={`${props.card.skill} ${signed(props.card.attack)} · dano ${damageLabel()} · crítico ${critLabel(props.card)}`}
        class="flex cursor-pointer flex-col items-center rounded-none border-2 border-destructive/50 bg-grimorio-panel p-2 text-center transition-colors hover:bg-destructive/10"
      >
        <span class="max-w-full truncate text-4xs font-bold uppercase tracking-widest text-destructive/80">
          {props.card.name}
        </span>
        <span class="mt-0.5 font-mono text-sm font-bold leading-tight text-foreground">
          {signed(props.card.attack)} · {damageLabel()}
        </span>
        <span class="text-3xs text-muted-foreground">{critLabel(props.card)}</span>
      </DialogTrigger>

      <DialogContent class="w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2 font-heading uppercase tracking-wide text-grimorio-gold">
            <Sword aria-hidden="true" class="size-3.5" />
            {props.card.name}
          </DialogTitle>
        </DialogHeader>
        <div class="space-y-3 text-sm">
          <div>
            <p class="text-xs font-bold uppercase tracking-widest text-destructive/80">
              Ataque ({props.card.skill}) {signed(props.card.attack)}
            </p>
            <div class="mt-1">
              <StatRowList rows={rows().attackRows} />
            </div>
          </div>
          <div>
            <p class="text-xs font-bold uppercase tracking-widest text-destructive/80">
              Dano {damageLabel()} · crítico {critLabel(props.card)}
            </p>
            <div class="mt-1">
              <Show
                when={rows().damageRows.length > 0}
                fallback={<p class="text-xs text-muted-foreground">Só o dado da arma.</p>}
              >
                <StatRowList rows={rows().damageRows} />
              </Show>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
