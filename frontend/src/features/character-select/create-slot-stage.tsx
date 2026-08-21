import type { Character } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { Kbd } from '@/shared/ui/kbd'
import { PeekPortrait } from './peek-portrait'
import { QuestionFrame } from './question-frame'
import { SectionLabel } from '@/shared/ui/section-label'

export type CreateSlotStageProps = {
  /** The last hero, peeking from the left — the way back into the roster. */
  prev: Character | null
  onStep: (delta: 1 | -1) => void
  onOpen: () => void
}

/**
 * The stage while the cursor sits on the trailing "+" slot: a dimmed frame with
 * a "?" where a portrait would be, and the single CTA into the Forge (ALE-98).
 *
 * It occupies the same stage position as a real character so arrowing to the
 * end doesn't collapse the layout — the nameplate, vitals and dossier simply
 * have nothing to say yet, which is the point of the empty frame.
 */
export function CreateSlotStage(props: CreateSlotStageProps) {
  return (
    <div class="relative flex min-h-0 flex-1 flex-col items-center justify-center gap-4 py-2">
      <div class="flex items-center justify-center gap-4 sm:gap-8">
        <PeekPortrait character={props.prev} side="left" onClick={() => props.onStep(-1)} />
        <button
          type="button"
          onClick={() => props.onOpen()}
          aria-label="Criar novo personagem"
          class="aspect-[3/4] w-44 rounded-sm border-2 border-dashed border-grimorio-iron transition-colors hover:border-grimorio-gold sm:w-56 lg:w-64"
        >
          <QuestionFrame />
        </button>
        <div class="w-20 sm:w-28 lg:w-32" aria-hidden="true" />
      </div>
      {/* The hero stage has a lit pedestal here. An empty slot has nothing to
          light, but it needs the same box: without it the centred column pulled
          the portrait row 7px down on arrival (ALE-99). Same classes, no glow. */}
      <div aria-hidden="true" class="pointer-events-none -mt-6 h-4 w-64 sm:w-80" />
      <div class="flex flex-col items-center gap-4 animate-in fade-in-0 slide-in-from-bottom-2 fill-mode-backwards duration-300 [animation-delay:80ms] motion-reduce:animate-none">
        <div class="max-w-2xl px-4 text-center">
          <h2 class="min-h-[2lh] font-display text-2xl uppercase tracking-[0.12em] text-muted-foreground sm:text-4xl">
            Novo personagem
          </h2>
          <SectionLabel class="mt-1 font-semibold">
            Uma vaga vazia no grupo
          </SectionLabel>
        </div>
        {/* An empty slot has no vitals and no summary line, and without their
            height the centred column pulled the portrait row down 40px on the
            way in. These two invisible rows keep the caption the same height as
            a hero's, so the cards never move (ALE-99). Mirrors VitalsRow and
            SummaryLine's type scale — change those, change these. */}
        <div aria-hidden="true" class="invisible flex flex-col items-center gap-4">
          <span class="font-mono text-lg font-semibold">0</span>
          <span class="text-xs">.</span>
        </div>
      </div>
      <div class="flex flex-wrap items-center justify-center gap-2">
        <Button size="lg" onClick={() => props.onOpen()}>
          Criar personagem <Kbd>⏎</Kbd>
        </Button>
      </div>
    </div>
  )
}
