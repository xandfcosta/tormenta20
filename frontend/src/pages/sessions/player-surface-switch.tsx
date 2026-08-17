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
    <div class="flex shrink-0 gap-1" role="group" aria-label="O que ver na sessão">
      <For each={SURFACES}>
        {(surface) => (
          <button
            type="button"
            aria-pressed={props.surface === surface.value}
            class={cn(
              'flex min-w-0 flex-1 items-center justify-center gap-1.5 rounded-sm border px-2 py-1.5 text-xs uppercase tracking-wide transition-colors',
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
