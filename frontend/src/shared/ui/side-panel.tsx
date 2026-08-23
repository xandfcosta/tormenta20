import { Dialog as KDialog } from '@kobalte/core/dialog'
import { X } from 'lucide-solid'
import { type JSX, Show, splitProps } from 'solid-js'
import { createMediaQuery } from '@/shared/lib/media-query'
import { useSceneContainer } from '@/shared/lib/scene-container'
import { cn } from '@/shared/lib/utils'
import { SectionTitle } from './section-label'

/** Above this the panel shares the screen instead of taking it. Chosen to match
 *  where keyboard navigation turns on — the laptop the GM actually runs from. */
const SHARES_SCREEN = '(min-width: 1280px)'

export type SidePanelProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  /** Live context pinned above the body — the tracker's peek, so the GM never
   *  loses the round while reading. */
  header?: JSX.Element
  children: JSX.Element
  class?: string
  /**
   * De que borda a gaveta entra, acima do degrau. Padrão direita, que é onde
   * moram as consultas do mestre (catálogo, bestiário). A ESQUERDA é da fila do
   * combate: ela mora no trilho esquerdo, e uma gaveta que entrasse pelo outro
   * lado faria o conteúdo atravessar a tela para longe do botão que o chamou
   * (ALE-198).
   */
  side?: 'left' | 'right'
}

/**
 * A side sheet built on the Kobalte Dialog, NOT a drag-gesture drawer.
 *
 * From 1280px up it is **non-modal**: no scrim, no focus trap, nothing hidden
 * from assistive tech — the scene behind stays live and clickable, which is the
 * whole point during combat (read the condition here, apply it there). Below
 * that there is no room for two things, so it becomes a real modal bottom sheet
 * with a full-width close bar: on a phone the X in the top corner is the
 * hardest place for a thumb to reach.
 *
 * ONE component switching by a WIDTH media query — never height, because the
 * virtual keyboard changes viewport height and a height-driven switch would
 * rebuild the panel mid-typing and lose what was being searched.
 *
 * Acima do degrau a largura CRESCE com a janela até um teto, e a entrada e a
 * saída são um deslize puro pela borda do `side` — um quadro de `transform` e
 * nada mais (ALE-207).
 *
 * @example <SidePanel open={open()} onOpenChange={setOpen} title="Catálogos">…</SidePanel>
 */
export function SidePanel(props: SidePanelProps) {
  const [local] = splitProps(props, [
    'open',
    'onOpenChange',
    'title',
    'description',
    'header',
    'children',
    'class',
    'side',
  ])
  const scene = useSceneContainer()
  const sharesScreen = createMediaQuery(SHARES_SCREEN)

  return (
    <KDialog
      open={local.open}
      onOpenChange={local.onOpenChange}
      // Non-modal where the screen is wide enough to hold both.
      modal={!sharesScreen()}
    >
      <KDialog.Portal mount={scene() ?? undefined}>
        {/* No scrim when sharing the screen: dimming what stays clickable is
            the interface lying about what it will accept. */}
        <Show when={!sharesScreen()}>
          <KDialog.Overlay class="fixed inset-0 z-50 bg-black/50 data-[closed]:animate-out data-[closed]:fade-out-0 data-[expanded]:animate-in data-[expanded]:fade-in-0" />
        </Show>

        <KDialog.Content
          // Read by `scene-nav`: an inline panel must NOT make the keyboard
          // driver stand down, or opening it would kill the scene's arrows.
          data-nav-inline={sharesScreen() ? '' : undefined}
          data-nav-region={sharesScreen() ? 'side-panel' : undefined}
          data-nav-layout="column"
          // Kobalte dismisses on outside interaction even when NOT modal, which
          // would undo the whole point: the GM clicks the tracker behind and the
          // panel they were reading vanishes. While sharing the screen the panel
          // closes only by Esc, the X, or the close bar.
          onInteractOutside={(event) => {
            if (sharesScreen()) event.preventDefault()
          }}
          class={cn(
            'fixed z-50 flex flex-col gap-3 border-grimorio-iron bg-grimorio-panel shadow-xl',
            // Só o deslize: um quadro de `transform` e nada de opacidade,
            // escala ou desfoque (ALE-207). A troca de conteúdo destas
            // superfícies custa tarefas longas de 55–133ms (ALE-174), e
            // animação sobre a main thread ocupada gagueja — um `transform`
            // puro fica no compositor e não disputa nada.
            'side-panel-slide',
            // Bottom sheet below the breakpoint; right-hand column above it.
            // The bottom padding grows past the home indicator under
            // `viewport-fit=cover`, keeping the p-3/p-4 floor of each layout.
            'inset-x-0 bottom-0 max-h-[92dvh] rounded-t-md border-t p-3 pb-[max(0.75rem,env(safe-area-inset-bottom))]',
            '[--side-panel-from-x:0] [--side-panel-from-y:100%]',
            // A largura CRESCE com a janela e tem teto: 26rem era 16% de uma
            // tela de 2560 para uma lista de magias que quer largura (ALE-207).
            // O piso é a largura de antes, e a 1280 o `clamp` cai nele — o
            // ponto de quebra não muda de tamanho. O teto existe porque acima
            // de 1280 o painel é NÃO modal de propósito (ALE-75): o mestre lê a
            // condição aqui e clica no rastreador atrás, e uma gaveta que
            // crescesse sem parar cobriria o que ele quer clicar.
            'xl:inset-y-0 xl:max-h-none xl:w-[clamp(26rem,30vw,44rem)] xl:rounded-none xl:border-t-0 xl:p-4 xl:pb-[max(1rem,env(safe-area-inset-bottom))]',
            // Escrito por ramo e não por `cn`: `xl:left-auto` e `xl:left-0` são
            // a MESMA propriedade, e deixar as duas na string entrega a decisão
            // à ordem do merge em vez de ao lado pedido. Os DOIS eixos do
            // deslize se reescrevem aqui pelo mesmo motivo, e por um a mais:
            // deixar o `--side-panel-from-y` da folha de baixo valendo acima do
            // `xl` foi exatamente o defeito da ALE-207 — a gaveta entrava na
            // diagonal, do canto inferior, atravessando a altura da janela.
            local.side === 'left'
              ? 'xl:left-0 xl:right-auto xl:border-r xl:[--side-panel-from-x:-100%] xl:[--side-panel-from-y:0]'
              : 'xl:left-auto xl:right-0 xl:border-l xl:[--side-panel-from-x:100%] xl:[--side-panel-from-y:0]',
            local.class,
          )}
        >
          <div class="flex shrink-0 items-start gap-2">
            <div class="min-w-0 flex-1 space-y-0.5">
              {/* `as` leva a peça do Kobalte: é ela que registra o rótulo do
                  diálogo, e trocá-la por uma tag solta tiraria isso. */}
              <SectionTitle as={KDialog.Title} class="text-sm">
                {local.title}
              </SectionTitle>
              <Show when={local.description}>
                {(description) => (
                  <KDialog.Description class="text-2xs text-muted-foreground">
                    {description()}
                  </KDialog.Description>
                )}
              </Show>
            </div>
            {/* Kobalte's CloseButton ships an English label — the app is pt-BR
                (gotcha #2). Hidden on the phone, where the bar below wins. */}
            <KDialog.CloseButton
              aria-label={`Fechar ${local.title}`}
              class="hidden shrink-0 rounded-none text-muted-foreground transition-colors hover:text-foreground xl:block"
            >
              <X aria-hidden="true" class="size-4" />
            </KDialog.CloseButton>
          </div>

          <Show when={local.header}>
            {(header) => (
              <div class="shrink-0 rounded-sm border border-grimorio-iron bg-muted/20 px-2 py-1 text-2xs">
                {header()}
              </div>
            )}
          </Show>

          <div class="min-h-0 flex-1 overflow-y-auto">{local.children}</div>

          {/* Full-width and anchored at the bottom: the reachable half of a
              phone screen. Redundant with Esc and the X, never the only way.
              The explicit `aria-label` is not decoration — Kobalte's own
              English "Dismiss" OVERRIDES the visible text (gotcha #2), so
              without it the bar announces a word the app never shows.

              O rótulo NOMEIA o painel, como o ✕ do topo já fazia: "Fechar" nu
              não diz fechar o quê, e dentro da gaveta da fila ele empatava com
              o "Fechar" do formulário de adicionar — dois botões com o mesmo
              nome acessível na mesma caixa (ALE-198). */}
          <KDialog.CloseButton
            aria-label={`Fechar ${local.title}`}
            class="shrink-0 rounded-sm border border-grimorio-iron px-3 py-2 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground xl:hidden"
          >
            Fechar
          </KDialog.CloseButton>
        </KDialog.Content>
      </KDialog.Portal>
    </KDialog>
  )
}
