import { useMutation, useQueryClient } from '@tanstack/react-query'
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
import { Input } from '@/shared/ui/input'
import type {
  AttributeKey,
  Character,
  CharacterExpertise,
} from '@/shared/api/api'
import { api } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { useCharacterEffects } from '@/entities/character/derived'
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
  const effects = useCharacterEffects(character)

  const customDefs: ExpertiseDef[] = character.expertises
    .filter((e) => e.custom)
    .map((e) => ({
      name: e.name,
      attribute: e.attribute,
      abbr: ATTRIBUTE_ABBR[e.attribute],
      trainedOnly: true,
    }))
  const allDefs: ExpertiseDef[] = [...EXPERTISES, ...customDefs]
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
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-amber-700/30 px-3 py-2 dark:border-amber-500/20 sm:px-4">
        <div className="flex items-baseline gap-3">
          <h2
            className={cn(
              'font-serif text-lg font-bold tracking-wide',
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
                  effects={effects}
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
  const [name, setName] = useState('')
  const [attribute, setAttribute] = useState<AttributeKey>('intelligence')
  const [error, setError] = useState<string | null>(null)

  const reset = () => {
    setName('')
    setAttribute('intelligence')
    setError(null)
  }

  const apply = () => {
    const trimmed = name.trim()
    if (!trimmed) {
      setError('Informe um nome.')
      return
    }
    onAdd({ name: trimmed, attribute }, (e) => {
      // ApiError fieldErrors surface generic message back into dialog
      setError(e.message)
    })
    setOpen(false)
    reset()
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o)
        if (!o) reset()
      }}
    >
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
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn('font-serif', accentStrong)}>
            Novo ofício
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-1">
            <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
              nome
            </span>
            <Input
              value={name}
              onChange={(e) => {
                setName(e.target.value)
                if (error) setError(null)
              }}
              placeholder="Ex: Carpintaria"
              autoFocus
              maxLength={40}
            />
          </div>
          <div className="space-y-1">
            <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
              atributo
            </span>
            <select
              value={attribute}
              onChange={(e) => setAttribute(e.target.value as AttributeKey)}
              className={cn(selectClass, 'h-9 w-full px-2 text-sm')}
            >
              {ATTRIBUTE_KEYS.map((k) => (
                <option key={k} value={k}>
                  {ATTRIBUTE_ABBR[k]} {signed(character[k])}
                </option>
              ))}
            </select>
          </div>
          {error && (
            <p className="text-xs text-destructive" role="alert">
              {error}
            </p>
          )}
          <p className={cn('text-[11px]', dimText)}>
            Ofícios só podem ser usados quando treinados.
          </p>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="button" onClick={apply}>
              Adicionar
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

