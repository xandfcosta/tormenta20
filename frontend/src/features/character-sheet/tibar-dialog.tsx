import { Pencil } from 'lucide-solid'
import { For, createSignal } from 'solid-js'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { FieldFrame, isInvalid } from '@/shared/ui/field-frame'
import { NumberInput } from '@/shared/ui/number-input'
import { cn } from '@/shared/lib/utils'
import { formatTibar } from '@/shared/lib/format-tibar'
import { toSubmitFailure } from '@/shared/lib/form-errors'
import {
  ITEM_DIALOG_CONTENT,
  ITEM_DIALOG_TITLE,
  ItemDialogFooter,
} from './item-dialog-kit'
import { type ModoDoTibar, VERBO_DO_MODO, erroDoTibar, saldoResultante } from './tibar-rules'
import type { SetTibar } from './tibar-write'

/** O mesmo teto do servidor (`maxTibar`), para a recusa aparecer antes da rede. */
const MAX_TIBAR = 1_000_000

const MODOS: readonly ModoDoTibar[] = ['receber', 'gastar', 'corrigir']

export type TibarDialogProps = {
  tibar: number
  onSave: SetTibar
}

/**
 * "Tibares" — ganhar, gastar e corrigir o dinheiro do personagem (ALE-224).
 *
 * Antes o campo era só o SALDO, e isso obrigava o jogador a fazer a conta de
 * cabeça toda vez que a mesa dizia "achamos 350" ou "paguei 80" — no meio da
 * sessão, com o mestre esperando. Os três gestos passam a conviver, e escrever
 * o total continua existindo porque é o número que a Forja preenche pela Tabela
 * 3-1 (p140) e é como se corrige um erro de digitação.
 *
 * O que torna os dois modos seguros é a PRÉVIA, e ela não é enfeite: com um
 * campo numérico só, digitar `350` querendo "ganhei 350" e gravar um saldo de
 * 350 sobre 1.200 é um erro silencioso que destrói dado. A linha
 * `T$ 1.200 → T$ 1.550` e o verbo no botão de confirmar dizem, os dois, o que
 * vai acontecer.
 *
 * O fio continua carregando SALDO (`PATCH /{id}/tibar`), nunca delta: quem soma
 * é esta tela, e a rota permanece idempotente — um delta reenviado somaria duas
 * vezes.
 *
 * @example <TibarDialog tibar={character.tibar} onSave={setTibar} />
 */
export function TibarDialog(props: TibarDialogProps) {
  const [open, setOpen] = createSignal(false)
  const [modo, setModo] = createSignal<ModoDoTibar>('receber')
  const [value, setValue] = createSignal(0)
  const [fieldError, setFieldError] = createSignal<string[]>([])
  const [formError, setFormError] = createSignal<string | null>(null)
  const [pending, setPending] = createSignal(false)

  const proximo = () => saldoResultante(props.tibar, modo(), value())

  /** Reabrir começa do zero em "Receber", nunca no rascunho da tentativa
   *  anterior — e nunca com um valor herdado que o novo modo leria como delta. */
  const start = () => {
    setModo('receber')
    setValue(0)
    setFieldError([])
    setFormError(null)
    setOpen(true)
  }

  /** Trocar de modo LIMPA o valor: `80` significa coisas diferentes em "gastar"
   *  e em "corrigir", e carregá-lo de um para o outro é o erro que a prévia
   *  existe para evitar — melhor não deixá-lo acontecer. */
  const escolheModo = (next: ModoDoTibar) => {
    setModo(next)
    setValue(next === 'corrigir' ? props.tibar : 0)
    setFieldError([])
  }

  const submit = async (event: SubmitEvent) => {
    event.preventDefault()
    // O NumberInput não apara o que se DIGITA — `min`/`max` governam só os
    // botões de passo —, então a recusa mora aqui (ALE-213).
    const erro = erroDoTibar(props.tibar, modo(), value(), MAX_TIBAR)
    if (erro) {
      setFieldError([erro])
      return
    }
    setFieldError([])
    setFormError(null)
    setPending(true)
    try {
      await props.onSave(proximo())
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
            {/* `fieldset` e não `div role="group"`: é o elemento que o biome
                pede, e dentro de um form ele é o certo para um conjunto de
                escolhas. O alternador em si é `aria-pressed` — `role="radio"`
                num botão é o que a regra do biome recusa. */}
            <fieldset class="flex gap-1 border-0 p-0">
              <legend class="sr-only">O que fazer com o dinheiro</legend>
              <For each={MODOS}>
                {(item) => (
                  <button
                    type="button"
                    aria-pressed={modo() === item}
                    onClick={() => escolheModo(item)}
                    class={cn(
                      'flex-1 rounded-sm border px-3 py-1 text-xs transition-colors',
                      modo() === item
                        ? 'border-grimorio-gold bg-accent text-grimorio-gold'
                        : 'border-grimorio-iron text-muted-foreground hover:bg-accent',
                    )}
                  >
                    {VERBO_DO_MODO[item]}
                  </button>
                )}
              </For>
            </fieldset>

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

            {/* A prévia é o que desfaz a ambiguidade dos dois modos, e por isso
                ela é TEXTO e não só cor: um leitor de tela precisa dela igual. */}
            <p class="text-xs text-muted-foreground">
              Saldo:{' '}
              <span class="font-mono">
                T$ {formatTibar(props.tibar)} → T$ {formatTibar(proximo())}
              </span>
            </p>

            <DialogInlineError message={formError()} />
            <ItemDialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                Cancelar
              </Button>
              {/* O verbo E o valor: o chip de modo já se chama "Gastar", e dois
                  botões com o mesmo nome acessível no mesmo diálogo é o que um
                  leitor de tela não consegue separar. Com o valor junto, o
                  botão de confirmar diz sozinho o que vai acontecer. */}
              <Button type="submit" disabled={pending()}>
                {pending()
                  ? 'Salvando…'
                  : `${VERBO_DO_MODO[modo()]} T$ ${formatTibar(value())}`}
              </Button>
            </ItemDialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}
