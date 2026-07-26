import type { FactCategory } from '@tormenta20/t20-data'
import {
  Eye,
  Footprints,
  type LucideIcon,
  Shield,
  ShieldCheck,
  Sparkles,
  Zap,
} from 'lucide-react'

/** Category → icon, so a list of abilities/facts gets scannable eye-anchors. */
export const ABILITY_ICON: Record<FactCategory, LucideIcon> = {
  sense: Eye,
  dr: Shield,
  immunity: ShieldCheck,
  movement: Footprints,
  action: Zap,
  other: Sparkles,
}

/**
 * One ability/benefit row: a category icon + bold name + a two-line-clamped
 * description. Shared by the character-select info panel and the creation
 * wizard's grant previews so abilities render identically across both screens.
 * Render inside a <ul>.
 */
export function AbilityLine({
  category,
  name,
  description,
}: {
  category?: FactCategory
  name: string
  description: string
}) {
  const Icon = category ? ABILITY_ICON[category] : Sparkles
  return (
    <li className="flex gap-2">
      <Icon
        className="mt-0.5 size-4 shrink-0 text-muted-foreground"
        aria-hidden
      />
      <div className="min-w-0">
        <p className="text-xs font-semibold">{name}</p>
        <p className="line-clamp-2 text-[11px] leading-snug text-muted-foreground">
          {description}
        </p>
      </div>
    </li>
  )
}
