import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  allGeneralPowers,
  caminhoSlotFor,
  classPowersFor,
  devotoOptionsFor,
  slotsForClassLevel,
} from '@tormenta20/t20-data'
import type {
  CaminhoOption,
  ClassChoiceBlob,
  ClassChoices,
  Deus,
  GeneralPower,
} from '@tormenta20/t20-data'
import { Combobox, type ComboboxOption } from '@/shared/ui/combobox'
import { api, type Character, type AbilityChoicesResult } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import {
  evaluatePrerequisite,
  parseClassChoices,
} from '@/entities/character/derived'
import { characterQueryOptions } from '@/entities/character/queries'
import { subtleText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { ClassPowerRow } from './class-power-row'
import {
  CollapsibleAbilityCard,
  type CardFocus,
} from './collapsible-ability-card'
import { GeneralPowersPool } from './general-powers-pool'
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
    AbilityChoicesResult,
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
    onSuccess: (delta) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, ...delta } : prev,
      )
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
            onChange={(value) => commit({ ...blob, devoto: value || undefined })}
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
 * One class card from `character.classes`. Owns the elective pool (class powers
 * + the virtualized general-powers pool) with slot-count enforcement, plus the
 * devoto/caminho picker when applicable.
 */
export function ClassesSection({
  entry,
  character,
  focus,
  pending,
}: {
  entry: { className: string; level: number }
  character: Character
  focus: CardFocus
  pending: number
}) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const allChosen = parseChoices(character.classPowers)
  const chosenSet = new Set(allChosen)
  const classChoices = parseClassChoices(character.classChoices)
  const pool = classPowersFor(entry.className)
  const auto = pool
    .filter(
      (p) => p.grantedAtLevel !== undefined && p.grantedAtLevel <= entry.level,
    )
    .sort((a, b) => (a.grantedAtLevel ?? 0) - (b.grantedAtLevel ?? 0))
  const classElectives = pool
    .filter((p) => p.grantedAtLevel === undefined)
    .sort((a, b) => (a.minLevel ?? 1) - (b.minLevel ?? 1))

  const slots = slotsForClassLevel(entry.className, entry.level)
  const slotCount = slots.length
  const generalPool = allGeneralPowers()
  const classElectiveIds = new Set(classElectives.map((p) => p.id))
  const generalIds = new Set(generalPool.map((p) => p.id))
  const ownedSlotPicks = allChosen.filter(
    (id) => classElectiveIds.has(id) || generalIds.has(id),
  ).length
  const slotsRemaining = Math.max(0, slotCount - ownedSlotPicks)

  const update = useMutation<
    AbilityChoicesResult,
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
    onSuccess: (delta) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, ...delta } : prev,
      )
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

  const generalLabel =
    [...new Set(generalPool.map((p) => p.kind))].join(', ') || 'sem pools'

  return (
    <CollapsibleAbilityCard
      id={`classe:${entry.className}`}
      title={`${entry.className} ${entry.level}`}
      count={
        slotCount > 0
          ? `${ownedSlotPicks}/${slotCount} poderes · ${slotsRemaining} restantes`
          : undefined
      }
      pending={pending}
      defaultOpen={pending > 0}
      focus={focus}
    >
      {pool.length === 0 ? (
        <p className="text-xs italic text-muted-foreground">
          Classe não está no catálogo.
        </p>
      ) : (
        <div className="space-y-3">
          <ClassChoicesPicker
            character={character}
            className={entry.className}
            level={entry.level}
            classChoices={classChoices}
          />
          {auto.length > 0 && (
            <div>
              <p
                className={cn(
                  'mb-1 text-[10px] font-semibold uppercase tracking-wide',
                  subtleText,
                )}
              >
                Concedidos
              </p>
              <ul className="space-y-1.5">
                {auto.map((power) => (
                  <ClassPowerRow
                    key={power.id}
                    power={power}
                    owned
                  />
                ))}
              </ul>
            </div>
          )}
          {classElectives.length > 0 && (
            <div>
              <p
                className={cn(
                  'mb-1 text-[10px] font-semibold uppercase tracking-wide',
                  subtleText,
                )}
              >
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
              <p
                className={cn(
                  'mb-1 text-[10px] font-semibold uppercase tracking-wide',
                  subtleText,
                )}
              >
                Poderes Gerais ({generalLabel})
              </p>
              <GeneralPowersPool
                powers={generalPool}
                isOwned={(id) => allChosen.includes(id)}
                isLocked={(power: GeneralPower) => {
                  const owned = allChosen.includes(power.id)
                  const tooHigh = (power.minLevel ?? 1) > entry.level
                  return tooHigh || (slotsRemaining <= 0 && !owned)
                }}
                onToggle={toggleElective}
                disabled={update.isPending}
              />
            </div>
          )}
        </div>
      )}
    </CollapsibleAbilityCard>
  )
}
