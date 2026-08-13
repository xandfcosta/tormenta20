import type { Deus, GrantedPowerOption } from '@/shared/api/catalog-types'
import { devotoEligible } from '@/entities/character/devoto-rules'
import { For, Show } from 'solid-js'
import { deuses } from '@/shared/lib/abilities-cache'
import { grantedPowerOptionsFor } from '@/shared/lib/divine-powers-cache'
import { cn } from '@/shared/lib/utils'
import { GrantBox } from './grant-panels'

const ACTION_LABEL: Record<string, string> = {
  padrao: 'ação padrão',
  movimento: 'ação de movimento',
  livre: 'ação livre',
  reacao: 'reação',
  gratuita: 'ação gratuita',
  completa: 'ação completa',
  passivo: 'passivo',
  varia: 'varia',
}

export type DevocaoPanelProps = {
  godName: string
  value: string
  onChange: (powerName: string) => void
  raceNames: string[]
  classNames: string[]
}

/**
 * Devoção panel of the Identidade step. Becoming devoto grants ONE poder
 * concedido of the player's choice from the god's list (p96) — this captures
 * that pick. Eligibility (race/class in the god's Devotos line) is ADVISORY: an
 * incompatible pick warns "negociado com o mestre" instead of blocking, because
 * that call belongs to the table and not to the form.
 */
export function DevocaoPanel(props: DevocaoPanelProps) {
  // ~20 deuses off the primed catalog: a linear find is cheaper than a
  // module-level Map, which would evaluate before priming (gotcha #13).
  const deus = () => deuses().find((d) => d.name === props.godName)
  const powers = () => {
    const god = deus()
    return god ? grantedPowerOptionsFor(god.id) : []
  }

  return (
    <Show when={deus()}>
      {(god) => (
        <Show when={powers().length > 0}>
          <GrantBox title={`✦ Devoção a ${god().name}`}>
            <DeusFacts deus={god()} />

            <Show when={!devotoEligible(god(), props.raceNames, props.classNames)}>
              <p class="text-[11px] text-[color:var(--hp-hurt)]">
                Raça/classe fora da lista de devotos de {god().name} (p96) — negociado com o
                mestre.
              </p>
            </Show>

            <div aria-label={`Poder concedido de ${god().name}`} class="grid gap-1.5">
              <p class="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                Poder concedido · escolha 1 (p96)
              </p>
              <For each={powers()}>
                {(power) => (
                  <GrantedPowerRow
                    power={power}
                    selected={props.value === power.name}
                    onPick={() => props.onChange(props.value === power.name ? '' : power.name)}
                  />
                )}
              </For>
            </div>

            <p class="text-[10px] text-muted-foreground">
              Devoto segue as Obrigações &amp; Restrições do deus; violá-las custa todos os PM
              (p96).
            </p>
          </GrantBox>
        </Show>
      )}
    </Show>
  )
}

function DeusFacts(props: { deus: Deus }) {
  const facts = () =>
    [
      props.deus.energia && `Energia: ${props.deus.energia}`,
      props.deus.armaPreferida && `Arma: ${props.deus.armaPreferida}`,
      props.deus.portfolio,
    ].filter(Boolean)

  return (
    <Show when={facts().length > 0}>
      <p class="text-[11px] text-muted-foreground">{facts().join(' · ')}</p>
    </Show>
  )
}

function GrantedPowerRow(props: {
  power: GrantedPowerOption
  selected: boolean
  onPick: () => void
}) {
  const meta = () =>
    [ACTION_LABEL[props.power.action] ?? props.power.action, pmLabel(props.power.pmCost)]
      .filter(Boolean)
      .join(' · ')

  return (
    <button
      type="button"
      // `aria-pressed`, not `role="radio"`: the house pattern for an exclusive
      // pick built out of rich rows, and the one biome accepts on a <button>.
      aria-pressed={props.selected}
      onClick={() => props.onPick()}
      class={cn(
        'flex items-start gap-2 rounded-md border p-2 text-left transition-colors',
        props.selected
          ? 'border-grimorio-gold bg-accent'
          : 'border-grimorio-iron hover:bg-accent',
      )}
    >
      <span class="mt-1 flex size-3.5 shrink-0 items-center justify-center rounded-full border border-grimorio-iron">
        <Show when={props.selected}>
          <span class="size-2 rounded-full bg-grimorio-gold" />
        </Show>
      </span>
      <span class="min-w-0 flex-1">
        <span class="flex flex-wrap items-baseline gap-1.5">
          <span class="text-xs font-semibold">{props.power.name}</span>
          <span class="text-[10px] text-muted-foreground">· {meta()}</span>
        </span>
        <span class="block text-[11px] leading-snug text-muted-foreground">
          {props.power.description}
        </span>
      </span>
    </button>
  )
}

function pmLabel(pmCost: GrantedPowerOption['pmCost']): string | null {
  if (pmCost === 'variavel') return 'PM variável'
  return pmCost > 0 ? `${pmCost} PM` : null
}
