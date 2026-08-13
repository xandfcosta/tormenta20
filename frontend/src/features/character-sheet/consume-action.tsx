import type { CatalogItem } from '@/shared/api/item-types'
import { type JSX, Show, createSignal } from 'solid-js'
import type { ConsumeItemInput } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { FieldFrame, isInvalid } from '@/shared/ui/field-frame'
import { NumberInput } from '@/shared/ui/number-input'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { type InstantRoll, rollValueSchema } from './consume-roll'

type Consumable = NonNullable<CatalogItem['consumable']>

const SCOPE_LABEL: Record<Consumable['scope'], string> = {
  instant: 'imediato',
  scene: '1 cena',
  day: '1 dia',
}

export type ConsumeActionProps = {
  consumable: Consumable
  itemName: string
  onConsume: (input?: ConsumeItemInput) => void
  /** Receives the opener — see `ConfirmDialog` on why these are render props. */
  trigger: (open: () => void) => JSX.Element
}

/**
 * "Usar" for a consumable. When the instant gain rolls a die (Bálsamo
 * restaurador = 2d4 PV) it asks for the player's result before spending the
 * unit; a fixed gain or a pure effect applies on the click.
 *
 * @example
 * <ConsumeAction consumable={catalog.consumable!} itemName={item.name}
 *   onConsume={(input) => actions().consume(item, input)}
 *   trigger={(open) => <Button onClick={open}>Usar</Button>} />
 */
export function ConsumeAction(props: ConsumeActionProps) {
  // Only an `instant` consumable rolls: a scene/day effect is applied by the
  // server from its own spec.
  const instant = () =>
    props.consumable.scope === 'instant' ? props.consumable.instant : undefined
  const hp = () => rollable(instant()?.hp)
  const mp = () => rollable(instant()?.mp)
  const needsRoll = () => Boolean(hp() || mp())

  return (
    <Show
      when={needsRoll()}
      fallback={
        <Tooltip>
          <TooltipTrigger as="span" class="contents">
            {props.trigger(() => props.onConsume())}
          </TooltipTrigger>
          <TooltipContent>Usar ({SCOPE_LABEL[props.consumable.scope]})</TooltipContent>
        </Tooltip>
      }
    >
      <ConsumeRollDialog
        itemName={props.itemName}
        hp={hp()}
        mp={mp()}
        trigger={props.trigger}
        onConsume={props.onConsume}
      />
    </Show>
  )
}

/** A roll is needed only when the die string isn't the fixed "0". */
function rollable(roll: InstantRoll | undefined): InstantRoll | undefined {
  return roll && roll.dice !== '0' ? roll : undefined
}

function ConsumeRollDialog(props: {
  itemName: string
  hp?: InstantRoll
  mp?: InstantRoll
  trigger: (open: () => void) => JSX.Element
  onConsume: (input?: ConsumeItemInput) => void
}) {
  const [open, setOpen] = createSignal(false)
  const [hpValue, setHpValue] = createSignal('')
  const [mpValue, setMpValue] = createSignal('')
  const [hpErrors, setHpErrors] = createSignal<string[]>([])
  const [mpErrors, setMpErrors] = createSignal<string[]>([])

  const reset = () => {
    setHpValue('')
    setMpValue('')
    setHpErrors([])
    setMpErrors([])
  }

  /** Messages for one rolled field, or [] when the die could produce it. */
  const validate = (roll: InstantRoll | undefined, raw: string): string[] => {
    if (!roll) return []
    const parsed = rollValueSchema(roll).safeParse(raw)
    return parsed.success ? [] : parsed.error.issues.map((issue) => issue.message)
  }

  const submit = (event: SubmitEvent) => {
    event.preventDefault()
    const hpIssues = validate(props.hp, hpValue())
    const mpIssues = validate(props.mp, mpValue())
    setHpErrors(hpIssues)
    setMpErrors(mpIssues)
    if (hpIssues.length > 0 || mpIssues.length > 0) return

    const input: ConsumeItemInput = {}
    if (props.hp) input.hpRolled = Number(hpValue()) + (props.hp.bonus ?? 0)
    if (props.mp) input.mpRolled = Number(mpValue()) + (props.mp.bonus ?? 0)
    props.onConsume(input)
    setOpen(false)
    reset()
  }

  return (
    <>
      {props.trigger(() => {
        reset()
        setOpen(true)
      })}
      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class="w-[calc(100vw-1.5rem)] max-w-sm">
          <DialogHeader>
            <DialogTitle>Usar {props.itemName}</DialogTitle>
          </DialogHeader>
          <form class="space-y-3" onSubmit={submit} noValidate>
            <Show when={props.hp}>
              {(roll) => (
                <RollField
                  slot="hp"
                  label="PV"
                  roll={roll()}
                  value={hpValue()}
                  onInput={setHpValue}
                  errors={hpErrors()}
                />
              )}
            </Show>
            <Show when={props.mp}>
              {(roll) => (
                <RollField
                  slot="mp"
                  label="PM"
                  roll={roll()}
                  value={mpValue()}
                  onInput={setMpValue}
                  errors={mpErrors()}
                />
              )}
            </Show>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                Cancelar
              </Button>
              <Button type="submit">Aplicar</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}

/** One die result, with the total it will restore spelled out below it — the
 *  player types the DIE, not the total, and the bonus is easy to double-add. */
function RollField(props: {
  slot: 'hp' | 'mp'
  label: string
  roll: InstantRoll
  value: string
  onInput: (value: string) => void
  errors: string[]
}) {
  const bonusLabel = () => (props.roll.bonus ? ` + ${props.roll.bonus}` : '')
  const total = () => (Number(props.value) || 0) + (props.roll.bonus ?? 0)

  return (
    <FieldFrame
      name={`roll-${props.slot}`}
      label={`Role ${props.roll.dice}${bonusLabel()} de ${props.label} e informe o resultado do dado`}
      errors={props.errors}
      hint={`${props.label} recuperado: ${total()}`}
    >
      <NumberInput
        id={`roll-${props.slot}`}
        min={0}
        max={999}
        value={props.value}
        onChange={(value) => props.onInput(String(value))}
        aria-invalid={isInvalid(props.errors)}
      />
    </FieldFrame>
  )
}
