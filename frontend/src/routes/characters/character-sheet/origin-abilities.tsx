import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Check } from 'lucide-react'
import { getOrigin } from '@tormenta20/t20-data'
import type {
  OriginBenefit,
  OriginDefinition,
} from '@tormenta20/t20-data'
import { api, type Character } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/shared/lib/character-cache'
import { characterQueryOptions } from '@/shared/lib/queries'
import { accentTitle, dimText, subtleText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { AbilitiesSection } from './abilities-section'
import { parseChoices } from './parse-choices'

const ORIGIN_BENEFIT_LIMIT = 2

/**
 * Origin section — lets the player pick `ORIGIN_BENEFIT_LIMIT`
 * benefits out of the origin's pool + the origin's unique power.
 * When the origin id isn't in the catalog, degrades to a message
 * rather than crashing the sheet.
 */
export function OriginAbilitySection({ character }: { character: Character }) {
  const origin = getOrigin(character.origin)
  if (!origin) {
    return (
      <AbilitiesSection title={`Origem: ${character.origin}`}>
        <p className={cn('text-xs italic', dimText)}>
          Origem não está no catálogo.
        </p>
      </AbilitiesSection>
    )
  }
  return <OriginPickerSection origin={origin} character={character} />
}

function OriginPickerSection({
  origin,
  character,
}: {
  origin: OriginDefinition
  character: Character
}) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const choices = parseChoices(character.originChoices)
  const pool: OriginBenefit[] = [...origin.benefits, origin.poderUnico]
  const benefitIds = new Set(pool.map((b) => b.id))
  const selected = choices.filter((id) => benefitIds.has(id))

  const update = useMutation<
    Character,
    Error,
    string[],
    { previous: Character | undefined }
  >({
    mutationFn: (next) =>
      api.characters.updateAbilityChoices(character.id, { originChoices: next }),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, originChoices: JSON.stringify(next) } : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (server) => {
      qc.setQueryData<Character>(queryKey, server)
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const toggle = (benefitId: string) => {
    const isSelected = selected.includes(benefitId)
    if (isSelected) {
      update.mutate(selected.filter((id) => id !== benefitId))
      return
    }
    if (selected.length >= ORIGIN_BENEFIT_LIMIT) return
    update.mutate([...selected, benefitId])
  }

  const remaining = ORIGIN_BENEFIT_LIMIT - selected.length

  return (
    <AbilitiesSection title={`Origem: ${origin.name}`}>
      <p className={cn('mb-2 text-[11px]', subtleText)}>
        Escolha {ORIGIN_BENEFIT_LIMIT} benefícios (perícia, poder geral, ou o
        poder único da origem). Restantes:{' '}
        <span className="font-semibold">{Math.max(0, remaining)}</span>
      </p>
      <ul className="space-y-1.5">
        {pool.map((benefit) => (
          <OriginBenefitRow
            key={benefit.id}
            benefit={benefit}
            isUnique={benefit.id === origin.poderUnico.id}
            selected={selected.includes(benefit.id)}
            atLimit={remaining <= 0}
            onToggle={() => toggle(benefit.id)}
            disabled={update.isPending}
          />
        ))}
      </ul>
    </AbilitiesSection>
  )
}

function OriginBenefitRow({
  benefit,
  isUnique,
  selected,
  atLimit,
  onToggle,
  disabled,
}: {
  benefit: OriginBenefit
  isUnique: boolean
  selected: boolean
  atLimit: boolean
  onToggle: () => void
  disabled: boolean
}) {
  const blocked = !selected && atLimit
  return (
    <li
      className={cn(
        'flex gap-2 rounded border p-2',
        selected
          ? 'border-amber-600 bg-amber-100/60 dark:border-amber-400 dark:bg-amber-500/10'
          : 'border-amber-700/20 dark:border-amber-500/20',
      )}
    >
      <button
        type="button"
        onClick={onToggle}
        disabled={disabled || blocked}
        className={cn(
          'mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border text-[10px]',
          selected
            ? 'border-amber-700 bg-amber-600 text-white'
            : 'border-amber-700/50 hover:bg-amber-100 dark:border-amber-500/50',
          (disabled || blocked) && 'cursor-not-allowed opacity-40',
        )}
        aria-pressed={selected}
        aria-label={selected ? 'Remover benefício' : 'Selecionar benefício'}
      >
        {selected ? <Check className="size-3" /> : null}
      </button>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1">
          <p className={cn('text-xs font-semibold', accentTitle)}>{benefit.name}</p>
          <span
            className={cn(
              'rounded px-1 text-[9px] uppercase tracking-wide',
              benefit.kind === 'pericia'
                ? 'bg-emerald-200/60 text-emerald-900 dark:bg-emerald-500/20 dark:text-emerald-100'
                : 'bg-violet-200/60 text-violet-900 dark:bg-violet-500/20 dark:text-violet-100',
            )}
          >
            {benefit.kind === 'pericia' ? 'Perícia' : 'Poder'}
          </span>
          {isUnique && (
            <span className="rounded bg-amber-300/60 px-1 text-[9px] font-semibold uppercase tracking-wide text-amber-900 dark:bg-amber-500/30 dark:text-amber-100">
              Único
            </span>
          )}
        </div>
        <p className={cn('mt-0.5 text-[11px] leading-snug', subtleText)}>
          {benefit.description}
        </p>
      </div>
    </li>
  )
}
