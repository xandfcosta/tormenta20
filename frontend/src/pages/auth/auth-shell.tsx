import type { JSX, ParentProps } from 'solid-js'
import { Show } from 'solid-js'
import { SceneShell } from '@/shared/layout/scene-shell'
import { FramedPanel } from '@/shared/ui/framed-panel'

/**
 * A porta do jogo: /login, /register e /redefinir-senha.
 *
 * Era o split-screen do shadcn — painel de marca à esquerda, formulário à
 * direita — e era a última superfície do app que ainda falava a língua do
 * template em vez da da casa (ALE-173). Agora usa a gramática da TELA-TÍTULO,
 * a mesma do Hub: pedra com vinheta, o nome do jogo em Cinzel, e o formulário
 * dentro de uma moldura de ferro no meio.
 *
 * O painel de marca morreu junto com a frase de vitrine que ele carregava
 * ("Sua mesa, organizada…"): numa tela-título ela é redundante — quem chegou
 * até a porta já sabe o que há dentro — e é linguagem de landing page, não a
 * voz do grimório. Sobra o rodapé legal, que existe por outro motivo.
 *
 * Isto não é só pintura: virando cena, estas três telas passam a herdar o
 * escopo `.scene-grimorio`, e com ele os tokens, a escala de raio e o
 * tratamento de foco de todo o resto do app. Era a única superfície que
 * resolvia no `:root` claro.
 */
export function AuthShell(
  props: ParentProps<{ title: string; subtitle?: string; footer?: JSX.Element }>,
) {
  return (
    <SceneShell title="Tormenta 20" kicker="— Grimório de Arton —">
      <div class="mx-auto flex w-full max-w-md flex-1 flex-col justify-center gap-5 py-6">
        <FramedPanel class="space-y-5">
          <div class="space-y-1.5">
            <h2 class="font-heading text-lg uppercase tracking-[0.16em] text-grimorio-gold">
              {props.title}
            </h2>
            <Show when={props.subtitle}>
              {(subtitle) => <p class="text-sm text-muted-foreground">{subtitle()}</p>}
            </Show>
          </div>
          {props.children}
        </FramedPanel>

        <Show when={props.footer}>
          {(footer) => <p class="text-center text-sm text-muted-foreground">{footer()}</p>}
        </Show>

        <p class="text-center text-xs text-muted-foreground">
          Gerenciador não-oficial de Tormenta 20.
        </p>
      </div>
    </SceneShell>
  )
}
