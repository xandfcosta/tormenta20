import { Tooltip as KTooltip } from '@kobalte/core/tooltip'
import { type ComponentProps, splitProps } from 'solid-js'
import { useSceneContainer } from '@/shared/lib/scene-container'
import { cn } from '@/shared/lib/utils'

/** O atraso de abertura da casa, herdado do `TooltipProvider` do React. */
const ATRASO_DA_CASA = 150

/**
 * Tooltip on Kobalte.
 *
 * There is no `TooltipProvider` to mount at the app root: Radix kept the shared
 * open-delay there, Kobalte puts it on each tooltip (`openDelay`). Este arquivo
 * PEDIA que cada chamador passasse `openDelay={150}`, e nenhum dos três passou:
 * desde o cutover para Solid todo tooltip do app abria com os 700ms padrão do
 * Kobalte, quase cinco vezes o que a casa queria. Uma intenção que depende de
 * alguém lembrar não é uma intenção, é um pedido — agora ela é o padrão, e quem
 * quiser outro atraso passa o dele.
 *
 * De quebra encolhe a janela de um defeito de TESTE: um clique agenda a
 * abertura, e se o arquivo termina antes de o timer disparar ele acorda com o
 * jsdom já desmontado e derruba o arquivo inteiro com `window is not defined`.
 *
 * @example
 * <Tooltip>
 *   <TooltipTrigger>?</TooltipTrigger>
 *   <TooltipContent>Defesa = 10 + DES + armadura</TooltipContent>
 * </Tooltip>
 */
export function Tooltip(props: ComponentProps<typeof KTooltip>) {
  return <KTooltip openDelay={ATRASO_DA_CASA} {...props} />
}

export const TooltipTrigger = KTooltip.Trigger

export type TooltipContentProps = ComponentProps<typeof KTooltip.Content> & {
  /** Portal target. Defaults to the enclosing grimório scene, else body. */
  mount?: Node
}

export function TooltipContent(props: TooltipContentProps) {
  const [local, rest] = splitProps(props, ['class', 'children', 'mount'])
  const scene = useSceneContainer()
  return (
    <KTooltip.Portal mount={local.mount ?? scene() ?? undefined}>
      <KTooltip.Content
        data-slot="tooltip-content"
        class={cn(
          'z-50 w-fit animate-in rounded-sm bg-foreground px-3 py-1.5 text-xs text-balance text-background fade-in-0 zoom-in-95 data-[closed]:animate-out data-[closed]:fade-out-0 data-[closed]:zoom-out-95',
          local.class,
        )}
        {...rest}
      >
        {local.children}
        <KTooltip.Arrow />
      </KTooltip.Content>
    </KTooltip.Portal>
  )
}
