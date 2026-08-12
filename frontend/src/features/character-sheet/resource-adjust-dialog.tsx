import { Minus, Pencil, Plus } from 'lucide-solid'
import { For, Show, createSignal } from 'solid-js'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { FieldFrame } from '@/shared/ui/field-frame'
import { Label } from '@/shared/ui/label'
import { NumberInput } from '@/shared/ui/number-input'
import { cn } from '@/shared/lib/utils'

/** Clamp a projected total into [0, max], flagging when it hit a bound so the
 *  preview can say "(limitado)" instead of silently lying. */
export function clampResource(raw: number, max: number): { value: number; clamped: boolean } {
  const value = Math.max(0, Math.min(max, raw))
  return { value, clamped: value !== raw }
}

/**
 * What the ✎ dialog will do to the bar, given the typed amount. Removals soak
 * the temp-PV pool first — the same routing the damage endpoint applies — so
 * the preview does not promise PV loss the server will absorb.
 *
 * @example adjustPreview({ mode: 'remove', amount: 8, current: 30, max: 40, tempTotal: 5 })
 * // { soak: 5, delta: -3, preview: 27, clamped: false }
 */
export function adjustPreview(input: {
  mode: 'add' | 'remove'
  amount: number
  current: number
  max: number
  tempTotal: number
}): { soak: number; delta: number; preview: number; clamped: boolean } {
  const soak = input.mode === 'remove' ? Math.min(input.amount, input.tempTotal) : 0
  // `soak - amount`, not `-(amount - soak)`: fully absorbed damage would come
  // out as -0, which reads as negative in any `Object.is` comparison downstream.
  const delta = input.mode === 'add' ? input.amount : soak - input.amount
  const { value, clamped } = clampResource(input.current + delta, input.max)
  return { soak, delta, preview: value, clamped }
}

export type TempPoolControl = { total: number; onSetManual: (value: number) => void }

export type ResourceAdjustDialogProps = {
  label: string
  current: number
  max: number
  onSetCurrent: (next: number) => void
  /** When set, "Remover" goes through the atomic damage endpoint (the pool
   *  drains first) instead of a plain vitals write. */
  onDamage?: (amount: number) => void
  /** Pool-aware VIDA dialog: current temp-PV total + the manual pool setter. */
  tempPool?: TempPoolControl
  triggerClass?: string
}

/** Most combat deltas land on one of these — no keyboard at the table. */
const QUICK_AMOUNTS = [5, 10, 20]

/**
 * Bulk PV/PM edit: pick remove/add, type (or tap) an amount, read the new total
 * before committing. The VIDA one also carries the GM's manual temp-PV field.
 */
export function ResourceAdjustDialog(props: ResourceAdjustDialogProps) {
  const [open, setOpen] = createSignal(false)
  const [mode, setMode] = createSignal<'add' | 'remove'>('remove')
  const [amount, setAmount] = createSignal(0)
  const [errors, setErrors] = createSignal<string[]>([])

  const close = () => {
    setOpen(false)
    setMode('remove')
    setAmount(0)
    setErrors([])
  }

  const preview = () =>
    adjustPreview({
      mode: mode(),
      amount: amount(),
      current: props.current,
      max: props.max,
      tempTotal: props.tempPool?.total ?? 0,
    })

  const submit = (event: SubmitEvent) => {
    event.preventDefault()
    if (!Number.isInteger(amount()) || amount() < 1) {
      setErrors(['Informe uma quantidade inteira ≥ 1.'])
      return
    }
    if (mode() === 'remove' && props.onDamage) {
      // Unclamped on purpose: the server soaks the pool first and floors at 0.
      props.onDamage(amount())
      close()
      return
    }
    props.onSetCurrent(preview().preview)
    close()
  }

  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="icon"
        class={cn('size-7', props.triggerClass)}
        aria-label={`Editar ${props.label}`}
        onClick={() => setOpen(true)}
      >
        <Pencil aria-hidden="true" class="size-3.5" />
      </Button>

      <Dialog open={open()} onOpenChange={(next) => (next ? setOpen(true) : close())}>
        <DialogContent class="w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6">
          <DialogHeader>
            <DialogTitle class="font-heading uppercase tracking-wide text-grimorio-gold">
              Ajustar {props.label}
            </DialogTitle>
          </DialogHeader>

          <form class="space-y-4" onSubmit={submit} noValidate>
            <div class="grid grid-cols-2 gap-2">
              <Button
                type="button"
                variant={mode() === 'remove' ? 'default' : 'outline'}
                class="gap-1"
                aria-pressed={mode() === 'remove'}
                onClick={() => setMode('remove')}
              >
                <Minus aria-hidden="true" class="size-4" /> Remover
              </Button>
              <Button
                type="button"
                variant={mode() === 'add' ? 'default' : 'outline'}
                class="gap-1"
                aria-pressed={mode() === 'add'}
                onClick={() => setMode('add')}
              >
                <Plus aria-hidden="true" class="size-4" /> Adicionar
              </Button>
            </div>

            <FieldFrame
              name="resource-amount"
              label="Quantidade"
              errors={errors()}
            >
              <NumberInput
                id="resource-amount"
                value={amount()}
                onChange={(value) => {
                  setAmount(value)
                  setErrors([])
                }}
                min={0}
                max={9999}
              />
              <div class="flex gap-1.5">
                <For each={QUICK_AMOUNTS}>
                  {(quick) => (
                    <button
                      type="button"
                      onClick={() => {
                        setAmount(quick)
                        setErrors([])
                      }}
                      class="rounded-md border border-border px-2.5 py-1 text-xs hover:bg-accent"
                    >
                      {quick}
                    </button>
                  )}
                </For>
              </div>
            </FieldFrame>

            <AdjustPreview
              current={props.current}
              max={props.max}
              result={preview()}
            />

            <div class="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={close}>
                Cancelar
              </Button>
              <Button type="submit">Aplicar</Button>
            </div>
          </form>

          <Show when={props.tempPool}>
            {(pool) => <ManualTempHpField pool={pool()} onDone={close} />}
          </Show>
        </DialogContent>
      </Dialog>
    </>
  )
}

function AdjustPreview(props: {
  current: number
  max: number
  result: ReturnType<typeof adjustPreview>
}) {
  return (
    <div class="flex items-center justify-between rounded-lg border border-border bg-muted px-4 py-2">
      <div class="flex flex-col">
        <span class="text-[10px] uppercase tracking-widest text-muted-foreground">novo total</span>
        <span class="text-[10px] text-muted-foreground">
          {props.current} {props.result.delta >= 0 ? '+' : '−'} {Math.abs(props.result.delta)}
          {props.result.clamped && ' (limitado)'}
          {props.result.soak > 0 && ` · PV temp. absorvem ${props.result.soak}`}
        </span>
      </div>
      {/* No aria-label: "novo total" is right beside it as visible text, and a
          bare span takes no accessible name anyway. */}
      <span class="font-mono text-2xl font-bold text-grimorio-gold">
        {props.result.preview}
        <span class="ml-1 text-sm font-normal text-muted-foreground">/ {props.max}</span>
      </span>
    </div>
  )
}

/**
 * The GM's ad-hoc temp-PV pool, inside the VIDA ✎ dialog. It SETS the pool to
 * the typed value (0 removes it); vale-o-maior (p256) is enforced server-side
 * and the helper line just reminds the table. Own state, not part of the
 * add/remove form — setting the pool is an independent action.
 */
function ManualTempHpField(props: { pool: TempPoolControl; onDone: () => void }) {
  const [value, setValue] = createSignal(props.pool.total)
  return (
    <div class="space-y-2 rounded-lg border border-border bg-muted p-3">
      <div class="flex items-baseline justify-between">
        <Label for="manual-temp-hp">PV temporários</Label>
        <span class="font-mono text-sm font-bold text-grimorio-gold">+{props.pool.total}</span>
      </div>
      <div class="flex items-center gap-2">
        <NumberInput
          id="manual-temp-hp"
          value={value()}
          onChange={setValue}
          min={0}
          max={9999}
          class="flex-1"
        />
        <Button
          type="button"
          variant="outline"
          onClick={() => {
            props.pool.onSetManual(value())
            props.onDone()
          }}
        >
          Definir
        </Button>
      </div>
      <p class="text-[10px] text-muted-foreground">
        vale o maior — não acumulam (p256) · 0 remove o valor manual
      </p>
    </div>
  )
}
