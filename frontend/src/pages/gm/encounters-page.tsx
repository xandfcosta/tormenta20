import { Link } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'
import { SendEncounterToSessionButton } from '@/features/gm-tools/send-encounter-to-session'
import { BESTIARY, encounterXp } from '@tormenta20/t20-data'
import {
  type EnrichedGroup,
  type EncounterEntry as Entry,
  enrichEncounter as enrich,
  encounterDifficulty as difficultyLabel,
} from '@/features/gm-tools/encounter'

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

  const groups = useMemo(() => enrich(entries), [entries])
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
      <div className="flex items-center gap-3">
        <Link to="/gm">
          <Button variant="outline" size="sm">
            ←
          </Button>
        </Link>
        <SectionHeading variant="kallyadranoch" as="h1">
          Construtor de encontros
        </SectionHeading>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="font-display text-base tracking-wide">
            Grupo
          </CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4 sm:grid-cols-2">
          <div>
            <label className="text-xs font-medium" htmlFor="party-level">
              Nível do grupo
            </label>
            <NumberInput
              id="party-level"
              min={1}
              max={20}
              value={partyLevel}
              onChange={setPartyLevel}
            />
          </div>
          <div>
            <label className="text-xs font-medium" htmlFor="party-size">
              Personagens
            </label>
            <NumberInput
              id="party-size"
              min={1}
              max={8}
              value={partySize}
              onChange={setPartySize}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="font-display text-base tracking-wide">
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

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="font-display text-base tracking-wide">
            Resultado
          </CardTitle>
          <SendEncounterToSessionButton
            groups={groups.map((g) => ({
              monster: g.monster,
              quantity: g.quantity,
            }))}
          />
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
          <Stat label="ND do encontro" value={formatNd(encounterNd)} />
          <Stat label="XP / personagem" value={totalXp} />
          <Stat label="Gap vs. grupo" value={signed(gap)} />
          <div>
            <p className="text-xs text-muted-foreground">Dificuldade</p>
            <Badge className="mt-1" variant={difficulty.variant}>
              {difficulty.label}
            </Badge>
          </div>
        </CardContent>
      </Card>
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
    <div className="flex flex-wrap items-center gap-2 rounded-md border p-2 text-sm">
      <div className="min-w-[140px] flex-1">
        <p className="font-medium">
          {group.monster.name}{' '}
          <Badge variant="secondary">ND {formatNd(group.monster.nd)}</Badge>
        </p>
        <p className="text-xs text-muted-foreground">
          Grupo: ND {formatNd(group.groupNd)}
        </p>
      </div>
      <div className="w-20">
        <NumberInput
          min={1}
          max={30}
          value={group.quantity}
          onChange={onQuantity}
        />
      </div>
      <Button variant="ghost" size="sm" onClick={onRemove}>
        Remover
      </Button>
    </div>
  )
}

// ─── Monster picker Dialog ──────────────────────────────────────

function MonsterPickerDialog({
  onPick,
}: {
  onPick: (id: string) => void
}) {
  const [query, setQuery] = useState('')
  const filtered = useMemo(() => {
    const q = normalize(query)
    return BESTIARY.filter((m) => !q || normalize(m.name).includes(q))
      .slice()
      .sort((a, b) => a.nd - b.nd || a.name.localeCompare(b.name))
  }, [query])

  return (
    <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>Escolher monstro</DialogTitle>
      </DialogHeader>
      <div className="space-y-2">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Buscar por nome…"
          autoFocus
        />
        <div className="space-y-1">
          {filtered.map((m) => (
            <button
              key={m.id}
              type="button"
              onClick={() => onPick(m.id)}
              className="flex w-full items-center justify-between rounded-md border p-2 text-sm transition hover:border-primary/40"
            >
              <span>{m.name}</span>
              <Badge variant="secondary">ND {formatNd(m.nd)}</Badge>
            </button>
          ))}
        </div>
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

function normalize(s: string): string {
  return s
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .trim()
}
