import { Swords } from 'lucide-solid'
import { Show } from 'solid-js'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { FieldLabel, SectionLabel } from '@/shared/ui/section-label'

/** Whose turn it is, from the player's point of view. */
export type LiveTurnState =
  | { kind: 'mine' }
  | { kind: 'other'; label: string }
  | { kind: 'idle' }

/**
 * De quem é a vez, do ponto de vista do jogador.
 *
 * Era privada da cena do jogador. Subiu para cá na ALE-201 porque o CABEÇALHO
 * passou a mostrar a vez, e a página precisa da mesma resposta que a cena —
 * duas cópias da mesma regra é como este repositório já mostrou 52/95 e 57/95
 * do mesmo combatente em duas telas (ALE-122).
 */
export function playerTurnState(
  rt: SessionRealtime,
  myCharacterIds: ReadonlySet<number>,
): LiveTurnState {
  const state = rt.state()
  const active = state.turnIndex >= 0 ? state.initiative[state.turnIndex] : undefined
  if (!active) return { kind: 'idle' }
  if (active.characterId !== undefined && myCharacterIds.has(active.characterId)) {
    return { kind: 'mine' }
  }
  return { kind: 'other', label: active.label }
}

/** `true` quando é a vez do jogador — o que faz o cabeçalho ACENDER. */
export function eAMinhaVez(turn: LiveTurnState): boolean {
  return turn.kind === 'mine'
}

/**
 * O que a sessão AO VIVO diz sobre si mesma, para caber no cabeçalho da cena
 * (ALE-201).
 *
 * Era uma FAIXA própria — o "Modo Jogo" da ALE-30, uma banda larga logo abaixo
 * do cabeçalho do app dizendo que aquilo era sessão e não edição de ficha. A
 * ALE-201 juntou as duas: a banda gastava ~38px de uma cena que no celular
 * deitado tem 390 de altura, e a ALE-146 já tinha medido que cada faixa a menos
 * é uma linha de combatente a mais.
 *
 * O que NÃO se perdeu na mudança, e era o risco: a faixa **acendia** quando
 * chegava a vez do jogador (ALE-200), e isso é sinal e não enfeite — dourado é
 * o que diz "a vez" na iniciativa e no tabuleiro. O aceso subiu junto e agora
 * pinta o cabeçalho INTEIRO, que é mais visível ainda com o telefone na mão.
 * Quem decide o aceso é o `MatchShell`, porque quem pinta é o cabeçalho; este
 * componente só diz o que a sessão está fazendo.
 *
 * E ele NÃO repete o número da sessão. A faixa antiga dizia "Ao vivo · Sessão
 * 4" porque vivia longe do título; no cabeçalho ela fica ao lado dele, que já
 * diz "Sessão 4 · <campanha>". Repetir custava a largura que fazia o texto ser
 * pintado para fora do pai a 390px — achado pelo guarda de transbordo, não por
 * inspeção.
 */
export function LiveSessionStatus(props: {
  round: number
  turn: LiveTurnState
}) {
  return (
    <span class="flex min-w-0 items-center gap-2">
      <LivePip />
      <SectionLabel as="span" tom="inherit" class="text-xs font-semibold text-[color:var(--hp-full)]">
        Ao vivo
      </SectionLabel>
      {/* A rodada só aparece quando alguém está na vez (ALE-213). Ela sobrevive
          ao fim de um combate — remover o último combatente devolve o turno a
          −1 mas não zera a rodada —, e a faixa anunciava "Rodada 1 · Aguardando
          iniciativa" com a fila vazia: um número de um combate que já acabou,
          ao lado da frase que diz que nenhum começou. `idle` é exatamente "não
          há vez de ninguém". */}
      <Show when={props.turn.kind !== 'idle'}>
        <span class="shrink-0 text-sm text-muted-foreground">· Rodada {props.round}</span>
      </Show>

      <Show
        when={eAMinhaVez(props.turn)}
        fallback={
          <span class="truncate text-sm text-muted-foreground">
            {props.turn.kind === 'other' ? `· Vez de ${props.turn.label}` : '· Aguardando iniciativa'}
          </span>
        }
      >
        <FieldLabel
          tom="inherit"
          class="text-xs flex shrink-0 animate-pulse items-center gap-1 rounded-none bg-primary px-2 py-0.5 font-bold text-primary-foreground"
        >
          <Swords aria-hidden="true" class="size-3.5" /> Sua vez
        </FieldLabel>
      </Show>
    </span>
  )
}

/** Pulsing "live" dot (ping halo + solid core), in the full-HP green. */
function LivePip() {
  return (
    <span class="relative flex size-2.5 shrink-0">
      <span class="absolute inline-flex size-full animate-ping rounded-full bg-[color:var(--hp-full)] opacity-75" />
      <span class="relative inline-flex size-2.5 rounded-full bg-[color:var(--hp-full)]" />
    </span>
  )
}
