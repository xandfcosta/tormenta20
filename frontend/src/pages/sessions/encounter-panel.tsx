import { Plus } from 'lucide-solid'
import { createMemo, createSignal, Show } from 'solid-js'
import { encounterInitiativeEntries, enrichEncounter } from '@/features/gm-tools/encounter'
import { EncounterComposer } from '@/features/gm-tools/encounter-composer'
import { createEncounterDraft } from '@/features/gm-tools/encounter-draft'
import { MonsterPickerList } from '@/features/gm-tools/monster-picker-list'
import { partyFromMembers } from '@/features/gm-tools/party-defaults'
import type { CampaignMember } from '@/shared/api/types'
import { allMonsters } from '@/shared/lib/bestiary-cache'
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
 * O leiaute é de TRÊS faixas e isso é a ALE-209: o rascunho rola em cima, a
 * lista do bestiário pega TODA a altura que sobra, e o "Mandar para a
 * iniciativa" fica ancorado embaixo. Antes o rodapé era desenhado pelo
 * compositor, então com o bestiário aberto ele ficava encalhado NO MEIO do
 * painel, com a lista pendurada abaixo dele e cortada em 256px fixos —
 * medido a 1440×900, o resto saía pela borda de baixo da janela.
 *
 * Quem abre é o TRILHO das consultas: um overlay por vez (ALE-198).
 */
export function EncounterPanel(props: {
  rt: SessionRealtime
  /** O elenco da campanha, para o grupo já nascer preenchido (ALE-209). Vem
   *  por prop e não por `useQuery` daqui de dentro: este painel vive num
   *  portal, e criar recurso num dono reativo novo DESANEXA a cena (ALE-199). */
  members: readonly CampaignMember[]
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const draft = createEncounterDraft(() => partyFromMembers(props.members))
  const [picking, setPicking] = createSignal(false)

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
          ? `${dropped} ficaram de fora — a fila aceita 50 entradas.`
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
      {/* O corpo do `SidePanel` já é uma coluna flex — as três faixas se
          distribuem nele sem prop nova.

          O rascunho rola por si: com dez criaturas ele é o que cresce, e é ele
          que deve ceder espaço à lista, não a lista a ele. */}
      <div class="min-h-0 shrink overflow-y-auto pr-1">
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
        />
      </div>

      {/* Inline, NOT a second side panel: the composer IS the panel here, and
          a nested one would land exactly on top of the composition being
          added to. */}
      <Show when={picking()}>
        {/* `min-h-64` é PISO, não teto — e é a diferença entre esta faixa e o
            `max-h-64` que ela substitui. Sem ele, o `flex-1` devolve ZERO
            altura no celular deitado (844×390): medido, a lista não pintava
            linha nenhuma, o que é pior do que os 256px fixos que estouravam.
            Com o piso, esse formato passa a rolar o painel inteiro — cabe mal,
            e caber mal ali é o assunto da ALE-230, que já mediu o mesmo em
            outras duas superfícies. Nos outros cinco formatos o piso nunca
            entra em jogo e a lista pega o que sobra.

            O piso CAI no celular deitado — `max-lg:landscape:`, a chave da casa
            para "largura de tablet, altura de telefone", que nem a largura
            sozinha nem `@container` enxergam. Com o piso cheio o "Mandar para a
            iniciativa" ficava 44px abaixo da dobra (medido): sobra menos lista,
            mas a saída do painel volta para dentro da tela. */}
        <div class="flex min-h-64 flex-1 flex-col border-t border-grimorio-iron pt-3 max-lg:landscape:min-h-48">
          {/* Aqui o clique ADICIONA, e continua assim de propósito (ALE-208):
              o que a issue consertou foi "clicar para ler joga na mesa AO
              VIVO", e aqui nada chega à mesa até o "Mandar para a
              iniciativa" — o rascunho logo acima é o desfazer, com
              quantidade e remover ao lado de cada linha. Pôr um diálogo no
              caminho custaria fricção justamente onde adicionar rápido é o
              propósito. */}
          <MonsterPickerList onPick={(monster) => draft.add(monster.id)} idPrefix="session-encounter" itemVerbo="Adicionar ao encontro:" />
        </div>
      </Show>

      {/* Ancorado: é a saída do painel, e ele não pode subir para o meio da
          tela quando a lista abre. */}
      <div class="shrink-0 border-t border-grimorio-iron pt-3">
        <Button
          type="button"
          size="sm"
          class="w-full"
          disabled={groups().length === 0 || !props.rt.isConnected()}
          onClick={send}
        >
          Mandar para a iniciativa
        </Button>
      </div>
    </SidePanel>
  )
}
