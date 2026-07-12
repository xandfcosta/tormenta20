import { useMemo, useState } from 'react'
import { toast } from 'sonner'
import { Skull } from 'lucide-react'
import { BESTIARY, type Monster } from '@tormenta20/t20-data'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/shared/ui/sheet'
import { rollD20 } from '@/shared/lib/dice'
import type { useSessionSocket } from '@/shared/realtime/realtime'
import {
  formatNd,
  MONSTER_TIPO_LABEL,
  normalizeMonsterName,
} from '@/features/gm-tools/monster-format'

/**
 * In-session bestiary launcher for the GM rail. Opens a side sheet to search
 * the bestiary and drop a monster straight into the live initiative — so the
 * GM never leaves the match to reach /gm/bestiary. Initiative is rolled
 * client-side (raw d20; monsters carry no DEX mod here) and HP is seeded from
 * the monster stat block. The sheet stays open across adds so an ambush of
 * several monsters is one trip.
 */
export function AddMonsterDrawer({
  rt,
}: {
  rt: ReturnType<typeof useSessionSocket>
}) {
  const [name, setName] = useState('')
  const filtered = useMemo(() => {
    const q = normalizeMonsterName(name)
    return BESTIARY.filter((m) => !q || normalizeMonsterName(m.name).includes(q))
      .slice()
      .sort((a, b) => a.nd - b.nd || a.name.localeCompare(b.name))
  }, [name])

  const addMonster = (monster: Monster) => {
    rt.addEntry({
      label: monster.name,
      initiative: rollD20(),
      type: 'npc',
      hpCurrent: monster.hp,
      hpMax: monster.hp,
    })
    toast(`${monster.name} entrou na iniciativa`, {
      description: `PV ${monster.hp} · iniciativa rolada (d20).`,
    })
  }

  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button size="sm" variant="secondary" className="w-full gap-1.5">
          <Skull className="size-4" /> Adicionar do bestiário
        </Button>
      </SheetTrigger>
      <SheetContent className="flex w-full flex-col gap-0 sm:max-w-md">
        <SheetHeader>
          <SheetTitle className="font-display tracking-wide">
            Bestiário
          </SheetTitle>
          <SheetDescription>
            Buscar e enviar um monstro para a iniciativa da sessão.
          </SheetDescription>
        </SheetHeader>
        <div className="border-b p-4">
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Buscar monstro…"
            disabled={!rt.isConnected}
          />
        </div>
        <div className="min-h-0 flex-1 space-y-2 overflow-y-auto p-4">
          {filtered.length === 0 ? (
            <p className="p-4 text-center text-sm text-muted-foreground">
              Nenhum monstro casa com a busca.
            </p>
          ) : (
            filtered.map((m) => (
              <MonsterRow
                key={m.id}
                monster={m}
                disabled={!rt.isConnected}
                onAdd={() => addMonster(m)}
              />
            ))
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

function MonsterRow({
  monster,
  disabled,
  onAdd,
}: {
  monster: Monster
  disabled: boolean
  onAdd: () => void
}) {
  return (
    <div className="flex items-center justify-between gap-2 rounded-md border p-2.5">
      <div className="min-w-0 flex-1 space-y-1">
        <p className="truncate font-medium">
          {monster.name}{' '}
          <Badge variant="secondary">ND {formatNd(monster.nd)}</Badge>
        </p>
        <p className="text-xs text-muted-foreground">
          {MONSTER_TIPO_LABEL[monster.tipo]} · PV {monster.hp} · Defesa{' '}
          {monster.defesa}
        </p>
      </div>
      <Button size="sm" variant="outline" onClick={onAdd} disabled={disabled}>
        Adicionar
      </Button>
    </div>
  )
}
