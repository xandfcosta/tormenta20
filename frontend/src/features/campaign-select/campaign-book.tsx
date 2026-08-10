import { ChevronRight } from 'lucide-react'
import {
  type AnimationEvent,
  type CSSProperties,
  useRef,
  useState,
} from 'react'
import type { Campaign } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { Kbd } from '@/shared/ui/kbd'
import { cn } from '@/shared/lib/utils'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { useMediaQuery, usePrefersReducedMotion } from '@/shared/lib/use-media-query'
import {
  campaignEmblemGradient,
  campaignInitials,
  roleLabel,
} from './campaign-select-helpers'

const NOOP = () => {}

type Page = { campaign: Campaign; isLive: boolean }
// `fast`: more picks are still queued behind this turn (rapid navigation) — play
// it quickly so the sequence drains; the settling turn (empty queue) runs normal.
type Turn = { from: Page; to: Page; dir: 1 | -1; fast: boolean }

/**
 * Page-turn state machine with an unbounded queue. `target` is the current
 * pick; `shown` is what's landed. Every new target is appended to `queue`
 * (deduped against the last known destination); the book turns to them one at a
 * time — when a turn ends, the next queued target turns in. So picks made during
 * animations all play, in order, never cutting or restarting. Direction comes
 * from the markers' order (`orderIds`): a later marker turns forward, an earlier
 * one backward. Disabled (instant sync, no leaf) on phones / reduced motion.
 */
function usePageTurns(target: Page, orderIds: number[]) {
  const wide = useMediaQuery('(min-width: 640px)')
  const reduced = usePrefersReducedMotion()
  const canTurn = wide && !reduced
  const [shown, setShown] = useState<Page>(target)
  const [turn, setTurn] = useState<Turn | null>(null)
  const [queue, setQueue] = useState<Page[]>([])
  const lastTargetId = useRef(target.campaign.id)

  const dirTo = (fromId: number, toId: number): 1 | -1 =>
    orderIds.indexOf(toId) >= orderIds.indexOf(fromId) ? 1 : -1

  // Drive the machine during render (guarded → no render loop). All setState
  // uses functional updates so an enqueue and a dequeue in the same render
  // compose instead of clobbering each other.
  if (!canTurn) {
    if (shown.campaign.id !== target.campaign.id) setShown(target)
  } else {
    // Enqueue each new target, deduped against the last known destination (the
    // queue's tail, else the turn's destination, else what's shown).
    if (lastTargetId.current !== target.campaign.id) {
      const lastDest = queue.at(-1) ?? turn?.to ?? shown
      if (lastDest.campaign.id !== target.campaign.id) {
        setQueue((q) => [...q, target])
      }
      lastTargetId.current = target.campaign.id
    }
    // Idle with work queued → start the next turn.
    if (!turn && queue.length > 0) {
      const next = queue[0]
      setTurn({
        from: shown,
        to: next,
        dir: dirTo(shown.campaign.id, next.campaign.id),
        fast: queue.length > 1,
      })
      setShown(next)
      setQueue((q) => q.slice(1))
    }
  }

  const onEnd = (e: AnimationEvent<HTMLDivElement>) => {
    if (
      e.animationName === 'grimorio-leaf-turn' ||
      e.animationName === 'grimorio-leaf-turn-rev'
    )
      // Go idle; the render logic above starts the next queued turn, if any.
      setTurn(null)
  }

  return { shown, turn, onEnd }
}

/**
 * The focused chronicle as an open tome: a leather cover holding two leaves
 * split by a shaded spine. The left leaf is the illustration (a hue emblem
 * until real cover art lands — object-cover fills the same plate later); the
 * right leaf is the info + primary action. Page margins grow with the viewport
 * (roomy real-book margins on desktop, tight on phones). Switching chronicles
 * turns a double-sided leaf the whole way over the spine (queued via
 * `usePageTurns`). On phones the leaves stack and switch instantly.
 */
export function CampaignBook({
  campaign,
  isLive,
  orderIds,
  onOpen,
  onResume,
}: {
  campaign: Campaign
  isLive: boolean
  /** The markers' current order (ids) — gives each turn its direction. */
  orderIds: number[]
  onOpen: () => void
  onResume: () => void
}) {
  const { shown, turn, onEnd } = usePageTurns({ campaign, isLive }, orderIds)
  const fwd = !turn || turn.dir === 1
  // Base spread. Idle: both pages = shown. Turning: the not-yet-covered page
  // keeps the OLD content and the revealed page is already NEW (the leaf brings
  // the new one down over the old, so nothing mismatches). Forward reveals the
  // info (right) and keeps the old illustration; backward mirrors it.
  const baseArtName = turn
    ? (fwd ? turn.from : turn.to).campaign.name
    : shown.campaign.name
  const baseInfo = turn ? (fwd ? turn.to : turn.from) : shown
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center py-1">
      <div className="grimorio-book w-full p-2.5 sm:p-3">
        <div className="grimorio-book-pages flex flex-col overflow-hidden sm:min-h-[26rem] sm:flex-row sm:items-stretch lg:min-h-[32rem] xl:min-h-[36rem]">
          <ArtPage name={baseArtName} className="shrink-0 sm:w-1/2" />
          <InfoPage
            campaign={baseInfo.campaign}
            isLive={baseInfo.isLive}
            onOpen={onOpen}
            onResume={onResume}
          />
          {/* The turning leaf. Forward: FRONT = outgoing info (lifts on the
              right), BACK = incoming illustration (lands on the left). Backward
              is the mirror (--rev): FRONT = outgoing illustration (lifts on the
              left), BACK = incoming info. It turns the whole way over. Spread
              only (sm+); phones switch instantly. */}
          {turn && (
            <div
              key={`${turn.from.campaign.id}-${turn.to.campaign.id}`}
              className={cn(
                'grimorio-leaf hidden sm:block',
                turn.dir === -1 && 'grimorio-leaf--rev',
              )}
              style={
                { '--grimorio-turn': turn.fast ? '0.22s' : '0.45s' } as CSSProperties
              }
              onAnimationEnd={onEnd}
            >
              <div className="grimorio-leaf-face grimorio-leaf-front">
                {turn.dir === 1 ? (
                  <InfoPage
                    campaign={turn.from.campaign}
                    isLive={turn.from.isLive}
                    onOpen={NOOP}
                    onResume={NOOP}
                  />
                ) : (
                  <ArtPage
                    name={turn.from.campaign.name}
                    className="h-full w-full"
                  />
                )}
                <span aria-hidden className="grimorio-leaf-shade" />
              </div>
              <div className="grimorio-leaf-face grimorio-leaf-back">
                {turn.dir === 1 ? (
                  <ArtPage
                    name={turn.to.campaign.name}
                    className="h-full w-full"
                  />
                ) : (
                  <InfoPage
                    campaign={turn.to.campaign}
                    isLive={turn.to.isLive}
                    onOpen={NOOP}
                    onResume={NOOP}
                  />
                )}
                <span aria-hidden className="grimorio-leaf-shade" />
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

/** Left leaf — the illustration. Full-bleed on phones; matted on a parchment
 *  margin as the viewport grows (roomy real-book plate). */
function ArtPage({ name, className }: { name: string; className?: string }) {
  const hue = hueFromName(name)
  return (
    <div
      className={cn(
        'grimorio-parchment-bg flex items-center justify-center p-0 sm:p-4 lg:p-7 xl:p-10',
        className,
      )}
    >
      <div
        className="relative aspect-[16/10] w-full overflow-hidden rounded-sm border border-[color:var(--grimorio-parchment-ink)]/35 sm:aspect-auto sm:h-full"
        style={{ background: campaignEmblemGradient(name) }}
      >
        <span
          aria-hidden
          className="absolute inset-0 flex select-none items-center justify-center font-display text-[4.5rem] leading-none text-white/15 sm:text-[7rem] lg:text-[8.5rem]"
          style={{ textShadow: `0 0 48px oklch(0.55 0.15 ${hue} / 0.5)` }}
        >
          {campaignInitials(name)}
        </span>
        <span
          aria-hidden
          className="pointer-events-none absolute inset-0 [box-shadow:inset_0_0_90px_14px_oklch(0_0_0/0.45)]"
        />
      </div>
    </div>
  )
}

/** Right leaf — parchment with dark ink. Margins grow with the viewport; muted
 *  text is opacity (dark scene tokens would vanish on cream), live is crimson. */
function InfoPage({
  campaign,
  isLive,
  onOpen,
  onResume,
}: {
  campaign: Campaign
  isLive: boolean
  onOpen: () => void
  onResume: () => void
}) {
  return (
    <div className="grimorio-parchment-bg flex flex-1 flex-col justify-center gap-4 p-6 sm:p-8 lg:p-12 xl:p-16">
      <StatusLine role={roleLabel(campaign.role)} isLive={isLive} />
      <h2 className="font-display text-2xl uppercase leading-tight tracking-[0.05em] sm:text-[2rem] lg:text-4xl">
        {campaign.name}
      </h2>
      <Synopsis text={campaign.description} />
      {campaign.character && <CharacterRow character={campaign.character} />}
      <Actions isLive={isLive} onOpen={onOpen} onResume={onResume} />
    </div>
  )
}

function StatusLine({ role, isLive }: { role: string; isLive: boolean }) {
  return (
    <p className="flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.22em] opacity-70">
      <span>{role}</span>
      {isLive && (
        <>
          <span aria-hidden>·</span>
          <span className="flex items-center gap-1.5 text-[color:var(--grimorio-crimson)] opacity-100">
            <span className="size-2 animate-pulse rounded-full bg-[color:var(--grimorio-crimson-bright)] motion-reduce:animate-none" />
            Sessão ao vivo
          </span>
        </>
      )}
    </p>
  )
}

function Synopsis({ text }: { text: string | null }) {
  if (!text)
    return (
      <p className="text-sm italic opacity-60">
        Esta crônica ainda não tem uma sinopse.
      </p>
    )
  return <p className="max-w-prose text-sm leading-relaxed opacity-85">{text}</p>
}

/** The caller's own PC in this chronicle — portrait + name + class/level. */
function CharacterRow({
  character,
}: {
  character: NonNullable<Campaign['character']>
}) {
  const classes = character.classes
    .map((c) => `${c.className} ${c.level}`)
    .join(' / ')
  return (
    <div className="flex w-full max-w-xs items-center gap-2 rounded-md border border-[color:var(--grimorio-parchment-ink)]/25 p-2">
      <CharacterPortrait name={character.name} size="sm" />
      <div className="min-w-0">
        <p className="truncate font-medium">{character.name}</p>
        <p className="truncate text-xs opacity-70">
          {classes || `Nv ${character.level}`}
        </p>
      </div>
    </div>
  )
}

function Actions({
  isLive,
  onOpen,
  onResume,
}: {
  isLive: boolean
  onOpen: () => void
  onResume: () => void
}) {
  return (
    <div className="flex flex-wrap items-center gap-2 pt-1">
      {isLive ? (
        <>
          <Button size="lg" onClick={onResume}>
            Continuar a sessão
            <ChevronRight aria-hidden className="ml-1 size-4" />
            <Kbd>⏎</Kbd>
          </Button>
          <Button variant="outline" onClick={onOpen}>
            Abrir crônica <Kbd>O</Kbd>
          </Button>
        </>
      ) : (
        <Button size="lg" onClick={onOpen}>
          Abrir crônica <Kbd>⏎</Kbd>
        </Button>
      )}
    </div>
  )
}
