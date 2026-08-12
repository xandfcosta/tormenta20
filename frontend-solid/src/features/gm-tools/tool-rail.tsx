import { BookMarked, Dices, Skull, Swords } from 'lucide-solid'
import type { LucideProps } from 'lucide-solid'
import { For, type Component } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { GM_TOOLS, type ToolSlug } from './gm-tools'

/** Icons live here, not in the registry: the registry is a rule and stays
 *  importable by a test without dragging an icon pack along. */
const TOOL_ICON: Record<ToolSlug, Component<LucideProps>> = {
  bestiario: Skull,
  encontros: Swords,
  improviso: Dices,
  catalogos: BookMarked,
}

export type ToolRailProps = {
  current: ToolSlug
  onPick: (slug: ToolSlug) => void
}

/**
 * The Mesa's tool rail: a column on wide viewports, a tab bar under the header
 * on narrow ones. ONE list either way — a `Show` with two branches would mean
 * two DOM trees to keep in step, which is how a rail and its phone twin drift.
 */
export function ToolRail(props: ToolRailProps) {
  return (
    <nav
      aria-label="Ferramentas do mestre"
      // Horizontal and scrollable below lg, a column at lg and up. Media query
      // by WIDTH only: on a phone the virtual keyboard changes viewport HEIGHT,
      // and a height-driven switch would rebuild the rail mid-typing.
      class="flex shrink-0 gap-1 overflow-x-auto pb-1 lg:w-44 lg:flex-col lg:overflow-x-visible lg:pb-0"
    >
      <For each={GM_TOOLS}>
        {(tool) => {
          const Icon = TOOL_ICON[tool.slug]
          const active = () => props.current === tool.slug
          return (
            <button
              type="button"
              aria-current={active() ? 'page' : undefined}
              onClick={() => props.onPick(tool.slug)}
              title={tool.hint}
              class={cn(
                'flex shrink-0 items-center gap-2 rounded-md border px-3 py-2 text-left text-xs transition-colors lg:w-full',
                active()
                  ? 'border-grimorio-gold bg-accent font-medium text-grimorio-gold'
                  : 'border-grimorio-iron text-muted-foreground hover:bg-accent hover:text-foreground',
              )}
            >
              <Icon aria-hidden="true" class="size-4 shrink-0" />
              <span class="truncate font-heading uppercase tracking-[0.12em]">{tool.label}</span>
            </button>
          )
        }}
      </For>
    </nav>
  )
}
