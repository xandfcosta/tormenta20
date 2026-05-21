import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { useDebouncedCallback } from '@tanstack/react-pacer'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NumberInput } from '@/components/ui/number-input'
import {
  AlertTriangle,
  Check,
  ChevronDown,
  ChevronUp,
  Crosshair,
  Gem,
  Info,
  Minus,
  Package,
  Pencil,
  Plus,
  Search,
  Shield,
  Shirt,
  SlidersHorizontal,
  Sparkles,
  Star,
  Sword,
  ToggleRight,
  Trash2,
  X,
  Zap,
} from 'lucide-react'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import { api } from '@/lib/api'
import { characterQueryOptions, meQueryOptions } from '@/lib/queries'
import type {
  AttributeKey,
  Character,
  CharacterExpertise,
  CharacterItem,
  CreateItemInput,
  UpdateItemInput,
  UpdateVitalsInput,
} from '@/lib/api'
import {
  ATTRIBUTE_ABBR,
  ATTRIBUTE_KEYS,
  EXPERTISES,
  expertiseStateFor,
  trainingBonusForLevel,
} from '@/lib/expertise'
import type { ExpertiseDef } from '@/lib/expertise'
import {
  defenseTotal,
  displacementTotal,
  evaluatePrerequisite,
  expertiseTotalWithItems,
  inventorySlotsTotal,
  isItemProficient,
  parseImprovementIds,
  pmCostMod,
  pmLimitTotal,
  spellDCBonus,
  useAllConditionals,
  useCharacterEffects,
  type ConditionalEntry,
  type PrerequisiteCheck,
} from '@/lib/derived'
import { useConditionalsStore } from '@/store/conditionals-store'
import {
  CATALOG_ITEMS,
  IMPROVEMENTS,
  MATERIALS,
  classPowersFor,
  characterProficiencies,
  familyFor,
  generalPowersByKinds,
  getCatalogItem,
  getOrigin,
  getRace,
  slotsForClassLevel,
  unlockedKinds,
} from '@tormenta20/t20-data'
import type {
  CatalogItem,
  ClassPower,
  ConditionalEffect,
  GeneralPower,
  ItemEffects,
  Modifier,
  OriginBenefit,
  OriginDefinition,
  PowerKind,
  ProficiencyEntry,
  RaceAbility,
  RaceDefinition,
} from '@tormenta20/t20-data'

export const Route = createFileRoute('/characters/$id')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(characterQueryOptions(Number(params.id))),
  component: CharacterViewPage,
})

function CharacterViewPage() {
  const { id } = Route.useParams()
  const character = useQuery(characterQueryOptions(Number(id)))

  if (character.isLoading) return <p className="p-6">Loading…</p>
  if (character.isError) {
    return (
      <p className="p-6 text-destructive">{(character.error as Error).message}</p>
    )
  }
  if (!character.data) return null

  return <CharacterSheet character={character.data} />
}

// Theme tokens — default = parchment light, dark: = current dark amber sheet.
const surface = 'border-2 border-amber-700/40 dark:border-amber-500/40'
const panelBg = 'bg-amber-50/70 dark:bg-zinc-900/40'
const sheetBg =
  'bg-gradient-to-br from-amber-50 via-orange-50 to-amber-100 text-zinc-900 dark:from-zinc-900 dark:via-zinc-950 dark:to-black dark:text-zinc-100'
const hoverRow = 'hover:bg-amber-100/60 dark:hover:bg-zinc-900/60'
const subtleText = 'text-zinc-600 dark:text-zinc-400'
const dimText = 'text-zinc-500 dark:text-zinc-500'
const accentStrong = 'text-amber-800 dark:text-amber-200'
const accentTitle = 'text-amber-900 dark:text-amber-50'
const accentBadge =
  'border-amber-700/50 bg-amber-200/60 text-amber-900 hover:bg-amber-200 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-200 dark:hover:bg-amber-500/20'

function CharacterSheet({ character }: { character: Character }) {
  return (
    <div
      className={cn(
        'grid h-full grid-rows-[auto_auto_1fr] gap-2 overflow-hidden p-2 sm:gap-3 sm:p-3 lg:grid-cols-[minmax(300px,26rem)_1fr] lg:grid-rows-[auto_1fr]',
        sheetBg,
      )}
    >
      <SheetHeader character={character} className="lg:col-span-2" />
      <VitalsAside character={character} />
      <RightPanel character={character} />
    </div>
  )
}

function RightPanel({ character }: { character: Character }) {
  return (
    <Tabs
      defaultValue="expertises"
      className="flex min-h-0 flex-col gap-2 overflow-hidden"
    >
      <TabsList className="self-start">
        <TabsTrigger value="expertises">Perícias</TabsTrigger>
        <TabsTrigger value="equipment" className="gap-1.5">
          <Shirt className="size-3.5" /> Equipado
        </TabsTrigger>
        <TabsTrigger value="inventory" className="gap-1.5">
          <Package className="size-3.5" /> Inventário
        </TabsTrigger>
        <TabsTrigger value="conditionals" className="gap-1.5">
          <ToggleRight className="size-3.5" /> Efeitos
          <EffectsCountBadge character={character} />
        </TabsTrigger>
        <TabsTrigger value="proficiencies" className="gap-1.5">
          <Shield className="size-3.5" /> Proficiências
        </TabsTrigger>
        <TabsTrigger value="abilities" className="gap-1.5">
          <Star className="size-3.5" /> Habilidades
        </TabsTrigger>
      </TabsList>
      <TabsContent
        value="expertises"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <ExpertisesPanel character={character} />
      </TabsContent>
      <TabsContent
        value="equipment"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <EquipmentPanel character={character} />
      </TabsContent>
      <TabsContent
        value="inventory"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <InventoryPanel character={character} />
      </TabsContent>
      <TabsContent
        value="conditionals"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <EffectsPanel character={character} />
      </TabsContent>
      <TabsContent
        value="proficiencies"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <ProficienciesPanel character={character} />
      </TabsContent>
      <TabsContent
        value="abilities"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <AbilitiesPanel character={character} />
      </TabsContent>
    </Tabs>
  )
}

function EffectsCountBadge({ character }: { character: Character }) {
  const all = useAllConditionals(character)
  const condActive = all.filter((e) => e.active).length
  const consumableActive = (character.activeEffects ?? []).length
  const total = condActive + consumableActive
  if (all.length === 0 && consumableActive === 0) return null
  return (
    <span
      className={cn(
        'ml-0.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full px-1 text-[10px] font-bold',
        total > 0
          ? 'bg-amber-700 text-amber-50 dark:bg-amber-500 dark:text-zinc-900'
          : 'bg-zinc-300 text-zinc-700 dark:bg-zinc-700 dark:text-zinc-300',
      )}
      aria-label={`${total} efeitos ativos`}
    >
      {total}
    </span>
  )
}

function describeConditionalTarget(
  target: Modifier['target'],
): string {
  switch (target.k) {
    case 'expertise':
      return target.name
    case 'expertiseAll':
      return 'todas perícias'
    case 'expertiseRemovePenalty':
      return `${target.name} (sem penalidade)`
    case 'expertiseByAttribute':
      return `Perícias de ${target.attribute}`
    case 'attribute':
      return target.name
    case 'defense':
      return 'Defesa'
    case 'defenseDexCap':
      return 'limite de Des na Defesa'
    case 'resistance':
      return 'Resistência'
    case 'fearResistance':
      return 'Resistência (medo)'
    case 'attack':
      return target.scope === 'this' ? 'Ataque (esta arma)' : 'Ataque'
    case 'damage':
      return target.scope === 'this' ? 'Dano (esta arma)' : 'Dano'
    case 'critRange':
      return 'Margem de crítico'
    case 'critMult':
      return 'Multiplicador de crítico'
    case 'pmLimit':
      return 'Limite de PM'
    case 'pmCost':
      return 'Custo de PM'
    case 'spellDC':
      return 'CD Magia'
    case 'inventorySlots':
      return 'Espaços'
    case 'displacement':
      return 'Deslocamento'
    case 'armorPenalty':
      return 'Penalidade de armadura'
    case 'armorPenaltyExpertises':
      return 'Penalidade em perícias afetadas'
    case 'tempHp':
      return 'PV temp.'
    case 'tempMp':
      return 'PM temp.'
    case 'maneuver':
      return `Manobra: ${target.name}`
    case 'flag':
      return `Estado: ${target.name}`
  }
}

function EffectsPanel({ character }: { character: Character }) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto pr-1">
      <ActiveEffectsSection character={character} />
      <ConditionalsSection character={character} />
    </div>
  )
}

function AbilitiesPanel({ character }: { character: Character }) {
  const races = character.races
    .map((r) => getRace(r.race))
    .filter((r): r is RaceDefinition => Boolean(r))
  const classes = character.classes
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto pr-1">
      {races.length === 0 ? (
        <AbilitiesSection title={`Raça: —`}>
          <p className={cn('text-xs italic', dimText)}>
            Raça do personagem não está no catálogo.
          </p>
        </AbilitiesSection>
      ) : (
        races.map((race) => (
          <RaceAbilitySection
            key={race.id}
            race={race}
            character={character}
          />
        ))
      )}
      <OriginAbilitySection character={character} />
      {classes.length === 0 ? (
        <AbilitiesSection title="Classes: —">
          <p className={cn('text-xs italic', dimText)}>
            Nenhuma classe atribuída.
          </p>
        </AbilitiesSection>
      ) : (
        classes.map((entry) => (
          <ClassesSection
            key={entry.className}
            entry={entry}
            character={character}
          />
        ))
      )}
    </div>
  )
}

function ClassesSection({
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
    onSuccess: (server) => qc.setQueryData<Character>(queryKey, server),
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
        <p className={cn('text-xs italic', dimText)}>
          Classe não está no catálogo.
        </p>
      ) : (
        <>
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
                    evaluatePrerequisite(p, character, chosenSet),
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

const RACE_ATTR_ABBR: Record<AttributeKey, string> = {
  strength: 'For',
  dexterity: 'Des',
  constitution: 'Con',
  intelligence: 'Int',
  wisdom: 'Sab',
  charisma: 'Car',
}

function formatAttributeBonuses(
  bonuses: Partial<Record<AttributeKey, number>>,
): string {
  const parts: string[] = []
  for (const [attr, amount] of Object.entries(bonuses)) {
    if (typeof amount !== 'number' || amount === 0) continue
    const sign = amount > 0 ? '+' : ''
    parts.push(`${RACE_ATTR_ABBR[attr as AttributeKey]} ${sign}${amount}`)
  }
  return parts.join(', ')
}

function RaceAbilitySection({
  race,
  character,
}: {
  race: RaceDefinition
  character: Character
}) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const choices = parseChoices(character.raceAbilityChoices)

  const update = useMutation<
    Character,
    Error,
    string[],
    { previous: Character | undefined }
  >({
    mutationFn: (next) =>
      api.characters.updateAbilityChoices(character.id, {
        raceAbilityChoices: next,
      }),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, raceAbilityChoices: JSON.stringify(next) } : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (server) => qc.setQueryData<Character>(queryKey, server),
  })

  const pickVariant = (ability: RaceAbility, variantId: string) => {
    const siblingIds = new Set(ability.variants?.map((v) => v.id) ?? [])
    const next = choices.filter((c) => !siblingIds.has(c))
    next.push(variantId)
    update.mutate(next)
  }

  const bonusLine = formatAttributeBonuses(race.attributeBonuses)
  return (
    <AbilitiesSection title={`Raça: ${race.name}`}>
      {bonusLine && (
        <p className={cn('mb-2 text-xs', subtleText)}>
          <span className="font-semibold">Modificadores:</span> {bonusLine}
        </p>
      )}
      <ul className="space-y-2">
        {race.abilities.map((ability) => (
          <li key={ability.id} className="rounded border border-amber-700/20 p-2 dark:border-amber-500/20">
            <p className={cn('text-xs font-semibold', accentTitle)}>{ability.name}</p>
            <p className={cn('mt-0.5 text-[11px] leading-snug', subtleText)}>
              {ability.description}
            </p>
            {ability.variants && (
              <RaceVariantPicker
                ability={ability}
                selected={ability.variants.find((v) => choices.includes(v.id))?.id}
                onPick={(id) => pickVariant(ability, id)}
                disabled={update.isPending}
              />
            )}
          </li>
        ))}
      </ul>
    </AbilitiesSection>
  )
}

function RaceVariantPicker({
  ability,
  selected,
  onPick,
  disabled,
}: {
  ability: RaceAbility
  selected: string | undefined
  onPick: (variantId: string) => void
  disabled: boolean
}) {
  return (
    <div className="mt-2 flex flex-wrap gap-1">
      {ability.variants?.map((variant) => {
        const active = variant.id === selected
        return (
          <button
            key={variant.id}
            type="button"
            disabled={disabled}
            onClick={() => onPick(variant.id)}
            title={variant.description}
            className={cn(
              'rounded border px-2 py-0.5 text-[11px] transition-colors',
              active
                ? 'border-amber-600 bg-amber-200 font-semibold text-amber-900 dark:border-amber-400 dark:bg-amber-500/20 dark:text-amber-100'
                : 'border-amber-700/30 text-amber-900/70 hover:bg-amber-100 dark:border-amber-500/30 dark:text-amber-100/60 dark:hover:bg-amber-500/10',
              disabled && 'cursor-not-allowed opacity-60',
            )}
          >
            {variant.name}
          </button>
        )
      })}
    </div>
  )
}

function parseChoices(raw: string): string[] {
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return parsed.filter((x): x is string => typeof x === 'string')
    }
    return []
  } catch {
    return []
  }
}

const ORIGIN_BENEFIT_LIMIT = 2

function OriginAbilitySection({ character }: { character: Character }) {
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
    onSuccess: (server) => qc.setQueryData<Character>(queryKey, server),
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

function AbilitiesSection({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <section className={cn('rounded-lg border p-3', surface)}>
      <h3 className={cn('font-serif text-sm font-bold', accentStrong)}>{title}</h3>
      <div className="mt-2">{children}</div>
    </section>
  )
}

function ProficienciesPanel({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const classNames = character.classes.map((c) => c.className)
  const classDefaults = characterProficiencies(classNames)
  const stored = parseProficiencySet(character.proficiencies)

  const update = useMutation<
    Character,
    Error,
    string[],
    { previous: Character | undefined }
  >({
    mutationFn: (next) =>
      api.characters.updateProficiencies(character.id, { proficiencies: next }),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, proficiencies: JSON.stringify(next) } : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (server) => qc.setQueryData<Character>(queryKey, server),
  })

  const toggle = (category: string) => {
    const next = new Set(stored)
    if (next.has(category)) next.delete(category)
    else next.add(category)
    update.mutate([...next])
  }

  const resetToDefaults = () => {
    const defaults = classDefaults.filter((e) => e.granted).map((e) => e.category)
    update.mutate(defaults)
  }

  const weapons = classDefaults.filter((e) => e.category.startsWith('armas-'))
  const armors = classDefaults.filter(
    (e) => e.category.startsWith('armaduras-') || e.category === 'escudos',
  )

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto pr-1">
      <div className="flex justify-end">
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-6 px-2 text-[11px]"
          onClick={resetToDefaults}
          disabled={update.isPending}
        >
          Restaurar padrão de classe
        </Button>
      </div>
      <ProficiencyGroup
        title="Armas"
        entries={weapons}
        stored={stored}
        onToggle={toggle}
        disabled={update.isPending}
      />
      <ProficiencyGroup
        title="Armaduras & Escudos"
        entries={armors}
        stored={stored}
        onToggle={toggle}
        disabled={update.isPending}
      />
    </div>
  )
}

function parseProficiencySet(raw: string): Set<string> {
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return new Set(parsed.filter((x): x is string => typeof x === 'string'))
    }
    return new Set()
  } catch {
    return new Set()
  }
}

function ProficiencyGroup({
  title,
  entries,
  stored,
  onToggle,
  disabled,
}: {
  title: string
  entries: ProficiencyEntry[]
  stored: Set<string>
  onToggle: (category: string) => void
  disabled: boolean
}) {
  return (
    <section className={cn('rounded-lg border p-3', surface)}>
      <h3 className={cn('font-serif text-sm font-bold', accentStrong)}>{title}</h3>
      <ul className="mt-2 space-y-1">
        {entries.map((entry) => (
          <ProficiencyRow
            key={entry.category}
            entry={entry}
            granted={stored.has(entry.category)}
            isClassDefault={entry.granted}
            onToggle={() => onToggle(entry.category)}
            disabled={disabled}
          />
        ))}
      </ul>
    </section>
  )
}

function ProficiencyRow({
  entry,
  granted,
  isClassDefault,
  onToggle,
  disabled,
}: {
  entry: ProficiencyEntry
  granted: boolean
  isClassDefault: boolean
  onToggle: () => void
  disabled: boolean
}) {
  return (
    <li
      className={cn(
        'flex items-center justify-between gap-2 rounded-md text-xs',
        granted
          ? 'bg-emerald-50 dark:bg-emerald-950/30'
          : 'bg-zinc-100 dark:bg-zinc-900/40',
      )}
    >
      <button
        type="button"
        onClick={onToggle}
        disabled={disabled}
        className="flex flex-1 items-center gap-2 rounded-md px-2 py-1 text-left hover:bg-amber-100/60 disabled:cursor-not-allowed disabled:opacity-60 dark:hover:bg-zinc-800/50"
        aria-pressed={granted}
        aria-label={`${granted ? 'Remover' : 'Adicionar'} proficiência: ${entry.label}`}
      >
        {granted ? (
          <Check className="size-3.5 text-emerald-700 dark:text-emerald-400" />
        ) : (
          <X className="size-3.5 text-zinc-400" />
        )}
        <span
          className={cn(
            granted
              ? 'text-zinc-800 dark:text-zinc-100'
              : cn('line-through', dimText),
          )}
        >
          {entry.label}
        </span>
        {isClassDefault ? (
          <span
            className={cn(
              'ml-1 rounded px-1 text-[9px] uppercase tracking-wider',
              'bg-amber-200/60 text-amber-900 dark:bg-amber-500/20 dark:text-amber-200',
            )}
            title={`Padrão: ${entry.sources.join(', ')}`}
          >
            classe
          </span>
        ) : null}
      </button>
    </li>
  )
}

function ActiveEffectsSection({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const effects = character.activeEffects ?? []

  const remove = useMutation<{ id: number }, Error, number>({
    mutationFn: (effectId) =>
      api.characters.removeActiveEffect(character.id, effectId),
    onSuccess: ({ id }) =>
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              activeEffects: prev.activeEffects.filter((e) => e.id !== id),
            }
          : prev,
      ),
  })
  const endScene = useMutation<Character, Error, void>({
    mutationFn: () => api.characters.endScene(character.id),
    onSuccess: (next) => qc.setQueryData<Character>(queryKey, next),
  })
  const endDay = useMutation<Character, Error, void>({
    mutationFn: () => api.characters.endDay(character.id),
    onSuccess: (next) => qc.setQueryData<Character>(queryKey, next),
  })

  return (
    <section
      className={cn(
        'rounded-lg border p-3',
        surface,
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <h3 className={cn('font-serif text-sm font-bold', accentStrong)}>
          Efeitos consumíveis ativos
        </h3>
        <div className="flex gap-1">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-6 px-2 text-[11px]"
            onClick={() => endScene.mutate()}
            disabled={endScene.isPending}
            aria-label="Encerrar cena"
          >
            Encerrar cena
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-6 px-2 text-[11px]"
            onClick={() => {
              if (confirm('Encerrar dia? Limpa efeitos de cena e dia.'))
                endDay.mutate()
            }}
            disabled={endDay.isPending}
            aria-label="Encerrar dia"
          >
            Encerrar dia
          </Button>
        </div>
      </div>
      {effects.length === 0 ? (
        <p className={cn('mt-2 text-xs', subtleText)}>
          Nenhum consumível ativo. Use itens consumíveis no inventário.
        </p>
      ) : (
        <ul className="mt-2 space-y-1">
          {effects.map((eff) => (
            <ActiveEffectRow
              key={eff.id}
              effect={eff}
              onRemove={() => remove.mutate(eff.id)}
            />
          ))}
        </ul>
      )}
    </section>
  )
}

function ActiveEffectRow({
  effect,
  onRemove,
}: {
  effect: Character['activeEffects'][number]
  onRemove: () => void
}) {
  const catalog = getCatalogItem(effect.catalogId)
  const name = catalog?.name ?? effect.catalogId
  return (
    <li
      className={cn(
        'flex items-center gap-2 rounded-md border px-2 py-1.5',
        'border-emerald-700/30 bg-emerald-50/60 dark:border-emerald-500/25 dark:bg-emerald-950/30',
      )}
    >
      <Sparkles className="size-3.5 shrink-0 text-emerald-700 dark:text-emerald-300" />
      <span className="flex-1 truncate text-sm text-zinc-800 dark:text-zinc-200">
        {name}
      </span>
      <span
        className={cn(
          'shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest',
          effect.scope === 'day'
            ? 'bg-amber-700/80 text-amber-50 dark:bg-amber-500/70 dark:text-zinc-900'
            : 'bg-emerald-700/80 text-emerald-50 dark:bg-emerald-500/70 dark:text-zinc-900',
        )}
      >
        {effect.scope === 'day' ? 'dia' : 'cena'}
      </span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-7 text-zinc-500 hover:bg-red-100 hover:text-red-700 dark:text-zinc-400 dark:hover:bg-red-950/40 dark:hover:text-red-400"
        onClick={onRemove}
        aria-label={`Remover ${name}`}
      >
        <X className="size-3.5" />
      </Button>
    </li>
  )
}

type ConditionalGroup =
  | { kind: 'single'; entry: ConditionalEntry }
  | {
      kind: 'flag'
      flag: string
      label: string
      source: string
      entries: ConditionalEntry[]
    }

/** Folds individual conditional entries that share a `flag` into one toggle
 *  row, so e.g. all four Fúria modifiers activate together. */
function groupConditionals(entries: ConditionalEntry[]): ConditionalGroup[] {
  const byFlag = new Map<string, ConditionalEntry[]>()
  const groups: ConditionalGroup[] = []
  for (const e of entries) {
    const f = e.effect.flag
    if (!f) {
      groups.push({ kind: 'single', entry: e })
      continue
    }
    const arr = byFlag.get(f) ?? []
    arr.push(e)
    byFlag.set(f, arr)
  }
  for (const [flag, list] of byFlag) {
    groups.push({
      kind: 'flag',
      flag,
      label: list[0].effect.note,
      source: list[0].effect.source,
      entries: list,
    })
  }
  return groups
}

function ConditionalsSection({ character }: { character: Character }) {
  const entries = useAllConditionals(character)
  const toggle = useConditionalsStore((s) => s.toggle)
  const setMany = useConditionalsStore((s) => s.setMany)
  const clear = useConditionalsStore((s) => s.clear)
  const activeCount = entries.filter((e) => e.active).length
  const groups = groupConditionals(entries)

  if (entries.length === 0) {
    return (
      <div
        className={cn(
          'flex flex-1 items-center justify-center rounded-lg border p-6 text-center text-sm',
          surface,
          subtleText,
        )}
      >
        Nenhum efeito condicional disponível. Equipe itens com modificadores
        situacionais (terreno, contexto, contra alvo) para vê-los aqui.
      </div>
    )
  }

  return (
    <div
      className={cn(
        'flex min-h-0 flex-1 flex-col gap-2 overflow-hidden rounded-lg border p-3',
        surface,
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <p className={cn('text-xs', subtleText)}>
          {activeCount} de {entries.length} ativos — opt-in por cena/contexto
        </p>
        {activeCount > 0 && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-xs"
            onClick={() => clear(character.id)}
          >
            Limpar
          </Button>
        )}
      </div>
      <ul className="flex-1 space-y-1 overflow-y-auto pr-1">
        {groups.map((g) =>
          g.kind === 'single' ? (
            <ConditionalRow
              key={g.entry.id}
              entry={g.entry}
              onToggle={() => toggle(character.id, g.entry.id)}
            />
          ) : (
            <FlagGroupRow
              key={g.flag}
              group={g}
              onToggle={(value) =>
                setMany(
                  character.id,
                  g.entries.map((e) => e.id),
                  value,
                )
              }
            />
          ),
        )}
      </ul>
    </div>
  )
}

function FlagGroupRow({
  group,
  onToggle,
}: {
  group: Extract<ConditionalGroup, { kind: 'flag' }>
  onToggle: (value: boolean) => void
}) {
  const allActive = group.entries.every((e) => e.active)
  const anyActive = group.entries.some((e) => e.active)
  return (
    <li>
      <button
        type="button"
        onClick={() => onToggle(!allActive)}
        className={cn(
          'flex w-full flex-col gap-1 rounded-md border px-2 py-1.5 text-left transition-colors',
          anyActive
            ? 'border-amber-700/50 bg-amber-100/70 dark:border-amber-500/40 dark:bg-amber-950/40'
            : 'border-amber-700/15 bg-zinc-100/40 hover:bg-amber-100/40 dark:border-amber-500/15 dark:bg-zinc-900/40 dark:hover:bg-zinc-800/60',
        )}
      >
        <div className="flex items-center gap-3">
          <span
            className={cn(
              'flex size-5 shrink-0 items-center justify-center rounded border',
              allActive
                ? 'border-amber-700 bg-amber-700 text-amber-50 dark:border-amber-400 dark:bg-amber-400 dark:text-zinc-900'
                : anyActive
                  ? 'border-amber-700 bg-amber-100 dark:border-amber-400 dark:bg-amber-900/60'
                  : 'border-zinc-400 bg-transparent dark:border-zinc-600',
            )}
            aria-hidden
          >
            {allActive && <Check className="size-3" />}
          </span>
          <span className="truncate text-sm font-medium text-zinc-800 dark:text-zinc-200">
            {group.source}
          </span>
          <span className={cn('ml-auto truncate text-[11px]', subtleText)}>
            {group.label}
          </span>
        </div>
        <ul className="ml-8 space-y-0.5 text-[11px]">
          {group.entries.map((e) => (
            <li key={e.id} className="flex items-center justify-between gap-2">
              <span className={cn('truncate', subtleText)}>
                {describeConditionalTarget(e.effect.target)}
              </span>
              <span
                className={cn(
                  'shrink-0 font-mono font-semibold',
                  e.effect.amount >= 0
                    ? 'text-emerald-700 dark:text-emerald-300'
                    : 'text-red-700 dark:text-red-300',
                )}
              >
                {signed(e.effect.amount)}
              </span>
            </li>
          ))}
        </ul>
      </button>
    </li>
  )
}

function ConditionalRow({
  entry,
  onToggle,
}: {
  entry: { id: string; effect: ConditionalEffect; active: boolean }
  onToggle: () => void
}) {
  const { effect, active } = entry
  const targetLabel = describeConditionalTarget(effect.target)
  return (
    <li>
      <button
        type="button"
        onClick={onToggle}
        className={cn(
          'flex w-full items-center gap-3 rounded-md border px-2 py-1.5 text-left transition-colors',
          active
            ? 'border-amber-700/50 bg-amber-100/70 dark:border-amber-500/40 dark:bg-amber-950/40'
            : 'border-amber-700/15 bg-zinc-100/40 hover:bg-amber-100/40 dark:border-amber-500/15 dark:bg-zinc-900/40 dark:hover:bg-zinc-800/60',
        )}
      >
        <span
          className={cn(
            'flex size-5 shrink-0 items-center justify-center rounded border',
            active
              ? 'border-amber-700 bg-amber-700 text-amber-50 dark:border-amber-400 dark:bg-amber-400 dark:text-zinc-900'
              : 'border-zinc-400 bg-transparent dark:border-zinc-600',
          )}
          aria-hidden
        >
          {active && <Check className="size-3" />}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <span className="truncate text-sm font-medium text-zinc-800 dark:text-zinc-200">
              {effect.source}
            </span>
            <span
              className={cn(
                'shrink-0 font-mono text-sm font-semibold',
                effect.amount >= 0
                  ? 'text-emerald-700 dark:text-emerald-300'
                  : 'text-red-700 dark:text-red-300',
              )}
            >
              {signed(effect.amount)} {targetLabel}
            </span>
          </div>
          <p className={cn('truncate text-[11px]', subtleText)}>
            {effect.note}
          </p>
        </div>
      </button>
    </li>
  )
}

function SheetHeader({
  character,
  className,
}: {
  character: Character
  className?: string
}) {
  const races = character.races.map((r) => r.race)
  const effects = useCharacterEffects(character)
  const disp = displacementTotal(character, effects)
  const fatigue = effects.flags.has('fatigue-on-sleep')
  return (
    <header
      className={cn(
        'relative flex flex-wrap items-center justify-between gap-3 overflow-hidden rounded-xl px-4 py-3 sm:px-6',
        surface,
        'bg-gradient-to-r from-amber-200/70 via-amber-100 to-amber-200/70 dark:from-amber-900/40 dark:via-zinc-900 dark:to-amber-900/40',
        className,
      )}
    >
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_30%_50%,rgba(217,119,6,0.18),transparent_50%)] dark:bg-[radial-gradient(circle_at_30%_50%,rgba(251,191,36,0.15),transparent_50%)]" />
      <div className="relative flex items-center gap-3">
        <Link to="/characters">
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              subtleText,
              'hover:bg-amber-200/60 hover:text-amber-900 dark:hover:bg-zinc-800/60 dark:hover:text-amber-200',
            )}
          >
            ←
          </Button>
        </Link>
        <div className="min-w-0">
          <p className="text-[10px] uppercase tracking-[0.4em] text-amber-800/80 dark:text-amber-400/80">
            Tormenta 20
          </p>
          <h1
            className={cn(
              'truncate font-serif text-2xl font-bold tracking-tight sm:text-3xl',
              accentTitle,
            )}
          >
            {character.name}
          </h1>
          <p className={cn('mt-0.5 truncate text-xs', subtleText)}>
            {races.join(' / ')} • {character.origin}
            {character.god && (
              <>
                {' '}
                •{' '}
                <span className="text-amber-700 dark:text-amber-300">
                  {character.god}
                </span>
              </>
            )}
            {' • '}
            {character.size} • <DisplacementBadge disp={disp} />
            {fatigue && (
              <>
                {' • '}
                <FatigueWarning />
              </>
            )}
          </p>
        </div>
      </div>
      <div className="relative flex items-center gap-2">
        <div className="flex flex-wrap justify-end gap-1">
          {character.classes.map((c) => (
            <Badge key={c.className} className={accentBadge}>
              {c.className} {c.level}
            </Badge>
          ))}
        </div>
        <LevelBadge character={character} />
      </div>
    </header>
  )
}

function LevelBadge({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const [pickerDir, setPickerDir] = useState<null | 'up' | 'down'>(null)

  const mutate = useMutation<
    Character,
    Error,
    { className: string; level: number },
    { previous: Character | undefined }
  >({
    mutationFn: (input) =>
      api.characters.updateClassLevel(character.id, input),
    onMutate: async (input) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) => {
        if (!prev) return prev
        const nextClasses = prev.classes.map((c) =>
          c.className === input.className ? { ...c, level: input.level } : c,
        )
        const total = nextClasses.reduce((s, c) => s + c.level, 0)
        return { ...prev, classes: nextClasses, level: total }
      })
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (server) => qc.setQueryData<Character>(queryKey, server),
  })

  const bumpClass = (className: string, delta: 1 | -1) => {
    const entry = character.classes.find((c) => c.className === className)
    if (!entry) return
    const next = entry.level + delta
    if (next < 1 || next > 20) return
    if (character.level + delta < 1 || character.level + delta > 20) return
    mutate.mutate({ className, level: next })
  }

  const trigger = (dir: 'up' | 'down') => {
    if (character.classes.length === 0) return
    if (character.classes.length === 1) {
      bumpClass(character.classes[0].className, dir === 'up' ? 1 : -1)
      return
    }
    setPickerDir(dir)
  }

  const atMin = character.level <= 1
  const atMax = character.level >= 20

  return (
    <>
      <div
        className={cn(
          'flex items-center gap-1 rounded-lg border px-2 py-1 text-center',
          'border-amber-700/40 bg-amber-50/80 dark:border-amber-500/40 dark:bg-zinc-950/80',
        )}
      >
        <button
          type="button"
          onClick={() => trigger('down')}
          disabled={atMin || mutate.isPending}
          aria-label="Diminuir nível"
          className={cn(
            'text-amber-700 transition-colors disabled:opacity-30 dark:text-amber-300',
            'hover:text-amber-900 dark:hover:text-amber-100',
          )}
        >
          <ChevronDown className="size-4" />
        </button>
        <div className="flex flex-col items-center leading-none">
          <p className={cn('text-[9px] uppercase tracking-widest', subtleText)}>
            Nv
          </p>
          <p
            className={cn(
              'w-7 text-center font-serif text-2xl font-bold leading-none text-amber-700 dark:text-amber-300',
            )}
            aria-label="Nível"
          >
            {character.level}
          </p>
        </div>
        <button
          type="button"
          onClick={() => trigger('up')}
          disabled={atMax || mutate.isPending}
          aria-label="Aumentar nível"
          className={cn(
            'text-amber-700 transition-colors disabled:opacity-30 dark:text-amber-300',
            'hover:text-amber-900 dark:hover:text-amber-100',
          )}
        >
          <ChevronUp className="size-4" />
        </button>
      </div>
      {pickerDir && (
        <ClassLevelPicker
          character={character}
          direction={pickerDir}
          onPick={(className) => {
            const delta = pickerDir === 'up' ? 1 : -1
            bumpClass(className, delta)
            setPickerDir(null)
          }}
          onClose={() => setPickerDir(null)}
        />
      )}
    </>
  )
}

function ClassLevelPicker({
  character,
  direction,
  onPick,
  onClose,
}: {
  character: Character
  direction: 'up' | 'down'
  onPick: (className: string) => void
  onClose: () => void
}) {
  const eligible = character.classes.filter((c) =>
    direction === 'up' ? c.level < 20 : c.level > 1,
  )
  const verb = direction === 'up' ? 'Subir' : 'Reduzir'
  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{verb} nível — escolha a classe</DialogTitle>
        </DialogHeader>
        {eligible.length === 0 ? (
          <p className={cn('text-xs italic', dimText)}>
            Nenhuma classe elegível.
          </p>
        ) : (
          <ul className="space-y-1.5">
            {eligible.map((c) => (
              <li key={c.className}>
                <button
                  type="button"
                  onClick={() => onPick(c.className)}
                  className={cn(
                    'flex w-full items-center justify-between rounded border px-3 py-2 text-left transition-colors',
                    'border-amber-700/30 hover:bg-amber-100 dark:border-amber-500/30 dark:hover:bg-amber-500/10',
                  )}
                >
                  <span className={cn('text-sm font-semibold', accentTitle)}>
                    {c.className}
                  </span>
                  <span className={cn('text-xs', subtleText)}>
                    {c.level} → {direction === 'up' ? c.level + 1 : c.level - 1}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </DialogContent>
    </Dialog>
  )
}

function DisplacementBadge({
  disp,
}: {
  disp: ReturnType<typeof displacementTotal>
}) {
  const changed = disp.itemBonus !== 0
  if (!changed) return <span>{disp.total}m</span>
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          className={cn(
            'cursor-help underline decoration-dotted underline-offset-2',
            disp.itemBonus < 0
              ? 'text-red-700 dark:text-red-300'
              : 'text-emerald-700 dark:text-emerald-300',
          )}
        >
          {disp.total}m
        </button>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        <div className="text-xs">
          <div>Base {disp.base}m</div>
          {disp.contributions.map((c, i) => (
            <div key={i}>
              {c.source} {signed(c.amount)}m
            </div>
          ))}
        </div>
      </TooltipContent>
    </Tooltip>
  )
}

function FatigueWarning() {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          className="cursor-help font-semibold text-amber-700 underline decoration-dotted underline-offset-2 dark:text-amber-300"
        >
          Fadiga ao dormir
        </button>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        <div className="max-w-[260px] text-xs">
          Dormir vestindo armadura pesada causa Fadiga (1 condição). Remova a
          armadura antes de descansar.
        </div>
      </TooltipContent>
    </Tooltip>
  )
}

function VitalsAside({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  // Snapshot taken at the start of an in-flight burst so we can roll back
  // on failure. Cleared after the debounced send settles.
  const rollbackSnapshot = useRef<Character | undefined>(undefined)

  const sendVitals = useDebouncedCallback(
    async (input: UpdateVitalsInput) => {
      try {
        const updated = await api.characters.updateVitals(character.id, input)
        qc.setQueryData(queryKey, updated)
      } catch {
        if (rollbackSnapshot.current) {
          qc.setQueryData(queryKey, rollbackSnapshot.current)
        }
      } finally {
        rollbackSnapshot.current = undefined
      }
    },
    { wait: 350 },
  )

  const setVital = (field: 'hpCurrent' | 'mpCurrent', max: number, next: number) => {
    const clamped = Math.max(0, Math.min(max, next))
    if (clamped === character[field]) return
    qc.cancelQueries({ queryKey })
    if (!rollbackSnapshot.current) {
      rollbackSnapshot.current = qc.getQueryData<Character>(queryKey)
    }
    qc.setQueryData<Character>(queryKey, (prev) =>
      prev ? { ...prev, [field]: clamped } : prev,
    )
    sendVitals({ [field]: clamped })
  }

  const setHp = (next: number) => setVital('hpCurrent', character.hpMax, next)
  const setMp = (next: number) => setVital('mpCurrent', character.mpMax, next)

  return (
    <aside
      className={cn(
        'flex min-h-0 flex-col gap-3 overflow-y-auto rounded-xl p-3 sm:p-4',
        surface,
        panelBg,
      )}
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:gap-3 lg:flex-col">
        <ResourceBar
          label="Vida"
          current={character.hpCurrent}
          max={character.hpMax}
          fromColor="from-red-700"
          toColor="to-red-500"
          accent="text-red-700 dark:text-red-300"
          className="sm:flex-1 lg:flex-none"
          onSetCurrent={setHp}
        />
        <ResourceBar
          label="Mana"
          current={character.mpCurrent}
          max={character.mpMax}
          fromColor="from-blue-700"
          toColor="to-blue-500"
          accent="text-blue-700 dark:text-blue-300"
          className="sm:flex-1 lg:flex-none"
          onSetCurrent={setMp}
        />
      </div>

      <div className="grid grid-cols-3 gap-2 sm:grid-cols-6 lg:grid-cols-3">
        <AttributeBox label="FOR" value={character.strength} />
        <AttributeBox label="DES" value={character.dexterity} />
        <AttributeBox label="CON" value={character.constitution} />
        <AttributeBox label="INT" value={character.intelligence} />
        <AttributeBox label="SAB" value={character.wisdom} />
        <AttributeBox label="CAR" value={character.charisma} />
      </div>

      <CombatStats character={character} />
      <MagicStats character={character} />
    </aside>
  )
}

type StatRow = { label: string; amount: number; muted?: boolean }

function CombatStats({ character }: { character: Character }) {
  const effects = useCharacterEffects(character)
  const def = defenseTotal(character, effects)
  const lutaState = expertiseStateFor(character, {
    name: 'Luta',
    attribute: 'strength',
    abbr: 'FOR',
  })
  const pontariaState = expertiseStateFor(character, {
    name: 'Pontaria',
    attribute: 'dexterity',
    abbr: 'DES',
  })
  const luta = expertiseTotalWithItems(character, lutaState, effects)
  const pontaria = expertiseTotalWithItems(character, pontariaState, effects)

  const defenseRows: StatRow[] = [{ label: 'Base', amount: 10 }]
  if (def.dexApplied) {
    defenseRows.push({ label: 'Destreza', amount: character.dexterity })
  } else {
    defenseRows.push({
      label: 'Destreza (bloqueada por armadura pesada)',
      amount: 0,
      muted: true,
    })
  }
  for (const c of def.contributions) {
    defenseRows.push({ label: c.source, amount: c.amount })
  }

  const attackRows = (
    e: ReturnType<typeof expertiseTotalWithItems>,
    attrAbbr: string,
  ): StatRow[] => {
    const rows: StatRow[] = [
      { label: '½ nível', amount: e.halfLevel },
      { label: attrAbbr, amount: e.attrValue },
    ]
    if (e.training) rows.push({ label: 'Treino', amount: e.training })
    for (const c of e.itemContributions) {
      rows.push({ label: c.source, amount: c.amount })
    }
    return rows
  }

  return (
    <div className="grid grid-cols-3 gap-2">
      <CombatBox
        label="Defesa"
        value={def.total}
        rows={defenseRows}
        icon={<Shield className="size-3.5" />}
      />
      <CombatBox
        label="Atq CaC"
        value={luta.total}
        rows={attackRows(luta, ATTRIBUTE_ABBR[lutaState.attribute])}
        icon={<Sword className="size-3.5" />}
        signed
      />
      <CombatBox
        label="Atq Dist"
        value={pontaria.total}
        rows={attackRows(pontaria, ATTRIBUTE_ABBR[pontariaState.attribute])}
        icon={<Crosshair className="size-3.5" />}
        signed
      />
    </div>
  )
}

function CombatBox({
  label,
  value,
  rows,
  icon,
  signed: showSigned,
}: {
  label: string
  value: number
  rows: StatRow[]
  icon: React.ReactNode
  signed?: boolean
}) {
  const display = showSigned ? signed(value) : String(value)
  return (
    <Dialog>
      <DialogTrigger asChild>
        <button
          type="button"
          className={cn(
            'relative flex cursor-pointer flex-col items-center rounded-lg border-2 p-2 text-center shadow-inner outline-none transition-colors',
            'border-red-800/50 bg-gradient-to-b from-red-100 to-red-50 text-red-900',
            'hover:from-red-200 hover:to-red-100',
            'dark:border-red-500/40 dark:from-red-950/40 dark:to-zinc-950 dark:text-red-200 dark:hover:from-red-900/40',
            'focus-visible:ring-2 focus-visible:ring-red-500/60',
          )}
          aria-label={`${label} ${display}`}
        >
          <span className="absolute left-1.5 top-1.5 text-red-700 dark:text-red-300">
            {icon}
          </span>
          <span className="text-[9px] font-bold uppercase tracking-widest text-red-800/80 dark:text-red-300/80">
            {label}
          </span>
          <span className="mt-0.5 font-serif text-2xl font-bold leading-none text-red-800 dark:text-red-100">
            {display}
          </span>
        </button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-red-700/40 bg-amber-50 text-zinc-900 dark:border-red-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle
            className={cn(
              'flex items-center gap-2 font-serif text-red-800 dark:text-red-200',
            )}
          >
            {icon}
            {label}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          <ul className="space-y-1">
            {rows.map((r, i) => (
              <li
                key={i}
                className={cn(
                  'flex items-center justify-between gap-2 border-b border-amber-700/15 pb-1 dark:border-amber-500/10',
                  r.muted && dimText,
                )}
              >
                <span className="truncate">{r.label}</span>
                <span className="shrink-0 font-mono">{signed(r.amount)}</span>
              </li>
            ))}
          </ul>
          <div
            className={cn(
              'flex items-center justify-between rounded-lg border px-3 py-2',
              'border-red-700/40 bg-red-100/60 dark:border-red-500/40 dark:bg-red-950/30',
            )}
          >
            <span className="text-xs uppercase tracking-widest text-red-800/80 dark:text-red-300/80">
              Total
            </span>
            <span className="font-mono text-2xl font-bold text-red-800 dark:text-red-100">
              {display}
            </span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function MagicStats({ character }: { character: Character }) {
  const effects = useCharacterEffects(character)
  const pmLimit = pmLimitTotal(character, effects)
  const dc = spellDCBonus(effects)
  const cost = pmCostMod(effects)

  const limitRows: StatRow[] = [
    { label: '½ nível (mín 1)', amount: pmLimit.base },
    ...pmLimit.contributions.map((c) => ({
      label: c.source,
      amount: c.amount,
    })),
  ]
  const dcRows: StatRow[] = dc.contributions.map((c) => ({
    label: c.source,
    amount: c.amount,
  }))
  const costRows: StatRow[] = cost.contributions.map((c) => ({
    label: c.source,
    amount: c.amount,
  }))

  const showDC = dc.total !== 0
  const showCost = cost.total !== 0

  return (
    <div className="grid grid-cols-3 gap-2">
      <MagicBox
        label="Limite PM"
        value={pmLimit.total}
        rows={limitRows}
        icon={<Zap className="size-3.5" />}
      />
      <MagicBox
        label="CD Magia"
        value={dc.total}
        rows={
          showDC
            ? dcRows
            : [{ label: 'Sem bônus de itens', amount: 0, muted: true }]
        }
        icon={<Sparkles className="size-3.5" />}
        signed
        prefix="+"
      />
      <MagicBox
        label="Custo PM"
        value={cost.total}
        rows={
          showCost
            ? costRows
            : [{ label: 'Sem mod de itens', amount: 0, muted: true }]
        }
        icon={<Sparkles className="size-3.5" />}
        signed
      />
    </div>
  )
}

function MagicBox({
  label,
  value,
  rows,
  icon,
  signed: showSigned,
  prefix,
}: {
  label: string
  value: number
  rows: StatRow[]
  icon: React.ReactNode
  signed?: boolean
  prefix?: string
}) {
  const display = showSigned ? signed(value) : `${prefix ?? ''}${value}`
  return (
    <Dialog>
      <DialogTrigger asChild>
        <button
          type="button"
          className={cn(
            'relative flex cursor-pointer flex-col items-center rounded-lg border-2 p-2 text-center shadow-inner outline-none transition-colors',
            'border-violet-800/50 bg-gradient-to-b from-violet-100 to-violet-50 text-violet-900',
            'hover:from-violet-200 hover:to-violet-100',
            'dark:border-violet-500/40 dark:from-violet-950/40 dark:to-zinc-950 dark:text-violet-200 dark:hover:from-violet-900/40',
            'focus-visible:ring-2 focus-visible:ring-violet-500/60',
          )}
          aria-label={`${label} ${display}`}
        >
          <span className="absolute left-1.5 top-1.5 text-violet-700 dark:text-violet-300">
            {icon}
          </span>
          <span className="text-[9px] font-bold uppercase tracking-widest text-violet-800/80 dark:text-violet-300/80">
            {label}
          </span>
          <span className="mt-0.5 font-serif text-2xl font-bold leading-none text-violet-800 dark:text-violet-100">
            {display}
          </span>
        </button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-violet-700/40 bg-amber-50 text-zinc-900 dark:border-violet-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle
            className={cn(
              'flex items-center gap-2 font-serif text-violet-800 dark:text-violet-200',
            )}
          >
            {icon}
            {label}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          <ul className="space-y-1">
            {rows.map((r, i) => (
              <li
                key={i}
                className={cn(
                  'flex items-center justify-between gap-2 border-b border-amber-700/15 pb-1 dark:border-amber-500/10',
                  r.muted && dimText,
                )}
              >
                <span className="truncate">{r.label}</span>
                <span className="shrink-0 font-mono">{signed(r.amount)}</span>
              </li>
            ))}
          </ul>
          <div
            className={cn(
              'flex items-center justify-between rounded-lg border px-3 py-2',
              'border-violet-700/40 bg-violet-100/60 dark:border-violet-500/40 dark:bg-violet-950/30',
            )}
          >
            <span className="text-xs uppercase tracking-widest text-violet-800/80 dark:text-violet-300/80">
              Total
            </span>
            <span className="font-mono text-2xl font-bold text-violet-800 dark:text-violet-100">
              {display}
            </span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function normalize(s: string): string {
  return s.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase()
}

function ExpertisesPanel({ character }: { character: Character }) {
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
    onSuccess: () => qc.invalidateQueries({ queryKey }),
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
        'flex min-h-0 flex-col overflow-hidden rounded-xl',
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
          <div className="grid gap-x-4 gap-y-0.5 xl:grid-cols-2">
            <ExpertiseHeader className="hidden sm:flex" />
            <ExpertiseHeader className="hidden xl:flex" />
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

function signed(n: number): string {
  return n >= 0 ? `+${n}` : String(n)
}

function ExpertiseHeader({ className }: { className?: string }) {
  const cell = cn('text-[9px] uppercase tracking-widest', dimText)
  return (
    <div
      className={cn(
        'items-center gap-2 border-b border-amber-700/20 px-2 pb-1 pt-2 dark:border-amber-500/15',
        className,
      )}
    >
      <span className="w-4 shrink-0" aria-hidden />
      <span className={cn(cell, 'flex-1')}>perícia</span>
      <span className={cn(cell, 'w-10 text-right')}>total</span>
      <span className={cn(cell, 'w-8 text-center')}>½lvl</span>
      <span className={cn(cell, 'w-16 text-center')}>atributo</span>
      <span className={cn(cell, 'w-10 text-center')}>treino</span>
      <span className={cn(cell, 'w-20 text-center')}>outros</span>
      <span className="size-7 shrink-0" aria-hidden />
    </div>
  )
}

const selectClass =
  'cursor-pointer rounded border bg-amber-50 text-zinc-900 outline-none focus:border-amber-700/60 dark:bg-zinc-900 dark:text-zinc-200 dark:focus:border-amber-500/60 border-amber-700/30 dark:border-zinc-700'

function ExpertiseRow({
  character,
  def,
  effects,
  onDelete,
}: {
  character: Character
  def: ExpertiseDef
  effects: ItemEffects
  onDelete?: () => void
}) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const state = expertiseStateFor(character, def)

  type ExpertisePatch = {
    attribute?: AttributeKey
    trained?: boolean
  }

  const mutation = useMutation<
    CharacterExpertise,
    Error,
    ExpertisePatch,
    { previous: Character | undefined }
  >({
    mutationFn: (input) =>
      api.characters.updateExpertise(character.id, { name: def.name, ...input }),
    onMutate: async (input) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) => {
        if (!prev) return prev
        return {
          ...prev,
          expertises: prev.expertises.map((e) =>
            e.name === def.name ? { ...e, ...input } : e,
          ),
        }
      })
      return { previous }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (updated) => {
      qc.setQueryData<Character>(queryKey, (prev) => {
        if (!prev) return prev
        return {
          ...prev,
          expertises: prev.expertises.map((e) =>
            e.name === updated.name ? updated : e,
          ),
        }
      })
    },
  })

  const detail = expertiseTotalWithItems(character, state, effects)
  const total = detail.total
  const halfLevel = Math.floor(character.level / 2)
  const trainBonus = state.trained ? trainingBonusForLevel(character.level) : 0
  const othersDisplay = detail.itemBonus

  const trainedToggle = (
    <input
      type="checkbox"
      className="h-4 w-4 cursor-pointer accent-amber-600 dark:accent-amber-500"
      checked={state.trained}
      onChange={(e) => mutation.mutate({ trained: e.target.checked })}
      aria-label={`${def.name} treinada`}
    />
  )

  const locked = !!def.trainedOnly && !state.trained

  const totalLabel = (
    <span
      className={cn(
        'font-mono text-base font-semibold',
        locked
          ? 'text-zinc-400 line-through dark:text-zinc-600'
          : accentStrong,
      )}
      title={locked ? 'Apenas treinada — não pode ser usada sem treino' : undefined}
    >
      {signed(total)}
    </span>
  )

  const nameNode = (
    <span
      className={cn(
        'flex flex-1 items-center gap-1.5 truncate text-sm',
        locked
          ? 'text-zinc-500 dark:text-zinc-500'
          : 'text-zinc-800 dark:text-zinc-200',
      )}
    >
      <span className="truncate">{def.name}</span>
      {def.trainedOnly && (
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              tabIndex={0}
              aria-label="Apenas treinada"
              className="inline-flex shrink-0 cursor-help"
            >
              <Star
                className={cn(
                  'size-3',
                  locked
                    ? 'fill-amber-500 text-amber-700 dark:fill-amber-400 dark:text-amber-300'
                    : 'fill-zinc-300 text-zinc-500 dark:fill-zinc-700 dark:text-zinc-500',
                )}
              />
            </button>
          </TooltipTrigger>
          <TooltipContent side="top">
            Pode ser usada apenas quando treinada
          </TooltipContent>
        </Tooltip>
      )}
    </span>
  )

  const othersInput = (
    <OthersDisplay
      total={othersDisplay}
      detail={detail}
      expertiseName={def.name}
    />
  )


  return (
    <>
      {/* Mobile: compact row + dialog */}
      <div
        className={cn(
          'flex items-center gap-2 rounded-md px-2 py-1.5 sm:hidden',
          hoverRow,
        )}
      >
        {trainedToggle}
        {nameNode}
        <span className="w-10 text-right">{totalLabel}</span>
        {onDelete && <DeleteExpertiseButton name={def.name} onDelete={onDelete} />}
        <Dialog>
          <DialogTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className={cn(
                'size-7',
                subtleText,
                'hover:bg-amber-200/60 hover:text-amber-900 dark:hover:bg-zinc-800/60 dark:hover:text-amber-200',
              )}
              aria-label={`Editar ${def.name}`}
            >
              <SlidersHorizontal className="size-4" />
            </Button>
          </DialogTrigger>
          <DialogContent
            className={cn(
              'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
              'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
            )}
          >
            <DialogHeader>
              <DialogTitle
                className={cn('flex items-center gap-2 font-serif', accentStrong)}
              >
                {def.name}
                {def.trainedOnly && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        type="button"
                        aria-label="Apenas treinada"
                        className="inline-flex cursor-help"
                      >
                        <Star
                          className={cn(
                            'size-4',
                            locked
                              ? 'fill-amber-500 text-amber-700 dark:fill-amber-400 dark:text-amber-300'
                              : 'fill-zinc-300 text-zinc-500 dark:fill-zinc-700 dark:text-zinc-500',
                          )}
                        />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>
                      Pode ser usada apenas quando treinada
                    </TooltipContent>
                  </Tooltip>
                )}
              </DialogTitle>
            </DialogHeader>
            {def.trainedOnly && !state.trained && (
              <p
                className={cn(
                  'rounded-md border px-3 py-2 text-xs',
                  'border-amber-700/40 bg-amber-100/60 text-amber-900 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200',
                )}
              >
                Esta perícia exige treino para ser usada.
              </p>
            )}
            <div className="space-y-4">
              <div
                className={cn(
                  'flex items-center justify-between rounded-lg border px-4 py-2',
                  'border-amber-700/30 bg-amber-100/70 dark:border-amber-500/30 dark:bg-zinc-900/60',
                )}
              >
                <span
                  className={cn(
                    'text-xs uppercase tracking-widest',
                    subtleText,
                  )}
                >
                  total
                </span>
                <span
                  className={cn(
                    'font-mono text-2xl font-bold',
                    locked
                      ? 'text-zinc-400 line-through dark:text-zinc-500'
                      : accentStrong,
                  )}
                >
                  {signed(total)}
                </span>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <DialogField label="½ nível">
                  <span className="font-mono text-sm text-zinc-800 dark:text-zinc-200">
                    {halfLevel}
                  </span>
                </DialogField>
                <DialogField label="atributo">
                  <select
                    value={state.attribute}
                    onChange={(e) =>
                      mutation.mutate({
                        attribute: e.target.value as AttributeKey,
                      })
                    }
                    className={cn(selectClass, 'h-7 px-2 font-mono text-xs')}
                    aria-label={`${def.name} atributo`}
                  >
                    {ATTRIBUTE_KEYS.map((k) => (
                      <option key={k} value={k}>
                        {ATTRIBUTE_ABBR[k]} {signed(character[k])}
                      </option>
                    ))}
                  </select>
                </DialogField>
                <DialogField label="treino">
                  <div className="flex items-center gap-2">
                    {trainedToggle}
                    <span className="font-mono text-sm text-zinc-800 dark:text-zinc-200">
                      {signed(trainBonus)}
                    </span>
                  </div>
                </DialogField>
                <DialogField label="outros">{othersInput}</DialogField>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* Desktop: full breakdown row */}
      <div
        className={cn(
          'hidden items-center gap-2 rounded-md px-2 py-1 sm:flex',
          hoverRow,
        )}
      >
        {trainedToggle}
        {nameNode}
        <span className="w-10 text-right">{totalLabel}</span>
        <span className="w-8 text-center font-mono text-xs text-zinc-700 dark:text-zinc-300">
          {halfLevel}
        </span>
        <select
          value={state.attribute}
          onChange={(e) =>
            mutation.mutate({ attribute: e.target.value as AttributeKey })
          }
          className={cn(selectClass, 'h-7 w-16 px-1 font-mono text-[11px]')}
          aria-label={`${def.name} atributo`}
        >
          {ATTRIBUTE_KEYS.map((k) => (
            <option key={k} value={k}>
              {ATTRIBUTE_ABBR[k]} {signed(character[k])}
            </option>
          ))}
        </select>
        <span className="w-10 text-center font-mono text-xs text-zinc-700 dark:text-zinc-300">
          {signed(trainBonus)}
        </span>
        <div className="w-20">
          <OthersDisplay
            total={othersDisplay}
            detail={detail}
            expertiseName={def.name}
          />
        </div>
        {onDelete ? (
          <DeleteExpertiseButton name={def.name} onDelete={onDelete} />
        ) : (
          <span className="size-7 shrink-0" aria-hidden />
        )}
      </div>
    </>
  )
}

function DeleteExpertiseButton({
  name,
  onDelete,
}: {
  name: string
  onDelete: () => void
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 text-zinc-500 hover:bg-red-100 hover:text-red-700 dark:text-zinc-400 dark:hover:bg-red-950/40 dark:hover:text-red-400"
          onClick={() => {
            if (confirm(`Remover ofício "${name}"?`)) onDelete()
          }}
          aria-label={`Remover ${name}`}
        >
          <Trash2 className="size-3.5" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>Remover ofício</TooltipContent>
    </Tooltip>
  )
}

function OthersDisplay({
  total,
  detail,
  expertiseName,
}: {
  total: number
  detail: ReturnType<typeof expertiseTotalWithItems>
  expertiseName: string
}) {
  const hasContribs = detail.itemContributions.length > 0
  const display = (
    <span
      className={cn(
        'block w-full rounded-md border bg-zinc-100/60 px-2 py-1 text-center font-mono text-xs',
        'border-amber-700/20 text-zinc-700 dark:border-amber-500/20 dark:bg-zinc-900/40 dark:text-zinc-300',
        total === 0 && 'text-zinc-400 dark:text-zinc-600',
        hasContribs &&
          'cursor-pointer hover:bg-amber-100/60 dark:hover:bg-zinc-800/60',
      )}
    >
      {signed(total)}
    </span>
  )
  if (!hasContribs) return display
  return (
    <Dialog>
      <DialogTrigger asChild>
        <button
          type="button"
          className="block w-full"
          aria-label={`Detalhes de Outros — ${expertiseName}`}
        >
          {display}
        </button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle
            className={cn('flex items-center gap-2 font-serif', accentStrong)}
          >
            Outros — {expertiseName}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          <ul className="space-y-1">
            {detail.itemContributions.map((c, i) => (
              <li
                key={i}
                className="flex items-center justify-between gap-2 border-b border-amber-700/15 pb-1 dark:border-amber-500/10"
              >
                <span className="truncate">{c.source}</span>
                <span className="shrink-0 font-mono">{signed(c.amount)}</span>
              </li>
            ))}
          </ul>
          <div
            className={cn(
              'flex items-center justify-between rounded-lg border px-3 py-2',
              'border-amber-700/40 bg-amber-100/60 dark:border-amber-500/40 dark:bg-amber-950/30',
            )}
          >
            <span
              className={cn(
                'text-xs uppercase tracking-widest',
                subtleText,
              )}
            >
              Total
            </span>
            <span
              className={cn(
                'font-mono text-2xl font-bold',
                accentStrong,
              )}
            >
              {signed(total)}
            </span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function DialogField({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-1">
      <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
        {label}
      </span>
      {children}
    </div>
  )
}

function ResourceBar({
  label,
  current,
  max,
  fromColor,
  toColor,
  accent,
  className,
  onSetCurrent,
}: {
  label: string
  current: number
  max: number
  fromColor: string
  toColor: string
  accent: string
  className?: string
  onSetCurrent?: (next: number) => void
}) {
  const pct = max > 0 ? Math.max(0, Math.min(100, (current / max) * 100)) : 0
  return (
    <div
      className={cn(
        'rounded-lg border p-2.5',
        'border-amber-700/30 bg-amber-100/60 dark:border-zinc-800 dark:bg-zinc-900/60',
        className,
      )}
    >
      <div className="flex items-baseline justify-between">
        <p className={cn('text-[10px] uppercase tracking-[0.3em]', accent)}>
          {label}
        </p>
        <p className="font-mono text-base">
          <span className="font-bold">{current}</span>
          <span className={dimText}> / {max}</span>
        </p>
      </div>
      <div className="mt-1.5 h-2.5 overflow-hidden rounded-full border border-amber-700/30 bg-amber-50/70 dark:border-zinc-800 dark:bg-zinc-950">
        <div
          className={`h-full bg-gradient-to-r ${fromColor} ${toColor} transition-all`}
          style={{ width: `${pct}%` }}
        />
      </div>
      {onSetCurrent && (
        <ResourceControls
          label={label}
          current={current}
          max={max}
          onSetCurrent={onSetCurrent}
        />
      )}
    </div>
  )
}

function ResourceControls({
  label,
  current,
  max,
  onSetCurrent,
}: {
  label: string
  current: number
  max: number
  onSetCurrent: (next: number) => void
}) {
  return (
    <div className="mt-2 flex items-center gap-1.5">
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="size-7"
        disabled={current <= 0}
        onClick={() => onSetCurrent(current - 1)}
        aria-label={`Reduzir ${label} em 1`}
      >
        <Minus className="size-3.5" />
      </Button>
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="size-7"
        disabled={current >= max}
        onClick={() => onSetCurrent(current + 1)}
        aria-label={`Aumentar ${label} em 1`}
      >
        <Plus className="size-3.5" />
      </Button>
      <ResourceAdjustDialog
        label={label}
        current={current}
        max={max}
        onSetCurrent={onSetCurrent}
      />
    </div>
  )
}

function ResourceAdjustDialog({
  label,
  current,
  max,
  onSetCurrent,
}: {
  label: string
  current: number
  max: number
  onSetCurrent: (next: number) => void
}) {
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<'add' | 'remove'>('remove')
  const [amount, setAmount] = useState<string>('')

  const parsedAmount = Math.max(0, Number(amount) || 0)
  const delta = mode === 'add' ? parsedAmount : -parsedAmount
  const previewRaw = current + delta
  const preview = Math.max(0, Math.min(max, previewRaw))
  const clamped = preview !== previewRaw

  const reset = () => {
    setAmount('')
    setMode('remove')
  }

  const apply = () => {
    if (parsedAmount === 0) return
    onSetCurrent(preview)
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
          variant="outline"
          size="icon"
          className="size-7"
          aria-label={`Editar ${label}`}
        >
          <Pencil className="size-3.5" />
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
            Ajustar {label}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-2">
            <Button
              type="button"
              variant={mode === 'remove' ? 'default' : 'outline'}
              onClick={() => setMode('remove')}
              className="gap-1"
            >
              <Minus className="size-4" /> Remover
            </Button>
            <Button
              type="button"
              variant={mode === 'add' ? 'default' : 'outline'}
              onClick={() => setMode('add')}
              className="gap-1"
            >
              <Plus className="size-4" /> Adicionar
            </Button>
          </div>

          <div className="space-y-1">
            <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
              quantidade
            </span>
            <NumberInput
              value={amount}
              onChange={(v) => setAmount(String(v))}
              min={0}
              max={9999}
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') apply()
              }}
              aria-label="Quantidade"
            />
          </div>

          <div
            className={cn(
              'flex items-center justify-between rounded-lg border px-4 py-2',
              'border-amber-700/30 bg-amber-100/70 dark:border-amber-500/30 dark:bg-zinc-900/60',
            )}
          >
            <div className="flex flex-col">
              <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
                novo total
              </span>
              <span className={cn('text-[10px]', dimText)}>
                {current} {delta >= 0 ? '+' : '−'} {Math.abs(delta)}
                {clamped && ' (limitado)'}
              </span>
            </div>
            <span
              className={cn(
                'font-mono text-2xl font-bold',
                accentStrong,
              )}
            >
              {preview}
              <span className={cn('ml-1 text-sm font-normal', dimText)}>
                / {max}
              </span>
            </span>
          </div>

          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="button" onClick={apply} disabled={parsedAmount === 0}>
              Aplicar
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function formatLoad(n: number): string {
  if (Number.isInteger(n)) return String(n)
  return n.toFixed(1).replace('.', ',')
}

function inventoryUsed(items: CharacterItem[]): number {
  return items.reduce((sum, it) => sum + it.quantity * it.slots, 0)
}

function InventoryPanel({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const effects = useCharacterEffects(character)
  const max = inventorySlotsTotal(character, effects)
  const used = inventoryUsed(character.items)
  const pct = max > 0 ? Math.min(100, (used / max) * 100) : 0
  const over = used > max

  const addItem = useMutation<
    CharacterItem,
    Error,
    CreateItemInput,
    { previous: Character | undefined; tempId: number }
  >({
    mutationFn: (input) => api.characters.addItem(character.id, input),
    onMutate: async (input) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      const tempId = -Date.now()
      const catalog = input.catalogId
        ? CATALOG_ITEMS.find((c) => c.id === input.catalogId)
        : undefined
      const optimistic: CharacterItem = {
        id: tempId,
        catalogId: input.catalogId ?? null,
        name: input.name ?? catalog?.name ?? '...',
        quantity: input.quantity,
        slots: input.slots ?? catalog?.slots ?? 1,
        equipped: input.equipped ?? null,
        improvements: JSON.stringify(input.improvements ?? []),
        material: input.material ?? null,
      }
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, items: [...prev.items, optimistic] } : prev,
      )
      return { previous, tempId }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (created, _v, ctx) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              items: prev.items.map((it) =>
                ctx && it.id === ctx.tempId ? created : it,
              ),
            }
          : prev,
      )
    },
  })

  const updateItem = useMutation<
    CharacterItem,
    Error,
    { itemId: number; input: UpdateItemInput },
    { previous: Character | undefined }
  >({
    mutationFn: ({ itemId, input }) =>
      api.characters.updateItem(character.id, itemId, input),
    onMutate: async ({ itemId, input }) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              items: prev.items.map((it) => {
                if (it.id !== itemId) return it
                const { improvements, ...rest } = input
                const merged: CharacterItem = { ...it, ...rest }
                if (improvements !== undefined) {
                  merged.improvements = JSON.stringify(improvements)
                }
                return merged
              }),
            }
          : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (updated) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              items: prev.items.map((it) => (it.id === updated.id ? updated : it)),
            }
          : prev,
      )
    },
  })

  const removeItem = useMutation<
    { id: number },
    Error,
    number,
    { previous: Character | undefined }
  >({
    mutationFn: (itemId) => api.characters.deleteItem(character.id, itemId),
    onMutate: async (itemId) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? { ...prev, items: prev.items.filter((it) => it.id !== itemId) }
          : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
  })

  const consumeItem = useMutation<Character, Error, number>({
    mutationFn: (itemId) => api.characters.consumeItem(character.id, itemId),
    onSuccess: (next) => qc.setQueryData<Character>(queryKey, next),
  })

  const items = character.items

  return (
    <section
      className={cn(
        'flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl',
        surface,
        panelBg,
      )}
    >
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-amber-700/30 px-3 py-2 dark:border-amber-500/20 sm:px-4">
        <div className="min-w-0">
          <h2
            className={cn(
              'font-serif text-lg font-bold tracking-wide',
              accentStrong,
            )}
          >
            Inventário
          </h2>
          <p className={cn('text-[10px] sm:text-xs', dimText)}>
            carga{' '}
            <span
              className={cn(
                'font-mono',
                over
                  ? 'text-red-700 dark:text-red-400'
                  : 'text-zinc-700 dark:text-zinc-300',
              )}
            >
              {formatLoad(used)}
            </span>{' '}
            / {max}
            {over && (
              <span className="ml-2 text-[10px] uppercase tracking-widest text-red-700 dark:text-red-400">
                sobrecarga
              </span>
            )}
            <span className="ml-2">• limite 10 + 2×|FOR|</span>
          </p>
        </div>
        <div className="flex items-center gap-2">
          <AddCatalogItemDialog
            onAdd={(input, fail) => addItem.mutate(input, { onError: fail })}
          />
          <ItemFormDialog
            title="Novo item"
            submitLabel="Adicionar"
            trigger={
              <Button
                type="button"
                size="sm"
                variant="outline"
                className="h-7 gap-1 text-xs"
                aria-label="Adicionar item custom"
              >
                <Plus className="size-3.5" />
                Custom
              </Button>
            }
            onSubmit={(input, fail) => addItem.mutate(input, { onError: fail })}
          />
        </div>
      </div>

      <div className="shrink-0 px-3 pb-2 pt-2 sm:px-4">
        <div className="h-2 overflow-hidden rounded-full border border-amber-700/30 bg-amber-50/70 dark:border-zinc-800 dark:bg-zinc-950">
          <div
            className={cn(
              'h-full transition-all',
              over
                ? 'bg-gradient-to-r from-red-700 to-red-500'
                : 'bg-gradient-to-r from-amber-700 to-amber-500',
            )}
            style={{ width: `${pct}%` }}
          />
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto px-2 py-1">
        {items.length === 0 ? (
          <p className={cn('px-2 py-3 text-center text-xs', dimText)}>
            Nenhum item. Use "+ Item" para adicionar.
          </p>
        ) : (
          <div className="grid gap-y-0.5">
            <InventoryHeader className="hidden sm:flex" />
            {items.map((it) => (
              <InventoryRow
                key={it.id}
                item={it}
                proficient={isItemProficient(character, it)}
                onUpdate={(input, fail) =>
                  updateItem.mutate({ itemId: it.id, input }, { onError: fail })
                }
                onDelete={() => removeItem.mutate(it.id)}
                onConsume={() => consumeItem.mutate(it.id)}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

function InventoryHeader({ className }: { className?: string }) {
  const cell = cn('text-[9px] uppercase tracking-widest', dimText)
  return (
    <div
      className={cn(
        'items-center gap-2 border-b border-amber-700/20 px-2 pb-1 pt-2 dark:border-amber-500/15',
        className,
      )}
    >
      <span className={cn(cell, 'flex-1')}>item</span>
      <span className={cn(cell, 'w-12 text-center')}>qtd</span>
      <span className={cn(cell, 'w-14 text-center')}>esp</span>
      <span className={cn(cell, 'w-14 text-right')}>total</span>
      <span className={cn(cell, 'w-20 text-center')}>equipar</span>
      <span className="size-7 shrink-0" aria-hidden />
      <span className="size-7 shrink-0" aria-hidden />
      <span className="size-7 shrink-0" aria-hidden />
    </div>
  )
}

const EQUIP_OPTIONS: { value: '' | 'vested' | 'wielded' | 'wielded2'; label: string }[] = [
  { value: '', label: '—' },
  { value: 'vested', label: 'Vestido' },
  { value: 'wielded', label: '1 mão' },
  { value: 'wielded2', label: '2 mãos' },
]

function EquipSelect({
  value,
  onChange,
  itemName,
  className,
}: {
  value: CharacterItem['equipped']
  onChange: (next: CharacterItem['equipped']) => void
  itemName: string
  className?: string
}) {
  return (
    <select
      value={value ?? ''}
      onChange={(e) => onChange((e.target.value || null) as CharacterItem['equipped'])}
      className={cn(selectClass, 'h-7 px-1 font-mono text-[11px]', className)}
      aria-label={`Equipar ${itemName}`}
    >
      {EQUIP_OPTIONS.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  )
}

function OverlayPickerDialog({
  item,
  onUpdate,
}: {
  item: CharacterItem
  onUpdate: (input: UpdateItemInput, onError: (e: Error) => void) => void
}) {
  const catalog = item.catalogId ? getCatalogItem(item.catalogId) : undefined
  const [open, setOpen] = useState(false)
  const [improvements, setImprovements] = useState<string[]>(
    parseImprovementIds(item.improvements),
  )
  const [material, setMaterial] = useState<string | null>(item.material)
  const [error, setError] = useState<string | null>(null)

  if (!catalog) return null
  if (
    catalog.category === 'consumable' ||
    catalog.category === 'meal' ||
    catalog.category === 'catalyst' ||
    catalog.category === 'improvement' ||
    catalog.category === 'material' ||
    catalog.category === 'animal' ||
    catalog.category === 'vehicle'
  ) {
    return null
  }

  const baseFamily = familyFor(catalog)
  const availableImprovements = IMPROVEMENTS.filter((imp) =>
    imp.appliesTo?.includes(baseFamily),
  )
  const availableMaterials = MATERIALS.filter((mat) =>
    mat.appliesTo?.includes(baseFamily),
  )

  const reset = () => {
    setImprovements(parseImprovementIds(item.improvements))
    setMaterial(item.material)
    setError(null)
  }

  const toggleImprovement = (id: string) => {
    setImprovements((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    )
  }

  const apply = () => {
    setError(null)
    onUpdate({ improvements, material }, (e) => setError(e.message))
    setOpen(false)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) reset()
      }}
    >
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            'size-7',
            subtleText,
            'hover:bg-amber-200/60 hover:text-amber-900 dark:hover:bg-zinc-800/60 dark:hover:text-amber-200',
          )}
          aria-label={`Melhorias e material de ${item.name}`}
        >
          <Gem className="size-3.5" />
        </Button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-md sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn('font-serif', accentStrong)}>
            Melhorias & Material — {item.name}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <section>
            <h3 className={cn('text-xs uppercase tracking-widest', dimText)}>
              Melhorias
            </h3>
            {availableImprovements.length === 0 ? (
              <p className={cn('mt-1 text-xs italic', dimText)}>
                Nenhuma melhoria compatível.
              </p>
            ) : (
              <ul className="mt-2 space-y-1">
                {availableImprovements.map((imp) => {
                  const checked = improvements.includes(imp.id)
                  return (
                    <li key={imp.id}>
                      <label className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-amber-100 dark:hover:bg-zinc-900/60">
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => toggleImprovement(imp.id)}
                          className="mt-0.5"
                        />
                        <span className="flex-1">
                          <span className="font-semibold">{imp.name}</span>
                          <span className={cn('ml-2', dimText)}>
                            {imp.modifiers
                              .map((m) => m.note ?? '')
                              .filter(Boolean)
                              .join(', ')}
                          </span>
                        </span>
                      </label>
                    </li>
                  )
                })}
              </ul>
            )}
          </section>
          <section>
            <h3 className={cn('text-xs uppercase tracking-widest', dimText)}>
              Material
            </h3>
            {availableMaterials.length === 0 ? (
              <p className={cn('mt-1 text-xs italic', dimText)}>
                Nenhum material compatível.
              </p>
            ) : (
              <ul className="mt-2 space-y-1">
                <li>
                  <label className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-amber-100 dark:hover:bg-zinc-900/60">
                    <input
                      type="radio"
                      name={`material-${item.id}`}
                      checked={material === null}
                      onChange={() => setMaterial(null)}
                      className="mt-0.5"
                    />
                    <span className={cn('flex-1 italic', dimText)}>nenhum</span>
                  </label>
                </li>
                {availableMaterials.map((mat) => (
                  <li key={mat.id}>
                    <label className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-amber-100 dark:hover:bg-zinc-900/60">
                      <input
                        type="radio"
                        name={`material-${item.id}`}
                        checked={material === mat.id}
                        onChange={() => setMaterial(mat.id)}
                        className="mt-0.5"
                      />
                      <span className="flex-1">
                        <span className="font-semibold">{mat.name}</span>
                        <span className={cn('ml-2', dimText)}>
                          {mat.modifiers
                            .map((m) => m.note ?? '')
                            .filter(Boolean)
                            .join(', ')}
                        </span>
                      </span>
                    </label>
                  </li>
                ))}
              </ul>
            )}
          </section>
          {error ? (
            <p className="text-xs text-red-700 dark:text-red-400">{error}</p>
          ) : null}
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => setOpen(false)}
            >
              Cancelar
            </Button>
            <Button type="button" onClick={apply}>
              Aplicar
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function InventoryRow({
  item,
  proficient,
  onUpdate,
  onDelete,
  onConsume,
}: {
  item: CharacterItem
  proficient: boolean
  onUpdate: (input: UpdateItemInput, onError: (e: Error) => void) => void
  onDelete: () => void
  onConsume: () => void
}) {
  const total = item.quantity * item.slots
  const catalog = item.catalogId ? getCatalogItem(item.catalogId) : undefined
  const consumable = catalog?.consumable
  const equipped = item.equipped !== null
  const proficiencyWarning = equipped && !proficient ? (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          className="inline-flex size-5 items-center justify-center rounded-full bg-red-100 text-red-700 dark:bg-red-950/40 dark:text-red-400"
          aria-label="Sem proficiência"
        >
          <AlertTriangle className="size-3" />
        </span>
      </TooltipTrigger>
      <TooltipContent>
        {catalog?.category.startsWith('weapon-')
          ? 'Sem proficiência: -5 nos testes de ataque'
          : 'Sem proficiência: não aplica Des na Defesa'}
      </TooltipContent>
    </Tooltip>
  ) : null
  const useButton = consumable ? (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 text-emerald-700 hover:bg-emerald-100 hover:text-emerald-800 dark:text-emerald-400 dark:hover:bg-emerald-950/40"
          onClick={onConsume}
          aria-label={`Usar ${item.name}`}
        >
          <Sparkles className="size-3.5" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>
        Usar ({consumable.scope === 'instant'
          ? 'imediato'
          : consumable.scope === 'scene'
            ? '1 cena'
            : '1 dia'}
        )
      </TooltipContent>
    </Tooltip>
  ) : null
  const editTrigger = (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className={cn(
        'size-7',
        subtleText,
        'hover:bg-amber-200/60 hover:text-amber-900 dark:hover:bg-zinc-800/60 dark:hover:text-amber-200',
      )}
      aria-label={`Editar ${item.name}`}
    >
      <Pencil className="size-3.5" />
    </Button>
  )
  const deleteButton = (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 text-zinc-500 hover:bg-red-100 hover:text-red-700 dark:text-zinc-400 dark:hover:bg-red-950/40 dark:hover:text-red-400"
          onClick={() => {
            if (confirm(`Remover "${item.name}"?`)) onDelete()
          }}
          aria-label={`Remover ${item.name}`}
        >
          <Trash2 className="size-3.5" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>Remover item</TooltipContent>
    </Tooltip>
  )
  return (
    <>
      <div
        className={cn(
          'hidden items-center gap-2 rounded-md px-2 py-1 sm:flex',
          hoverRow,
        )}
      >
        <span className="flex flex-1 items-center gap-1.5 truncate text-sm text-zinc-800 dark:text-zinc-200">
          <span className="truncate">{item.name}</span>
          {proficiencyWarning}
        </span>
        <span className="w-12 text-center font-mono text-xs text-zinc-700 dark:text-zinc-300">
          {item.quantity}
        </span>
        <span className="w-14 text-center font-mono text-xs text-zinc-700 dark:text-zinc-300">
          {formatLoad(item.slots)}
        </span>
        <span
          className={cn(
            'w-14 text-right font-mono text-xs font-semibold',
            accentStrong,
          )}
        >
          {formatLoad(total)}
        </span>
        <EquipSelect
          value={item.equipped}
          onChange={(v) => onUpdate({ equipped: v }, () => {})}
          itemName={item.name}
          className="w-20"
        />
        <ItemInfoDialog item={item} />
        <OverlayPickerDialog item={item} onUpdate={onUpdate} />
        {useButton}
        <ItemFormDialog
          title={`Editar ${item.name}`}
          submitLabel="Salvar"
          trigger={editTrigger}
          initial={{ name: item.name, quantity: item.quantity, slots: item.slots }}
          onSubmit={(input, fail) => onUpdate(input, fail)}
        />
        {deleteButton}
      </div>
      <div
        className={cn(
          'flex items-center gap-2 rounded-md px-2 py-1.5 sm:hidden',
          hoverRow,
        )}
      >
        <div className="flex min-w-0 flex-1 flex-col leading-tight">
          <span className="flex items-center gap-1.5 truncate text-sm text-zinc-800 dark:text-zinc-200">
            <span className="truncate">{item.name}</span>
            {proficiencyWarning}
          </span>
          <span className={cn('truncate text-[10px]', dimText)}>
            {item.quantity} × {formatLoad(item.slots)} ={' '}
            <span className={cn('font-semibold', accentStrong)}>
              {formatLoad(total)}
            </span>
          </span>
        </div>
        <EquipSelect
          value={item.equipped}
          onChange={(v) => onUpdate({ equipped: v }, () => {})}
          itemName={item.name}
          className="w-20"
        />
        <ItemInfoDialog item={item} />
        <OverlayPickerDialog item={item} onUpdate={onUpdate} />
        {useButton}
        <ItemFormDialog
          title={`Editar ${item.name}`}
          submitLabel="Salvar"
          trigger={editTrigger}
          initial={{ name: item.name, quantity: item.quantity, slots: item.slots }}
          onSubmit={(input, fail) => onUpdate(input, fail)}
        />
        {deleteButton}
      </div>
    </>
  )
}

function AddCatalogItemDialog({
  onAdd,
}: {
  onAdd: (input: CreateItemInput, onError: (e: Error) => void) => void
}) {
  const [open, setOpen] = useState(false)
  const [catalogId, setCatalogId] = useState('')
  const [search, setSearch] = useState('')
  const [quantity, setQuantity] = useState<string>('1')
  const [equipped, setEquipped] = useState<'' | 'vested' | 'wielded' | 'wielded2'>('')
  const [error, setError] = useState<string | null>(null)

  const reset = () => {
    setCatalogId('')
    setSearch('')
    setQuantity('1')
    setEquipped('')
    setError(null)
  }

  const selected = catalogId
    ? CATALOG_ITEMS.find((c) => c.id === catalogId)
    : undefined

  const filtered =
    search.trim() === ''
      ? CATALOG_ITEMS
      : CATALOG_ITEMS.filter(
          (c) =>
            normalize(c.name).includes(normalize(search)) ||
            normalize(c.category).includes(normalize(search)),
        )

  const apply = () => {
    if (!selected) {
      setError('Selecione um item do catálogo.')
      return
    }
    const qty = Number(quantity)
    if (!Number.isInteger(qty) || qty < 1) {
      setError('Quantidade deve ser inteiro ≥ 1.')
      return
    }
    onAdd(
      {
        catalogId: selected.id,
        quantity: qty,
        equipped: equipped || undefined,
      },
      (e) => setError(e.message),
    )
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
          aria-label="Adicionar do catálogo"
        >
          <Plus className="size-3.5" />
          Catálogo
        </Button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-md sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn('font-serif', accentStrong)}>
            Adicionar do catálogo
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-1">
            <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
              item
            </span>
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Buscar pelo nome ou categoria..."
              autoFocus
            />
            <div
              className={cn(
                'mt-1 max-h-56 overflow-y-auto rounded-md border',
                'border-amber-700/30 bg-amber-50/80 dark:border-amber-500/20 dark:bg-zinc-900/60',
              )}
            >
              {filtered.length === 0 ? (
                <p className={cn('px-3 py-4 text-center text-xs', dimText)}>
                  Nenhum item.
                </p>
              ) : (
                filtered.map((opt) => {
                  const active = catalogId === opt.id
                  return (
                    <button
                      key={opt.id}
                      type="button"
                      onClick={() => {
                        setCatalogId(opt.id)
                        if (error) setError(null)
                      }}
                      className={cn(
                        'flex w-full items-center justify-between gap-2 px-3 py-1.5 text-left text-sm transition-colors',
                        active
                          ? 'bg-amber-200/70 text-amber-900 dark:bg-amber-500/20 dark:text-amber-200'
                          : 'hover:bg-amber-100/60 dark:hover:bg-zinc-800/60',
                      )}
                    >
                      <span className="truncate">{opt.name}</span>
                      <span className={cn('shrink-0 text-[10px]', dimText)}>
                        {opt.category}
                      </span>
                    </button>
                  )
                })
              )}
            </div>
          </div>
          {selected && (
            <div
              className={cn(
                'rounded-md border px-3 py-2 text-[11px]',
                'border-amber-700/30 bg-amber-100/70 dark:border-amber-500/30 dark:bg-zinc-900/60',
              )}
            >
              <p className={cn('font-semibold', accentStrong)}>{selected.name}</p>
              <p className={dimText}>
                {selected.category} • esp {formatLoad(selected.slots)} • T${' '}
                {selected.price}
              </p>
              {selected.weapon && (
                <p className={dimText}>
                  dano {selected.weapon.damage} • crit {selected.weapon.critRange}
                  /×{selected.weapon.critMult}
                </p>
              )}
              {selected.armor && (
                <p className={dimText}>
                  Def +{selected.armor.defense} • penalidade{' '}
                  {selected.armor.penalty} •{' '}
                  {selected.armor.heavy ? 'pesada' : 'leve'}
                </p>
              )}
              {selected.shield && (
                <p className={dimText}>
                  Def +{selected.shield.defense} • penalidade{' '}
                  {selected.shield.penalty}
                </p>
              )}
            </div>
          )}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <span
                className={cn('text-[10px] uppercase tracking-widest', dimText)}
              >
                quantidade
              </span>
              <NumberInput
                value={quantity}
                onChange={(v) => setQuantity(String(v))}
                min={1}
                max={9999}
                step={1}
                aria-label="Quantidade"
              />
            </div>
            <div className="space-y-1">
              <span
                className={cn('text-[10px] uppercase tracking-widest', dimText)}
              >
                equipar
              </span>
              <select
                value={equipped}
                onChange={(e) =>
                  setEquipped(e.target.value as typeof equipped)
                }
                className={cn(selectClass, 'h-9 w-full px-2 text-sm')}
                aria-label="Equipar"
              >
                {EQUIP_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
          {error && (
            <p className="text-xs text-destructive" role="alert">
              {error}
            </p>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="button" onClick={apply} disabled={!selected}>
              Adicionar
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function describeModifierTarget(t: Modifier['target']): string {
  switch (t.k) {
    case 'expertise':
      return `Perícia ${t.name}`
    case 'expertiseAll':
      return 'Todas perícias'
    case 'expertiseRemovePenalty':
      return `Remove penalidade em ${t.name}`
    case 'expertiseByAttribute':
      return `Perícias de ${t.attribute}`
    case 'attribute':
      return `Atributo ${t.name}`
    case 'defense':
      return 'Defesa'
    case 'defenseDexCap':
      return 'Limite de Des na Defesa'
    case 'resistance':
      return 'Resistências'
    case 'fearResistance':
      return 'Resistência a medo'
    case 'attack':
      return `Ataque (${t.scope})`
    case 'damage':
      return `Dano (${t.scope})`
    case 'critRange':
      return 'Margem de ameaça'
    case 'critMult':
      return 'Multiplicador crítico'
    case 'pmLimit':
      return 'Limite de PM por magia'
    case 'pmCost':
      return 'Custo em PM'
    case 'spellDC':
      return 'CD de magias'
    case 'inventorySlots':
      return 'Espaços de carga'
    case 'displacement':
      return 'Deslocamento'
    case 'armorPenalty':
      return 'Penalidade de armadura'
    case 'armorPenaltyExpertises':
      return 'Penalidade em perícias'
    case 'tempHp':
      return 'PV temporários'
    case 'tempMp':
      return 'PM temporários'
    case 'maneuver':
      return `Manobra ${t.name}`
    case 'flag':
      return `Efeito: ${t.name}`
  }
}

function describeCondition(m: Modifier): string | null {
  if (!m.condition) return null
  switch (m.condition.c) {
    case 'always':
      return null
    case 'wielded':
      return 'enquanto empunhado'
    case 'vested':
      return 'enquanto vestido'
    case 'terrain':
      return `terreno: ${m.condition.type}`
    case 'against':
      return `contra: ${m.condition.trait}`
    case 'context':
      return m.condition.note
    case 'flagOn':
      return m.condition.label
  }
}

function CatalogInfoBody({ catalog }: { catalog: CatalogItem }) {
  return (
    <div className="space-y-3 text-xs">
      <div>
        <p className={cn('font-semibold', accentStrong)}>{catalog.name}</p>
        <p className={dimText}>
          {catalog.category} • esp {formatLoad(catalog.slots)} • T${' '}
          {catalog.price} •{' '}
          {catalog.equip === 'either' ? 'qualquer equipar' : catalog.equip}
          {catalog.hands ? ` • ${catalog.hands} mão(s)` : ''}
        </p>
      </div>
      {catalog.weapon && (
        <div className="space-y-0.5">
          <p className={cn('text-[10px] uppercase tracking-widest', dimText)}>
            arma
          </p>
          <p>
            dano <span className="font-mono">{catalog.weapon.damage}</span> •
            crítico{' '}
            <span className="font-mono">
              {catalog.weapon.critRange}/×{catalog.weapon.critMult}
            </span>{' '}
            • {catalog.weapon.type} • {catalog.weapon.purpose}
            {catalog.weapon.range ? ` (${catalog.weapon.range})` : ''}
          </p>
          {catalog.weapon.traits.length > 0 && (
            <p className={dimText}>
              propriedades: {catalog.weapon.traits.join(', ')}
            </p>
          )}
        </div>
      )}
      {catalog.armor && (
        <div className="space-y-0.5">
          <p className={cn('text-[10px] uppercase tracking-widest', dimText)}>
            armadura
          </p>
          <p>
            Defesa +{catalog.armor.defense} • penalidade{' '}
            {catalog.armor.penalty} •{' '}
            {catalog.armor.heavy ? 'pesada' : 'leve'}
          </p>
        </div>
      )}
      {catalog.shield && (
        <div className="space-y-0.5">
          <p className={cn('text-[10px] uppercase tracking-widest', dimText)}>
            escudo
          </p>
          <p>
            Defesa +{catalog.shield.defense} • penalidade{' '}
            {catalog.shield.penalty}
          </p>
        </div>
      )}
      <div className="space-y-1">
        <p className={cn('text-[10px] uppercase tracking-widest', dimText)}>
          modificadores
        </p>
        {catalog.modifiers.length === 0 ? (
          <p className={dimText}>Nenhum.</p>
        ) : (
          <ul className="space-y-0.5">
            {catalog.modifiers.map((m, i) => {
              const cond = describeCondition(m)
              const sign = m.amount >= 0 ? '+' : ''
              return (
                <li key={i} className="flex flex-wrap gap-x-1">
                  <span className="font-mono">
                    {sign}
                    {m.amount}
                  </span>
                  <span>{describeModifierTarget(m.target)}</span>
                  <span className={cn('text-[10px]', dimText)}>
                    [{m.bonusType}]
                  </span>
                  {cond && (
                    <span className={cn('text-[10px]', dimText)}>— {cond}</span>
                  )}
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}

function ItemInfoDialog({ item }: { item: CharacterItem }) {
  const catalog = item.catalogId ? getCatalogItem(item.catalogId) : undefined
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            'size-7',
            subtleText,
            'hover:bg-amber-200/60 hover:text-amber-900 dark:hover:bg-zinc-800/60 dark:hover:text-amber-200',
          )}
          aria-label={`Informações de ${item.name}`}
        >
          <Info className="size-3.5" />
        </Button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-md sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn('font-serif', accentStrong)}>
            {item.name}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div
            className={cn(
              'rounded-md border px-3 py-2 text-xs',
              'border-amber-700/30 bg-amber-100/70 dark:border-amber-500/30 dark:bg-zinc-900/60',
            )}
          >
            <p>
              quantidade <span className="font-mono">{item.quantity}</span> •
              espaços <span className="font-mono">{formatLoad(item.slots)}</span>{' '}
              • total{' '}
              <span className={cn('font-mono font-semibold', accentStrong)}>
                {formatLoad(item.quantity * item.slots)}
              </span>
            </p>
            <p className={dimText}>
              equipado:{' '}
              {item.equipped
                ? item.equipped === 'wielded'
                  ? '1 mão'
                  : item.equipped === 'wielded2'
                    ? '2 mãos'
                    : 'vestido'
                : '—'}
            </p>
          </div>
          {catalog ? (
            <CatalogInfoBody catalog={catalog} />
          ) : (
            <p className={cn('text-xs', dimText)}>
              Item customizado, sem dados de catálogo.
            </p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

function ItemFormDialog({
  title,
  submitLabel,
  trigger,
  initial,
  onSubmit,
}: {
  title: string
  submitLabel: string
  trigger: React.ReactNode
  initial?: Partial<CreateItemInput>
  onSubmit: (input: CreateItemInput, onError: (e: Error) => void) => void
}) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState(initial?.name ?? '')
  const [quantity, setQuantity] = useState<string>(String(initial?.quantity ?? 1))
  const [slots, setSlots] = useState<string>(String(initial?.slots ?? 1))
  const [error, setError] = useState<string | null>(null)

  const reset = () => {
    setName(initial?.name ?? '')
    setQuantity(String(initial?.quantity ?? 1))
    setSlots(String(initial?.slots ?? 1))
    setError(null)
  }

  const apply = () => {
    const trimmed = name.trim()
    const qty = Number(quantity)
    const sl = Number(slots)
    if (!trimmed) {
      setError('Informe um nome.')
      return
    }
    if (!Number.isInteger(qty) || qty < 1) {
      setError('Quantidade deve ser inteiro ≥ 1.')
      return
    }
    if (!Number.isFinite(sl) || sl < 0.5 || !Number.isInteger(sl * 2)) {
      setError('Espaços deve ser múltiplo de 0,5 (mínimo 0,5).')
      return
    }
    onSubmit({ name: trimmed, quantity: qty, slots: sl }, (e) =>
      setError(e.message),
    )
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
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn('font-serif', accentStrong)}>
            {title}
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
              placeholder="Ex: Espada longa"
              autoFocus
              maxLength={80}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <span
                className={cn('text-[10px] uppercase tracking-widest', dimText)}
              >
                quantidade
              </span>
              <NumberInput
                value={quantity}
                onChange={(v) => setQuantity(String(v))}
                min={1}
                max={9999}
                step={1}
                aria-label="Quantidade"
              />
            </div>
            <div className="space-y-1">
              <span
                className={cn('text-[10px] uppercase tracking-widest', dimText)}
              >
                espaços
              </span>
              <NumberInput
                value={slots}
                onChange={(v) => setSlots(String(v))}
                min={0.5}
                max={9999}
                step={0.5}
                aria-label="Espaços"
              />
            </div>
          </div>
          <p className={cn('text-[11px]', dimText)}>
            Espaços é múltiplo de 0,5 (ex.: 0,5 / 1 / 1,5).
          </p>
          {error && (
            <p className="text-xs text-destructive" role="alert">
              {error}
            </p>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="button" onClick={apply}>
              {submitLabel}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function EquipmentPanel({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey

  const unequip = useMutation<
    CharacterItem,
    Error,
    number,
    { previous: Character | undefined }
  >({
    mutationFn: (itemId) =>
      api.characters.updateItem(character.id, itemId, { equipped: null }),
    onMutate: async (itemId) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              items: prev.items.map((it) =>
                it.id === itemId ? { ...it, equipped: null } : it,
              ),
            }
          : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
  })

  const vested = character.items.filter((i) => i.equipped === 'vested')
  const wielded = character.items.filter((i) => i.equipped === 'wielded')
  const twoHand = character.items.find((i) => i.equipped === 'wielded2')

  return (
    <section
      className={cn(
        'flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl',
        surface,
        panelBg,
      )}
    >
      <div className="flex shrink-0 items-center justify-between gap-2 border-b border-amber-700/30 px-3 py-2 dark:border-amber-500/20 sm:px-4">
        <h2
          className={cn(
            'font-serif text-lg font-bold tracking-wide',
            accentStrong,
          )}
        >
          Equipado
        </h2>
        <p className={cn('text-[10px] sm:text-xs', dimText)}>
          {vested.length}/4 vestidos •{' '}
          {twoHand ? '2/2' : `${wielded.length}/2`} mãos
        </p>
      </div>

      <div className="flex min-h-0 flex-1 items-center justify-center overflow-y-auto p-4">
        <div className="grid w-full max-w-xs gap-3">
          <div className="flex justify-center">
            <EquipmentSlot
              label="Cabeça"
              item={vested[0]}
              onUnequip={(id) => unequip.mutate(id)}
              size="md"
            />
          </div>

          <div className="grid grid-cols-3 items-stretch gap-3">
            {twoHand ? (
              <EquipmentSlot
                label="Duas mãos"
                item={twoHand}
                onUnequip={(id) => unequip.mutate(id)}
                size="md"
                className="col-span-3"
              />
            ) : (
              <>
                <EquipmentSlot
                  label="Mão principal"
                  item={wielded[0]}
                  onUnequip={(id) => unequip.mutate(id)}
                  size="md"
                />
                <EquipmentSlot
                  label="Tronco"
                  item={vested[1]}
                  onUnequip={(id) => unequip.mutate(id)}
                  size="md"
                />
                <EquipmentSlot
                  label="Mão secundária"
                  item={wielded[1]}
                  onUnequip={(id) => unequip.mutate(id)}
                  size="md"
                />
              </>
            )}
          </div>

          {twoHand && (
            <div className="flex justify-center">
              <EquipmentSlot
                label="Tronco"
                item={vested[1]}
                onUnequip={(id) => unequip.mutate(id)}
                size="md"
              />
            </div>
          )}

          <div className="grid grid-cols-2 gap-3">
            <EquipmentSlot
              label="Pés"
              item={vested[2]}
              onUnequip={(id) => unequip.mutate(id)}
              size="md"
            />
            <EquipmentSlot
              label="Acessório"
              item={vested[3]}
              onUnequip={(id) => unequip.mutate(id)}
              size="md"
            />
          </div>
        </div>
      </div>
    </section>
  )
}

function EquipmentSlot({
  label,
  item,
  onUnequip,
  size,
  className,
}: {
  label: string
  item: CharacterItem | undefined
  onUnequip: (itemId: number) => void
  size: 'sm' | 'md'
  className?: string
}) {
  const filled = !!item
  return (
    <div
      className={cn(
        'relative flex flex-col items-center justify-center rounded-lg border-2 px-2 py-2 text-center transition-colors',
        size === 'md' ? 'min-h-[64px]' : 'min-h-[44px]',
        filled
          ? 'border-amber-700/60 bg-amber-100/80 dark:border-amber-500/60 dark:bg-zinc-900/70'
          : 'border-dashed border-amber-700/30 bg-amber-50/40 dark:border-amber-500/20 dark:bg-zinc-900/30',
        className,
      )}
    >
      <span
        className={cn(
          'text-[9px] uppercase tracking-widest',
          dimText,
        )}
      >
        {label}
      </span>
      {filled ? (
        <span
          className={cn(
            'mt-0.5 line-clamp-2 text-xs font-semibold',
            accentStrong,
          )}
          title={item.name}
        >
          {item.name}
        </span>
      ) : (
        <span className={cn('mt-0.5 text-xs', dimText)}>—</span>
      )}
      {filled && (
        <button
          type="button"
          className={cn(
            'absolute right-1 top-1 inline-flex size-5 items-center justify-center rounded-full text-zinc-500 hover:bg-red-100 hover:text-red-700 dark:text-zinc-400 dark:hover:bg-red-950/40 dark:hover:text-red-400',
          )}
          onClick={() => onUnequip(item.id)}
          aria-label={`Desequipar ${item.name}`}
        >
          <X className="size-3" />
        </button>
      )}
    </div>
  )
}

function AttributeBox({ label, value }: { label: string; value: number }) {
  const sign = value >= 0 ? '+' : ''
  return (
    <div
      className={cn(
        'relative rounded-lg border-2 p-2 text-center shadow-inner',
        'border-amber-700/30 bg-gradient-to-b from-amber-100 to-amber-50 dark:border-amber-500/30 dark:from-zinc-900 dark:to-zinc-950',
      )}
    >
      <p className="text-[9px] font-bold uppercase tracking-widest text-amber-800/80 dark:text-amber-400/70">
        {label}
      </p>
      <p
        className={cn(
          'mt-0.5 font-serif text-2xl font-bold leading-none',
          accentTitle,
        )}
      >
        {sign}
        {value}
      </p>
    </div>
  )
}
