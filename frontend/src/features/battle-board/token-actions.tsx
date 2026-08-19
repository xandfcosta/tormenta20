import { Copy, Eye, EyeOff, Pencil, Trash2, Undo2 } from 'lucide-solid'
import { For } from 'solid-js'
import { Show } from 'solid-js'
import type { BoardMarker, BoardToken } from '@/shared/realtime/realtime'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { TokenDialog } from './token-dialog'

/**
 * O que o mestre faz com a peça selecionada (ALE-178).
 *
 * O servidor já sabia fazer tudo isto desde a primeira fatia — `board-token-add`,
 * `remove` e `update` existem com porta de mestre e teste —, e nada disso tinha
 * botão. O caso mais caro era o esconder: a redação da emboscada
 * (`redactBoardForPlayers`) está implementada e testada no Go, e estava MORTA
 * porque a tela não tinha como ligá-la.
 *
 * Barra e não menu: são cinco ações de um clique, e um menu esconderia atrás de
 * dois cliques o que o mestre faz no meio do combate. Só o renomear/redimensionar
 * abre diálogo, porque digitar numa lista que se recria a cada broadcast perde o
 * foco a cada tecla (ALE-122).
 *
 * @example <TokenActions token={peça} onUpdate={…} onRemove={…} onUndo={…} />
 */
export function TokenActions(props: {
  token: BoardToken
  onUpdate: (patch: Partial<Omit<BoardToken, 'id'>>) => void
  onRemove: () => void
  /** Ausente = a peça não foi movida nesta sessão de tela, então não há para onde voltar. */
  onUndo?: () => void
  /** Ausente = duplicar não está disponível (o rascunho do lugar guarda a cena
   *  inteira de uma vez e numera na gravação, não por mensagem). */
  onDuplicate?: () => void
}) {
  return (
    <div class="flex flex-wrap items-center gap-1 border-t border-grimorio-iron px-3 py-1.5">
      <span class="mr-1 min-w-0 truncate text-[11px] text-grimorio-gold">{props.token.label}</span>

      <TokenDialog
        token={props.token}
        onSave={(patch) => props.onUpdate(patch)}
        trigger={(open) => (
          <Button size="sm" variant="ghost" aria-label={`Editar ${props.token.label}`} onClick={open}>
            <Pencil aria-hidden="true" class="size-4" />
          </Button>
        )}
      />

      {/* Esconder é o que liga a emboscada: a peça some INTEIRA da cópia do
          jogador, e não vira uma peça anônima — a existência dela é a
          informação (ALE-124). O rótulo diz o que vai acontecer, não o estado. */}
      <Button
        size="sm"
        variant="ghost"
        aria-label={props.token.hidden ? `Mostrar ${props.token.label}` : `Esconder ${props.token.label}`}
        onClick={() => props.onUpdate({ hidden: !props.token.hidden })}
      >
        <Show when={props.token.hidden} fallback={<Eye aria-hidden="true" class="size-4" />}>
          <EyeOff aria-hidden="true" class="size-4 text-grimorio-gold" />
        </Show>
      </Button>

      {/* "Mais um zumbi" é a operação mais repetida ao montar encontro, e ela
          custava abrir a forma e digitar o nome de uma criatura idêntica à que
          já está ali ao lado. Quem NUMERA é o servidor (ALE-192). */}
      <Show when={props.onDuplicate}>
        {(duplicar) => (
          <Button
            size="sm"
            variant="ghost"
            aria-label={`Duplicar ${props.token.label}`}
            onClick={() => duplicar()()}
          >
            <Copy aria-hidden="true" class="size-4" />
          </Button>
        )}
      </Show>

      <Show when={props.onUndo}>
        {(undo) => (
          <Button
            size="sm"
            variant="ghost"
            aria-label={`Voltar ${props.token.label} para onde estava`}
            onClick={() => undo()()}
          >
            <Undo2 aria-hidden="true" class="size-4" />
          </Button>
        )}
      </Show>

      <ConfirmDialog
        title={`Tirar ${props.token.label} do tabuleiro?`}
        description="A peça sai da cena. A linha da iniciativa e os PV continuam como estão."
        confirmLabel="Tirar"
        destructive
        onConfirm={props.onRemove}
        trigger={(open) => (
          <Button size="sm" variant="ghost" aria-label={`Tirar ${props.token.label}`} onClick={open}>
            <Trash2 aria-hidden="true" class="size-4" />
          </Button>
        )}
      />

      <Show when={props.token.hidden}>
        <span class="ml-auto text-[10px] uppercase tracking-widest text-grimorio-gold">
          escondida dos jogadores
        </span>
      </Show>
    </div>
  )
}

/** As cores que o servidor aceita, com o nome que a mesa usa. */
const MARKER_COLORS: { id: BoardMarker['color']; label: string; swatch: string }[] = [
  { id: 'ouro', label: 'Ouro', swatch: 'bg-grimorio-gold' },
  { id: 'carmim', label: 'Carmim', swatch: 'bg-[color:var(--primary)]' },
  { id: 'azul', label: 'Azul', swatch: 'bg-[#3f6fb0]' },
  { id: 'verde', label: 'Verde', swatch: 'bg-[#3f8f52]' },
]

/**
 * O que o mestre faz com o LUGAR marcado (ALE-195).
 *
 * O verbo que importa é REVELAR: o marcador nasce escondido porque marcar a
 * armadilha antes de a mesa chegar nela é o motivo de ele existir, e mostrar é
 * o gesto do momento em que alguém pisa.
 *
 * A cor é escolha de UM clique e não um seletor: são quatro, e o mestre está no
 * meio da cena.
 */
export function MarkerActions(props: {
  marker: BoardMarker
  onUpdate: (patch: Partial<Omit<BoardMarker, 'id' | 'x' | 'y'>>) => void
  onRemove: () => void
}) {
  return (
    <div class="flex flex-wrap items-center gap-1 border-t border-grimorio-iron px-3 py-1.5">
      <span class="mr-1 text-[11px] text-grimorio-gold">
        Marcador {props.marker.text || '—'}
      </span>

      <For each={MARKER_COLORS}>
        {(cor) => (
          <Button
            size="sm"
            variant="ghost"
            aria-label={`Cor ${cor.label}`}
            aria-pressed={props.marker.color === cor.id}
            onClick={() => props.onUpdate({ color: cor.id })}
          >
            <span
              aria-hidden="true"
              class={cn(
                'size-3.5 rounded-full',
                cor.swatch,
                props.marker.color === cor.id && 'ring-2 ring-white',
              )}
            />
          </Button>
        )}
      </For>

      <Button
        size="sm"
        variant="ghost"
        aria-label={props.marker.hidden ? `Mostrar marcador ${props.marker.text}` : `Esconder marcador ${props.marker.text}`}
        onClick={() => props.onUpdate({ hidden: !props.marker.hidden })}
      >
        <Show when={props.marker.hidden} fallback={<Eye aria-hidden="true" class="size-4" />}>
          <EyeOff aria-hidden="true" class="size-4 text-grimorio-gold" />
        </Show>
      </Button>

      <Button
        size="sm"
        variant="ghost"
        aria-label={`Apagar marcador ${props.marker.text}`}
        onClick={() => props.onRemove()}
      >
        <Trash2 aria-hidden="true" class="size-4" />
      </Button>
    </div>
  )
}

/**
 * A próxima letra livre para um marcador novo (ALE-195).
 *
 * Quem está apontando a armadilha no meio da cena não quer digitar, e "A", "B",
 * "C" é como a mesa fala de lugares num mapa. Esgotadas as letras, cai em "??"
 * — que é feio de propósito: com 26 marcadores na tela, o rótulo já não é o que
 * distingue nada.
 */
export function nextMarkerText(markers: readonly BoardMarker[]): string {
  const usadas = new Set(markers.map((m) => m.text))
  for (let i = 0; i < 26; i++) {
    const letra = String.fromCharCode(65 + i)
    if (!usadas.has(letra)) return letra
  }
  return '??'
}
