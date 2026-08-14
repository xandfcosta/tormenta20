import { BookMarked, NotebookPen, Skull, UserRound } from 'lucide-solid'
import { Show } from 'solid-js'
import { CatalogBrowser } from '@/features/gm-tools/catalog-browser'
import { MonsterPickerList } from '@/features/gm-tools/monster-picker-list'
import { SessionNotes } from '@/features/session-tracker/session-notes'
import type { Session } from '@/shared/api/api'
import type { InitiativeEntry, SessionRealtime } from '@/shared/realtime/realtime'
import { rollD20 } from '@/shared/lib/dice'
import { toast } from '@/shared/ui/sonner'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { CombatantPanel } from './combatant-panel'
import { EncounterPanel } from './encounter-panel'

export type WorkspaceTab = 'combatente' | 'bestiario' | 'catalogos' | 'notas'

export type SessionWorkspaceProps = {
  campaignId: number
  session: Session
  rt: SessionRealtime
  tab: WorkspaceTab
  onTabChange: (tab: WorkspaceTab) => void
  selected: InitiativeEntry | null
  onCloseCombatant: () => void
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
  const addMonster = (monster: { name: string; hp: number }) => {
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
    <Tabs
      value={props.tab}
      onChange={(value) => props.onTabChange(value as WorkspaceTab)}
      class="flex h-full min-h-0 min-w-0 flex-col rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)]"
    >
      <TabsList class="@container flex w-full shrink-0 gap-1 border-b border-grimorio-iron p-1">
        <WorkspaceTabTrigger value="combatente" icon={UserRound} label="Combatente" />
        <WorkspaceTabTrigger value="bestiario" icon={Skull} label="Bestiário" />
        <WorkspaceTabTrigger value="catalogos" icon={BookMarked} label="Catálogos" />
        <WorkspaceTabTrigger value="notas" icon={NotebookPen} label="Notas" />
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
              onClose={props.onCloseCombatant}
              onApplyEffect={(spellId) => props.rt.applyEffect(entry().id, spellId)}
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
