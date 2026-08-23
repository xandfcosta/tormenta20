import { PanelLeftOpen } from 'lucide-solid'
import { type JSX, For, Show } from 'solid-js'
import type { JanelaDaFila } from '@/features/session-tracker/rail-geometry'
import { initials } from '@/shared/lib/initials'
import { cn } from '@/shared/lib/utils'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import { hpFillVar } from '@/shared/ui/vital-bar'

/**
 * A fila do combate como bloco do trilho esquerdo (ALE-198, refeita na ALE-211).
 *
 * A iniciativa era a espinha da tela e comia 7/12 dela mesmo fora de combate.
 * Ela virou consulta de um instante — "de quem é a vez", "quanto o ogro ainda
 * aguenta" —, e um instante não paga 420px de largura permanente. A lista
 * inteira, com os botões de dano, ordem e condição, vive na gaveta.
 *
 * A ALE-211 tirou a ROLAGEM daqui, e essa é a mudança que carrega a regra: em
 * vez de uma lista que rola, o bloco mostra uma JANELA centrada em quem está na
 * vez — quem já agiu acima, escurecido, e quem ainda vai abaixo. Rolar obrigava
 * o mestre a procurar a vez no meio de uma lista que se move sozinha; a janela
 * responde "onde estamos" sem gesto nenhum.
 *
 * Quantos vizinhos cabem é conta MEDIDA e não CSS (`rail-geometry.ts`), porque
 * a decisão muda o que a tela AFIRMA: com um número errado, o trilho mente
 * sobre quem já jogou. Quem mede é o `SessionRail`, que é dono da altura.
 *
 * Cada item ABRE a ficha, e é o mesmo gesto da peça no tabuleiro: a linha e a
 * peça são a mesma criatura, e o mestre não deveria ter dois caminhos com
 * resultados diferentes para o mesmo clique.
 *
 * @example <InitiativeRail entries={live().initiative} janela={janela()} … />
 */
export function InitiativeRail(props: {
  entries: readonly InitiativeEntry[]
  /** A linha na vez: o item ganha a marca dourada da cena. */
  activeEntryId?: string | null
  /**
   * O índice da vez, para saber quem JÁ AGIU. Vem separado do `activeEntryId`
   * porque "antes da vez" é pergunta de POSIÇÃO, e o id não a responde.
   */
  turnIndex: number
  /** A faixa de índices que cabe no bloco (`rail-geometry.ts`). */
  janela: JanelaDaFila
  onOpenCombatant: (entryId: string) => void
  onExpand: () => void
  /** Acende a peça correspondente no tabuleiro (ALE-189). */
  onHoverEntry?: (entryId: string | null) => void
  /**
   * O fim do ciclo da cena, ancorado no pé (ALE-210). Vem de fora porque o
   * trilho não sabe o que é uma cena — ele desenha a fila; quem compõe fila e
   * ciclo é a página.
   */
  footer?: JSX.Element
  class?: string
}) {
  /**
   * A janela chega como ÍNDICES e sai como linhas com o índice junto: quem
   * desenha precisa saber a posição de cada uma para dizer se ela já agiu, e
   * recortar o array perderia justamente essa informação.
   */
  const naJanela = () =>
    props.entries
      .slice(props.janela.inicio, props.janela.fim + 1)
      .map((linha, deslocamento) => ({ linha, indice: props.janela.inicio + deslocamento }))

  return (
    <nav
      aria-label="Fila do combate"
      class={cn(
        'grimorio-frame flex shrink-0 flex-col gap-1 bg-grimorio-panel p-1',
        props.class,
      )}
    >
      <button
        type="button"
        aria-label="Abrir a iniciativa"
        onClick={props.onExpand}
        class="flex shrink-0 items-center justify-center rounded-sm border border-grimorio-iron py-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      >
        <PanelLeftOpen aria-hidden="true" class="size-4" />
      </button>

      {/* RECORTA, não rola (ALE-211). A lista desenha só a janela que a
          geometria disse caber, e o `overflow-hidden` é a rede: se a constante
          de altura do item ficar defasada do CSS, um vizinho é cortado em vez
          de a fila transbordar o bloco. */}
      <ul class="flex min-h-0 flex-1 flex-col gap-1 overflow-hidden">
        <For each={naJanela()}>
          {(entry) => (
            <li>
              <RailEntry
                entry={entry.linha}
                active={entry.indice === props.turnIndex}
                jaAgiu={props.turnIndex >= 0 && entry.indice < props.turnIndex}
                onOpen={() => props.onOpenCombatant(entry.linha.id)}
                onHover={props.onHoverEntry}
              />
            </li>
          )}
        </For>
      </ul>

      {/* Ancorado no PÉ e fora do `ul` que rola: com doze combatentes o
          encerrar sairia da tela junto com a lista, que é o mesmo defeito que a
          ALE-131 consertou na coluna e que o botão de abrir resolve no topo. */}
      <Show when={props.footer}>
        <div class="flex shrink-0 flex-col gap-1 border-t border-grimorio-iron pt-1">
          {props.footer}
        </div>
      </Show>
    </nav>
  )
}

/**
 * Um combatente em 56px: iniciais, o filete de vida e a marca da vez.
 *
 * O nome inteiro vive no `aria-label` com os PV junto, porque duas letras não
 * são um nome — quem usa leitor de tela ouve "Arcanista Erudito, PV 42 de 42" e
 * não "AE". O `title` faz o mesmo pelo ponteiro.
 */
function RailEntry(props: {
  entry: InitiativeEntry
  active: boolean
  /** Já jogou nesta rodada: o item escurece (ALE-211). */
  jaAgiu: boolean
  onOpen: () => void
  onHover?: (entryId: string | null) => void
}) {
  const hp = () => props.entry.hpCurrent
  const hpMax = () => props.entry.hpMax
  /**
   * "já agiu" entra no NOME e não só na tinta. Escurecer é o sinal para quem
   * enxerga; quem usa leitor de tela ouviria dois itens idênticos e não saberia
   * que um deles já jogou — e essa é a informação que o bloco existe para dar.
   */
  const rotulo = () => {
    const vida = hp() !== undefined && hpMax() ? ` — PV ${hp()} de ${hpMax()}` : ''
    const quando = props.active ? ' — na vez' : props.jaAgiu ? ' — já agiu' : ''
    return `Abrir ${props.entry.label}${vida}${quando}`
  }

  return (
    <button
      type="button"
      aria-label={rotulo()}
      title={props.entry.label}
      aria-current={props.active ? 'true' : undefined}
      onClick={props.onOpen}
      onPointerEnter={() => props.onHover?.(props.entry.id)}
      onPointerLeave={() => props.onHover?.(null)}
      class={cn(
        'flex w-full flex-col items-center gap-1 rounded-sm border px-1 py-1.5 transition-colors',
        props.active
          ? 'border-grimorio-gold bg-accent text-grimorio-gold'
          : 'border-grimorio-iron text-muted-foreground hover:bg-accent hover:text-foreground',
        // Quem já jogou APAGA, e volta ao normal no hover: escurecido é estado,
        // não desabilitado — o mestre continua abrindo a ficha de quem já agiu,
        // e um item que não reage ao ponteiro diria o contrário.
        props.jaAgiu && !props.active && 'opacity-45 hover:opacity-100',
      )}
    >
      {/* Sans e não Cinzel: duas maiúsculas a 12px estão abaixo do piso de
          leitura de 14px da serifada de display (ALE-173). */}
      <span class="text-xs font-semibold tabular-nums">{initials(props.entry.label)}</span>
      <Show when={hpMax()}>{(max) => <RailVital current={hp() ?? 0} max={max()} />}</Show>
    </button>
  )
}

/**
 * O filete de vida do trilho.
 *
 * Não é a `VitalBar`: aquela desenha rótulo, barra e "42/42" numa linha, e nada
 * disso cabe em 56px. O que ela empresta é o que importa — `hpFillVar`, a
 * mesma função que decide a COR por faixa, para que "quase morto" tenha o
 * mesmo vermelho aqui e na gaveta. Um segundo limiar escrito à mão seria duas
 * verdades sobre o mesmo combatente.
 *
 * Sem `role="progressbar"`: o número já vai no nome acessível do botão, e um
 * medidor aninhado num botão faria o leitor de tela anunciar a vida duas vezes.
 */
function RailVital(props: { current: number; max: number }) {
  const percent = () => Math.max(0, Math.min(100, (props.current / props.max) * 100))

  return (
    <div aria-hidden="true" class="h-1 w-full overflow-hidden rounded-full bg-muted">
      <div
        class="h-full rounded-full transition-[width]"
        style={{ width: `${percent()}%`, background: `var(${hpFillVar(percent())})` }}
      />
    </div>
  )
}
