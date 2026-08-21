import {
  LogOut,
  type LucideIcon,
  Maximize,
  Minimize,
  Settings,
  ShieldCheck,
  UserPlus,
  Volume2,
  VolumeX,
} from 'lucide-solid'
import { Dynamic } from 'solid-js/web'
import { type ComponentProps, Show, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/popover'
import { Slider } from '@/shared/ui/slider'

export type HubFooterProps = {
  name: string
  onLogout: () => void
  logoutPending?: boolean
  sfxEnabled: boolean
  onToggleSfx: () => void
  /** 0–100. Só aparece com o som ligado: um slider sobre o mudo é controle
   *  morto, e o que ele resolve é o sino do "Sua vez" alto demais (ALE-180). */
  volume: number
  onVolumeChange: (percent: number) => void
  /** False on iPhone Safari, which has no Fullscreen API for elements: the item
   *  is hidden rather than shown dead. Those players get a chrome-less app via
   *  "Adicionar à Tela de Início" (the meta tags in index.html). */
  fullscreenSupported?: boolean
  fullscreenActive?: boolean
  onToggleFullscreen?: () => void
  /** True only for an ADMIN_EMAILS account — the real gate is the server (ALE-120). */
  canInvite?: boolean
  onInvite?: () => void
  /** Mesma origem do `canInvite`: o servidor diz, a UI só mostra (ALE-120). */
  canAdminister?: boolean
  onAdminister?: () => void
  class?: string
}

/**
 * The player's identity strip at the foot of the Hub. The initialed portrait +
 * name triggers a quick menu (popover): the sound toggle, tela cheia (when the
 * browser has the API), Configurações (placeholder until a settings screen
 * exists), and Sair. Pure — the page injects the name and the callbacks.
 *
 * Fullscreen lives here, and not per scene, because it survives client-side
 * navigation: the player turns it on once at the Hub and the whole session
 * stays chrome-less.
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
          class="group flex flex-1 items-center gap-3 rounded-none p-1 text-left"
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
          <Show when={props.sfxEnabled}>
            <Slider
              class="px-2 pb-2 pt-1"
              label="Volume"
              value={props.volume}
              onChange={props.onVolumeChange}
              format={(percent) => `${percent}%`}
            />
          </Show>
          <Show when={props.fullscreenSupported}>
            <QuickMenuItem
              icon={props.fullscreenActive ? Minimize : Maximize}
              onClick={() => props.onToggleFullscreen?.()}
            >
              {props.fullscreenActive ? 'Sair da tela cheia' : 'Tela cheia'}
            </QuickMenuItem>
          </Show>
          <Show when={props.canInvite}>
            <QuickMenuItem icon={UserPlus} onClick={() => props.onInvite?.()}>
              Convidar jogador
            </QuickMenuItem>
          </Show>
          <Show when={props.canAdminister}>
            <QuickMenuItem icon={ShieldCheck} onClick={() => props.onAdminister?.()}>
              Administração
            </QuickMenuItem>
          </Show>
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
        'flex w-full items-center gap-2.5 rounded-none px-2.5 py-2 text-left font-heading text-sm tracking-wide transition-colors',
        'hover:bg-accent focus-visible:bg-accent',
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
