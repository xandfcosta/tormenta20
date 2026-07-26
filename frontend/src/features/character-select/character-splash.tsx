import type { Character } from '@/shared/api/api'
import { Badge } from '@/shared/ui/badge'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'
import { primaryClass } from './select-helpers'

/** First letter of the name — the giant faded "keystone" behind the splash. */
function keystone(name: string): string {
  return name.trim()[0]?.toUpperCase() ?? '?'
}

/**
 * The dominant character "splash". With no image field (reskin-only), identity
 * comes from a deterministic name→hue oklch gradient plus a giant faded
 * keystone initial — a Valorant-style hero panel built from data alone. The
 * gradient hue must be inline (Tailwind can't template an arbitrary hue).
 * Decorative layers are aria-hidden; the visible name lives in a real <h2>.
 */
export function CharacterSplash({ character }: { character: Character }) {
  const hue = hueFromName(character.name)
  return (
    <div
      className="relative min-h-40 flex-1 overflow-hidden rounded-xl md:min-h-56 lg:min-h-64"
      style={{
        background: `linear-gradient(155deg, oklch(0.55 0.15 ${hue}) 0%, oklch(0.30 0.09 ${hue}) 70%, oklch(0.22 0.06 ${hue}) 100%)`,
      }}
    >
      <span
        aria-hidden
        className="pointer-events-none absolute inset-0 flex select-none items-center justify-center font-black leading-none text-white/15 text-[7rem] sm:text-[14rem]"
      >
        {keystone(character.name)}
      </span>
      {/* Fixed dark scrim (not theme background) so the white name keeps
          contrast on light hues. */}
      <div className="absolute inset-x-0 bottom-0 flex items-end justify-between gap-2 bg-gradient-to-t from-black/75 via-black/25 to-transparent p-4 pt-10">
        <div className="min-w-0">
          <h2
            className={cn(
              'truncate bg-gradient-to-r from-white to-white/80 bg-clip-text text-2xl font-semibold tracking-tight text-transparent drop-shadow sm:text-3xl',
            )}
          >
            {character.name}
          </h2>
          <p className="truncate text-sm text-white/85">
            {primaryClass(character)}
          </p>
        </div>
        <Badge variant="secondary" className="shrink-0 self-end">
          Nv {character.level}
        </Badge>
      </div>
    </div>
  )
}
