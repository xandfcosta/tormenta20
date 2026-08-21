import { CircleCheckIcon, InfoIcon, Loader2Icon, OctagonXIcon, TriangleAlertIcon } from 'lucide-solid'
import { Toaster as SolidSonner, type ToasterProps } from 'solid-sonner'

export { toast } from 'solid-sonner'

/**
 * Toasts, on solid-sonner. Same styling contract as the React kit (tokens via
 * CSS vars, our own icon set).
 *
 * As cores vêm dos tokens (`--popover` e irmãos), então quem manda é o escopo
 * onde o aviso nasce. Isso já mordeu: o `Toaster` é irmão do `Outlet`, fica
 * fora de qualquer cena, e enquanto a RAIZ era clara todo aviso do jogo pintava
 * branco puro sobre a mesa escura (ALE-173).
 *
 * @example <Toaster theme="dark" />
 */
export function Toaster(props: ToasterProps) {
  return (
    <SolidSonner
      class="toaster group"
      theme={props.theme}
      icons={{
        success: <CircleCheckIcon class="size-4" />,
        info: <InfoIcon class="size-4" />,
        warning: <TriangleAlertIcon class="size-4" />,
        error: <OctagonXIcon class="size-4" />,
        loading: <Loader2Icon class="size-4 animate-spin" />,
      }}
      style={{
        '--normal-bg': 'var(--popover)',
        '--normal-text': 'var(--popover-foreground)',
        '--normal-border': 'var(--border)',
        '--border-radius': 'var(--radius)',
      }}
      {...props}
    />
  )
}
