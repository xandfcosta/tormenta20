import { LogOut, type LucideIcon, Settings, Volume2, VolumeX } from 'lucide-solid'
import { Dynamic } from 'solid-js/web'
import { type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/popover'

export type HubFooterProps = {
  name: string
  onLogout: () => void
  logoutPending?: boolean
  sfxEnabled: boolean
  onToggleSfx: () => void
  class?: string
}

/**
 * The player's identity strip at the foot of the Hub. The initialed portrait +
 * name triggers a quick menu (popover): the sound toggle, Configurações
 * (placeholder until a settings screen exists), and Sair. Pure — the page
 * injects the name and the callbacks.
 *
 * No theme toggle here on purpose: a game scene is intrinsically dark
 * (`.scene-grimorio`), so a light/dark switch would read as a dead control.
 */
export function HubFooter(props: HubFooterProps) {
  const initial = () => props.name.trim().charAt(0).toUpperCase() || '?'
  return (
    <footer class={cn('mx-auto flex w-full max-w-md border-t border-grimorio-iron pt-4', props.class)}>
      <Popover placement="top-start">
        <PopoverTrigger
          aria-label={`Menu de ${props.name}`}
          class="group flex flex-1 items-center gap-3 rounded-sm p-1 text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-grimorio-gold"
        >
          <span
            aria-hidden="true"
            class="grid size-10 place-items-center rounded-full border-2 border-grimorio-iron-light font-heading font-bold text-grimorio-gold transition-colors group-hover:border-grimorio-gold"
          >
            {initial()}
          </span>
          <span class="min-w-0 flex-1 truncate font-heading tracking-wide text-foreground">
            {props.name}
          </span>
          <Settings
            aria-hidden="true"
            class="size-4 text-muted-foreground transition-colors group-hover:text-grimorio-gold"
          />
        </PopoverTrigger>
        {/* No `.scene-grimorio` re-application here, unlike the React version:
            the popover now portals INTO the scene (ALE-66), so it inherits the
            token scope instead of escaping to the body. */}
        <PopoverContent class="w-56 p-1.5">
          <QuickMenuItem
            icon={props.sfxEnabled ? Volume2 : VolumeX}
            onClick={() => props.onToggleSfx()}
          >
            {props.sfxEnabled ? 'Som ligado' : 'Som desligado'}
          </QuickMenuItem>
          <QuickMenuItem icon={Settings} disabled>
            Configurações
          </QuickMenuItem>
          <QuickMenuItem
            icon={LogOut}
            onClick={() => props.onLogout()}
            disabled={props.logoutPending}
            danger
          >
            Sair
          </QuickMenuItem>
        </PopoverContent>
      </Popover>
    </footer>
  )
}

type QuickMenuItemProps = ComponentProps<'button'> & {
  icon: LucideIcon
  danger?: boolean
}

function QuickMenuItem(props: QuickMenuItemProps) {
  const [local, rest] = splitProps(props, ['icon', 'children', 'danger', 'class'])
  return (
    <button
      type="button"
      class={cn(
        'flex w-full items-center gap-2.5 rounded-sm px-2.5 py-2 text-left font-heading text-sm tracking-wide transition-colors',
        'hover:bg-accent focus-visible:bg-accent focus-visible:outline-none',
        'disabled:pointer-events-none disabled:opacity-50',
        local.danger ? 'text-grimorio-crimson-bright' : 'text-foreground',
        local.class,
      )}
      {...rest}
    >
      <Dynamic component={local.icon} aria-hidden="true" class="size-4" />
      {local.children}
    </button>
  )
}
