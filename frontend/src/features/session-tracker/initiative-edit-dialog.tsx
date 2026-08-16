import { type JSX, createSignal } from 'solid-js'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { NumberInput } from '@/shared/ui/number-input'

/** A faixa jogável de iniciativa, a mesma do formulário de adicionar. */
const MIN = -5
const MAX = 40

/**
 * Corrige a iniciativa de um combatente já na lista (ALE-122).
 *
 * "Adicionar grupo" entra com 0 e o mestre não tinha como consertar: o
 * `initiative-update` existia no cliente e nunca era chamado, então a única
 * saída era remover e adicionar de novo — perdendo PV e condições no caminho.
 *
 * É DIÁLOGO e não campo na linha porque a lista se recria a cada broadcast do
 * socket: um input vivo ali perderia o foco a cada tecla, que é a armadilha do
 * `For` documentada no guia do front.
 *
 * @example <InitiativeEditDialog label="Ogro" current={12} onSave={…} trigger={…} />
 */
export function InitiativeEditDialog(props: {
  label: string
  current: number
  onSave: (initiative: number) => void
  trigger: (open: () => void) => JSX.Element
}) {
  const [open, setOpen] = createSignal(false)
  const [value, setValue] = createSignal(props.current)

  const start = () => {
    setValue(props.current) // sempre o valor de AGORA, não o de quando montou
    setOpen(true)
  }

  const save = () => {
    props.onSave(value())
    setOpen(false)
  }

  return (
    <>
      {props.trigger(start)}
      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class="max-w-xs">
          <DialogHeader>
            <DialogTitle>Iniciativa de {props.label}</DialogTitle>
            <DialogDescription>
              A ordem se reordena sozinha, e quem está na vez continua na vez.
            </DialogDescription>
          </DialogHeader>
          <NumberInput
            aria-label={`Iniciativa de ${props.label}`}
            min={MIN}
            max={MAX}
            value={value()}
            onChange={setValue}
          />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="button" onClick={save}>
              Salvar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
