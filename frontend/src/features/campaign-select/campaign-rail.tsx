import { useEffect, useRef } from 'react'
import type { Campaign } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import {
  campaignEmblemGradient,
  campaignInitials,
  roleLabel,
} from './campaign-select-helpers'

/**
 * The chronicle selector as bookmark tabs down the tome's fore-edge (the
 * campaign analog of the character filmstrip). A vertical stack beside the book
 * on desktop, a horizontal strip on phones. The active tab takes the parchment
 * of an open page and merges with the book; each tab carries its emblem, name,
 * role, and an ember when a session is live. The selected one auto-centers.
 */
export function CampaignRail({
  campaigns,
  selectedId,
  activeByCampaign,
  onSelect,
  onHover,
  className,
}: {
  campaigns: Campaign[]
  selectedId: number
  activeByCampaign: Record<number, number>
  onSelect: (id: number) => void
  /** Fired when the pointer enters a tab — a subtle hover cue. */
  onHover?: () => void
  className?: string
}) {
  const railRef = useRef<HTMLDivElement>(null)
  // biome-ignore lint/correctness/useExhaustiveDependencies: recenter on selection change only.
  useEffect(() => {
    railRef.current
      ?.querySelector('[aria-current="true"]')
      ?.scrollIntoView({ block: 'nearest', inline: 'center', behavior: 'smooth' })
  }, [selectedId])

  return (
    <div
      ref={railRef}
      role="listbox"
      aria-label="Crônicas"
      className={cn(
        'flex gap-1.5 overflow-x-auto overflow-y-hidden pb-1',
        // Desktop: a vertical stack of markers glued to the book's fore-edge.
        // overflow-visible (not hidden/auto) so the active marker's wider edge +
        // focus outline are never clipped; the column is wider than any marker,
        // so there's room to protrude. Centered against the book's height.
        'lg:flex-col lg:items-start lg:justify-center lg:gap-0 lg:overflow-visible lg:pb-0',
        className,
      )}
    >
      {campaigns.map((c) => (
        <TabEntry
          key={c.id}
          campaign={c}
          active={c.id === selectedId}
          live={activeByCampaign[c.id] != null}
          onSelect={() => onSelect(c.id)}
          onHover={onHover}
        />
      ))}
    </div>
  )
}

function TabEntry({
  campaign,
  active,
  live,
  onSelect,
  onHover,
}: {
  campaign: Campaign
  active: boolean
  live: boolean
  onSelect: () => void
  onHover?: () => void
}) {
  return (
    <button
      type="button"
      role="option"
      aria-selected={active}
      aria-current={active}
      data-active={active}
      title={campaign.name}
      onClick={onSelect}
      onMouseEnter={onHover}
      className={cn(
        'grimorio-book-tab flex min-w-52 shrink-0 items-center gap-2.5 rounded-md px-3 py-2 text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-grimorio-gold',
        // Desktop: a contiguous thumb-index down the book's fore-edge. Tabs
        // touch (gap-0) with collapsed borders (-mt-px), forming one solid
        // strip; the outer corners round. The active marker is pulled out
        // (wider, w-52 vs w-44), takes the parchment of the open page, rounds
        // its protruding right tip, and sits on top (z-10) so its edges draw
        // over its neighbours. The column is wider than w-52, so it never clips.
        'lg:min-w-0 lg:rounded-none lg:[&:not(:first-child)]:-mt-px lg:first:rounded-tr-lg lg:last:rounded-br-lg',
        active
          ? 'lg:z-10 lg:w-52 lg:rounded-r-lg lg:border-l-0'
          : 'lg:w-44',
      )}
    >
      <span
        aria-hidden
        className="flex size-8 shrink-0 items-center justify-center rounded-sm border border-grimorio-iron font-display text-xs text-white/85"
        style={{ background: campaignEmblemGradient(campaign.name) }}
      >
        {campaignInitials(campaign.name)}
      </span>
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="truncate font-heading text-sm tracking-wide">
          {campaign.name}
        </span>
        <span className="text-[10px] uppercase tracking-[0.18em] opacity-65">
          {roleLabel(campaign.role)}
        </span>
      </span>
      {live && (
        <span
          role="img"
          className="size-2 shrink-0 rounded-full bg-[color:var(--hp-full)]"
          title="Sessão ao vivo"
          aria-label="Sessão ao vivo"
        />
      )}
    </button>
  )
}
