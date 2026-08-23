import { ChevronDown, ChevronUp } from 'lucide-solid'
import { type ComponentProps, createSignal, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { Input } from './input'

export type NumberInputProps = Omit<
  ComponentProps<'input'>,
  'value' | 'onChange' | 'type'
> & {
  value: number | string
  onChange: (value: number) => void
  min?: number
  max?: number
  step?: number
  /** Names the spinner buttons ("Aumentar deslocamento"). Required in practice
   *  wherever a screen holds more than one — three bare "Aumentar" buttons are
   *  three identical announcements. */
  spinnerLabel?: string
}

/** Keeps a spinner step inside the field's own bounds. */
export function clampToRange(n: number, min?: number, max?: number): number {
  if (typeof min === 'number' && n < min) return min
  if (typeof max === 'number' && n > max) return max
  return n
}

/**
 * Numeric field with its own spinner column — a `0,5` step is a tap, not a
 * keyboard exercise (T20 tracks encumbrance in half-espaço steps). The native
 * spinner is unstyleable and invisible on touch, hence the pair of buttons.
 *
 * The React kit also carried an `onCommit` for "changed by spinner, not by
 * typing"; nothing in this app used it on a bag field, so it is not ported.
 *
 * O campo guarda um RASCUNHO dos estados que ainda não são número (ALE-236).
 * Sem ele, apagar para digitar outro valor é impossível: `Number('')` é ZERO,
 * o zero vai ao chamador, o chamador clampa para o mínimo dele, e o valor
 * controlado reescreve o campo no meio da digitação. Medido no Chrome, em
 * "Personagens do grupo" (mín 1, máx 8): partindo de 4, um Backspace deixava
 * `1` na tela em vez de vazio, e digitar `3` produzia `13` — clampado a `8`.
 * Quem quis 3 gravou 8, sem aviso nenhum, em TODO campo numérico do app.
 *
 * @example <NumberInput value={slots()} onChange={setSlots} min={0.5} step={0.5} />
 */
export function NumberInput(props: NumberInputProps) {
  const [local, rest] = splitProps(props, [
    'class',
    'value',
    'onChange',
    'min',
    'max',
    'step',
    'disabled',
    'spinnerLabel',
    'onBlur',
  ])
  // `null` significa "mostre o valor comprometido"; uma string significa que a
  // pessoa está no meio de digitar algo que ainda não é número.
  const [rascunho, setRascunho] = createSignal<string | null>(null)
  const exibido = () => rascunho() ?? String(local.value)
  const numeric = () => (typeof local.value === 'number' ? local.value : Number(local.value) || 0)
  const step = () => local.step ?? 1
  const atMin = () => typeof local.min === 'number' && numeric() <= local.min
  const atMax = () => typeof local.max === 'number' && numeric() >= local.max

  const adjust = (delta: number) => {
    if (local.disabled) return
    // O spinner COMPROMETE: ele parte do valor de verdade, então o rascunho
    // deixa de fazer sentido e a tela volta a mostrar o modelo.
    setRascunho(null)
    local.onChange(clampToRange(numeric() + delta, local.min, local.max))
  }

  /** Vazio e "-" são passos do caminho até um número, não valores — mostram-se
   *  na tela e NÃO sobem para o chamador. */
  const digitou = (texto: string) => {
    if (texto === '' || texto === '-') {
      setRascunho(texto)
      return
    }
    setRascunho(null)
    const numero = Number(texto)
    if (Number.isNaN(numero)) return
    local.onChange(numero)
  }

  /** Sair do campo desfaz o rascunho: um campo vazio na tela sobre um valor
   *  gravado é mentira. Nenhum chamador passa `onBlur` hoje, mas engolir o
   *  dele calado seria a próxima armadilha. */
  const saiu = (event: FocusEvent & { currentTarget: HTMLInputElement; target: HTMLInputElement }) => {
    setRascunho(null)
    const handler = local.onBlur
    if (typeof handler === 'function') handler(event)
    else if (handler) handler[0](handler[1], event)
  }

  return (
    <div class={cn('relative', local.class)}>
      <Input
        {...rest}
        type="number"
        inputMode="numeric"
        value={exibido()}
        onInput={(event) => digitou(event.currentTarget.value)}
        onBlur={saiu}
        min={local.min}
        max={local.max}
        step={step()}
        disabled={local.disabled}
        class="pr-7"
      />
      <div class="pointer-events-none absolute inset-y-0 right-0 flex w-6 flex-col">
        <SpinnerButton
          direction="up"
          name={local.spinnerLabel}
          disabled={local.disabled || atMax()}
          onClick={() => adjust(step())}
        />
        <SpinnerButton
          direction="down"
          name={local.spinnerLabel}
          disabled={local.disabled || atMin()}
          onClick={() => adjust(-step())}
        />
      </div>
    </div>
  )
}

/** Pointer affordance only — `tabIndex={-1}` leaves the keyboard to the field's
 *  own arrow keys, and the labels are pt-BR like the rest of the app. */
function SpinnerButton(props: {
  direction: 'up' | 'down'
  name?: string
  disabled?: boolean
  onClick: () => void
}) {
  const label = () =>
    [props.direction === 'up' ? 'Aumentar' : 'Diminuir', props.name].filter(Boolean).join(' ')

  return (
    <button
      type="button"
      tabIndex={-1}
      disabled={props.disabled}
      onClick={() => props.onClick()}
      aria-label={label()}
      class={cn(
        'pointer-events-auto flex flex-1 items-center justify-center text-muted-foreground transition-colors',
        'hover:bg-accent/50 hover:text-foreground',
        'disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-transparent',
        props.direction === 'up' ? 'rounded-tr-sm' : 'rounded-br-sm',
      )}
    >
      {props.direction === 'up' ? (
        <ChevronUp aria-hidden="true" class="size-3" />
      ) : (
        <ChevronDown aria-hidden="true" class="size-3" />
      )}
    </button>
  )
}
