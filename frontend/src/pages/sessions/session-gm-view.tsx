import { Settings2 } from 'lucide-solid'
import { Show, createMemo, createSignal } from 'solid-js'
import type { Session } from '@/shared/api/api'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { DeleteSessionButton } from '@/features/session-tracker/delete-session-button'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'
import { NotesCard } from '@/features/session-tracker/notes-card'
import { PartyRoster } from '@/features/session-tracker/party-roster'
import { createMediaQuery } from '@/shared/lib/media-query'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { AddMonsterPanel } from './add-monster-panel'
import { CatalogPanel } from './catalog-panel'
import { CombatantPanel } from './combatant-panel'
import { EncounterPanel } from './encounter-panel'
import { MatchControls, MatchPeek } from './match-rail'

/**
 * A tela do mestre numa sessão ao vivo.
 *
 * Duas superfícies, não três (ALE-122): o rastreador e o combatente aberto. Os
 * controles da sessão — bestiário, encontro, catálogos, notas, excluir — saíram
 * da terceira coluna para o cabeçalho, porque são de baixa frequência e estavam
 * ocupando largura permanente durante o combate.
 *
 * Os breakpoints deste app são de JANELA, não de caixa, então uma coluna
 * estreita continuaria recebendo o layout largo: medido, a linha da iniciativa
 * numa coluna de 24rem fica 4× mais alta E estoura na horizontal. Por isso as
 * duas colunas só entram quando cabem de verdade (≥1280); abaixo disso o
 * combatente OCUPA a coluna do rastreador, com a volta explícita.
 *
 * A largura cheia (o `max-w-6xl` saiu) devolve os ~40% de tela que morriam em
 * 1920 — e devolve HAJA combatente aberto ou não.
 */
export function SessionGmView(props: {
  campaignId: number
  sessionId: number
  session: Session
  rt: SessionRealtime
  myCharacterIds: ReadonlySet<number>
}) {
  const [selectedId, setSelectedId] = createSignal<string | null>(null)
  // Derivado do estado ao vivo, não guardado: os vitais da entrada mudam a cada
  // pancada, e uma cópia mostraria o número de quando ela foi aberta.
  const selected = createMemo(
    () => props.rt.state().initiative.find((entry) => entry.id === selectedId()) ?? null,
  )
  // 1700 e não 1280: MEDIDO. A ficha dentro do painel só fica limpa a partir de
  // ~940px, porque os painéis internos dela (a grade de perícias, por exemplo)
  // também decidem por JANELA — a 812px são 22 elementos cortados, a 1092 são 0.
  // Abaixo disso vale a regra que o app já usa: uma superfície por vez.
  const sideBySide = createMediaQuery('(min-width: 1700px)')
  const trackerVisible = () => selected() === null || sideBySide()

  return (
    <div class="flex h-full min-h-0 flex-col gap-3 p-3 pb-20 sm:p-4 lg:pb-4">
      <SessionToolbar {...props} />

      <div
        class={cn(
          'grid min-h-0 flex-1 gap-4',
          // Assimétrico: o combate tolera estreitar melhor que a ficha.
          selected() && sideBySide() && 'grid-cols-[minmax(0,5fr)_minmax(0,7fr)]',
        )}
      >
        <Show when={trackerVisible()}>
          <div class="min-w-0 space-y-4 overflow-y-auto">
            <InitiativeCard
              rt={props.rt}
              isGm
              myCharacterIds={props.myCharacterIds}
              onSelect={(entryId) => setSelectedId((current) => (current === entryId ? null : entryId))}
              selectedId={selectedId()}
            />
            <PartyRoster campaignId={props.campaignId} />
          </div>
        </Show>

        <Show when={selected()}>
          {(entry) => (
            // `min-w-0`: sem isso o painel adota a largura MÍNIMA DO CONTEÚDO
            // e vaza da tela — em 390px ele ficava com 509px de largura, 131
            // fora do visor, e uma checagem que compara os filhos contra a
            // caixa DELE não acusa nada, porque a caixa é que está grande. Quem
            // pegou foi o screenshot (ALE-122).
            <div class="flex min-h-0 w-full min-w-0 flex-col gap-2">
              <Show when={!sideBySide()}>
                <Button
                  size="sm"
                  variant="outline"
                  class="self-start"
                  onClick={() => setSelectedId(null)}
                >
                  ◀ Voltar ao combate
                </Button>
              </Show>
              <CombatantPanel entry={entry()} onClose={() => setSelectedId(null)} />
            </div>
          )}
        </Show>
      </div>
    </div>
  )
}

/**
 * Os controles da sessão, atrás de um gatilho no cabeçalho. Mesma gramática que
 * o rail já usava no telefone — só que agora em todas as larguras, porque a
 * largura permanente pertence ao combate.
 */
function SessionToolbar(props: {
  campaignId: number
  sessionId: number
  session: Session
  rt: SessionRealtime
}) {
  return (
    <div class="flex flex-wrap items-center justify-between gap-2">
      <p class="text-sm text-muted-foreground">
        <MatchPeek rt={props.rt} />
      </p>
      <MatchControls
        title="Controles da sessão"
        trigger={(open) => (
          <Button size="sm" variant="outline" class="gap-1.5" onClick={open}>
            <Settings2 aria-hidden="true" class="size-4" />
            Controles
          </Button>
        )}
      >
        <div class="space-y-2">
          <AddMonsterPanel rt={props.rt} />
          <EncounterPanel rt={props.rt} />
          <CatalogPanel rt={props.rt} />
        </div>
        <HeaderCard campaignId={props.campaignId} session={props.session} isGm />
        <NotesCard campaignId={props.campaignId} session={props.session} />
        <div class="flex justify-end">
          <DeleteSessionButton
            campaignId={props.campaignId}
            sessionId={props.sessionId}
            sessionNumber={props.session.sessionNumber}
          />
        </div>
      </MatchControls>
    </div>
  )
}
