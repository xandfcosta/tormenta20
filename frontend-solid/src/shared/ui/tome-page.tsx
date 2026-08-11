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
            'grimorio-frame--stone flex min-h-[calc(100dvh-7rem)] flex-col gap-6 rounded-sm p-6 sm:p-10',
            // A phone held sideways has ~390px of height; full padding there
            // pushes the content of a short leaf clean off the screen.
            '[@media(max-height:520px)]:gap-3 [@media(max-height:520px)]:p-4',
            props.class,
          )}
        >
          {props.children}
        </div>
      </div>
    </div>
  )
}
