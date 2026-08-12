import { useQuery } from '@tanstack/solid-query'
import { Plus, Swords } from 'lucide-solid'
import { Show, createMemo, createSignal } from 'solid-js'
import { bestiaryCatalogQueryOptions } from '@/entities/catalog/queries'
import { enrichEncounter, encounterInitiativeLabels } from '@/features/gm-tools/encounter'
import { EncounterComposer } from '@/features/gm-tools/encounter-composer'
import { createEncounterDraft } from '@/features/gm-tools/encounter-draft'
import { MonsterPickerList } from '@/features/gm-tools/monster-picker-list'
import { rollD20 } from '@/shared/lib/dice'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { SidePanel } from '@/shared/ui/side-panel'
import { toast } from '@/shared/ui/sonner'
import { MatchPeek } from './match-rail'

/**
 * Composes an encounter mid-session and pushes it into the initiative in one
 * go. Each creature enters as its OWN entry — four goblins are "Goblin 1".."4"
 * — because the GM tracks each one's PV separately.
 *
 * The server caps the tracker at 50 entries, so the push counts what is already
 * there and SAYS what did not fit instead of letting the surplus die as a
 * silent socket error halfway through the loop.
 */
export function EncounterPanel(props: { rt: SessionRealtime }) {
  const bestiary = useQuery(() => bestiaryCatalogQueryOptions)
  const draft = createEncounterDraft()
  const [open, setOpen] = createSignal(false)
  const [picking, setPicking] = createSignal(false)

  const groups = createMemo(() => enrichEncounter(draft.entries(), bestiary.data ?? []))

  const send = () => {
    const { labels, dropped } = encounterInitiativeLabels(
      groups(),
      props.rt.state().initiative.length,
    )
    for (const label of labels) {
      props.rt.addEntry({ label, initiative: rollD20(), type: 'npc' })
    }
    draft.clear()
    toast(`${labels.length} criaturas entraram na iniciativa`, {
      description:
        dropped > 0
          ? `${dropped} ficaram de fora — o rastreador aceita 50 entradas.`
          : 'Iniciativa rolada para cada uma (d20).',
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
        <Swords aria-hidden="true" class="size-4" /> Montar encontro
      </Button>

      <SidePanel
        open={open()}
        onOpenChange={setOpen}
        title="Montar encontro"
        description="Combine criaturas e mande tudo para a iniciativa."
        header={<MatchPeek rt={props.rt} />}
      >
        <EncounterComposer
          groups={groups()}
          partyLevel={draft.partyLevel()}
          partySize={draft.partySize()}
          onPartyLevel={draft.setPartyLevel}
          onPartySize={draft.setPartySize}
          onQuantity={draft.setQuantity}
          onRemove={draft.remove}
          addControl={
            <Button
              type="button"
              variant="outline"
              size="sm"
              aria-expanded={picking()}
              onClick={() => setPicking((was) => !was)}
            >
              <Plus aria-hidden="true" class="mr-1 size-4" />
              {picking() ? 'Fechar bestiário' : 'Adicionar criatura'}
            </Button>
          }
          footer={
            <Button
              type="button"
              size="sm"
              class="w-full"
              disabled={groups().length === 0 || !props.rt.isConnected()}
              onClick={send}
            >
              Mandar para a iniciativa
            </Button>
          }
        />

        {/* Inline, NOT a second side panel: the composer IS the panel here, and
            a nested one would land exactly on top of the composition being
            added to. The list is bounded so the ledger above stays on screen. */}
        <Show when={picking()}>
          <div class="mt-3 border-t border-grimorio-iron pt-3">
            <MonsterPickerList
              onPick={(monster) => draft.add(monster.id)}
              idPrefix="session-encounter"
              listClass="max-h-64"
            />
          </div>
        </Show>
      </SidePanel>
    </>
  )
}
