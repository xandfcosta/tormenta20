import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from '@tanstack/react-form'
import { z } from 'zod'
import { Plus, Search } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { Field, FieldError, FieldLabel } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import type {
  AttributeKey,
  Character,
  CharacterExpertise,
} from '@/shared/api/api'
import { api } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { useComputedSheet } from '@/entities/character/computed-sheet'
import type { ExpertiseDef } from '@/entities/character/expertise'
import {
  ATTRIBUTE_ABBR,
  ATTRIBUTE_KEYS,
  EXPERTISES,
  trainingBonusForLevel,
} from '@/entities/character/expertise'
import { characterQueryOptions } from '@/entities/character/queries'
import {
  accentStrong,
  dimText,
  panelBg,
  selectClass,
  surface,
} from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { ExpertiseRow } from './expertise-row'
import { normalize } from './normalize'
import { signed } from './signed'

export function ExpertisesPanel({ character }: { character: Character }) {
  const trainingBonus = trainingBonusForLevel(character.level)
  const [query, setQuery] = useState('')
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const sheet = useComputedSheet(character)

  const customDefs: ExpertiseDef[] = character.expertises
    .filter((e) => e.custom)
    .map((e) => ({
      name: e.name,
      attribute: e.attribute,
      abbr: ATTRIBUTE_ABBR[e.attribute],
      trainedOnly: true,
    }))
  // Resistências first (audit: "teste de Reflexos!" is the hottest lookup) —
  // pinned above the alphabetical rest.
  const RESISTENCIAS = ['Fortitude', 'Reflexos', 'Vontade']
  const pinned = EXPERTISES.filter((e) => RESISTENCIAS.includes(e.name))
  const rest = EXPERTISES.filter((e) => !RESISTENCIAS.includes(e.name))
  const allDefs: ExpertiseDef[] = [...pinned, ...rest, ...customDefs]
  const filtered =
    query.trim() === ''
      ? allDefs
      : allDefs.filter((d) => normalize(d.name).includes(normalize(query)))

  const addCustom = useMutation<
    CharacterExpertise,
    Error,
    { name: string; attribute: AttributeKey },
    { previous: Character | undefined }
  >({
    mutationFn: (input) => api.characters.addExpertise(character.id, input),
    onMutate: async (input) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) => {
        if (!prev) return prev
        return {
          ...prev,
          expertises: [
            ...prev.expertises,
            {
              name: input.name.trim(),
              attribute: input.attribute,
              trained: true,
              custom: true,
            },
          ],
        }
      })
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey })
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const removeCustom = useMutation<
    { name: string },
    Error,
    string,
    { previous: Character | undefined }
  >({
    mutationFn: (name) => api.characters.deleteExpertise(character.id, name),
    onMutate: async (name) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) => {
        if (!prev) return prev
        return {
          ...prev,
          expertises: prev.expertises.filter((e) => e.name !== name),
        }
      })
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
  })

  return (
    <section
      className={cn(
        'flex h-full min-h-0 flex-1 flex-col overflow-hidden rounded-xl',
        surface,
        panelBg,
      )}
    >
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-border px-3 py-2  sm:px-4">
        <div className="flex items-baseline gap-3">
          <h2
            className={cn(
              'text-lg font-bold tracking-wide',
              accentStrong,
            )}
          >
            Perícias
          </h2>
          <p className={cn('text-[10px] sm:text-xs', dimText)}>
            treino +{trainingBonus} • ½ nível {Math.floor(character.level / 2)}
          </p>
        </div>
        <div className="flex w-full items-center gap-2 sm:w-auto">
          <div className="relative flex-1 sm:w-56 sm:flex-none">
            <Search
              className={cn(
                'pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2',
                dimText,
              )}
            />
            <Input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Buscar perícia"
              className="h-7 pl-7 text-xs"
              aria-label="Buscar perícia"
            />
          </div>
          <AddCustomExpertiseDialog
            character={character}
            onAdd={(input, fail) => addCustom.mutate(input, { onError: fail })}
          />
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-auto px-2 py-1">
        {filtered.length === 0 ? (
          <p className={cn('px-2 py-3 text-center text-xs', dimText)}>
            Nenhuma perícia para "{query}"
          </p>
        ) : (
          <div className="grid gap-2 xl:grid-cols-2">
            {filtered.map((def) => {
              const isCustom = !EXPERTISES.some((b) => b.name === def.name)
              return (
                <ExpertiseRow
                  key={def.name}
                  character={character}
                  def={def}
                  sheet={sheet}
                  onDelete={
                    isCustom ? () => removeCustom.mutate(def.name) : undefined
                  }
                />
              )
            })}
          </div>
        )}
      </div>
    </section>
  )
}

/** A custom "ofício": a free-text name plus the attribute it keys off. The
 *  attribute comes from a fixed select, so only the name needs validating. */
const customExpertiseSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, 'Informe um nome.')
    .max(40, 'Máximo 40 caracteres.'),
  attribute: z.enum(ATTRIBUTE_KEYS),
})

function AddCustomExpertiseDialog({
  character,
  onAdd,
}: {
  character: Character
  onAdd: (
    input: { name: string; attribute: AttributeKey },
    onError: (e: Error) => void,
  ) => void
}) {
  const [open, setOpen] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const form = useForm({
    defaultValues: {
      name: '',
      attribute: 'intelligence' as AttributeKey,
    },
    validators: { onSubmit: customExpertiseSchema },
    onSubmit: ({ value }) => {
      setFormError(null)
      // ApiError fieldErrors surface a generic message back into the dialog.
      onAdd({ name: value.name.trim(), attribute: value.attribute }, (e) =>
        setFormError(e.message),
      )
      setOpen(false)
      form.reset()
    },
  })

  const close = (next: boolean) => {
    setOpen(next)
    if (!next) {
      form.reset()
      setFormError(null)
    }
  }

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogTrigger asChild>
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="h-7 gap-1 text-xs"
          aria-label="Adicionar ofício"
        >
          <Plus className="size-3.5" />
          Ofício
        </Button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-border bg-muted text-foreground   ',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn(accentStrong)}>
            Novo ofício
          </DialogTitle>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            e.stopPropagation()
            form.handleSubmit()
          }}
        >
          <form.Field
            name="name"
            validators={{ onChange: customExpertiseSchema.shape.name }}
          >
            {(f) => {
              const invalid = f.state.meta.isTouched && !f.state.meta.isValid
              return (
                <Field data-invalid={invalid}>
                  <FieldLabel htmlFor={f.name}>Nome</FieldLabel>
                  <Input
                    id={f.name}
                    value={f.state.value}
                    onChange={(e) => f.handleChange(e.target.value)}
                    onBlur={f.handleBlur}
                    placeholder="Ex: Carpintaria"
                    autoFocus
                    maxLength={40}
                    aria-invalid={invalid}
                  />
                  {invalid && <FieldError errors={f.state.meta.errors} />}
                </Field>
              )
            }}
          </form.Field>
          <form.Field name="attribute">
            {(f) => (
              <Field>
                <FieldLabel htmlFor={f.name}>Atributo</FieldLabel>
                <select
                  id={f.name}
                  value={f.state.value}
                  onChange={(e) => f.handleChange(e.target.value as AttributeKey)}
                  className={cn(selectClass, 'h-9 w-full px-2 text-sm')}
                >
                  {ATTRIBUTE_KEYS.map((k) => (
                    <option key={k} value={k}>
                      {ATTRIBUTE_ABBR[k]} {signed(character[k])}
                    </option>
                  ))}
                </select>
              </Field>
            )}
          </form.Field>
          {formError && (
            <p className="text-xs text-destructive" role="alert">
              {formError}
            </p>
          )}
          <p className={cn('text-[11px]', dimText)}>
            Ofícios só podem ser usados quando treinados.
          </p>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => close(false)}>
              Cancelar
            </Button>
            <form.Subscribe
              selector={(s) => s.canSubmit}
              children={(canSubmit) => (
                <Button type="submit" disabled={!canSubmit}>
                  Adicionar
                </Button>
              )}
            />
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

