import { useMemo, useState } from 'react'
import { Trash2 } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/shared/ui/dialog'
import { NumberInput } from '@/shared/ui/number-input'
import { PageChrome } from '@/shared/ui/page-chrome'
import { clampToRange } from '@/shared/lib/bounded-number'
import { VirtualList } from '@/shared/ui/virtual-list'
import { GmPageHeader } from '@/features/gm-tools/gm-page-header'
import {
  MonsterFilters,
  useMonsterFilter,
} from '@/features/gm-tools/monster-filter'
import { SendEncounterToSessionButton } from '@/features/gm-tools/send-encounter-to-session'
import { encounterXp } from '@tormenta20/t20-data'
import {
  type EnrichedGroup,
  type EncounterEntry as Entry,
  enrichEncounter as enrich,
  encounterDifficulty as difficultyLabel,
} from '@/features/gm-tools/encounter'
import { useBestiary } from '@/entities/catalog/use-bestiary'

/**
 * Encounter builder — party level + size, monster composition, live
 * ND + XP + difficulty. Book Cap 7 p282 rules:
 *   - ND < 1: group ND = monster.nd × quantity
 *   - ND >= 1: group ND = monster.nd + 2 × log2(quantity)
 * Encounter ND (mixed): sum of group NDs. Book is silent on mixed
 * composition — sum is the permissive default; GM can eyeball.
 */

export function EncounterBuilderPage() {
  const [partyLevel, setPartyLevel] = useState(1)
  const [partySize, setPartySize] = useState(4)
  const [entries, setEntries] = useState<Entry[]>([])
  const [pickerOpen, setPickerOpen] = useState(false)

  const bestiary = useBestiary().data ?? []
  const groups = useMemo(() => enrich(entries, bestiary), [entries, bestiary])
  const encounterNd = useMemo(
    () => groups.reduce((sum, g) => sum + g.groupNd, 0),
    [groups],
  )
  const totalXp = useMemo(
    () =>
      encounterXp({
        nd: encounterNd,
        partyLevel,
        partySize,
        outcome: 'win',
      }),
    [encounterNd, partyLevel, partySize],
  )
  const gap = encounterNd - partyLevel
  const difficulty = difficultyLabel(gap)

  const addEntry = (monsterId: string) => {
    setEntries((prev) => {
      const idx = prev.findIndex((e) => e.monsterId === monsterId)
      if (idx >= 0) {
        const next = [...prev]
        next[idx] = { ...next[idx]!, quantity: next[idx]!.quantity + 1 }
        return next
      }
      return [...prev, { monsterId, quantity: 1 }]
    })
    setPickerOpen(false)
  }

  const setQuantity = (monsterId: string, q: number) => {
    setEntries((prev) =>
      prev.map((e) =>
        e.monsterId === monsterId ? { ...e, quantity: Math.max(1, q) } : e,
      ),
    )
  }

  const removeEntry = (monsterId: string) => {
    setEntries((prev) => prev.filter((e) => e.monsterId !== monsterId))
  }

  return (
    <PageChrome className="space-y-4">
      <GmPageHeader title="Construtor de encontros" />

      <div className="grid gap-4 lg:grid-cols-[1fr_20rem] lg:items-start">
        <div className="space-y-4">
      <Card className="py-3 lg:py-4">
        <CardContent className="flex flex-wrap items-center justify-between gap-3">
          <CardTitle className="text-base">Grupo</CardTitle>
          <div className="ml-auto flex flex-wrap items-end gap-4">
            <div>
              <label className="text-xs font-medium" htmlFor="party-level">
                Nível do grupo
              </label>
              <div className="w-24">
                <NumberInput
                  id="party-level"
                  min={1}
                  max={20}
                  value={partyLevel}
                  onChange={(v) => setPartyLevel(clampToRange(v, { min: 1, max: 20 }))}
                />
              </div>
            </div>
            <div>
              <label className="text-xs font-medium" htmlFor="party-size">
                Personagens
              </label>
              <div className="w-24">
                <NumberInput
                  id="party-size"
                  min={1}
                  max={8}
                  value={partySize}
                  onChange={(v) => setPartySize(clampToRange(v, { min: 1, max: 8 }))}
                />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="gap-3 py-4 lg:gap-6 lg:py-6">
        <CardHeader className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle className="text-base">
            Composição
          </CardTitle>
          <div className="flex gap-2">
            <Dialog open={pickerOpen} onOpenChange={setPickerOpen}>
              <DialogTrigger asChild>
                <Button>+ Adicionar monstro</Button>
              </DialogTrigger>
              <MonsterPickerDialog onPick={addEntry} />
            </Dialog>
            <Button
              variant="outline"
              onClick={() => setEntries([])}
              disabled={entries.length === 0}
            >
              Reset
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-2">
          {entries.length === 0 && (
            <p className="text-sm text-muted-foreground">
              Sem criaturas ainda.
            </p>
          )}
          {groups.map((g) => (
            <EntryRow
              key={g.monster.id}
              group={g}
              onQuantity={(q) => setQuantity(g.monster.id, q)}
              onRemove={() => removeEntry(g.monster.id)}
            />
          ))}
        </CardContent>
      </Card>
        </div>

        <Card className="sticky top-0 z-10 order-first gap-3 self-start py-4 lg:order-none lg:gap-6 lg:py-6 lg:top-4">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">Resultado</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid grid-cols-4 gap-2 text-sm lg:grid-cols-2 lg:gap-3">
              <Stat label="ND" value={formatNd(encounterNd)} />
              <Stat label="XP" value={totalXp} />
              <Stat label="Gap" value={signed(gap)} />
              <div>
                <p className="text-xs text-muted-foreground">Dif.</p>
                <Badge className="mt-1" variant={difficulty.variant}>
                  {difficulty.label}
                </Badge>
              </div>
            </div>
            <SendEncounterToSessionButton
              groups={groups.map((g) => ({
                monster: g.monster,
                quantity: g.quantity,
              }))}
            />
          </CardContent>
        </Card>
      </div>
    </PageChrome>
  )
}

// ─── Row ────────────────────────────────────────────────────────

function EntryRow({
  group,
  onQuantity,
  onRemove,
}: {
  group: EnrichedGroup
  onQuantity: (q: number) => void
  onRemove: () => void
}) {
  return (
    <div className="flex items-center gap-2 rounded-md border p-2 text-sm">
      <div className="min-w-0 flex-1">
        <p className="truncate font-medium">
          {group.monster.name}{' '}
          <Badge variant="secondary">ND {formatNd(group.monster.nd)}</Badge>
        </p>
        <p className="text-xs text-muted-foreground">
          Grupo: ND {formatNd(group.groupNd)}
        </p>
      </div>
      <div className="w-16 shrink-0">
        <NumberInput
          min={1}
          max={30}
          value={group.quantity}
          onChange={(v) => onQuantity(clampToRange(v, { min: 1, max: 30 }))}
        />
      </div>
      <Button
        variant="ghost"
        size="icon"
        className="shrink-0"
        onClick={onRemove}
        aria-label={`Remover ${group.monster.name}`}
      >
        <Trash2 className="size-4" />
      </Button>
    </div>
  )
}

// ─── Monster picker Dialog ──────────────────────────────────────

/** Picker uses the same bestiary filter (fuzzy name + tipo + ND) as the
 * bestiary page, over a virtualized result list. */
function MonsterPickerDialog({
  onPick,
}: {
  onPick: (id: string) => void
}) {
  const { filtered, controls } = useMonsterFilter()

  return (
    <DialogContent className="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>Escolher monstro</DialogTitle>
      </DialogHeader>
      <div className="space-y-3">
        <MonsterFilters {...controls} idPrefix="picker" />
        {filtered.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            Nenhum monstro casa com os filtros.
          </p>
        ) : (
          <VirtualList
            items={filtered}
            estimateSize={44}
            gap={4}
            className="max-h-[50vh]"
            getKey={(m) => m.id}
            renderItem={(m) => (
              <button
                type="button"
                onClick={() => onPick(m.id)}
                className="flex w-full items-center justify-between rounded-md border p-2 text-sm transition-colors hover:border-primary"
              >
                <span>{m.name}</span>
                <Badge variant="secondary">ND {formatNd(m.nd)}</Badge>
              </button>
            )}
          />
        )}
      </div>
    </DialogContent>
  )
}

// ─── Helpers ────────────────────────────────────────────────────

function Stat({
  label,
  value,
}: {
  label: string
  value: string | number
}) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-lg font-semibold">{value}</p>
    </div>
  )
}

function signed(n: number): string {
  return n >= 0 ? `+${n.toFixed(1).replace(/\.0$/, '')}` : `${n.toFixed(1).replace(/\.0$/, '')}`
}

function formatNd(nd: number): string {
  if (nd === 0) return '0'
  if (Math.abs(nd - 0.25) < 0.001) return '1/4'
  if (Math.abs(nd - 0.5) < 0.001) return '1/2'
  if (Number.isInteger(nd)) return String(nd)
  return nd.toFixed(1)
}
