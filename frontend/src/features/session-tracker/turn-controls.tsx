import { ChevronLeft, ChevronRight } from 'lucide-solid'
import { Show } from 'solid-js'
import type { SessionRuntimeState } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'

/**
 * Voltar e avançar o turno, como PAR (ALE-132).
 *
 * Eram um `‹` fantasma ao lado de um botão vermelho com texto: a mesma família
 * — os dois andam na ordem do combate — desenhada como coisas de naturezas
 * diferentes. Agora são dois ícones do mesmo tamanho num grupo com moldura
 * comum, e a hierarquia continua legível porque só o de avançar é preenchido.
 *
 * O tooltip carrega o que o texto dizia: o dono pediu ícone para economizar
 * espaço, e sem `title` o botão mais clicado da mesa viraria adivinhação.
 *
 * @example <TurnControls rt={rt} class="hidden lg:flex" />
 */
export function TurnControls(props: {
  connected: boolean
  onPrevious: () => void
  onNext: () => void
  /** Só o avanço — a faixa fixa abaixo de 1024, onde a iniciativa some da tela. */
  onlyNext?: boolean
  class?: string
}) {
  return (
    <div
      class={cn(
        'flex shrink-0 items-center gap-px rounded-md border border-grimorio-iron p-px',
        props.class,
      )}
    >
      <Show when={!props.onlyNext}>
        <Button
          size="sm"
          variant="ghost"
          class="size-8 rounded-[5px]"
          disabled={!props.connected}
          aria-label="Turno anterior"
          title="Turno anterior"
          onClick={() => props.onPrevious()}
        >
          <ChevronLeft aria-hidden="true" class="size-4" />
        </Button>
      </Show>
      <Button
        size="sm"
        class="size-8 rounded-[5px]"
        disabled={!props.connected}
        aria-label="Próximo turno"
        title="Próximo turno"
        onClick={() => props.onNext()}
      >
        <ChevronRight aria-hidden="true" class="size-4" />
      </Button>
    </div>
  )
}

/**
 * "Rodada 2 · Turno 6/7 · 14 no total" (ALE-142).
 *
 * A posição na rodada sai do estado que já existia; o total vem CONTADO do
 * servidor, porque derivar de rodada × tamanho da lista mente assim que alguém
 * entra ou morre no meio do combate.
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
      <Show when={(props.state.turnsTaken ?? 0) > 0}>
        {' · '}
        {props.state.turnsTaken} no total
      </Show>
    </span>
  )
}
