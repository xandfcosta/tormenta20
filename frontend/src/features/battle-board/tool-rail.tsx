import { Eraser, Brush, MapPin, Radar, Ruler as RulerIcon, MousePointer2 } from 'lucide-solid'
import { For, Show } from 'solid-js'
import { Button } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'

/**
 * O TRILHO DE FERRAMENTAS do tabuleiro (ALE-203, fatia 1).
 *
 * Antes eram onze controles numa fileira horizontal que já tinha enrolado uma
 * vez — a ALE-178 registra o ✕ ficando inalcançável a 390px. Mas o problema
 * maior não era largura: era o clique num quadrado significar coisas diferentes
 * conforme um modo escondido em botões que não pareciam modo.
 *
 * Aqui a ferramenta ATIVA é o estado explícito da cena. Uma por vez, marcada, e
 * o que o clique vai fazer é legível ANTES do clique.
 *
 * ORIENTAÇÃO: vertical de `lg` para cima, horizontal abaixo. É a MESMA árvore
 * trocando classe, e não dois `Show` — dois ramos com a mesma lista divergem com
 * o tempo, e o guia da casa proíbe exatamente isso. Abaixo de `lg` a cena já tem
 * uma fileira de trilhos acima do mapa (ALE-198), e a decisão do dono foi que
 * este encoste no mapa em vez de virar gaveta: trocar de ferramenta é o gesto
 * mais frequente da cena, e uma gaveta poria um toque a mais em cada troca —
 * além de tirar da vista qual está ativa, que é o que esta issue existe para
 * consertar.
 */
export type Ferramenta = 'peca' | 'regua' | 'gabarito' | 'marcador' | 'terreno' | 'borracha'

type Entrada = {
  id: Ferramenta
  rotulo: string
  Icone: typeof RulerIcon
  /** Ferramenta que muda a cena — o jogador não pinta terreno nem marca lugar. */
  soMestre?: boolean
}

/**
 * A ordem é a do uso, não a do alfabeto: pousar peça primeiro porque é o padrão
 * e o retorno de toda outra; medir e mirar em seguida, que são de TODO MUNDO
 * («dá para acertar daqui?» é pergunta de quem ataca); e as do mestre por
 * último, agrupadas.
 */
const FERRAMENTAS: readonly Entrada[] = [
  { id: 'peca', rotulo: 'Pousar a peça', Icone: MousePointer2 },
  { id: 'regua', rotulo: 'Régua', Icone: RulerIcon },
  { id: 'gabarito', rotulo: 'Gabarito de área', Icone: Radar },
  { id: 'marcador', rotulo: 'Marcar um lugar', Icone: MapPin, soMestre: true },
  // 'Terreno' e não 'Terreno difícil': o GLOSSARIO define `terreno` como "o
  // chão difícil que o pincel pinta", então o adjetivo seria uma segunda
  // palavra para o mesmo conceito. Escrevi 'Terreno difícil' primeiro e os
  // testes acusaram — a tabela fez o trabalho dela.
  { id: 'terreno', rotulo: 'Terreno', Icone: Brush, soMestre: true },
  { id: 'borracha', rotulo: 'Apagar terreno', Icone: Eraser, soMestre: true },
]

export function ToolRail(props: {
  ativa: Ferramenta
  isGm: boolean
  onEscolher: (ferramenta: Ferramenta) => void
  /** As AÇÕES do trilho — centralizar, tela cheia, o acervo. Vêm de fora porque
   *  cada uma tem dono próprio, e o trilho só as hospeda embaixo da régua. */
  children?: unknown
}) {
  return (
    <nav
      aria-label="Ferramentas do tabuleiro"
      class={cn(
        'flex shrink-0 gap-1 border-grimorio-iron bg-grimorio-panel/40 p-1',
        // Horizontal e rolável no pequeno; vertical e colada à esquerda no
        // grande. `overflow-x-auto` porque a fileira pode passar de 390px com o
        // mestre vendo as seis, e conteúdo que não alcança é o defeito da
        // ALE-178 outra vez.
        'flex-row overflow-x-auto border-b lg:flex-col lg:overflow-x-visible lg:border-b-0 lg:border-r',
      )}
    >
      <For each={FERRAMENTAS}>
        {(entrada) => (
          <Show when={props.isGm || !entrada.soMestre}>
            <Button
              size="sm"
              variant={props.ativa === entrada.id ? 'default' : 'ghost'}
              aria-pressed={props.ativa === entrada.id}
              aria-label={entrada.rotulo}
              title={entrada.rotulo}
              class="shrink-0"
              onClick={() => props.onEscolher(entrada.id)}
            >
              <entrada.Icone aria-hidden="true" class="size-4" />
            </Button>
          </Show>
        )}
      </For>
      {/* A régua separa FERRAMENTA de AÇÃO: acima, o que muda o que o clique
          faz; abaixo, o que acontece uma vez e acaba. Sem a divisa, centralizar
          a vista parece um modo que se liga. */}
      <div class="my-auto h-px w-4 shrink-0 self-center bg-grimorio-iron lg:h-4 lg:w-px" />
      {props.children as never}
    </nav>
  )
}
