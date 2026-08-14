import { Settings2, Swords } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import type { Session } from '@/shared/api/api'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { DeleteSessionButton } from '@/features/session-tracker/delete-session-button'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'
import { RestControls } from '@/features/session-tracker/rest-controls'
import { createMediaQuery } from '@/shared/lib/media-query'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { MatchControls } from './match-rail'
import { type WorkspaceTab, SessionWorkspace } from './session-workspace'

/**
 * A cena do mestre como SHELL: cabe numa tela, sem rolagem de página — que é a
 * premissa de "sensação de jogo" do app (ALE-122).
 *
 * Duas regiões e uma faixa. A faixa do turno fica FIXA no topo, porque o botão
 * mais clicado da sessão saía da tela ao rolar e "de quem é a vez" estava
 * desenhado como metadado de 13px no canto. À esquerda, a iniciativa — a única
 * coisa que rola. À direita, a mesa: combatente, bestiário, catálogos e notas
 * como abas irmãs, no lugar dos side sheets que se empilhavam.
 *
 * O card "Grupo" saiu: eram os mesmos PCs com os mesmos números que já estão na
 * iniciativa, ocupando a maior área da tela para não dizer nada novo.
 *
 * Abaixo de lg não cabem duas regiões, então uma por vez com um seletor
 * visível — nunca uma escondida atrás da outra.
 */
export function SessionGmView(props: {
  campaignId: number
  sessionId: number
  session: Session
  rt: SessionRealtime
  myCharacterIds: ReadonlySet<number>
}) {
  const [selectedId, setSelectedId] = createSignal<string | null>(null)
  const [tab, setTab] = createSignal<WorkspaceTab>('combatente')
  const [region, setRegion] = createSignal<'combate' | 'mesa'>('combate')
  // Derivado do estado ao vivo: os vitais mudam a cada pancada, e uma cópia
  // mostraria o número de quando o combatente foi aberto.
  const selected = createMemo(
    () => props.rt.state().initiative.find((entry) => entry.id === selectedId()) ?? null,
  )
  const sideBySide = createMediaQuery('(min-width: 1024px)')

  const select = (entryId: string) => {
    setSelectedId((current) => (current === entryId ? null : entryId))
    setTab('combatente')
    setRegion('mesa')
  }

  const showTracker = () => sideBySide() || region() === 'combate'
  const showWorkspace = () => sideBySide() || region() === 'mesa'

  return (
    <div class="flex h-full min-h-0 flex-col gap-2 p-2 sm:gap-3 sm:p-3">
      <TurnBar
        campaignId={props.campaignId}
        sessionId={props.sessionId}
        session={props.session}
        rt={props.rt}
      />

      <Show when={!sideBySide()}>
        <RegionSwitch region={region()} onRegion={setRegion} />
      </Show>

      <div
        class={cn(
          'grid min-h-0 flex-1 gap-3',
          sideBySide() && 'grid-cols-[minmax(0,5fr)_minmax(0,7fr)]',
        )}
      >
        <Show when={showTracker()}>
          <div class="flex min-h-0 min-w-0 flex-col overflow-y-auto">
            <InitiativeCard
              rt={props.rt}
              isGm
              myCharacterIds={props.myCharacterIds}
              turnControls={false}
              onSelect={select}
              selectedId={selectedId()}
            />
          </div>
        </Show>

        <Show when={showWorkspace()}>
          <SessionWorkspace
            campaignId={props.campaignId}
            session={props.session}
            rt={props.rt}
            tab={tab()}
            onTabChange={setTab}
            selected={selected()}
            onCloseCombatant={() => setSelectedId(null)}
          />
        </Show>
      </div>
    </div>
  )
}

/**
 * Rodada, de quem é a vez e o avanço do turno — FIXO, porque é o que muda a
 * cada dez segundos e era o primeiro a sumir ao rolar.
 *
 * "Reiniciar" pede confirmação e sai de perto do "Próximo turno": ele apaga o
 * combate inteiro e estava encostado no botão mais clicado da sessão, enquanto
 * "Excluir sessão" — muito menos provável — já tinha confirmação. A proteção
 * estava no lugar errado.
 */
function TurnBar(props: {
  campaignId: number
  sessionId: number
  session: Session
  rt: SessionRealtime
}) {
  const state = () => props.rt.state()
  const active = () => (state().turnIndex >= 0 ? state().initiative[state().turnIndex] : undefined)

  return (
    <div class="flex shrink-0 flex-wrap items-center gap-2 rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)] px-3 py-2">
      <span class="font-mono text-sm tabular-nums text-muted-foreground">
        Rodada {state().round}
      </span>
      <Show when={active()}>
        {(entry) => (
          <span class="flex min-w-0 items-center gap-1.5 font-heading text-sm uppercase tracking-wide text-grimorio-gold">
            <Swords aria-hidden="true" class="size-4 text-[color:var(--primary)]" />
            <span class="truncate">Vez de {entry().label}</span>
          </span>
        )}
      </Show>

      <div class="ml-auto flex items-center gap-2">
        <Button size="sm" disabled={!props.rt.isConnected()} onClick={props.rt.nextTurn}>
          Próximo turno
        </Button>
        <ConfirmDialog
          title="Reiniciar o combate?"
          description="A iniciativa, a rodada e o turno voltam a zero, e os combatentes saem da lista."
          confirmLabel="Reiniciar"
          destructive
          onConfirm={props.rt.resetInitiative}
          trigger={(open) => (
            <Button size="sm" variant="outline" disabled={!props.rt.isConnected()} onClick={open}>
              Reiniciar
            </Button>
          )}
        />
        <MatchControls
          title="Sessão"
          trigger={(open) => (
            <Button size="sm" variant="ghost" aria-label="Configurações da sessão" onClick={open}>
              <Settings2 aria-hidden="true" class="size-4" />
            </Button>
          )}
        >
          <RestControls rt={props.rt} />
          <HeaderCard campaignId={props.campaignId} session={props.session} isGm />
          <div class="flex justify-end">
            <DeleteSessionButton
              campaignId={props.campaignId}
              sessionId={props.sessionId}
              sessionNumber={props.session.sessionNumber}
            />
          </div>
        </MatchControls>
      </div>
    </div>
  )
}

/** Uma região por vez onde não cabem duas — visível, nunca escondida. */
function RegionSwitch(props: {
  region: 'combate' | 'mesa'
  onRegion: (region: 'combate' | 'mesa') => void
}) {
  return (
    <div class="flex shrink-0 gap-1">
      <For each={['combate', 'mesa'] as const}>
        {(value) => (
          <Button
            size="sm"
            variant={props.region === value ? 'default' : 'outline'}
            aria-pressed={props.region === value}
            class="flex-1 capitalize"
            onClick={() => props.onRegion(value)}
          >
            {value}
          </Button>
        )}
      </For>
    </div>
  )
}
