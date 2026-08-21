import { For, Show } from 'solid-js'
import { createMediaQuery } from '@/shared/lib/media-query'
import { Skeleton } from '@/shared/ui/skeleton'
import { SHEET_PANELS } from './sheet-sections'

/**
 * The sheet's silhouette while it loads: content beside the block rail over the
 * HUD strip on wide viewports, one block over the bottom bar on a phone.
 *
 * It switches on the SAME media query as `CharacterSheet` itself, so the
 * placeholder and the thing it stands for can never disagree about which layout
 * is coming — a phone showing a desktop skeleton would reflow on arrival.
 *
 * Why a shaped placeholder and not one filled block: it only started rendering
 * at all with ALE-96 (before that the pending read suspended the whole route
 * match and this never painted), and a flat rectangle over the play surface
 * reads as an empty panel mid-session rather than as loading.
 *
 * @example <Show when={sheet()} fallback={<CharacterSheetSkeleton />}>
 */
export function CharacterSheetSkeleton() {
  const isDesktop = createMediaQuery('(min-width: 1024px)')

  return (
    <Show when={isDesktop()} fallback={<PhoneSheetSkeleton />}>
      <DesktopSheetSkeleton />
    </Show>
  )
}

/** Widths that read as prose rather than as a bar chart — no data implied. */
const CONTENT_LINES = ['w-2/5', 'w-4/5', 'w-3/5', 'w-11/12', 'w-1/2', 'w-3/4']

function ContentLines() {
  return (
    <div class="space-y-3 p-3">
      <For each={CONTENT_LINES}>{(width) => <Skeleton class={`h-4 ${width}`} />}</For>
    </div>
  )
}

/** Mirrors `CharacterSheetDesktop`: content + block rail, HUD in the auto row. */
function DesktopSheetSkeleton() {
  return (
    <div class="grid h-full min-h-0 grid-rows-[1fr_auto] gap-3 overflow-hidden">
      <div class="flex min-h-0 items-stretch gap-3">
        <div class="min-w-0 flex-1">
          <ContentLines />
        </div>
        <div class="flex h-full w-32 shrink-0 flex-col gap-1 rounded-md border bg-card p-1">
          <For each={SHEET_PANELS}>{() => <Skeleton class="h-full flex-1" />}</For>
        </div>
      </div>
      <Skeleton class="h-20 rounded-none" />
    </div>
  )
}

/** Mirrors `CharacterSheetMobile`: one block, the HUD, then the bottom bar. */
function PhoneSheetSkeleton() {
  return (
    <div class="flex h-full min-h-0 w-full min-w-0 flex-col">
      <div class="min-h-40 min-w-0 flex-1 overflow-hidden">
        <ContentLines />
      </div>
      <Skeleton class="h-16 shrink-0 rounded-none" />
      <div class="flex h-14 w-full shrink-0 items-stretch gap-0 border-t border-border/60 landscape:h-11">
        <For each={SHEET_PANELS}>
          {() => (
            <div class="flex flex-1 items-center justify-center">
              <Skeleton class="size-5 rounded-none" />
            </div>
          )}
        </For>
      </div>
    </div>
  )
}
