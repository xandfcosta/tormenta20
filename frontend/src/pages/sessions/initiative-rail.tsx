import { PanelLeftOpen } from 'lucide-solid'
import { For, Show } from 'solid-js'
import { initials } from '@/shared/lib/initials'
import { cn } from '@/shared/lib/utils'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import { hpFillVar } from '@/shared/ui/vital-bar'

/**
 * A fila do combate como TRILHO: o que fica permanentemente na cena do mestre
 * (ALE-198).
 *
 * A iniciativa era a espinha da tela e comia 7/12 dela mesmo fora de combate.
 * Ela virou consulta de um instante — "de quem é a vez", "quanto o ogro ainda
 * aguenta" —, e um instante não paga 420px de largura permanente. O que a
 * responde é este trilho de 64px; a lista inteira, com os botões de dano, ordem
 * e condição, vive na gaveta que o topo abre.
 *
 * Cada item ABRE a ficha, e é o mesmo gesto da peça no tabuleiro: a linha e a
 * peça são a mesma criatura, e o mestre não deveria ter dois caminhos com
 * resultados diferentes para o mesmo clique.
 *
 * @example <InitiativeRail entries={live().initiative} activeEntryId={vez()} … />
 */
export function InitiativeRail(props: {
  entries: readonly InitiativeEntry[]
  /** A linha na vez: o item ganha a marca dourada da cena. */
  activeEntryId?: string | null
  onOpenCombatant: (entryId: string) => void
  onExpand: () => void
  /** Acende a peça correspondente no tabuleiro (ALE-189). */
  onHoverEntry?: (entryId: string | null) => void
  class?: string
}) {
  return (
    <nav
      aria-label="Fila do combate"
      class={cn(
        'grimorio-frame flex w-16 shrink-0 flex-col gap-1 bg-grimorio-panel p-1',
        props.class,
      )}
    >
      <button
        type="button"
        aria-label="Abrir a iniciativa"
        onClick={props.onExpand}
        class="flex shrink-0 items-center justify-center rounded-sm border border-grimorio-iron py-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      >
        <PanelLeftOpen aria-hidden="true" class="size-4" />
      </button>

      {/* Rola só a FILA, nunca o trilho: o botão de abrir a iniciativa é o
          primeiro item do topo, e com doze combatentes ele sairia da tela junto
          com a lista — o mesmo defeito que a ALE-131 consertou na coluna. */}
      <ul class="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto">
        <For each={props.entries}>
          {(entry) => (
            <li>
              <RailEntry
                entry={entry}
                active={entry.id === props.activeEntryId}
                onOpen={() => props.onOpenCombatant(entry.id)}
                onHover={props.onHoverEntry}
              />
            </li>
          )}
        </For>
      </ul>
    </nav>
  )
}

/**
 * Um combatente em 56px: iniciais, o filete de vida e a marca da vez.
 *
 * O nome inteiro vive no `aria-label` com os PV junto, porque duas letras não
 * são um nome — quem usa leitor de tela ouve "Arcanista Erudito, PV 42 de 42" e
 * não "AE". O `title` faz o mesmo pelo ponteiro.
 */
function RailEntry(props: {
  entry: InitiativeEntry
  active: boolean
  onOpen: () => void
  onHover?: (entryId: string | null) => void
}) {
  const hp = () => props.entry.hpCurrent
  const hpMax = () => props.entry.hpMax
  const rotulo = () => {
    const vida = hp() !== undefined && hpMax() ? ` — PV ${hp()} de ${hpMax()}` : ''
    return `Abrir ${props.entry.label}${vida}${props.active ? ' — na vez' : ''}`
  }

  return (
    <button
      type="button"
      aria-label={rotulo()}
      title={props.entry.label}
      aria-current={props.active ? 'true' : undefined}
      onClick={props.onOpen}
      onPointerEnter={() => props.onHover?.(props.entry.id)}
      onPointerLeave={() => props.onHover?.(null)}
      class={cn(
        'flex w-full flex-col items-center gap-1 rounded-sm border px-1 py-1.5 transition-colors',
        props.active
          ? 'border-grimorio-gold bg-accent text-grimorio-gold'
          : 'border-grimorio-iron text-muted-foreground hover:bg-accent hover:text-foreground',
      )}
    >
      {/* Sans e não Cinzel: duas maiúsculas a 12px estão abaixo do piso de
          leitura de 14px da serifada de display (ALE-173). */}
      <span class="text-xs font-semibold tabular-nums">{initials(props.entry.label)}</span>
      <Show when={hpMax()}>{(max) => <RailVital current={hp() ?? 0} max={max()} />}</Show>
    </button>
  )
}

/**
 * O filete de vida do trilho.
 *
 * Não é a `VitalBar`: aquela desenha rótulo, barra e "42/42" numa linha, e nada
 * disso cabe em 56px. O que ela empresta é o que importa — `hpFillVar`, a
 * mesma função que decide a COR por faixa, para que "quase morto" tenha o
 * mesmo vermelho aqui e na gaveta. Um segundo limiar escrito à mão seria duas
 * verdades sobre o mesmo combatente.
 *
 * Sem `role="progressbar"`: o número já vai no nome acessível do botão, e um
 * medidor aninhado num botão faria o leitor de tela anunciar a vida duas vezes.
 */
function RailVital(props: { current: number; max: number }) {
  const percent = () => Math.max(0, Math.min(100, (props.current / props.max) * 100))

  return (
    <div aria-hidden="true" class="h-1 w-full overflow-hidden rounded-full bg-muted">
      <div
        class="h-full rounded-full transition-[width]"
        style={{ width: `${percent()}%`, background: `var(${hpFillVar(percent())})` }}
      />
    </div>
  )
}
