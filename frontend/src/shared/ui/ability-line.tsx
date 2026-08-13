import type { FactCategory } from '@/shared/api/display-facts'
import { Eye, Footprints, Shield, ShieldCheck, Sparkles, Zap } from 'lucide-solid'
import { Dynamic } from 'solid-js/web'
import type { Component } from 'solid-js'

/** Category → icon, so a list of abilities/facts gets scannable eye-anchors. */
export const ABILITY_ICON: Record<FactCategory, Component<{ class?: string }>> = {
  sense: Eye,
  dr: Shield,
  immunity: ShieldCheck,
  movement: Footprints,
  action: Zap,
  other: Sparkles,
}

/**
 * One ability/benefit row: a category icon + bold name + a two-line-clamped
 * description. Shared by every surface that lists abilities, so they read
 * identically wherever they appear. Render inside a `<ul>`.
 *
 * @example <AbilityLine category="movement" name="Deslocamento" description="9m" />
 */
export function AbilityLine(props: {
  category?: FactCategory
  name: string
  description: string
}) {
  const icon = () => (props.category ? ABILITY_ICON[props.category] : Sparkles)
  return (
    <li class="flex gap-2">
      <Dynamic
        component={icon()}
        class="mt-0.5 size-4 shrink-0 text-muted-foreground"
        aria-hidden="true"
      />
      <div class="min-w-0">
        <p class="text-xs font-semibold">{props.name}</p>
        <p class="line-clamp-2 text-[11px] leading-snug text-muted-foreground">
          {props.description}
        </p>
      </div>
    </li>
  )
}
