import { useMemo, useState } from 'react'
import { toast } from 'sonner'
import { Swords } from 'lucide-react'
import { BESTIARY, encounterXp } from '@tormenta20/t20-data'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/shared/ui/sheet'
import { rollD20 } from '@/shared/lib/dice'
import type { useSessionSocket } from '@/shared/realtime/realtime'
import {
  type EncounterEntry,
  INITIATIVE_MAX_ENTRIES,
  encounterDifficulty,
  enrichEncounter,
} from '@/features/gm-tools/encounter'
import { formatNd, normalizeMonsterName } from '@/features/gm-tools/monster-format'

/**
 * In-session encounter builder for the GM rail. Compose a group of monsters
 * with live ND + difficulty (against a party level/size the GM sets), then
 * batch the whole encounter straight into the live initiative — the
 * multi-monster counterpart to AddMonsterDrawer. Initiative is rolled
 * client-side (raw d20) and HP seeded from each stat block; duplicates get a
 * " #N" suffix so they stay distinguishable. The batch is clamped to the
 * server's INITIATIVE_MAX_ENTRIES so the GM sees the cutoff, not a silent
 * WS error mid-loop.
 */
export function EncounterBuilderDrawer({
  rt,
}: {
  rt: ReturnType<typeof useSessionSocket>
}) {
  const [open, setOpen] = useState(false)
  const [partyLevel, setPartyLevel] = useState(1)
  const [partySize, setPartySize] = useState(4)
  const [entries, setEntries] = useState<EncounterEntry[]>([])

  const groups = useMemo(() => enrichEncounter(entries), [entries])
  const encounterNd = groups.reduce((sum, g) => sum + g.groupNd, 0)
  const totalXp = encounterXp({
    nd: encounterNd,
    partyLevel,
    partySize,
    outcome: 'win',
  })
  const difficulty = encounterDifficulty(encounterNd - partyLevel)
  const wanted = groups.reduce((sum, g) => sum + g.quantity, 0)

  const addMonster = (monsterId: string) => {
    setEntries((prev) => {
      const found = prev.find((e) => e.monsterId === monsterId)
      if (found)
        return prev.map((e) =>
          e.monsterId === monsterId ? { ...e, quantity: e.quantity + 1 } : e,
        )
      return [...prev, { monsterId, quantity: 1 }]
    })
  }
  const setQuantity = (monsterId: string, q: number) =>
    setEntries((prev) =>
      prev.map((e) =>
        e.monsterId === monsterId ? { ...e, quantity: Math.max(1, q) } : e,
      ),
    )
  const removeEntry = (monsterId: string) =>
    setEntries((prev) => prev.filter((e) => e.monsterId !== monsterId))

  const send = () => {
    const remaining = Math.max(
      0,
      INITIATIVE_MAX_ENTRIES - rt.state.initiative.length,
    )
    let count = 0
    for (const g of groups) {
      for (let i = 1; i <= g.quantity; i++) {
        if (count >= remaining) break
        const label = g.quantity > 1 ? `${g.monster.name} #${i}` : g.monster.name
        rt.addEntry({
          label,
          initiative: rollD20(),
          type: 'npc',
          hpCurrent: g.monster.hp,
          hpMax: g.monster.hp,
        })
        count++
      }
      if (count >= remaining) break
    }
    toast(`${count} combatente${count === 1 ? '' : 's'} na iniciativa`, {
      description:
        count < wanted
          ? `Tracker no limite (${INITIATIVE_MAX_ENTRIES}); enviados ${count} de ${wanted}.`
          : 'Iniciativa rolada (d20), PV do bestiário.',
    })
    setEntries([])
    setOpen(false)
  }

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button size="sm" variant="secondary" className="w-full gap-1.5">
          <Swords className="size-4" /> Montar encontro
        </Button>
      </SheetTrigger>
      <SheetContent className="flex w-full flex-col gap-0 sm:max-w-md">
        <SheetHeader>
          <SheetTitle className="font-display tracking-wide">
            Montar encontro
          </SheetTitle>
        </SheetHeader>
        <div className="grid grid-cols-2 gap-3 border-b p-4">
          <div>
            <label className="text-xs font-medium" htmlFor="enc-party-level">
              Nível do grupo
            </label>
            <NumberInput
              id="enc-party-level"
              min={1}
              max={20}
              value={partyLevel}
              onChange={setPartyLevel}
            />
          </div>
          <div>
            <label className="text-xs font-medium" htmlFor="enc-party-size">
              Personagens
            </label>
            <NumberInput
              id="enc-party-size"
              min={1}
              max={8}
              value={partySize}
              onChange={setPartySize}
            />
          </div>
        </div>
        <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-4">
          {groups.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Adicione monstros abaixo para montar o encontro.
            </p>
          ) : (
            groups.map((g) => (
              <div
                key={g.monster.id}
                className="flex items-center gap-2 rounded-md border p-2 text-sm"
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium">
                    {g.monster.name}{' '}
                    <Badge variant="secondary">
                      grupo ND {formatNd(g.groupNd)}
                    </Badge>
                  </p>
                  <p className="text-xs text-muted-foreground">
                    PV {g.monster.hp}
                  </p>
                </div>
                <div className="w-16">
                  <NumberInput
                    min={1}
                    max={30}
                    value={g.quantity}
                    onChange={(q) => setQuantity(g.monster.id, q)}
                  />
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => removeEntry(g.monster.id)}
                >
                  ×
                </Button>
              </div>
            ))
          )}
          <MonsterPicker onPick={addMonster} disabled={!rt.isConnected} />
        </div>
        <div className="space-y-3 border-t p-4">
          <div className="grid grid-cols-3 gap-2 text-center text-sm">
            <Stat label="ND" value={formatNd(encounterNd)} />
            <Stat label="XP/pers." value={totalXp} />
            <div>
              <p className="text-xs text-muted-foreground">Dificuldade</p>
              <Badge className="mt-0.5" variant={difficulty.variant}>
                {difficulty.label}
              </Badge>
            </div>
          </div>
          <Button
            className="w-full"
            onClick={send}
            disabled={!rt.isConnected || wanted === 0}
          >
            Enviar {wanted > 0 ? `${wanted} ` : ''}para a iniciativa
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  )
}

function MonsterPicker({
  onPick,
  disabled,
}: {
  onPick: (id: string) => void
  disabled: boolean
}) {
  const [query, setQuery] = useState('')
  const filtered = useMemo(() => {
    const q = normalizeMonsterName(query)
    return BESTIARY.filter((m) => !q || normalizeMonsterName(m.name).includes(q))
      .slice()
      .sort((a, b) => a.nd - b.nd || a.name.localeCompare(b.name))
  }, [query])

  return (
    <div className="space-y-2 rounded-md border border-dashed p-2">
      <Input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Buscar monstro para adicionar…"
        disabled={disabled}
      />
      <div className="max-h-48 space-y-1 overflow-y-auto">
        {filtered.map((m) => (
          <button
            key={m.id}
            type="button"
            disabled={disabled}
            onClick={() => onPick(m.id)}
            className="flex w-full items-center justify-between rounded-md border p-2 text-sm transition hover:border-[color:var(--primary)]/40 disabled:opacity-50"
          >
            <span className="truncate">{m.name}</span>
            <Badge variant="secondary">ND {formatNd(m.nd)}</Badge>
          </button>
        ))}
      </div>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-lg font-semibold">{value}</p>
    </div>
  )
}
