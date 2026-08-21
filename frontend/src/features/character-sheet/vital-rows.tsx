import { useQueryClient } from '@tanstack/solid-query'
import { Minus, Plus } from 'lucide-solid'
import { Show, createMemo, createSignal, onCleanup } from 'solid-js'
import { tempHpPool } from '@/entities/character/temp-hp-pool'
import type { Character } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'
import { hpFillVar } from '@/shared/ui/vital-bar'
import { type TempPoolControl, ResourceAdjustDialog } from '@/shared/ui/resource-adjust-dialog'
import { createVitalActions } from './vital-mutations'

/**
 * PV e PM com os controles de ajuste — o par que o HUD da ficha e a faixa do
 * combatente compartilham (ALE-145). Saiu do `CharacterHud` porque a faixa
 * precisa exatamente disto e de nada mais do HUD.
 *
 * `class` decide a FORMA: a faixa passa uma consulta de contêiner para pôr os
 * dois lado a lado quando a coluna dá, o HUD deixa empilhado.
 *
 * @example <VitalRows character={character} class="@[34rem]:grid-cols-2" />
 */
export function VitalRows(props: { character: Character; class?: string }) {
  const queryClient = useQueryClient()

  // Nasce UMA vez (o corpo roda uma só vez no Solid): as ações são donas do
  // timer do debounce e do instantâneo de rollback da rajada de cliques inteira.
  const vitals = createVitalActions(queryClient, () => props.character)

  const pool = createMemo(() => tempHpPool(props.character))

  /**
   * Baixar PV é DANO, não escrita de vitais: o servidor passa pelo pool de PV
   * temporários antes (uma requisição atômica). Subir é cura comum.
   */
  const applyHp = (next: number) => {
    const damage = props.character.hpCurrent - next
    if (damage <= 0) {
      vitals.setHp(next)
      return
    }
    void vitals.applyDamage(damage)
  }

  return (
    <div class={cn('grid gap-1', props.class)}>
      <VitalRow
        label="Vida"
        current={props.character.hpCurrent}
        max={props.character.hpMax}
        kind="hp"
        onSet={applyHp}
        onDamage={(amount) => void vitals.applyDamage(amount)}
        temp={pool().total}
        tempTitle={pool()
          .slices.map((slice) => slice.label)
          .join(', ')}
        tempPool={{
          total: pool().total,
          onSetManual: (value) => void vitals.setManualTempHp(value),
        }}
      />
      <VitalRow
        label="Mana"
        current={props.character.mpCurrent}
        max={props.character.mpMax}
        kind="mp"
        onSet={vitals.setMp}
      />
    </div>
  )
}

type VitalRowProps = {
  label: string
  current: number
  max: number
  kind: 'hp' | 'mp'
  onSet: (next: number) => void
  /** Routes "−" through the atomic damage endpoint (temp-first). */
  onDamage?: (amount: number) => void
  /** Debitable temp-PV pool shown as "+N". */
  temp?: number
  tempTitle?: string
  tempPool?: TempPoolControl
}

/**
 * One PV/PM row: label · decay-colored bar · current/max · −/+ · bulk edit.
 * A real `progressbar`, so the number is readable by assistive tech and by the
 * E2E suite.
 */
function VitalRow(props: VitalRowProps) {
  const percent = () =>
    props.max > 0 ? Math.max(0, Math.min(100, (props.current / props.max) * 100)) : 0
  const fillVar = () => (props.kind === 'hp' ? hpFillVar(percent()) : '--mp-arcane')
  const temp = () => props.temp ?? 0
  const delta = createVitalDelta(() => props.current)
  // Shift-click steps ±5 — combat deltas are rarely 1.
  const stepOf = (event: MouseEvent) => (event.shiftKey ? 5 : 1)

  return (
    <div class="flex items-center gap-1.5 sm:gap-2">
      <span
        class="w-9 shrink-0 text-3xs font-bold uppercase tracking-wider"
        style={{ color: `var(${fillVar()})` }}
      >
        {props.label}
      </span>
      {/* − sits on the far side of + so a greasy thumb never heals when it
          meant to hurt. Damage goes out UNCLAMPED: the server routes it
          temp-first, so a shift−5 at 3 PV still drains 5 from the pool. */}
      <Button
        type="button"
        variant="outline"
        size="icon"
        class="size-9 shrink-0 lg:size-6"
        disabled={props.current <= 0 && temp() <= 0}
        onClick={(event) =>
          props.onDamage
            ? props.onDamage(stepOf(event))
            : props.onSet(Math.max(0, props.current - stepOf(event)))
        }
        aria-label={`Reduzir ${props.label} (shift: 5)`}
      >
        <Minus aria-hidden="true" class="size-4 lg:size-3" />
      </Button>

      <div
        role="progressbar"
        aria-label={props.label}
        aria-valuenow={props.current}
        aria-valuemin={0}
        aria-valuemax={props.max}
        // O piso da barra é `min-w-4` e não `min-w-8`, e isto é conserto medido
        // (ALE-196): a 375px a linha pedia 252px numa coluna de 243, e o piso
        // antigo de 32px era o que não deixava caber — ela pintava para fora.
        //
        // Quem cede espaço é a BARRA, e não os botões: eles são alvo de toque (a
        // ALE-177 mede que 56% dos alvos da ficha estão abaixo do mínimo) e a
        // barra é indicador, com o número exato ao lado. A 375px ela fica com
        // 23px — estreita, mas desenhada.
        //
        // Um `min-w-0` na linha também foi tentado e SAIU: com o piso corrigido
        // ele não muda nada, e o teste de regressão continuou verde sem ele.
        class="relative h-3.5 min-w-4 flex-1 overflow-hidden rounded-full border border-border bg-muted lg:h-2.5"
      >
        <div
          class="h-full transition-[width,background-color] duration-500 ease-out"
          style={{ width: `${percent()}%`, 'background-color': `var(${fillVar()})` }}
        />
      </div>

      <span class="relative shrink-0 font-mono text-base tabular-nums lg:text-xs">
        <span class="font-bold">{props.current}</span>
        <span class="text-muted-foreground">/{props.max}</span>
        <Show when={temp() > 0}>
          <span class="ml-1 font-bold text-emerald-400" title={props.tempTitle ?? 'PV temporários'}>
            +{temp()}
          </span>
        </Show>
        <Show when={delta() !== null}>
          <span
            class={cn(
              'absolute -top-4 right-0 text-3xs font-bold',
              (delta() ?? 0) < 0
                ? 'text-[color:var(--hp-critical)]'
                : 'text-[color:var(--hp-full)]',
            )}
          >
            {(delta() ?? 0) > 0 ? `+${delta()}` : delta()}
          </span>
        </Show>
      </span>

      <div class="flex shrink-0 items-center gap-1 lg:gap-0.5">
        <Button
          type="button"
          variant="outline"
          size="icon"
          class="size-9 lg:size-6"
          disabled={props.current >= props.max}
          onClick={(event) => props.onSet(Math.min(props.max, props.current + stepOf(event)))}
          aria-label={`Aumentar ${props.label} (shift: 5)`}
        >
          <Plus aria-hidden="true" class="size-4 lg:size-3" />
        </Button>
        <ResourceAdjustDialog
          label={props.label}
          current={props.current}
          max={props.max}
          onSetCurrent={props.onSet}
          onDamage={props.onDamage}
          tempPool={props.tempPool}
          triggerClass="size-9 lg:size-6"
        />
      </div>
    </div>
  )
}

/**
 * Lingering delta chip: ANY change to the value — the player's own click or an
 * external one (the GM's socket damage, an expired effect) — shows "+X/−X" for
 * a few seconds, so someone who looked away still learns WHAT changed instead
 * of only seeing a different number.
 */
function createVitalDelta(current: () => number) {
  const [delta, setDelta] = createSignal<number | null>(null)
  let previous = current()
  let timer: ReturnType<typeof setTimeout> | undefined

  createMemo(() => {
    const value = current()
    const diff = value - previous
    previous = value
    if (diff === 0) return
    setDelta(diff)
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => setDelta(null), 3000)
  })

  onCleanup(() => timer && clearTimeout(timer))
  return delta
}
