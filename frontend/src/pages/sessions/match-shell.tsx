import { Link } from '@tanstack/solid-router'
import { LogOut, Volume2, VolumeX } from 'lucide-solid'
import { type JSX, Show, createSignal } from 'solid-js'
import { SceneContainerProvider } from '@/shared/lib/scene-container'
import { Button, buttonVariants } from '@/shared/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { cn } from '@/shared/lib/utils'

/**
 * Full-screen frame for a live session ("match mode"): UMA faixa com a
 * identidade da sessão, o que ela está fazendo agora, a presença e a saída,
 * sobre um corpo rolável. As telas de cada papel entram como filhos.
 *
 * ERAM DUAS FAIXAS (ALE-201): esta, com o título, e logo abaixo a do "Modo
 * Jogo" dizendo rodada e de quem era a vez. Duas bandas empilhadas falando do
 * mesmo assunto, e a de baixo custava ~38px de uma cena que no celular deitado
 * tem 390 de altura — a ALE-146 já tinha medido que cada faixa a menos é uma
 * linha de combatente a mais.
 *
 * O que a junção NÃO podia perder é o ACESO: a faixa de baixo brilhava quando
 * chegava a vez do jogador (ALE-200), e dourado é o que diz "a vez" na
 * iniciativa e no tabuleiro. Ele subiu junto e agora pinta a faixa inteira.
 */
export function MatchShell(props: {
  campaignId: number
  title: string
  /** O que a sessão está fazendo agora — sobe da faixa que sumiu (ALE-201). */
  live?: JSX.Element
  /** `true` acende a faixa: é a vez de quem está olhando (ALE-200). */
  minhaVez?: boolean
  /** Right-hand slot — presence chips today. */
  bar?: JSX.Element
  /** O som se liga e desliga AQUI, e não só no Hub: mudar de ideia sobre áudio
   *  no meio de uma sessão ao vivo não pode custar sair da mesa (ALE-165). */
  sfxEnabled: boolean
  onToggleSfx: () => void
  children: JSX.Element
}) {
  // Publicado para os overlays (Dialog/Select/Popover) portarem para DENTRO da
  // cena. A partida carrega o escopo de tokens mas não o publicava, então todo
  // diálogo da sessão ao vivo — "Reiniciar o combate?", o ajuste de PV, o
  // descanso de dia — abria em shadcn CLARO sobre a mesa escura (ALE-122).
  const [sceneEl, setSceneEl] = createSignal<HTMLElement | null>(null)

  return (
    // `scene-grimorio` is the TOKEN SCOPE, not a skin: without it the whole
    // match renders in shadcn's light defaults. Match mode owns the viewport,
    // so it carries the scope itself instead of going through SceneShell,
    // which brings a back button and a title bar this screen already has.
    <div
      ref={setSceneEl}
      class="scene-grimorio flex h-dvh min-h-0 flex-col bg-background text-foreground"
    >
      <header
        class={cn(
          'flex items-center gap-2 border-b px-2 py-2 transition-colors sm:px-3',
          props.minhaVez
            ? 'border-grimorio-gold/60 bg-grimorio-gold/10'
            : 'border-grimorio-iron bg-grimorio-panel',
        )}
      >
        {/* A saída vem PRIMEIRO e sem texto (ALE-201): ícone e tooltip bastam,
            e à esquerda ela fica onde a volta sempre esteve nesta casa. Perder
            o rótulo é o que abre espaço para o estado ao vivo caber na mesma
            linha. O `Tooltip` da casa já traz o atraso certo (ALE-183) — nada
            de `openDelay` próprio aqui.

            Não é `asChild`: no Solid um link com cara de botão É um link
            vestindo as classes do botão. */}
        <Tooltip>
          {/* O gatilho é um `span` e o link vai DENTRO: `as={Link}` faz o
              Kobalte repassar as props, e a tipagem de rota do TanStack não
              sobrevive à travessia — `params` deixa de ser conferido contra o
              caminho, que é justamente o que ela existe para conferir. */}
          <TooltipTrigger as="span" class="inline-flex shrink-0">
            <Link
              to="/campaigns/$id"
              params={{ id: String(props.campaignId) }}
              aria-label="Sair da sessão"
              class={cn(buttonVariants({ variant: 'outline', size: 'icon-sm' }))}
            >
              <LogOut aria-hidden="true" class="size-4" />
            </Link>
          </TooltipTrigger>
          <TooltipContent>Sair da sessão</TooltipContent>
        </Tooltip>

        <p class="min-w-0 shrink truncate font-heading tracking-wide text-grimorio-gold">
          {props.title}
        </p>

        {/* `min-w-0` na cadeia inteira: o estado ao vivo trunca o nome de quem
            está na vez, e `truncate` só encolhe se TODO ancestral flex puder
            encolher — o `min-width: auto` padrão de um item flex é o
            min-content dele, que num texto sem quebra é a frase inteira
            (ALE-184). */}
        <div class="ml-auto flex min-w-0 items-center gap-2">
          {props.live}
          {props.bar}
          <Button
            variant="outline"
            size="icon-sm"
            class="shrink-0"
            aria-label={props.sfxEnabled ? 'Desligar o som' : 'Ligar o som'}
            aria-pressed={props.sfxEnabled}
            onClick={() => props.onToggleSfx()}
          >
            <Show when={props.sfxEnabled} fallback={<VolumeX aria-hidden="true" class="size-4" />}>
              <Volume2 aria-hidden="true" class="size-4" />
            </Show>
          </Button>
        </div>
      </header>
      <div class="min-h-0 flex-1 overflow-y-auto">
        <SceneContainerProvider element={sceneEl}>{props.children}</SceneContainerProvider>
      </div>
    </div>
  )
}
