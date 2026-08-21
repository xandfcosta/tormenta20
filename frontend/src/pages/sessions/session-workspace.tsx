import { BookMarked, LayoutGrid, NotebookPen, Skull, UserRound } from 'lucide-solid'
import { Show } from 'solid-js'
import { CatalogBrowser } from '@/features/gm-tools/catalog-browser'
import { MonsterPickerList } from '@/features/gm-tools/monster-picker-list'
import { SessionNotes } from '@/features/session-tracker/session-notes'
import type { Session } from '@/shared/api/api'
import type { InitiativeEntry, SessionRealtime } from '@/shared/realtime/realtime'
import { rollD20 } from '@/shared/lib/dice'
import { toast } from '@/shared/ui/sonner'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import type { BoardViewport } from '@/features/battle-board/board-viewport'
import { BoardRegion } from './board-region'
import { CombatantPanel } from './combatant-panel'
import { EncounterPanel } from './encounter-panel'

export type WorkspaceTab = 'combatente' | 'bestiario' | 'catalogos' | 'notas' | 'tabuleiro'

export type SessionWorkspaceProps = {
  campaignId: number
  session: Session
  rt: SessionRealtime
  tab: WorkspaceTab
  onTabChange: (tab: WorkspaceTab) => void
  selected: InitiativeEntry | null
  onCloseCombatant: () => void
  /** A janela do tabuleiro, quando ele divide esta coluna (abaixo de 1536). */
  boardView?: BoardViewport
  activeEntryId?: string | null
  /** A linha sob o ponteiro na iniciativa: a peça dela acende (ALE-189). */
  highlightEntryId?: string | null
}

/**
 * A mesa do mestre: o combatente aberto, o bestiário, os catálogos e as notas —
 * quatro abas irmãs de UMA região do shell (ALE-122).
 *
 * Eram side sheets flutuantes, e é daí que vinham quatro queixas de uma vez:
 * abrir o bestiário exigia passar por um menu, um painel abria POR CIMA do
 * outro em vez de trocar, clicar fora não fechava (de propósito — fechar
 * enquanto o mestre clica no rastreador atrás seria pior), e a lista tinha um
 * teto de altura que deixava vão morto. Como região do layout, nada disso
 * existe: a aba troca, o conteúdo ocupa a altura, e nada cobre o combate.
 */
export function SessionWorkspace(props: SessionWorkspaceProps) {
  const addMonster = (monster: { id: string; name: string; hp: number }) => {
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
    <Tabs
      value={props.tab}
      onChange={(value) => props.onTabChange(value as WorkspaceTab)}
      class="grimorio-frame flex h-full min-h-0 min-w-0 flex-col bg-grimorio-panel"
    >
      <TabsList class="@container flex w-full shrink-0 gap-1 border-b border-grimorio-iron p-1">
        <WorkspaceTabTrigger value="combatente" icon={UserRound} label="Combatente" />
        <WorkspaceTabTrigger value="bestiario" icon={Skull} label="Bestiário" />
        <WorkspaceTabTrigger value="catalogos" icon={BookMarked} label="Catálogos" />
        <WorkspaceTabTrigger value="notas" icon={NotebookPen} label="Notas" />
        {/* O tabuleiro só é aba quando NÃO tem coluna própria (abaixo de 1536).
            Uma faixa separada atravessando a tela inteira para trocar só a
            coluna da direita era um controle desalinhado do próprio efeito —
            aqui a barra fica exatamente sobre o que ela troca (ALE-130). */}
        <Show when={props.boardView}>
          <WorkspaceTabTrigger value="tabuleiro" icon={LayoutGrid} label="Tabuleiro" />
        </Show>
      </TabsList>

      <TabsContent value="combatente" class="flex min-h-0 flex-1 flex-col">
        <Show
          when={props.selected}
          fallback={
            <p class="p-4 text-sm text-muted-foreground">
              Clique no nome de um combatente na iniciativa para abrir a ficha dele aqui.
            </p>
          }
        >
          {(entry) => (
            <CombatantPanel
              entry={entry()}
              campaignId={props.campaignId}
              onClose={props.onCloseCombatant}
              onApplyEffect={(spellId) => props.rt.applyEffect(entry().id, spellId)}
              onConditions={(conditions) => props.rt.updateEntry(entry().id, { conditions })}
              onLinkCreature={(creature) => {
                const linha = entry()
                // A linha herda a VIDA do bloco quando não tem nenhuma: sem
                // isto a tela se contradiz — o painel dizia "não tem vida
                // registrada" com o bloco declarando 30 PV logo abaixo, e a
                // barra do rastreador não teria o que rastrear (ALE-137).
                // Quem já tem vida não é tocado: ela é estado de combate, e o
                // bloco descreve a criatura, não o dano de agora.
                props.rt.updateEntry(linha.id, {
                  creatureId: creature.id,
                  ...(linha.hpMax === undefined
                    ? { hpCurrent: creature.block.hp, hpMax: creature.block.hp }
                    : {}),
                })
              }}
            />
          )}
        </Show>
      </TabsContent>

      <TabsContent value="bestiario" class="flex min-h-0 flex-1 flex-col gap-2 p-2">
        {/* O encontro nasce do bestiário, então mora ao lado dele. */}
        <div class="shrink-0">
          <EncounterPanel rt={props.rt} />
        </div>
        <MonsterPickerList onPick={addMonster} idPrefix="mesa" />
      </TabsContent>

      <TabsContent value="catalogos" class="flex min-h-0 flex-1 flex-col p-2">
        <CatalogBrowser listClass="min-h-0 flex-1 pr-1" />
      </TabsContent>

      <TabsContent value="notas" class="flex min-h-0 flex-1 flex-col p-2">
        <SessionNotes campaignId={props.campaignId} session={props.session} />
      </TabsContent>

      {/* `Tabs` desmonta o conteúdo inativo, e é por isso que a JANELA do
          tabuleiro (origem e zoom) mora na página: ela sobrevive à troca de aba.
          O que se perde ao trocar é a peça selecionada, que é escolha de um
          instante e não estado de cena. */}
      <Show when={props.boardView}>
        {(view) => (
          <TabsContent value="tabuleiro" class="flex min-h-0 flex-1 flex-col p-2">
            <BoardRegion
              rt={props.rt}
              isGm
              view={view()}
              activeEntryId={props.activeEntryId}
              highlightEntryId={props.highlightEntryId}
            />
          </TabsContent>
        )}
      </Show>
    </Tabs>
  )
}

/**
 * Ícone + rótulo: no meio de um combate, ícone sem rótulo é adivinhação.
 *
 * O aperto é medido no CONTÊINER, não na viewport: as abas ocupam 7/12 da tela
 * no desktop e a tela inteira no telefone, então a mesma largura de tela dá
 * duas folgas diferentes. Sem `min-w-0` a última aba saía 15px para fora da
 * faixa a 390px — o `flex-1` não encolhe abaixo do conteúdo (ALE-122).
 */
function WorkspaceTabTrigger(props: {
  value: WorkspaceTab
  icon: typeof Skull
  label: string
}) {
  return (
    <TabsTrigger value={props.value} class="min-w-0 flex-1 gap-1 px-1 @sm:gap-1.5 @sm:px-3">
      <props.icon aria-hidden="true" class="size-4" />
      <span class="truncate text-xs">{props.label}</span>
    </TabsTrigger>
  )
}
