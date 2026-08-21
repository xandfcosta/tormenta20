import type { ParentProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * A leaf of the grimório, opened: a leather cover holding one page that fills
 * the scene, with the turn-to animation. Every campaign scene is a page of the
 * same book — the chronicle you're reading (CampaignTome), the blank one you're
 * about to write (nova campanha) and the letter tucked in it (convite).
 *
 * Carries `data-tome-root`, which `createSceneNav` queries as the scene's
 * keyboard-navigation root.
 *
 * @example
 * <TomePage>
 *   <h1>Abrir nova crônica</h1>
 *   <CampaignForm … />
 * </TomePage>
 */
export function TomePage(props: ParentProps<{ class?: string }>) {
  return (
    <div class="w-full">
      <div class="grimorio-leather grimorio-page-in p-2.5 sm:p-3">
        <div
          data-tome-root
          class={cn(
            'grimorio-frame--stone flex min-h-[calc(100dvh-7rem)] flex-col gap-6 rounded-none p-6 sm:p-10',
            // A phone held sideways has ~390px of height; full padding there
            // pushes the content of a short leaf clean off the screen — the
            // "Abrir crônica" button ends at y=389 with this and at ~411
            // without it. The key is width + ORIENTATION and not height
            // (ALE-176): a `max-height` query also matches a phone held
            // UPRIGHT with the virtual keyboard open (390x494), and these
            // leaves host text fields — the spacing collapsed under the
            // finger mid-typing. Same key as ALE-162, and for the same
            // reason: a phone sideways has a tablet's width and a phone's
            // height, which no width query alone can tell apart.
            'max-lg:landscape:gap-3 max-lg:landscape:p-4',
            props.class,
          )}
        >
          {props.children}
        </div>
      </div>
    </div>
  )
}
