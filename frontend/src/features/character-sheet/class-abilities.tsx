import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Check } from 'lucide-react'
import {
  caminhoSlotFor,
  classPowersFor,
  devotoOptionsFor,
  generalPowersByKinds,
  slotsForClassLevel,
  unlockedKinds,
} from '@tormenta20/t20-data'
import type {
  CaminhoOption,
  ClassChoiceBlob,
  ClassChoices,
  ClassPower,
  Deus,
  GeneralPower,
  PowerKind,
} from '@tormenta20/t20-data'
import { Combobox, type ComboboxOption } from '@/shared/ui/combobox'
import { api, type Character } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import {
  evaluatePrerequisite,
  parseClassChoices,
  type PrerequisiteCheck,
} from '@/entities/character/derived'
import { characterQueryOptions } from '@/entities/character/queries'
import { accentTitle, subtleText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { AbilitiesSection } from './abilities-section'
import { parseChoices } from './parse-choices'

/**
 * Picker for per-class subpath choices — devoto (Clérigo/Paladino/Druida)
 * and caminho (Arcanista L1, Paladino L5, Cavaleiro L5). Returns null when
 * the class has no slot or the player has not reached the caminho minLevel.
 * Empty value clears the choice; sending a blob with no fields removes the
 * class key from the persisted blob so the row stays minimal.
 */
function ClassChoicesPicker({
  character,
  className,
  level,
  classChoices,
}: {
  character: Character
  className: string
  level: number
  classChoices: ClassChoices
}) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const devotoOpts: Deus[] | null = devotoOptionsFor(className)
  const caminhoSlot = caminhoSlotFor(className)
  const showDevoto = devotoOpts !== null
  const showCaminho = caminhoSlot !== null && level >= caminhoSlot.minLevel
  const blob: ClassChoiceBlob = classChoices[className] ?? {}

  const mutate = useMutation<
    Character,
    Error,
    ClassChoices,
    { previous: Character | undefined }
  >({
    mutationFn: (next) =>
      api.characters.updateAbilityChoices(character.id, { classChoices: next }),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, classChoices: JSON.stringify(next) } : prev,
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

  function commit(nextBlob: ClassChoiceBlob) {
    const next: ClassChoices = { ...classChoices }
    if (nextBlob.devoto || nextBlob.caminho) next[className] = nextBlob
    else delete next[className]
    mutate.mutate(next)
  }

  if (!showDevoto && !showCaminho) return null

  const devotoOptions: ComboboxOption[] = (devotoOpts ?? []).map((d) => ({
    value: d.id,
    label: d.name,
  }))
  const caminhoOptions: ComboboxOption[] = (caminhoSlot?.options ?? []).map(
    (c: CaminhoOption) => ({ value: c.id, label: c.name }),
  )

  return (
    <div className="mb-3 space-y-2">
      <p
        className={cn(
          'text-[10px] font-semibold uppercase tracking-wide',
          subtleText,
        )}
      >
        Escolhas
      </p>
      {showDevoto && (
        <div>
          <p className={cn('mb-1 text-[11px]', subtleText)}>Devoto</p>
          <Combobox
            options={devotoOptions}
            value={blob.devoto ?? ''}
            onChange={(value) =>
              commit({ ...blob, devoto: value || undefined })
            }
            placeholder="Escolher devoto…"
            searchPlaceholder="Buscar deus…"
            emptyMessage="Nenhum deus."
            allowClear
            clearLabel="Sem devoto"
          />
        </div>
      )}
      {showCaminho && (
        <div>
          <p className={cn('mb-1 text-[11px]', subtleText)}>Caminho</p>
          <Combobox
            options={caminhoOptions}
            value={blob.caminho ?? ''}
            onChange={(value) =>
              commit({ ...blob, caminho: value || undefined })
            }
            placeholder="Escolher caminho…"
            searchPlaceholder="Buscar caminho…"
            emptyMessage="Nenhum caminho."
            allowClear
            clearLabel="Não escolhido"
          />
        </div>
      )}
    </div>
  )
}

/**
 * Renders one class row from `character.classes`. Owns the elective
 * pool (class powers + general powers) with slot-count enforcement,
 * plus the devoto/caminho picker when applicable.
 */
export function ClassesSection({
  entry,
  character,
}: {
  entry: { className: string; level: number }
  character: Character
}) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const allChosen = parseChoices(character.classPowers)
  const chosenSet = new Set(allChosen)
  const classChoices = parseClassChoices(character.classChoices)
  const pool = classPowersFor(entry.className)
  const auto = pool
    .filter((p) => p.grantedAtLevel !== undefined && p.grantedAtLevel <= entry.level)
    .sort((a, b) => (a.grantedAtLevel ?? 0) - (b.grantedAtLevel ?? 0))
  const classElectives = pool
    .filter((p) => p.grantedAtLevel === undefined)
    .sort((a, b) => (a.minLevel ?? 1) - (b.minLevel ?? 1))

  const slots = slotsForClassLevel(entry.className, entry.level)
  const slotCount = slots.length
  const kinds = unlockedKinds(entry.className, entry.level)
  const generalPool = generalPowersByKinds(kinds)
  const classElectiveIds = new Set(classElectives.map((p) => p.id))
  const generalIds = new Set(generalPool.map((p) => p.id))
  const ownedSlotPicks = allChosen.filter(
    (id) => classElectiveIds.has(id) || generalIds.has(id),
  ).length
  const slotsRemaining = Math.max(0, slotCount - ownedSlotPicks)

  const update = useMutation<
    Character,
    Error,
    string[],
    { previous: Character | undefined }
  >({
    mutationFn: (next) =>
      api.characters.updateAbilityChoices(character.id, { classPowers: next }),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, classPowers: JSON.stringify(next) } : prev,
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

  const toggleElective = (powerId: string) => {
    const isSelected = allChosen.includes(powerId)
    if (!isSelected && slotsRemaining <= 0) return
    const next = isSelected
      ? allChosen.filter((id) => id !== powerId)
      : [...allChosen, powerId]
    update.mutate(next)
  }

  return (
    <AbilitiesSection title={`Classe: ${entry.className} ${entry.level}`}>
      {pool.length === 0 ? (
        <p className={cn('text-xs italic text-muted-foreground')}>
          Classe não está no catálogo.
        </p>
      ) : (
        <>
          <ClassChoicesPicker
            character={character}
            className={entry.className}
            level={entry.level}
            classChoices={classChoices}
          />
          {auto.length > 0 && (
            <div className="mb-3">
              <p className={cn('mb-1 text-[10px] font-semibold uppercase tracking-wide', subtleText)}>
                Concedidos
              </p>
              <ul className="space-y-1.5">
                {auto.map((power) => (
                  <ClassPowerRow key={power.id} power={power} owned />
                ))}
              </ul>
            </div>
          )}
          {slotCount > 0 && (
            <p className={cn('mb-2 text-[11px]', subtleText)}>
              Poderes:{' '}
              <span className="font-semibold">
                {ownedSlotPicks} / {slotCount}
              </span>{' '}
              · Restantes:{' '}
              <span className="font-semibold">{slotsRemaining}</span>
            </p>
          )}
          {classElectives.length > 0 && (
            <div className="mb-3">
              <p className={cn('mb-1 text-[10px] font-semibold uppercase tracking-wide', subtleText)}>
                Poderes de {entry.className}
              </p>
              <ul className="space-y-1.5">
                {classElectives.map((power) => {
                  const owned = allChosen.includes(power.id)
                  const tooHigh = (power.minLevel ?? 1) > entry.level
                  const prereqChecks = (power.prerequisites ?? []).map((p) =>
                    evaluatePrerequisite(p, character, chosenSet, classChoices),
                  )
                  const missingPrereq = prereqChecks.some((c) => !c.met)
                  const noSlot = slotsRemaining <= 0
                  return (
                    <ClassPowerRow
                      key={power.id}
                      power={power}
                      owned={owned}
                      locked={tooHigh || missingPrereq || (noSlot && !owned)}
                      prereqChecks={prereqChecks}
                      onToggle={() => toggleElective(power.id)}
                      disabled={update.isPending}
                    />
                  )
                })}
              </ul>
            </div>
          )}
          {generalPool.length > 0 && (
            <div>
              <p className={cn('mb-1 text-[10px] font-semibold uppercase tracking-wide', subtleText)}>
                Poderes Gerais ({kinds.filter((k) => k !== entry.className.toLowerCase()).join(', ') || 'sem pools'})
              </p>
              <ul className="space-y-1.5">
                {generalPool.map((power) => {
                  const owned = allChosen.includes(power.id)
                  const tooHigh = (power.minLevel ?? 1) > entry.level
                  const noSlot = slotsRemaining <= 0
                  return (
                    <GeneralPowerRow
                      key={power.id}
                      power={power}
                      owned={owned}
                      locked={tooHigh || (noSlot && !owned)}
                      onToggle={() => toggleElective(power.id)}
                      disabled={update.isPending}
                    />
                  )
                })}
              </ul>
            </div>
          )}
        </>
      )}
    </AbilitiesSection>
  )
}

function GeneralPowerRow({
  power,
  owned,
  locked,
  onToggle,
  disabled,
}: {
  power: GeneralPower
  owned: boolean
  locked?: boolean
  onToggle: () => void
  disabled?: boolean
}) {
  return (
    <li
      className={cn(
        'flex gap-2 rounded border p-2',
        owned
          ? 'border-amber-600 bg-amber-100/60 dark:border-amber-400 dark:bg-amber-500/10'
          : 'border-amber-700/20 dark:border-amber-500/20',
      )}
    >
      <button
        type="button"
        onClick={onToggle}
        disabled={disabled || (locked && !owned)}
        className={cn(
          'mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border text-[10px]',
          owned
            ? 'border-amber-700 bg-amber-600 text-white'
            : 'border-amber-700/50 hover:bg-amber-100 dark:border-amber-500/50',
          (disabled || (locked && !owned)) && 'cursor-not-allowed opacity-40',
        )}
        aria-pressed={owned}
        aria-label={owned ? 'Remover poder' : 'Selecionar poder'}
      >
        {owned ? <Check className="size-3" /> : null}
      </button>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1">
          <p className={cn('text-xs font-semibold', accentTitle)}>{power.name}</p>
          <span className={cn(
            'rounded px-1 text-[9px] uppercase tracking-wide',
            kindBadgeClass(power.kind),
          )}>
            {power.kind}
          </span>
          {power.minLevel !== undefined && power.minLevel > 1 && (
            <span className="rounded bg-zinc-200/60 px-1 text-[9px] uppercase tracking-wide text-zinc-700 dark:bg-zinc-500/20 dark:text-zinc-200">
              ≥L{power.minLevel}
            </span>
          )}
          {locked && !owned && (
            <span className="rounded bg-red-200/60 px-1 text-[9px] uppercase tracking-wide text-red-800 dark:bg-red-500/20 dark:text-red-100">
              Bloqueado
            </span>
          )}
        </div>
        <p className={cn('mt-0.5 text-[11px] leading-snug', subtleText)}>
          {power.description}
        </p>
      </div>
    </li>
  )
}

function kindBadgeClass(kind: PowerKind): string {
  switch (kind) {
    case 'combate':
      return 'bg-rose-200/60 text-rose-900 dark:bg-rose-500/20 dark:text-rose-100'
    case 'magia':
      return 'bg-indigo-200/60 text-indigo-900 dark:bg-indigo-500/20 dark:text-indigo-100'
    case 'destino':
      return 'bg-emerald-200/60 text-emerald-900 dark:bg-emerald-500/20 dark:text-emerald-100'
    case 'tormenta':
      return 'bg-fuchsia-200/60 text-fuchsia-900 dark:bg-fuchsia-500/20 dark:text-fuchsia-100'
    default:
      return 'bg-amber-200/60 text-amber-900 dark:bg-amber-500/20 dark:text-amber-100'
  }
}

function ClassPowerRow({
  power,
  owned,
  locked,
  prereqChecks,
  onToggle,
  disabled,
}: {
  power: ClassPower
  owned: boolean
  locked?: boolean
  prereqChecks?: PrerequisiteCheck[]
  onToggle?: () => void
  disabled?: boolean
}) {
  const isAuto = power.grantedAtLevel !== undefined
  return (
    <li
      className={cn(
        'flex gap-2 rounded border p-2',
        owned
          ? 'border-amber-600 bg-amber-100/60 dark:border-amber-400 dark:bg-amber-500/10'
          : 'border-amber-700/20 dark:border-amber-500/20',
      )}
    >
      {!isAuto && onToggle && (
        <button
          type="button"
          onClick={onToggle}
          disabled={disabled || (locked && !owned)}
          className={cn(
            'mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border text-[10px]',
            owned
              ? 'border-amber-700 bg-amber-600 text-white'
              : 'border-amber-700/50 hover:bg-amber-100 dark:border-amber-500/50',
            (disabled || (locked && !owned)) && 'cursor-not-allowed opacity-40',
          )}
          aria-pressed={owned}
          aria-label={owned ? 'Remover poder' : 'Selecionar poder'}
        >
          {owned ? <Check className="size-3" /> : null}
        </button>
      )}
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1">
          <p className={cn('text-xs font-semibold', accentTitle)}>{power.name}</p>
          {isAuto && (
            <span className="rounded bg-emerald-200/60 px-1 text-[9px] uppercase tracking-wide text-emerald-900 dark:bg-emerald-500/20 dark:text-emerald-100">
              L{power.grantedAtLevel}
            </span>
          )}
          {!isAuto && power.minLevel !== undefined && power.minLevel > 1 && (
            <span className="rounded bg-zinc-200/60 px-1 text-[9px] uppercase tracking-wide text-zinc-700 dark:bg-zinc-500/20 dark:text-zinc-200">
              ≥L{power.minLevel}
            </span>
          )}
          {locked && !owned && (
            <span className="rounded bg-red-200/60 px-1 text-[9px] uppercase tracking-wide text-red-800 dark:bg-red-500/20 dark:text-red-100">
              Bloqueado
            </span>
          )}
        </div>
        {!owned && prereqChecks && prereqChecks.length > 0 && (
          <p className="mt-0.5 text-[10px] leading-snug">
            <span className={cn('font-semibold', subtleText)}>Requer: </span>
            {prereqChecks.map((c, i) => (
              <span key={i}>
                {i > 0 && <span className={subtleText}>, </span>}
                <span
                  className={cn(
                    c.prereq.kind === 'note'
                      ? 'text-amber-700 dark:text-amber-300'
                      : c.met
                        ? 'text-emerald-700 dark:text-emerald-300'
                        : 'text-red-700 dark:text-red-300',
                  )}
                >
                  {c.reason}
                </span>
              </span>
            ))}
          </p>
        )}
        <p className={cn('mt-0.5 text-[11px] leading-snug', subtleText)}>
          {power.description}
        </p>
      </div>
    </li>
  )
}
