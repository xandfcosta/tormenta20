import { Check, Crosshair, Eye, Maximize, Minimize, Undo2, X } from 'lucide-solid'
import { For, Show } from 'solid-js'
import type { BoardState } from '@/shared/realtime/realtime'
import type { BoardAreaKind, BoardMeasurement } from '@/shared/lib/engine-wasm'
import { NumberInput } from '@/shared/ui/number-input'
import { Button } from '@/shared/ui/button'
import type { FullscreenController } from '@/shared/lib/fullscreen'
import { type BoardViewport, SQUARE_METRES } from './board-viewport'

/**
 * As tiras do tabuleiro: o que ele DIZ enquanto a mesa joga.
 *
 * Saíram da cena da sessão (ALE-191) porque não decidem nada — recebem o que
 * mostrar e para quem avisar. O que ficou lá é a composição: o socket, a
 * seleção, a lente e os controles do mestre.
 */

/**
 * A tira da lente do mestre (ALE-193).
 *
 * Existe porque um modo que se esquece é pior que nenhum: o mestre que não
 * percebe que está na vista da mesa não vê a peça que ele mesmo escondeu, e vai
 * concluir que ela sumiu. Por isso ela é PERSISTENTE, nomeia o modo em texto e
 * carrega a própria saída.
 *
 * E diz o NÚMERO de peças escondidas, que é a pergunta que trouxe o mestre até
 * aqui — "a emboscada está mesmo invisível?". Contar o que sumiu da tela não é
 * resposta: ele não sabe o que não está vendo.
 */
export function PlayerLensBar(props: { hidden: number; onExit: () => void }) {
  return (
    <div
      role="status"
      class="flex shrink-0 flex-wrap items-center gap-2 border-b border-grimorio-gold/40 bg-grimorio-gold/10 px-3 py-1 text-[11px] text-grimorio-gold"
    >
      <Eye aria-hidden="true" class="size-3.5 shrink-0" />
      <p>
        Você está vendo a cena como a mesa.
        {props.hidden > 0
          ? ` ${props.hidden} ${props.hidden === 1 ? 'peça escondida não aparece' : 'peças escondidas não aparecem'}.`
          : ' Nenhuma peça escondida nesta cena.'}
      </p>
      <Button size="sm" variant="ghost" class="ml-auto" onClick={() => props.onExit()}>
        Voltar à vista do mestre
      </Button>
    </div>
  )
}

/**
 * A barra do movimento proposto (ALE-124).
 *
 * Diz o custo em QUADRADOS e em metros: quadrado é a unidade da regra (T20
 * p236) e metro é a unidade da conversa na mesa. E diz o orçamento ao lado,
 * porque "4" sem "de 6" não responde a pergunta que o jogador tem.
 *
 * Quem não decide continua lendo a barra: a mesa inteira vê para onde a peça
 * está indo, que é a razão de o provisório ser estado e não arraste privado.
 */
export function MoveBar(props: {
  move: NonNullable<BoardState['pending']>
  canDecide: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  const metres = () => (props.move.cost * SQUARE_METRES).toFixed(1).replace('.', ',')

  return (
    <div class="flex shrink-0 flex-wrap items-center gap-2 border-t border-grimorio-iron px-3 py-1.5">
      <p class="font-mono text-[11px] tabular-nums text-grimorio-gold">
        {props.move.cost} {props.move.cost === 1 ? 'quadrado' : 'quadrados'} ({metres()}m)
        {props.move.budget >= 0 ? ` de ${props.move.budget}` : ' · sem limite de turno'}
      </p>
      <Show when={props.canDecide} fallback={<span class="text-[11px] text-muted-foreground">Aguardando confirmação.</span>}>
        <div class="ml-auto flex items-center gap-1">
          <Button size="sm" variant="ghost" onClick={() => props.onCancel()}>
            <Undo2 aria-hidden="true" class="size-4" />
            Refazer
          </Button>
          <Button size="sm" onClick={() => props.onConfirm()}>
            <Check aria-hidden="true" class="size-4" />
            Confirmar
          </Button>
        </div>
      </Show>
    </div>
  )
}

/**
 * Mover a vista e o zoom. Num plano INFINITO isto não é enfeite: sem uma forma
 * de andar com a janela, metade da cena fica inalcançável.
 *
 * "Centralizar" enquadra as PEÇAS, e não a origem: o centro de um plano infinito
 * não significa nada — o que o mestre quer é achar o grupo.
 */
/**
 * Os controles da vista. As quatro setas e o −/+ saíram (ALE-124): arrastar e a
 * roda/pinça fazem o mesmo melhor e sem ocupar seis lugares num cabeçalho que
 * já quebrava linha no telefone. Fica o que gesto nenhum faz — achar o grupo
 * num plano infinito — e entra a tela cheia, que é do TABULEIRO e não da
 * página: em tela cheia da página o mapa continuaria dividindo espaço com a
 * iniciativa, que é justamente o que se quer sair.
 */
export function ViewControls(props: {
  view: BoardViewport
  onFit: () => void
  fullscreen: FullscreenController
}) {
  return (
    <div class="flex items-center gap-0.5">
      <Button size="sm" variant="ghost" aria-label="Centralizar nas peças" onClick={props.onFit}>
        <Crosshair aria-hidden="true" class="size-4" />
      </Button>
      <Show when={props.fullscreen.supported}>
        <Button
          size="sm"
          variant="ghost"
          aria-label={props.fullscreen.active() ? 'Sair da tela cheia' : 'Tabuleiro em tela cheia'}
          onClick={props.fullscreen.toggle}
        >
          <Show
            when={props.fullscreen.active()}
            fallback={<Maximize aria-hidden="true" class="size-4" />}
          >
            <Minimize aria-hidden="true" class="size-4" />
          </Show>
        </Button>
      </Show>
    </div>
  )
}

/**
 * A leitura da régua (ALE-124, fatia 6).
 *
 * Diz a distância em QUADRADOS e em metros — quadrado é a unidade da regra
 * (p236), metro é a unidade da conversa — e, principalmente, a FAIXA de alcance
 * do livro (p224). É a faixa que responde a pergunta real: "10,5m" obriga o
 * jogador a lembrar que curto são 9m, enquanto "alcance médio" já é a resposta.
 */
export function RulerBar(props: { reading: BoardMeasurement; onClose: () => void }) {
  const metres = () => props.reading.metres.toFixed(1).replace('.', ',')
  const faixa = () =>
    props.reading.band === 'além'
      ? 'além do alcance longo'
      : `alcance ${props.reading.band}`

  return (
    <div
      role="status"
      class="flex shrink-0 flex-wrap items-center gap-2 border-t border-grimorio-iron px-3 py-1.5"
    >
      <p class="font-mono text-[11px] tabular-nums text-grimorio-gold">
        {props.reading.squares} {props.reading.squares === 1 ? 'quadrado' : 'quadrados'} ({metres()}m) · {faixa()}
      </p>
      <Button size="sm" variant="ghost" class="ml-auto" aria-label="Guardar a régua" onClick={() => props.onClose()}>
        <X aria-hidden="true" class="size-4" />
      </Button>
    </div>
  )
}

/** Os quatro tipos de área do livro (p225), na ordem em que a mesa os usa. */
const AREA_KINDS: { id: BoardAreaKind; label: string; medida: string }[] = [
  { id: 'esfera', label: 'Esfera', medida: 'raio' },
  { id: 'cone', label: 'Cone', medida: 'alcance' },
  { id: 'linha', label: 'Linha', medida: 'comprimento' },
  { id: 'quadrado', label: 'Quadrado', medida: 'lado' },
]

/**
 * A barra do gabarito de área (ALE-124, fatia 6b).
 *
 * Ela existe para responder, sem discussão, a pergunta que trava o turno do
 * conjurador: "se eu soltar aqui, quem pega?". Por isso a LISTA de quem está
 * dentro é o que ela tem de maior — o desenho no mapa mostra onde, mas é o nome
 * que resolve a dúvida, e é o nome que a mesa fala em voz alta.
 *
 * O tamanho é dito em quadrados E em metros: quadrado é a unidade do gabarito
 * (p236) e metro é a unidade em que a magia está escrita na ficha.
 */
export function AreaBar(props: {
  kind: BoardAreaKind
  onKind: (kind: BoardAreaKind) => void
  size: number
  onSize: (size: number) => void
  needsDirection: boolean
  hasOrigin: boolean
  hasDirection: boolean
  inside: readonly { id: string; label: string }[]
  onClose: () => void
}) {
  const medida = () => AREA_KINDS.find((k) => k.id === props.kind)?.medida ?? 'tamanho'
  const metros = () => (props.size * SQUARE_METRES).toFixed(1).replace('.', ',')
  const dica = () => {
    if (!props.hasOrigin) return 'Clique numa casa para pôr o gabarito.'
    if (props.needsDirection && !props.hasDirection) return 'Clique de novo para apontar.'
    return props.inside.length === 0 ? 'Ninguém dentro.' : ''
  }

  return (
    <div
      role="status"
      class="flex shrink-0 flex-wrap items-center gap-2 border-t border-grimorio-iron px-3 py-1.5"
    >
      <div class="flex items-center gap-0.5">
        <For each={AREA_KINDS}>
          {(area) => (
            <Button
              size="sm"
              variant={props.kind === area.id ? 'default' : 'ghost'}
              aria-pressed={props.kind === area.id}
              onClick={() => props.onKind(area.id)}
            >
              {area.label}
            </Button>
          )}
        </For>
      </div>

      <label class="text-[11px] text-muted-foreground" for="area-size">
        {medida()}
      </label>
      <NumberInput
        id="area-size"
        class="w-20"
        value={props.size}
        min={1}
        max={60}
        spinnerLabel={medida()}
        onChange={(valor) => props.onSize(valor)}
      />
      <span class="font-mono text-[11px] tabular-nums text-muted-foreground">{metros()}m</span>

      <Show
        when={props.inside.length > 0}
        fallback={<span class="text-[11px] text-muted-foreground">{dica()}</span>}
      >
        <p class="min-w-0 flex-1 truncate text-[11px] text-grimorio-gold">
          Pega {props.inside.length === 1 ? '1 peça' : `${props.inside.length} peças`}:{' '}
          {props.inside.map((peca) => peca.label).join(', ')}
        </p>
      </Show>

      <Button size="sm" variant="ghost" class="ml-auto" aria-label="Guardar o gabarito" onClick={() => props.onClose()}>
        <X aria-hidden="true" class="size-4" />
      </Button>
    </div>
  )
}
