import type { CSSProperties } from 'react'
import { Toaster as SonnerToaster } from 'sonner'
import { useUiStore } from '@/shared/stores/ui-store'

/**
 * App-themed toast host. Wraps sonner with Controlled Decay tokens —
 * card surface, ember-tinted shadow, Cinzel titles — so a notification
 * reads as part of the sheet rather than a stock widget. Mounted once at
 * the app root; fire toasts anywhere via `toast()` from 'sonner'.
 *
 * Colors come through sonner's CSS-var hooks (reliable override); fonts
 * + radius via classNames. `top-center` matches the "notify the active
 * player" intent — the cue lands where the eye already is.
 */
export function Toaster() {
  const theme = useUiStore((s) => s.theme)
  return (
    <SonnerToaster
      theme={theme}
      position="top-center"
      style={
        {
          '--normal-bg': 'var(--card)',
          '--normal-text': 'var(--card-foreground)',
          '--normal-border': 'var(--border)',
        } as CSSProperties
      }
      toastOptions={{
        classNames: {
          toast:
            'font-serif rounded-md shadow-[0_18px_50px_-30px_var(--primary)]',
          title: 'font-display tracking-wide',
          description: 'text-muted-foreground',
          actionButton: 'bg-primary text-primary-foreground',
        },
      }}
    />
  )
}
