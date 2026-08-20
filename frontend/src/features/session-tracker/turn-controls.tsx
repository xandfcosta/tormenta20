import { ChevronLeft, ChevronRight } from 'lucide-solid'
import { Show } from 'solid-js'
import type { SessionRuntimeState } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'
import { nextTurnTarget } from './tracker-rules'

/**
 * Avançar o turno — o controle mais clicado do app (ALE-184).
 *
 * Era um `▶` de 32px no cabeçalho da coluna, do mesmo tamanho dos botões de
 * descanso, que se usam uma vez por sessão. Três coisas mudaram e cada uma tem
 * um porquê:
 *
 * 1. **Alvo grande** (44px, o mínimo de toque) — ele é clicado uma vez por
 *    combatente, por rodada, a noite inteira.
 * 2. **Diz o NOME de quem entra**: com `▶` o mestre contava a lista para saber
 *    para onde ia. Agora ele LÊ.
 * 3. **Fica no PÉ da coluna**, ancorado: entre o cabeçalho e a linha da vez
 *    havia a lista inteira rolando.
 *
 * O par com o `‹` continua sendo par (ALE-132): voltar e avançar andam na
 * mesma ordem do combate. O que muda é o peso — só o avanço é preenchido e
 * largo, e o `‹` fica do tamanho de um ícone, porque desfazer turno é raro.
 *
 * @example <TurnAdvance state={rt.state()} connected onPrevious={…} onNext={…} />
 */
export function TurnAdvance(props: {
  state: SessionRuntimeState
  connected: boolean
  onPrevious: () => void
  onNext: () => void
  /** Sem o `‹` — a faixa fixa abaixo de 1024, onde só cabe o avanço. */
  onlyNext?: boolean
  class?: string
}) {
  const alvo = () => nextTurnTarget(props.state.initiative, props.state.turnIndex)

  return (
    // `min-w-0` e NÃO `shrink-0` (ALE-184): o `shrink-0` veio do par de ícones,
    // onde o conteúdo media 32px fixos. Agora o conteúdo é um rótulo de tamanho
    // VARIÁVEL — o nome do próximo combatente —, e um invólucro que não encolhe
    // fica com a largura de max-content: "Começar: Zumbi Putrefato Ancião do
    // Pântano" mediu 460px numa janela de 390 e pintou para fora da faixa.
    // Encolhendo, o rótulo trunca, que é para isso que ele tem `truncate`.
    <div class={cn('flex min-w-0 items-center gap-1', props.class)}>
      <Show when={!props.onlyNext}>
        <Button
          variant="outline"
          class="size-11 shrink-0"
          disabled={!props.connected}
          aria-label="Turno anterior"
          title="Turno anterior"
          onClick={() => props.onPrevious()}
        >
          <ChevronLeft aria-hidden="true" class="size-4" />
        </Button>
      </Show>
      {/* `min-w-0` + `truncate` no rótulo: "Zumbi Putrefato Ancião" cabe na
          coluna de 22rem empurrando a seta para fora sem isto. */}
      <Button
        class="h-11 min-w-0 flex-1 justify-between gap-2 px-3 text-sm"
        disabled={!props.connected || alvo().entry === null}
        aria-label={alvo().label}
        onClick={() => props.onNext()}
      >
        <span class="truncate">{alvo().label}</span>
        <ChevronRight aria-hidden="true" class="size-4 shrink-0" />
      </Button>
    </div>
  )
}

/**
 * "Rodada 2 · Turno 6/7".
 *
 * A posição na rodada sai do estado que já existia. O TOTAL contado pelo
 * servidor saiu do desenho na ALE-184: "Turno 9/9 · 6 no total" com nove linhas
 * na lista lia como contradição — os dois números respondem perguntas
 * diferentes (onde estou na volta × quantos turnos já se jogaram, que difere
 * quando alguém entra ou morre no meio) e ninguém decifra isso no combate. O
 * campo continua no estado; o que saiu foi a tinta.
 *
 * Antes do primeiro turno não há o que contar, então só a rodada aparece —
 * "Turno 0/7" diria que a rodada começou quando ela não começou.
 */
export function TurnCounter(props: { state: SessionRuntimeState; class?: string }) {
  const emCombate = () => props.state.turnIndex >= 0 && props.state.initiative.length > 0

  return (
    <span class={cn('font-mono text-sm tabular-nums text-muted-foreground', props.class)}>
      Rodada {props.state.round}
      <Show when={emCombate()}>
        {' · '}Turno {props.state.turnIndex + 1}/{props.state.initiative.length}
      </Show>
    </span>
  )
}
