import { BookMarked, NotebookPen, Skull, Swords, Users } from 'lucide-solid'
import type { LucideProps } from 'lucide-solid'
import { type Component, type JSX, For } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/** As consultas do mestre DENTRO da sessão, na ordem do trilho. Cada uma é um
 *  overlay, menos as notas, que abrem coluna (ALE-198). */
export const SESSION_TOOLS = [
  // O elenco vem PRIMEIRO: é quem existe na crônica, e as outras quatro são
  // material de fora dela (o livro, os catálogos) ou o caderno da noite. A
  // ALE-211 depois o recolhe no trilho da ESQUERDA, junto da fila.
  { slug: 'elenco', label: 'Elenco', hint: 'Os jogadores e os NPCs desta crônica.' },
  { slug: 'bestiario', label: 'Bestiário', hint: 'Pôr uma criatura do livro na iniciativa.' },
  { slug: 'encontros', label: 'Encontros', hint: 'Combinar criaturas e mandar tudo de uma vez.' },
  { slug: 'catalogos', label: 'Catálogos', hint: 'Condições, magias, poderes e itens.' },
  { slug: 'notas', label: 'Notas', hint: 'O que aconteceu nesta sessão, ao lado do mapa.' },
] as const

export type SessionTool = (typeof SESSION_TOOLS)[number]['slug']

/** O ícone mora aqui e não no registro: o registro é uma regra, e um teste que
 *  o importa não deveria arrastar um pacote de ícones junto. */
const TOOL_ICON: Record<SessionTool, Component<LucideProps>> = {
  elenco: Users,
  bestiario: Skull,
  encontros: Swords,
  catalogos: BookMarked,
  notas: NotebookPen,
}

/**
 * O trilho das consultas do mestre: a borda direita da cena da sessão
 * (ALE-198).
 *
 * Ele existe porque a decisão de produto foi inverter a hierarquia — o
 * tabuleiro é o que fica, e bestiário, encontros e catálogo são coisas que se
 * consultam e se fecham. Consulta com casa fixa não precisa de aba, precisa de
 * um lugar previsível de onde ser chamada; o trilho é esse lugar.
 *
 * UMA lista nas duas larguras, trocando por classe: coluna à direita do mapa a
 * partir de 1024, fileira acima dele abaixo disso. Duas árvores num `Show` por
 * largura é como um trilho e o gêmeo de telefone divergem.
 *
 * Só ícone, e por isso cada botão carrega `aria-label` e `title`: em 56px um
 * rótulo escrito só caberia truncado, e "Best…" não é um nome. O estado ligado
 * é `aria-pressed` e não `role="checkbox"` — é o padrão da casa para alternador
 * rico, e o que o biome aceita.
 *
 * @example <GmToolRail isOpen={(t) => t === aberta()} onPick={alternar} />
 */
export function GmToolRail(props: {
  /** Se a consulta está aberta agora. Predicado e não um valor só porque as
   *  notas convivem com um overlay: elas não cobrem nada, abrem coluna. */
  isOpen: (tool: SessionTool) => boolean
  onPick: (tool: SessionTool) => void
  /** O que abre a fila do combate onde o trilho da esquerda não cabe. */
  leading?: JSX.Element
  class?: string
}) {
  return (
    <nav
      aria-label="Consultas do mestre"
      class={cn(
        'grimorio-frame flex shrink-0 gap-1 overflow-x-auto bg-grimorio-panel p-1',
        'lg:w-14 lg:flex-col lg:overflow-x-visible',
        props.class,
      )}
    >
      {props.leading}
      <For each={SESSION_TOOLS}>
        {(tool) => {
          const Icon = TOOL_ICON[tool.slug]
          const aberto = () => props.isOpen(tool.slug)
          return (
            <button
              type="button"
              aria-label={tool.label}
              aria-pressed={aberto()}
              title={tool.hint}
              onClick={() => props.onPick(tool.slug)}
              class={cn(
                'flex shrink-0 items-center justify-center rounded-sm border px-2.5 py-2.5 transition-colors lg:px-0',
                aberto()
                  ? 'border-grimorio-gold bg-accent text-grimorio-gold'
                  : 'border-grimorio-iron text-muted-foreground hover:bg-accent hover:text-foreground',
              )}
            >
              <Icon aria-hidden="true" class="size-4" />
            </button>
          )
        }}
      </For>
    </nav>
  )
}
