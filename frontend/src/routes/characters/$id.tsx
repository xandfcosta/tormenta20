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
  ChevronDown,
  ChevronUp,
  Crosshair,
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
import {
  accentBadge,
  accentStrong,
  accentTitle,
  dimText,
  hoverRow,
  panelBg,
  sheetBg,
  subtleText,
  surface,
} from '@/lib/sheet-theme'
import { api } from '@/lib/api'
import {
  characterQueryOptions,
  meQueryOptions,
} from '@/lib/queries'
import { invalidateCharacterDependents } from '@/lib/character-cache'
import { AbilitiesPanel } from './character-sheet/abilities-panel'
import { CampaignsPanel } from './character-sheet/campaigns-panel'
import { EffectsCountBadge } from './character-sheet/effects-count-badge'
import { EffectsPanel } from './character-sheet/effects-panel'
import { EquipmentPanel } from './character-sheet/equipment-panel'
import { InventoryPanel } from './character-sheet/inventory-panel'
import { normalize } from './character-sheet/normalize'
import { ProficienciesPanel } from './character-sheet/proficiencies-panel'
import { signed } from './character-sheet/signed'
import type {
  AttributeKey,
  Character,
  CharacterExpertise,
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
  expertiseTotalWithItems,
  pmCostMod,
  pmLimitTotal,
  spellDCBonus,
  useCharacterEffects,
} from '@/lib/derived'
import type { ItemEffects } from '@tormenta20/t20-data'

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
        <TabsTrigger value="campaigns" className="gap-1.5">
          Campanhas
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
      <TabsContent
        value="campaigns"
        className="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <CampaignsPanel characterId={character.id} />
      </TabsContent>
    </Tabs>
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
        <Link
          to="/characters/$id/sheet"
          params={{ id: String(character.id) }}
        >
          <Button variant="outline" size="sm">
            Ficha computada
          </Button>
        </Link>
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
    onSuccess: (server) => {
      qc.setQueryData<Character>(queryKey, server)
      invalidateCharacterDependents(qc, character.id)
    },
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
        invalidateCharacterDependents(qc, character.id)
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
      invalidateCharacterDependents(qc, character.id)
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
