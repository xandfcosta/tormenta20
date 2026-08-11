import { ChevronRight } from 'lucide-solid'
import { Show } from 'solid-js'
import type { Campaign } from '@/shared/api/api'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { Kbd } from '@/shared/ui/kbd'
import { campaignEmblemGradient, campaignInitials, roleLabel } from './campaign-select-helpers'
import { createPageTurns } from './page-turns'

const NOOP = () => {}

export type CampaignBookProps = {
  campaign: Campaign
  isLive: boolean
  /** The markers' current order (ids) — gives each turn its direction. */
  orderIds: number[]
  onOpen: () => void
  onResume: () => void
}

/**
 * The focused chronicle as an open tome: a leather cover holding two leaves
 * split by a shaded spine. The left leaf is the illustration (a hue emblem
 * until real cover art lands); the right leaf is the info + primary action.
 * Page margins grow with the viewport. Switching chronicles turns a
 * double-sided leaf the whole way over the spine (queued via `createPageTurns`).
 * On phones the leaves switch instantly.
 */
export function CampaignBook(props: CampaignBookProps) {
  const { shown, turn, finishTurn } = createPageTurns(
    () => ({ campaign: props.campaign, isLive: props.isLive }),
    () => props.orderIds,
  )

  const forward = () => {
    const t = turn()
    return !t || t.dir === 1
  }
  // Base spread. Idle: both pages = shown. Turning: the not-yet-covered page
  // keeps the OLD content and the revealed page is already NEW (the leaf brings
  // the new one down over the old, so nothing mismatches). Forward reveals the
  // info (right) and keeps the old illustration; backward mirrors it.
  const baseArtName = () => {
    const t = turn()
    if (!t) return shown().campaign.name
    return (forward() ? t.from : t.to).campaign.name
  }
  const baseInfo = () => {
    const t = turn()
    if (!t) return shown()
    return forward() ? t.to : t.from
  }

  const onAnimationEnd = (event: AnimationEvent) => {
    if (
      event.animationName === 'grimorio-leaf-turn' ||
      event.animationName === 'grimorio-leaf-turn-rev'
    ) {
      finishTurn()
    }
  }

  return (
    <div class="flex min-h-0 flex-1 items-center justify-center py-1">
      <div class="grimorio-book w-full p-2.5 sm:p-3">
        <div class="grimorio-book-pages flex flex-col overflow-hidden sm:min-h-[26rem] sm:flex-row sm:items-stretch lg:min-h-[32rem] xl:min-h-[36rem]">
          <ArtPage name={baseArtName()} class="shrink-0 sm:w-1/2" />
          <InfoPage
            campaign={baseInfo().campaign}
            isLive={baseInfo().isLive}
            onOpen={props.onOpen}
            onResume={props.onResume}
          />
          {/* The turning leaf. Forward: FRONT = outgoing info (lifts on the
              right), BACK = incoming illustration (lands on the left). Backward
              is the mirror (--rev). Spread only (sm+); phones switch instantly. */}
          <Show when={turn()}>
            {(activeTurn) => (
              <div
                class={cn(
                  'grimorio-leaf hidden sm:block',
                  activeTurn().dir === -1 && 'grimorio-leaf--rev',
                )}
                style={{ '--grimorio-turn': activeTurn().fast ? '0.22s' : '0.45s' }}
                on:animationend={onAnimationEnd}
              >
                <div class="grimorio-leaf-face grimorio-leaf-front">
                  <Show
                    when={activeTurn().dir === 1}
                    fallback={<ArtPage name={activeTurn().from.campaign.name} class="h-full w-full" />}
                  >
                    <InfoPage
                      campaign={activeTurn().from.campaign}
                      isLive={activeTurn().from.isLive}
                      onOpen={NOOP}
                      onResume={NOOP}
                    />
                  </Show>
                  <span aria-hidden="true" class="grimorio-leaf-shade" />
                </div>
                <div class="grimorio-leaf-face grimorio-leaf-back">
                  <Show
                    when={activeTurn().dir === 1}
                    fallback={
                      <InfoPage
                        campaign={activeTurn().to.campaign}
                        isLive={activeTurn().to.isLive}
                        onOpen={NOOP}
                        onResume={NOOP}
                      />
                    }
                  >
                    <ArtPage name={activeTurn().to.campaign.name} class="h-full w-full" />
                  </Show>
                  <span aria-hidden="true" class="grimorio-leaf-shade" />
                </div>
              </div>
            )}
          </Show>
        </div>
      </div>
    </div>
  )
}

/** Left leaf — the illustration. Full-bleed on phones; matted on a parchment
 *  margin as the viewport grows (roomy real-book plate). */
function ArtPage(props: { name: string; class?: string }) {
  const hue = () => hueFromName(props.name)
  return (
    <div
      class={cn(
        'grimorio-parchment-bg flex items-center justify-center p-0 sm:p-4 lg:p-7 xl:p-10',
        props.class,
      )}
    >
      <div
        class="relative aspect-[16/10] w-full overflow-hidden rounded-sm border border-[color:var(--grimorio-parchment-ink)]/35 sm:aspect-auto sm:h-full"
        style={{ background: campaignEmblemGradient(props.name) }}
      >
        <span
          aria-hidden="true"
          class="absolute inset-0 flex select-none items-center justify-center font-display text-[4.5rem] leading-none text-white/15 sm:text-[7rem] lg:text-[8.5rem]"
          style={{ 'text-shadow': `0 0 48px oklch(0.55 0.15 ${hue()} / 0.5)` }}
        >
          {campaignInitials(props.name)}
        </span>
        <span
          aria-hidden="true"
          class="pointer-events-none absolute inset-0 [box-shadow:inset_0_0_90px_14px_oklch(0_0_0/0.45)]"
        />
      </div>
    </div>
  )
}

/** Right leaf — parchment with dark ink. Muted text is opacity (dark scene
 *  tokens would vanish on cream); live is crimson. */
function InfoPage(props: {
  campaign: Campaign
  isLive: boolean
  onOpen: () => void
  onResume: () => void
}) {
  return (
    <div class="grimorio-parchment-bg flex flex-1 flex-col justify-center gap-4 p-6 sm:p-8 lg:p-12 xl:p-16">
      <StatusLine role={roleLabel(props.campaign.role)} isLive={props.isLive} />
      <h2 class="font-display text-2xl uppercase leading-tight tracking-[0.05em] sm:text-[2rem] lg:text-4xl">
        {props.campaign.name}
      </h2>
      <Synopsis text={props.campaign.description} />
      <Show when={props.campaign.character}>
        {(character) => <CharacterRow character={character()} />}
      </Show>
      <Actions isLive={props.isLive} onOpen={props.onOpen} onResume={props.onResume} />
    </div>
  )
}

function StatusLine(props: { role: string; isLive: boolean }) {
  return (
    <p class="flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.22em] opacity-70">
      <span>{props.role}</span>
      <Show when={props.isLive}>
        <span aria-hidden="true">·</span>
        <span class="flex items-center gap-1.5 text-[color:var(--grimorio-crimson)] opacity-100">
          <span class="size-2 animate-pulse rounded-full bg-[color:var(--grimorio-crimson-bright)] motion-reduce:animate-none" />
          Sessão ao vivo
        </span>
      </Show>
    </p>
  )
}

function Synopsis(props: { text: string | null }) {
  return (
    <Show
      when={props.text}
      fallback={
        <p class="text-sm italic opacity-60">Esta crônica ainda não tem uma sinopse.</p>
      }
    >
      {(text) => <p class="max-w-prose text-sm leading-relaxed opacity-85">{text()}</p>}
    </Show>
  )
}

/** The caller's own PC in this chronicle — portrait + name + class/level. */
function CharacterRow(props: { character: NonNullable<Campaign['character']> }) {
  const classes = () =>
    props.character.classes.map((c) => `${c.className} ${c.level}`).join(' / ')
  return (
    <div class="flex w-full max-w-xs items-center gap-2 rounded-md border border-[color:var(--grimorio-parchment-ink)]/25 p-2">
      <CharacterPortrait name={props.character.name} size="sm" />
      <div class="min-w-0">
        <p class="truncate font-medium">{props.character.name}</p>
        <p class="truncate text-xs opacity-70">
          {classes() || `Nv ${props.character.level}`}
        </p>
      </div>
    </div>
  )
}

function Actions(props: { isLive: boolean; onOpen: () => void; onResume: () => void }) {
  return (
    <div class="flex flex-wrap items-center gap-2 pt-1">
      <Show
        when={props.isLive}
        fallback={
          <Button size="lg" onClick={() => props.onOpen()}>
            Abrir crônica <Kbd>⏎</Kbd>
          </Button>
        }
      >
        <Button size="lg" onClick={() => props.onResume()}>
          Continuar a sessão
          <ChevronRight aria-hidden="true" class="ml-1 size-4" />
          <Kbd>⏎</Kbd>
        </Button>
        <Button variant="outline" onClick={() => props.onOpen()}>
          Abrir crônica <Kbd>O</Kbd>
        </Button>
      </Show>
    </div>
  )
}
