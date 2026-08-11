import { CircleCheckIcon, InfoIcon, Loader2Icon, OctagonXIcon, TriangleAlertIcon } from 'lucide-solid'
import { Toaster as SolidSonner, type ToasterProps } from 'solid-sonner'
import type { Theme } from '@/shared/stores/ui-store'

export { toast } from 'solid-sonner'

/**
 * Toasts, on solid-sonner. Same styling contract as the React kit (tokens via
 * CSS vars, our own icon set).
 *
 * Theme comes in as a prop instead of being read from a global store — the
 * root owns the UI store and passes it down, keeping this a leaf.
 *
 * @example <Toaster theme={ui.theme()} />
 */
export function Toaster(props: ToasterProps & { theme?: Theme }) {
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
