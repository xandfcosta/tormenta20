import { ATTRIBUTE_ABBR, ATTRIBUTE_KEYS, type AttributeKey } from '@/shared/api/attribute-keys'
import { For, type JSX, Show } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { FieldLabel } from '@/shared/ui/section-label'

/**
 * O seletor do bônus de atributo de uma raça — as pastilhas de `+1` e os
 * cartões de ascendência.
 *
 * Vive em `shared/ui` porque DUAS telas precisam dele e elas são features
 * irmãs, que a FSD proíbe de se importarem: a forja, onde a escolha nasce, e a
 * ficha, onde ela pode ser terminada. Antes só a forja tinha o controle, e o
 * Resumo prometia "dá para criar assim e terminar na ficha" para uma ficha que
 * não tinha onde (ALE-169).
 *
 * Não sabe nada de catálogo: recebe a conta pronta. Quem sabe qual raça pede o
 * quê é `raceChoiceMeta`, e cada lado o chama do seu lugar.
 */
export type FloatingPickerProps = {
  /** Quantas escolhas a raça pede, e quanto cada uma vale. */
  count: number
  value: number
  /** Atributo que esta raça não pode aumentar, se houver. */
  exclude?: AttributeKey
  /** Penalidade fixa que acompanha o bônus, se houver. */
  penalty?: { attribute: AttributeKey; value: number }
  picks: readonly AttributeKey[]
  onChange: (next: AttributeKey[]) => void
}

export function RaceFloatingPicker(props: FloatingPickerProps) {
  // O atributo excluído nunca conta para a cota, mesmo que um rascunho velho o
  // carregue — senão o contador reivindicaria uma escolha que não concede nada.
  const placed = () => props.picks.filter((a) => a !== props.exclude).length

  const toggle = (attr: AttributeKey) => {
    if (attr === props.exclude) return
    if (props.picks.includes(attr)) {
      props.onChange(props.picks.filter((a) => a !== attr))
      return
    }
    if (placed() < props.count) props.onChange([...props.picks, attr])
  }

  return (
    <div class="space-y-1.5">
      <FieldLabel as="p" class="text-2xs font-semibold">
        Distribua +{props.value} em {props.count} atributos · {placed()}/{props.count}
      </FieldLabel>
      <div class="flex flex-wrap gap-1.5">
        <For each={ATTRIBUTE_KEYS}>
          {(attr) => {
            const excluded = () => attr === props.exclude
            const selected = () => props.picks.includes(attr)
            const full = () => placed() >= props.count && !selected()
            return (
              <button
                type="button"
                aria-pressed={selected()}
                disabled={excluded() || full()}
                onClick={() => toggle(attr)}
                title={excluded() ? `Não pode aumentar ${ATTRIBUTE_ABBR[attr]}` : undefined}
                class={cn(
                  'rounded-sm border px-2 py-1 font-mono text-xs transition-colors',
                  selected()
                    ? 'border-grimorio-gold bg-accent text-grimorio-gold'
                    : 'border-grimorio-iron',
                  (excluded() || full()) && 'opacity-40',
                )}
              >
                {ATTRIBUTE_ABBR[attr]}
              </button>
            )
          }}
        </For>
      </div>
      <Show when={props.penalty}>
        {(penalty) => (
          <p class="text-2xs text-muted-foreground">
            Penalidade fixa:{' '}
            <span class="font-mono">
              −{Math.abs(penalty().value)} {ATTRIBUTE_ABBR[penalty().attribute]}
            </span>
          </p>
        )}
      </Show>
    </div>
  )
}

export type SubracePickerProps = {
  options: readonly string[]
  value?: string
  onChange: (next: string) => void
  /** Prévia opcional do que a ascendência vale. A forja mostra os deltas; a
   *  ficha não, porque lá o bloco de atributos ao lado já os mostra aplicados. */
  preview?: (option: string) => JSX.Element
}

export function RaceSubracePicker(props: SubracePickerProps) {
  return (
    <div class="space-y-1.5">
      <FieldLabel as="p" class="text-2xs font-semibold">
        Escolha a ascendência
      </FieldLabel>
      <div class="grid gap-1.5 sm:grid-cols-2">
        <For each={props.options}>
          {(option) => (
            <button
              type="button"
              aria-pressed={props.value === option}
              onClick={() => props.onChange(option)}
              class={cn(
                'space-y-1 rounded-sm border p-2 text-left transition-colors',
                props.value === option
                  ? 'border-grimorio-gold bg-accent'
                  : 'border-grimorio-iron hover:bg-accent',
              )}
            >
              <p class="text-xs font-semibold capitalize">{option}</p>
              {props.preview?.(option)}
            </button>
          )}
        </For>
      </div>
    </div>
  )
}
