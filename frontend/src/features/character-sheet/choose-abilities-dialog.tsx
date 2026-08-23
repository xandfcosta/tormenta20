import type { RaceDefinition } from '@/shared/api/catalog-types'
import { Plus } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { getRace } from '@/shared/lib/abilities-cache'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { CountBadge } from '@/shared/ui/count-badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { ClassesSection } from './class-abilities'
import type { CardFocus } from './collapsible-ability-card'
import { OriginAbilitySection } from './origin-abilities'
import {
  type Pendencia,
  type PendenciaSource,
  computePendencias,
  escolhasPendentes,
} from './pendencias'
import { PendenciasCallout } from './pendencias-callout'
import { RaceAbilitySection } from './race-abilities'
import { SectionTitle } from '@/shared/ui/section-label'

const SOURCE_TABS: { value: PendenciaSource; label: string }[] = [
  { value: 'raca', label: 'Raça' },
  { value: 'origem', label: 'Origem' },
  { value: 'classe', label: 'Classe' },
]

/**
 * "Escolher poderes" — a administração da ficha, num DIÁLOGO (ALE-217).
 *
 * Ela era um MODO da tela de Poderes, e o modo é o que a tornava difícil de
 * usar. Medido a 390×844: o cromo do modo de edição — o cabeçalho, o aviso de
 * pendências, a busca, o botão de voltar e as três abas de fonte — comia 235px
 * dos 530 do painel, 44%, sobrando cinco linhas para os 26 a 34 poderes que um
 * personagem de nível 20 possui. E a tela ABRIA nele sempre que havia pendência
 * — que é o estado normal de quem acabou de subir de nível —, então em sessão o
 * jogador caía na tela de administração em vez da de jogo.
 *
 * Escolher poder acontece uma vez por nível. Virar meia tela para isso é caro
 * demais; um botão que abre diálogo é a gramática que a casa padronizou
 * (ALE-169, ALE-216), e ela devolve os 235px para a lista.
 *
 * As três abas de FONTE ficam aqui dentro, e aqui elas fazem sentido: no
 * preparo a pergunta é "de onde vem o que eu ainda posso escolher". Na mesa a
 * pergunta é outra, e por isso a lista de trás não tem abas.
 *
 * @example <ChooseAbilitiesDialog character={character} />
 */
export function ChooseAbilitiesDialog(props: { character: Character }) {
  const [open, setOpen] = createSignal(false)
  const pendencias = createMemo(() => computePendencias(props.character))

  const pendingByCard = createMemo(() => {
    const byCard = new Map<string, number>()
    for (const pendencia of pendencias()) {
      byCard.set(pendencia.cardId, (byCard.get(pendencia.cardId) ?? 0) + 1)
    }
    return byCard
  })

  const [tab, setTab] = createSignal<PendenciaSource>('raca')
  const [focus, setFocus] = createSignal<CardFocus>(null)

  /** Abrir aterrissa na primeira fonte que ainda deve escolha, e não na Raça:
   *  quem abriu o diálogo veio pela pendência, e fazê-lo caçar a aba certa é
   *  gastar o clique que ele acabou de dar. */
  const start = () => {
    setTab(pendencias()[0]?.source ?? 'raca')
    setFocus(null)
    setOpen(true)
  }

  const jump = (pendencia: Pendencia) => {
    setTab(pendencia.source)
    // Um nonce novo para reclicar a mesma pendência reabrir o cartão.
    setFocus({ id: pendencia.cardId, nonce: (focus()?.nonce ?? 0) + 1 })
  }

  const countFor = (source: PendenciaSource) =>
    pendencias().filter((pendencia) => pendencia.source === source).length

  const races = createMemo(() =>
    props.character.races
      .map((entry) => getRace(entry.race))
      .filter((race): race is RaceDefinition => Boolean(race)),
  )

  return (
    <>
      <Button
        type="button"
        variant={pendencias().length > 0 ? 'default' : 'outline'}
        size="sm"
        class="h-8 shrink-0 gap-1 text-xs"
        aria-label={
          pendencias().length > 0
            ? `Escolher poderes, ${escolhasPendentes(pendencias().length)}`
            : 'Escolher poderes'
        }
        onClick={start}
      >
        <Plus aria-hidden="true" class="size-3.5" />
        Escolher
        <Show when={pendencias().length > 0}>
          <CountBadge count={pendencias().length} anunciadoPeloPai />
        </Show>
      </Button>

      <Dialog open={open()} onOpenChange={setOpen}>
        {/* A altura mora na regra `.dialogo-de-ficha`, a mesma do diálogo da
            ficha e do combatente: dentro vai uma lista de cartões que decide o
            piso, e é medida de OBJETO. Como valor arbitrário ela sumia do CSS
            de forma intermitente e derrubou o e2e três vezes (ALE-200). */}
        <DialogContent class="dialogo-de-ficha flex w-full max-w-3xl flex-col gap-0 overflow-hidden p-0 sm:max-w-3xl">
          <DialogHeader class="shrink-0 border-b border-grimorio-iron px-3 py-2 sm:px-4">
            <DialogTitle>
              <SectionTitle as="span" contexto="painel">
                Escolher poderes
              </SectionTitle>
            </DialogTitle>
          </DialogHeader>

          <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden p-3 sm:p-4">
            <PendenciasCallout pendencias={pendencias()} onJump={jump} />
            <SourceTabs active={tab()} countFor={countFor} onPick={setTab} />

            <div class="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
              <Show when={tab() === 'raca'}>
                <Show
                  when={races().length > 0}
                  fallback={<EmptyHint>Raça do personagem não está no catálogo.</EmptyHint>}
                >
                  <For each={races()}>
                    {(race) => (
                      <RaceAbilitySection
                        race={race}
                        character={props.character}
                        focus={focus()}
                        pending={pendingByCard().get(`raca:${race.id}`) ?? 0}
                      />
                    )}
                  </For>
                </Show>
              </Show>

              <Show when={tab() === 'origem'}>
                <OriginAbilitySection
                  character={props.character}
                  focus={focus()}
                  pending={pendingByCard().get('origem') ?? 0}
                />
              </Show>

              <Show when={tab() === 'classe'}>
                <Show
                  when={props.character.classes.length > 0}
                  fallback={<EmptyHint>Nenhuma classe atribuída.</EmptyHint>}
                >
                  <For each={props.character.classes}>
                    {(entry) => (
                      <ClassesSection
                        entry={entry}
                        character={props.character}
                        focus={focus()}
                        pending={pendingByCard().get(`classe:${entry.className}`) ?? 0}
                      />
                    )}
                  </For>
                </Show>
              </Show>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}

/**
 * Fileira de pílulas em vez do `Tabs` da casa: este bloco vive dentro das abas
 * VERTICAIS da ficha, e uma `TabsList` aninhada herda aquela orientação e
 * empilha.
 */
function SourceTabs(props: {
  active: PendenciaSource
  countFor: (source: PendenciaSource) => number
  onPick: (source: PendenciaSource) => void
}) {
  return (
    // overflow-y-hidden: um `overflow-x-auto` sozinho deixa o navegador
    // promover o overflow-y para `auto`, e 1px da borda da aba ativa fazia
    // nascer uma barra de rolagem vertical nesta faixa de uma linha só.
    <div class="flex shrink-0 gap-1 overflow-x-auto overflow-y-hidden border-b border-grimorio-iron">
      <For each={SOURCE_TABS}>
        {(source) => (
          <button
            type="button"
            onClick={() => props.onPick(source.value)}
            aria-pressed={props.active === source.value}
            // O nome vem do CONTROLE, não de uma linha `sr-only` dentro dele: o
            // cálculo do nome acessível concatena os filhos sem separador, e a
            // aba anunciava "Origem2" (ALE-173, P6).
            aria-label={
              props.countFor(source.value) > 0
                ? `${source.label}, ${escolhasPendentes(props.countFor(source.value))}`
                : source.label
            }
            class={cn(
              '-mb-px flex items-center gap-1.5 whitespace-nowrap border-b-2 px-3 py-1.5 text-sm font-medium transition-colors',
              props.active === source.value
                ? 'border-grimorio-crimson-bright text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground',
            )}
          >
            {source.label}
            <Show when={props.countFor(source.value) > 0}>
              <CountBadge count={props.countFor(source.value)} anunciadoPeloPai />
            </Show>
          </button>
        )}
      </For>
    </div>
  )
}

function EmptyHint(props: { children: string }) {
  return <p class="text-xs italic text-muted-foreground">{props.children}</p>
}
