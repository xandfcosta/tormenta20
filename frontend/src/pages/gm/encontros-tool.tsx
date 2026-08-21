import { useQuery } from '@tanstack/solid-query'
import { Plus } from 'lucide-solid'
import { createMemo, createSignal } from 'solid-js'
import { bestiaryCatalogQueryOptions } from '@/entities/catalog/queries'
import { enrichEncounter } from '@/features/gm-tools/encounter'
import { EncounterComposer } from '@/features/gm-tools/encounter-composer'
import { createEncounterDraft } from '@/features/gm-tools/encounter-draft'
import { MonsterPickerPanel } from '@/features/gm-tools/monster-picker-panel'
import { Button } from '@/shared/ui/button'
import { SectionTitle } from '@/shared/ui/section-label'

/**
 * Construtor de encontros — the party, the creatures, and what the fight adds
 * up to (Cap 7 p282). Composing here is planning between sessions; the same
 * composer runs inside a live session, where it can also push the result
 * straight into the initiative tracker.
 */
export function EncontrosTool() {
  const bestiary = useQuery(() => bestiaryCatalogQueryOptions)
  const draft = createEncounterDraft()
  const [picking, setPicking] = createSignal(false)

  const groups = createMemo(() => enrichEncounter(draft.entries(), bestiary.data ?? []))

  return (
    <section class="flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="mesa-encontros">
      <SectionTitle
        id="mesa-encontros"
       
      >
        Construtor de encontros
      </SectionTitle>

      <div class="min-h-0 flex-1 overflow-y-auto pr-1">
        <EncounterComposer
          groups={groups()}
          partyLevel={draft.partyLevel()}
          partySize={draft.partySize()}
          onPartyLevel={draft.setPartyLevel}
          onPartySize={draft.setPartySize}
          onQuantity={draft.setQuantity}
          onRemove={draft.remove}
          addControl={
            <Button type="button" variant="outline" size="sm" onClick={() => setPicking(true)}>
              <Plus aria-hidden="true" class="mr-1 size-4" />
              Adicionar criatura
            </Button>
          }
        />
      </div>

      <MonsterPickerPanel
        open={picking()}
        onOpenChange={setPicking}
        title="Adicionar criatura"
        description="Escolha quantas quiser — o painel fica aberto."
        onPick={(monster) => draft.add(monster.id)}
      />
    </section>
  )
}
