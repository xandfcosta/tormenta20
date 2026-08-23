import { Show, createSignal } from 'solid-js'
import { SceneCycleControls } from '@/features/session-tracker/scene-cycle-controls'
import { cabemNaFila, janelaDaFila } from '@/features/session-tracker/rail-geometry'
import { createElementSize } from '@/shared/lib/element-size'
import type { InitiativeEntry, PresenceUser } from '@/shared/realtime/realtime'
import { CastRail } from './cast-rail'
import { InitiativeRail } from './initiative-rail'

/**
 * O TRILHO ESQUERDO do mestre: dois blocos e o ciclo da cena (ALE-211).
 *
 * Ele deixou de ser só a fila. Agora hospeda a fila do combate e o ELENCO
 * (ALE-212), e a repartição da altura entre os dois é a parte que carrega
 * regra: nenhum encolhe abaixo de metade do trilho.
 *
 * Quem faz isso é o CSS, e de propósito — `max-h-1/2` na fila, `flex-1` no
 * elenco. Com poucos combatentes a fila fica menor que metade e o elenco recebe
 * a sobra; com muitos ela para na metade. Escrever isso em JS seria medir para
 * decidir uma altura que o navegador já sabe calcular.
 *
 * O que o JS precisa medir é OUTRA coisa: **quantos combatentes cabem**. A fila
 * não rola (a posição carrega significado), então o número de vizinhos
 * desenhados é uma decisão, e decisão sobre o que a tela AFIRMA se mede por
 * observador — a lição da ALE-139. A conta mora em `rail-geometry.ts` e parte
 * da altura dos BLOCOS, nunca da altura do bloco da fila: medir o bloco para
 * decidir o que pôr nele é um laço (desenha menos, encolhe, cabem mais).
 *
 * O ciclo da cena fica FORA da caixa medida, no pé da coluna. Se ele entrasse,
 * a altura medida mudaria quando o mestre encerrasse a cena — e a fila
 * recalcularia a vizinhança por causa de um botão que sumiu.
 *
 * A largura foi de 64 para 80px (decisão do dono): o retrato de iniciais tem
 * 44px, e com o contorno de conexão em volta dele 64 não sobrava respiro.
 *
 * Abaixo de 1024 este trilho NÃO EXISTE (ALE-198) — quem o monta decide isso.
 */
export function SessionRail(props: {
  campaignId: number
  entries: readonly InitiativeEntry[]
  turnIndex: number
  activeEntryId: string | null
  present: PresenceUser[]
  connected: boolean
  sceneActive: boolean
  onOpenCombatant: (entryId: string) => void
  onOpenQueue: () => void
  onOpenCast: () => void
  onOpenCharacter: (characterId: number) => void
  onHoverEntry?: (entryId: string | null) => void
  onEndScene: () => void
  onResetScene: () => void
}) {
  const [blocos, setBlocos] = createSignal<HTMLDivElement>()
  const tamanho = createElementSize(blocos)
  const janela = () =>
    janelaDaFila({
      total: props.entries.length,
      turnIndex: props.turnIndex,
      cabem: cabemNaFila(tamanho().height),
    })
  const temFimDeCiclo = () => props.sceneActive || props.entries.length > 0

  return (
    <div class="flex w-20 shrink-0 flex-col gap-2">
      <div ref={setBlocos} class="flex min-h-0 flex-1 flex-col gap-2">
        {/* `max-h-1/2` e não `h-1/2`: com três combatentes a fila ocupa três
            itens e devolve o resto ao elenco. O piso de metade é do ELENCO, que
            é quem cresce — e ele o ganha justamente por a fila ter teto. */}
        <InitiativeRail
          class="max-h-1/2"
          entries={props.entries}
          turnIndex={props.turnIndex}
          janela={janela()}
          activeEntryId={props.activeEntryId}
          onOpenCombatant={props.onOpenCombatant}
          onExpand={props.onOpenQueue}
          onHoverEntry={props.onHoverEntry}
        />
        {/* O elenco cresce até o que sobra, mas a MOLDURA para onde o conteúdo
            para. Com `flex-1` direto no `nav` ele pintava meia tela de quadro
            vazio abaixo de quatro jogadores — o vício do painel vazio que a
            ALE-171 nomeou. O invólucro fica com a sobra (que é fundo, e fundo
            vazio não incomoda) e o bloco herda dela só o TETO, então ele rola
            quando a mesa for grande e encolhe quando for pequena. */}
        <div class="flex min-h-0 flex-1 flex-col">
          <CastRail
            class="min-h-0 max-h-full"
            campaignId={props.campaignId}
            present={props.present}
            onOpenCharacter={props.onOpenCharacter}
            onExpand={props.onOpenCast}
          />
        </div>
      </div>

      {/* Fora da caixa medida, e fora dos dois blocos: o ciclo é da CENA e não
          da fila nem do elenco. Ele some inteiro quando não há o que encerrar
          nem o que reiniciar — elemento JSX é sempre verdadeiro, então quem
          decide é aqui e não um `Show` lá dentro (ALE-210). */}
      <Show when={temFimDeCiclo()}>
        <div class="flex shrink-0 flex-col gap-1">
          <SceneCycleControls
            compact
            connected={props.connected}
            sceneActive={props.sceneActive}
            hasQueue={props.entries.length > 0}
            onEnd={props.onEndScene}
            onReset={props.onResetScene}
          />
        </div>
      </Show>
    </div>
  )
}
