import { Copy, Eye, EyeOff, Pencil, Trash2, Undo2 } from 'lucide-solid'
import { Show } from 'solid-js'
import type { BoardToken } from '@/shared/realtime/realtime'
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
