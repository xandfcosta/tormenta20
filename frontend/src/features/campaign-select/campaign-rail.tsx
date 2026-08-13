import { For, Show, createEffect } from 'solid-js'
import type { Campaign } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import { campaignEmblemGradient, campaignInitials, roleLabel } from './campaign-select-helpers'

export type CampaignRailProps = {
  campaigns: Campaign[]
  selectedId: number
  activeByCampaign: Record<number, number>
  onSelect: (id: number) => void
  /** Fired when the pointer enters a tab — a subtle hover cue. */
  onHover?: () => void
  class?: string
}

/**
 * The chronicle selector as bookmark tabs down the tome's fore-edge (the
 * campaign analog of the character filmstrip). A vertical stack beside the book
 * on desktop, a horizontal strip on phones. The active tab takes the parchment
 * of an open page and merges with the book; each tab carries its emblem, name,
 * role, and an ember when a session is live. The selected one auto-centers.
 */
export function CampaignRail(props: CampaignRailProps) {
  let rail: HTMLDivElement | undefined

  createEffect(() => {
    const id = props.selectedId
    rail
      ?.querySelector(`[data-tab-id="${id}"]`)
      ?.scrollIntoView({ block: 'nearest', inline: 'center', behavior: 'smooth' })
  })

  return (
    <div
      ref={rail}
      role="listbox"
      aria-label="Crônicas"
      class={cn(
        'flex gap-1.5 overflow-x-auto overflow-y-hidden pb-1',
        // Desktop: a vertical stack of markers glued to the book's fore-edge.
        // overflow-visible (not hidden/auto) so the active marker's wider edge +
        // focus outline are never clipped; the column is wider than any marker.
        'lg:flex-col lg:items-start lg:justify-center lg:gap-0 lg:overflow-visible lg:pb-0',
        props.class,
      )}
    >
      <For each={props.campaigns}>
        {(campaign) => (
          <TabEntry
            campaign={campaign}
            active={campaign.id === props.selectedId}
            live={props.activeByCampaign[campaign.id] != null}
            onSelect={() => props.onSelect(campaign.id)}
            onHover={() => props.onHover?.()}
          />
        )}
      </For>
    </div>
  )
}

function TabEntry(props: {
  campaign: Campaign
  active: boolean
  live: boolean
  onSelect: () => void
  onHover: () => void
}) {
  return (
    <button
      type="button"
      role="option"
      data-tab-id={props.campaign.id}
      aria-selected={props.active}
      aria-current={props.active}
      data-active={props.active}
      title={props.campaign.name}
      onClick={() => props.onSelect()}
      onMouseEnter={() => props.onHover()}
      class={cn(
        'grimorio-book-tab flex min-w-52 shrink-0 items-center gap-2.5 rounded-md px-3 py-2 text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-grimorio-gold',
        // Desktop: a contiguous thumb-index down the book's fore-edge. Tabs
        // touch (gap-0) with collapsed borders (-mt-px), forming one solid
        // strip; the outer corners round. The active marker is pulled out
        // (wider), takes the parchment of the open page, rounds its protruding
        // right tip, and sits on top (z-10) so its edges draw over neighbours.
        'lg:min-w-0 lg:rounded-none lg:[&:not(:first-child)]:-mt-px lg:first:rounded-tr-lg lg:last:rounded-br-lg',
        props.active ? 'lg:z-10 lg:w-52 lg:rounded-r-lg lg:border-l-0' : 'lg:w-44',
      )}
    >
      <span
        aria-hidden="true"
        class="flex size-8 shrink-0 items-center justify-center rounded-sm border border-grimorio-iron font-display text-xs text-white/85"
        style={{ background: campaignEmblemGradient(props.campaign.name) }}
      >
        {campaignInitials(props.campaign.name)}
      </span>
      <span class="flex min-w-0 flex-1 flex-col">
        <span class="truncate font-heading text-sm tracking-wide">{props.campaign.name}</span>
        <span class="text-[10px] uppercase tracking-[0.18em] opacity-65">
          {roleLabel(props.campaign.role)}
        </span>
      </span>
      <Show when={props.live}>
        <span
          role="img"
          class="size-2 shrink-0 rounded-full bg-[color:var(--hp-full)]"
          title="Sessão ao vivo"
          aria-label="Sessão ao vivo"
        />
      </Show>
    </button>
  )
}
