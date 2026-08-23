import { allMonsters } from '@/shared/lib/bestiary-cache'
import { Plus } from 'lucide-solid'
import { Show, createMemo, createSignal } from 'solid-js'
import { enrichEncounter, encounterInitiativeEntries } from '@/features/gm-tools/encounter'
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
 *
 * Quem abre é o TRILHO das consultas: um overlay por vez (ALE-198).
 */
export function EncounterPanel(props: {
  rt: SessionRealtime
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const draft = createEncounterDraft()
  const [picking, setPicking] = createSignal(false)

  // `settledQuery` e não `bestiary.data ?? []`: a leitura é que suspende, antes
  // de o `??` importar — e o suspend desanexa a CENA inteira (ALE-96). Era o
  // último branco ao entrar na sessão, porque o bestiário só é buscado depois
  // da montagem.
  const groups = createMemo(() => enrichEncounter(draft.entries(), allMonsters()))

  const send = () => {
    const { entries, dropped } = encounterInitiativeEntries(
      groups(),
      props.rt.state().initiative.length,
    )
    for (const entry of entries) {
      props.rt.addEntry({ ...entry, initiative: rollD20(), type: 'npc' })
    }
    draft.clear()
    toast(`${entries.length} criaturas entraram na iniciativa`, {
      description:
        dropped > 0
          ? `${dropped} ficaram de fora — o rastreador aceita 50 entradas.`
          : 'Iniciativa rolada para cada uma (d20).',
    })
  }

  return (
      <SidePanel
        open={props.open}
        onOpenChange={props.onOpenChange}
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
            {/* Aqui o clique ADICIONA, e continua assim de propósito (ALE-208):
                o que a issue consertou foi "clicar para ler joga na mesa AO
                VIVO", e aqui nada chega à mesa até o "Mandar para a
                iniciativa" — o rascunho logo acima é o desfazer, com
                quantidade e remover ao lado de cada linha. Pôr um diálogo no
                caminho custaria fricção justamente onde adicionar rápido é o
                propósito. */}
            <MonsterPickerList
              onPick={(monster) => draft.add(monster.id)}
              idPrefix="session-encounter"
              listClass="max-h-64"
              itemVerbo="Adicionar ao encontro:"
            />
          </div>
        </Show>
      </SidePanel>
  )
}
