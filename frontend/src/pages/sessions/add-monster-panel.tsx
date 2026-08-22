import type { Monster } from '@/shared/api/catalog-types'
import { MonsterPickerPanel } from '@/features/gm-tools/monster-picker-panel'
import { rollD20 } from '@/shared/lib/dice'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { toast } from '@/shared/ui/sonner'
import { MatchPeek } from './match-rail'

/**
 * Drops a creature from the bestiary straight into the live initiative, so the
 * GM never leaves the match to reach the Mesa. Initiative is a raw d20 (the
 * stat blocks carry no DEX mod) and PV is seeded from the block.
 *
 * The panel STAYS OPEN across adds: an ambush is one trip, not six. On a wide
 * screen it is non-modal, so the tracker keeps taking clicks behind it — and
 * the round/turn peek rides in the panel header so the GM never loses the fio.
 *
 * Quem abre é o TRILHO das consultas: um overlay por vez (ALE-198).
 */
export function AddMonsterPanel(props: {
  rt: SessionRealtime
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const add = (monster: Monster) => {
    props.rt.addEntry({
      label: monster.name,
      initiative: rollD20(),
      type: 'npc',
      monsterId: monster.id,
      hpCurrent: monster.hp,
      hpMax: monster.hp,
    })
    toast(`${monster.name} entrou na iniciativa`, {
      description: `PV ${monster.hp} · iniciativa rolada (d20).`,
    })
  }

  return (
      <MonsterPickerPanel
        open={props.open}
        onOpenChange={props.onOpenChange}
        title="Adicionar do bestiário"
        description="Entra na iniciativa com PV cheio e d20 rolado."
        header={<MatchPeek rt={props.rt} />}
        onPick={add}
      />
  )
}
