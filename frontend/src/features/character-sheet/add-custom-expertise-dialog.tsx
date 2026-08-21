import { Plus } from 'lucide-solid'
import { For, Show, createSignal } from 'solid-js'
import { z } from 'zod'
import { ATTRIBUTE_ABBR, ATTRIBUTE_KEYS } from '@/entities/character/expertise'
import type { AttributeKey } from '@/shared/api/api'
import { type FieldErrors, toSubmitFailure } from '@/shared/lib/form-errors'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { TextField } from '@/shared/ui/text-field'

/**
 * A custom "ofício": free-text name plus the attribute it keys off. The
 * attribute comes from a fixed list, so only the name needs validating.
 */
const customExpertiseSchema = z.object({
  name: z.string().trim().min(1, 'Informe um nome.').max(40, 'Máximo 40 caracteres.'),
})

export type AddCustomExpertiseDialogProps = {
  onAdd: (input: { name: string; attribute: AttributeKey }) => Promise<void>
}

/** Mints a perícia the book does not have — a smith's trade, a sailor's craft. */
export function AddCustomExpertiseDialog(props: AddCustomExpertiseDialogProps) {
  const [open, setOpen] = createSignal(false)
  const [name, setName] = createSignal('')
  const [attribute, setAttribute] = createSignal<AttributeKey>('intelligence')
  const [fieldErrors, setFieldErrors] = createSignal<FieldErrors>({})
  const [formError, setFormError] = createSignal<string | null>(null)
  const [pending, setPending] = createSignal(false)

  const close = () => {
    setOpen(false)
    setName('')
    setFieldErrors({})
    setFormError(null)
  }

  const submit = async (event: SubmitEvent) => {
    event.preventDefault()
    setFormError(null)
    const parsed = customExpertiseSchema.safeParse({ name: name() })
    if (!parsed.success) {
      setFieldErrors(z.flattenError(parsed.error).fieldErrors as FieldErrors)
      return
    }
    setFieldErrors({})
    setPending(true)
    try {
      await props.onAdd({ name: parsed.data.name, attribute: attribute() })
      close()
    } catch (failure) {
      setFormError(toSubmitFailure(failure).formError ?? 'Não foi possível criar a perícia.')
    } finally {
      setPending(false)
    }
  }

  return (
    <Dialog open={open()} onOpenChange={(next) => (next ? setOpen(true) : close())}>
      <DialogTrigger as={Button} variant="outline" size="sm" aria-label="Nova perícia">
        <Plus aria-hidden="true" class="size-4" />
      </DialogTrigger>
      <DialogContent class="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle class="font-heading uppercase tracking-wide text-grimorio-gold">
            Nova perícia
          </DialogTitle>
        </DialogHeader>
        <form class="space-y-4" onSubmit={submit} noValidate>
          <TextField
            name="expertise-name"
            label="Nome"
            value={name()}
            onInput={setName}
            errors={fieldErrors().name}
          />
          <div class="space-y-2">
            <label
              for="expertise-attribute"
              class="text-sm font-medium leading-none text-foreground"
            >
              Atributo
            </label>
            <select
              id="expertise-attribute"
              value={attribute()}
              onChange={(event) => setAttribute(event.currentTarget.value as AttributeKey)}
              class="h-9 w-full cursor-pointer rounded-sm border border-input bg-transparent px-3 text-sm outline-none focus:ring-2 focus:ring-ring"
            >
              <For each={ATTRIBUTE_KEYS}>
                {(key) => <option value={key}>{ATTRIBUTE_ABBR[key]}</option>}
              </For>
            </select>
          </div>
          <Show when={formError()}>
            {(message) => <p class="text-sm text-destructive">{message()}</p>}
          </Show>
          <div class="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={close}>
              Cancelar
            </Button>
            <Button type="submit" disabled={pending()}>
              {pending() ? 'Criando…' : 'Criar'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
