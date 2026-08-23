import { Show } from 'solid-js'
import type { LoadBreakdown } from '@/shared/lib/computed-sheet-v2'
import { cn } from '@/shared/lib/utils'
import { signed } from './signed'

/**
 * A barra de carga e o que acontece quando ela estoura. Todo número aqui vem do
 * motor (`sheet().carga`, p141) — inclusive os DOIS da penalidade: a mochila
 * anuncia exatamente o que a ficha já aplicou no deslocamento e nas perícias, e
 * não uma frase decorada que pode divergir dela.
 *
 * @example <CargaMeter carga={sheet().carga} />
 */
export function CargaMeter(props: { carga: LoadBreakdown }) {
  const percent = () =>
    props.carga.limit > 0 ? Math.min(100, (props.carga.used / props.carga.limit) * 100) : 0

  return (
    <div class="space-y-1">
      <div
        role="progressbar"
        aria-label="Espaços de carga"
        aria-valuenow={props.carga.used}
        aria-valuemin={0}
        aria-valuemax={props.carga.limit}
        class="h-2 overflow-hidden rounded-full border border-grimorio-iron bg-muted"
      >
        <div
          // `--hp-critical` e não `--hp-hurt`: o âmbar do matiz 70 está a um fio
          // do dourado do preenchimento, e uma barra ACIMA do limite pintada com
          // ele lê como cheia em vez de estourada.
          class={cn(
            'h-full transition-all',
            props.carga.overloaded ? 'bg-[var(--hp-critical)]' : 'bg-grimorio-gold',
          )}
          style={{ width: `${percent()}%` }}
        />
      </div>
      <Show when={props.carga.overloaded}>
        {/* A BARRA acima continua `--hp-critical`; este aviso é TEXTO, e em
            10px ele precisa da tinta da casa — o token da barra rende 3,59:1
            sobre o painel (ALE-240). */}
        <p class="text-3xs font-semibold text-grimorio-crimson-bright sm:text-2xs">
          Sobrecarregado (p141): {signed(props.carga.armorPenalty)} em Acrobacia, Furtividade e
          Ladinagem · {signed(props.carga.displacementPenalty)}m de deslocamento
          <Show when={props.carga.overMax}>
            {' '}
            · acima de {props.carga.max} espaços o livro diz que não dá para carregar
          </Show>
        </p>
      </Show>
    </div>
  )
}
