import type { Monster } from '@tormenta20/t20-data'
import { Skull } from 'lucide-solid'
import { createSignal } from 'solid-js'
import { MonsterPickerPanel } from '@/features/gm-tools/monster-picker-panel'
import { rollD20 } from '@/shared/lib/dice'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
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
 */
export function AddMonsterPanel(props: { rt: SessionRealtime }) {
  const [open, setOpen] = createSignal(false)

  const add = (monster: Monster) => {
    props.rt.addEntry({
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
    <>
      <Button
        type="button"
        size="sm"
        variant="secondary"
        class="w-full gap-1.5"
        disabled={!props.rt.isConnected()}
        onClick={() => setOpen(true)}
      >
        <Skull aria-hidden="true" class="size-4" /> Adicionar do bestiário
      </Button>

      <MonsterPickerPanel
        open={open()}
        onOpenChange={setOpen}
        title="Adicionar do bestiário"
        description="Entra na iniciativa com PV cheio e d20 rolado."
        header={<MatchPeek rt={props.rt} />}
        onPick={add}
      />
    </>
  )
}
