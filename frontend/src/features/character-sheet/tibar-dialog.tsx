import { Pencil } from 'lucide-solid'
import { createSignal } from 'solid-js'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { FieldFrame, isInvalid } from '@/shared/ui/field-frame'
import { NumberInput } from '@/shared/ui/number-input'
import { toSubmitFailure } from '@/shared/lib/form-errors'
import {
  ITEM_DIALOG_CONTENT,
  ITEM_DIALOG_TITLE,
  ItemDialogFooter,
} from './item-dialog-kit'
import type { SetTibar } from './tibar-write'

/** O mesmo teto do servidor (`maxTibar`), para a recusa aparecer antes da rede. */
const MAX_TIBAR = 1_000_000

export type TibarDialogProps = {
  tibar: number
  onSave: SetTibar
}

/**
 * "Editar tibares" — o único caminho de atualizar o dinheiro depois da Forja.
 *
 * O campo é o SALDO, e não um par ganhar/gastar: é o mesmo controle e o mesmo
 * número que a Forja preenche pela Tabela 3-1 (p140), e duas gramáticas para um
 * campo só fariam a ficha e a Forja discordarem sobre o que "tibares" quer
 * dizer.
 *
 * @example <TibarDialog tibar={character.tibar} onSave={setTibar} />
 */
export function TibarDialog(props: TibarDialogProps) {
  const [open, setOpen] = createSignal(false)
  const [value, setValue] = createSignal(props.tibar)
  const [fieldError, setFieldError] = createSignal<string[]>([])
  const [formError, setFormError] = createSignal<string | null>(null)
  const [pending, setPending] = createSignal(false)

  /** Reabrir mostra o saldo de AGORA, nunca o rascunho da tentativa anterior. */
  const start = () => {
    setValue(props.tibar)
    setFieldError([])
    setFormError(null)
    setOpen(true)
  }

  const submit = async (event: SubmitEvent) => {
    event.preventDefault()
    // O NumberInput não apara o que se DIGITA — `min`/`max` governam só os
    // botões de passo —, então a recusa mora aqui (ALE-213).
    if (!Number.isFinite(value()) || value() < 0 || value() > MAX_TIBAR) {
      setFieldError([`Informe um valor entre 0 e ${MAX_TIBAR.toLocaleString('pt-BR')}.`])
      return
    }
    setFieldError([])
    setFormError(null)
    setPending(true)
    try {
      await props.onSave(value())
      setOpen(false)
    } catch (failure) {
      setFormError(toSubmitFailure(failure).formError ?? 'Não foi possível salvar os tibares.')
    } finally {
      setPending(false)
    }
  }

  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="icon"
        class="size-6"
        aria-label="Editar tibares"
        onClick={start}
      >
        <Pencil aria-hidden="true" class="size-3" />
      </Button>

      {/* Só o botão abre — reabrir por aqui reiniciaria o rascunho no meio. */}
      <Dialog open={open()} onOpenChange={(next) => setOpen(next)}>
        <DialogContent class={ITEM_DIALOG_CONTENT}>
          <DialogHeader>
            <DialogTitle class={ITEM_DIALOG_TITLE}>Tibares</DialogTitle>
          </DialogHeader>
          <form class="space-y-4" onSubmit={submit} noValidate>
            <FieldFrame
              name="tibar-amount"
              label="T$"
              hint="Cada mil moedas ocupam 1 espaço na mochila (p141)."
              errors={fieldError()}
            >
              <NumberInput
                id="tibar-amount"
                value={value()}
                onChange={(next) => {
                  setValue(next)
                  setFieldError([])
                }}
                min={0}
                max={MAX_TIBAR}
                step={1}
                spinnerLabel="tibares"
                aria-invalid={isInvalid(fieldError())}
              />
            </FieldFrame>
            <DialogInlineError message={formError()} />
            <ItemDialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                Cancelar
              </Button>
              <Button type="submit" disabled={pending()}>
                {pending() ? 'Salvando…' : 'Salvar'}
              </Button>
            </ItemDialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}
