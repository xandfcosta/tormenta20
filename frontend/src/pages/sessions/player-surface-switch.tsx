import { LayoutGrid, ScrollText, Users } from 'lucide-solid'
import { For } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/** As três superfícies da sessão para o jogador. */
export type PlayerSurface = 'ficha' | 'mesa' | 'tabuleiro'

const SURFACES: { value: PlayerSurface; label: string; icon: typeof Users }[] = [
  { value: 'ficha', label: 'Minha ficha', icon: ScrollText },
  { value: 'mesa', label: 'Mesa', icon: Users },
  { value: 'tabuleiro', label: 'Tabuleiro', icon: LayoutGrid },
]

/**
 * O seletor de superfície do jogador (ALE-129).
 *
 * Dois níveis, não um: aqui se escolhe A SUPERFÍCIE (minha ficha, a mesa, o
 * tabuleiro), e dentro da ficha continuam as seções dela. Uma fileira única com
 * as seis abas da ficha mais as da sessão misturaria "partes de mim" com
 * "partes da mesa" e, no telefone, oito abas numa linha viram ilegíveis.
 *
 * Fica ANCORADO no topo, sempre visível: a premissa do app é tela cheia e
 * ninguém rolando a página atrás de menu.
 *
 * @example <PlayerSurfaceSwitch surface={surface()} onSurface={setSurface} />
 */
export function PlayerSurfaceSwitch(props: {
  surface: PlayerSurface
  onSurface: (surface: PlayerSurface) => void
}) {
  return (
    // `flex-1` e não `shrink-0`: com `shrink-0` o grupo tomava a largura do
    // CONTEÚDO, e como cada aba é `flex-1 min-w-0 truncate`, esse mínimo é
    // quase zero — as abas saíam "MINHA …" e "TABULE…" numa tela de 1920 com
    // 85% da linha vazia ao lado (ALE-168). Ocupando a linha, os rótulos cabem
    // onde há espaço e continuam truncando só onde falta.
    <div class="flex min-w-0 flex-1 gap-1" role="group" aria-label="O que ver na sessão">
      <For each={SURFACES}>
        {(surface) => (
          <button
            type="button"
            aria-pressed={props.surface === surface.value}
            class={cn(
              'flex min-w-0 flex-1 items-center justify-center gap-1.5 rounded-none border px-2 py-1.5 text-xs uppercase tracking-wide transition-colors',
              props.surface === surface.value
                ? 'border-grimorio-gold bg-[color:var(--primary)]/15 text-grimorio-gold'
                : 'border-grimorio-iron text-muted-foreground hover:text-foreground',
            )}
            onClick={() => props.onSurface(surface.value)}
          >
            <surface.icon aria-hidden="true" class="size-4 shrink-0" />
            <span class="truncate">{surface.label}</span>
          </button>
        )}
      </For>
    </div>
  )
}
