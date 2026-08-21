import { initials } from '@/shared/lib/initials'
import { portraitGradient } from './select-helpers'
import { Show } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { Kbd } from '@/shared/ui/kbd'
import { PeekPortrait } from './peek-portrait'
import { primaryRole } from './select-helpers'

export type CharacterStageProps = {
  selected: Character
  prev: Character | null
  next: Character | null
  /** Last navigation direction — the new portrait slides in from that side. */
  direction: 1 | -1
  /** Server-computed defense; null while the sheet is still loading. */
  defense: number | null
  onStep: (delta: 1 | -1) => void
  onOpen: () => void
  onDossier: () => void
  dossierOpen: boolean
}

/**
 * Center stage of the character selector (design "palco + dossiê"): the
 * selected character's portrait on a spotlit pedestal, prev/next peeking
 * dimmed from the sides, engraved Cinzel nameplate + vitals beneath, and the
 * page's single primary CTA. Peeks are restrained coverflow — ±1 at reduced
 * scale, no 3D.
 *
 * `defense` arrives as a prop rather than being read from a sheet hook inside
 * (as the React version did), so the stage stays presentational.
 */
export function CharacterStage(props: CharacterStageProps) {
  const hue = () => hueFromName(props.selected.name)
  // Directional slide for the incoming portrait; the nameplate fades + rises
  // with a small stagger. `animate-in` fires on MOUNT and only on mount, so the
  // two blocks are `keyed` on the character id: swapping a character then
  // REBUILDS them and the entrance replays. Without the key Solid reuses the
  // nodes and the stage swaps in dead silence (ALE-97) — which is what happened
  // once the scene stopped being re-inserted wholesale by the ALE-95 bug.
  // Keyed on the id, not on the character: the computed sheet landing must not
  // count as a selection change. motion-reduce collapses it to instant.
  const slideIn = () => (props.direction === 1 ? 'slide-in-from-right-8' : 'slide-in-from-left-8')

  return (
    <div class="relative flex min-h-0 flex-1 flex-col items-center justify-center gap-4 py-2">
      {/* spotlight wash in the character's hue */}
      <div
        aria-hidden="true"
        class="pointer-events-none absolute inset-0 -z-10 animate-in fade-in-0 duration-500 motion-reduce:animate-none"
        style={{
          background: `radial-gradient(ellipse 60% 50% at 50% 42%, oklch(0.55 0.15 ${hue()} / 0.14), transparent 70%)`,
        }}
      />
      <div class="flex items-center justify-center gap-4 sm:gap-8">
        <PeekPortrait character={props.prev} side="left" onClick={() => props.onStep(-1)} />
        {/* The parameter is load-bearing, not decoration: Solid only re-invokes
            a keyed Show's child when the function DECLARES one (it branches on
            `child.length > 0`). Written `{() => …}` the block silently never
            rebuilds and the entrance never replays. */}
        <Show when={props.selected.id} keyed>
          {(_id) => (
            <div
              class={cn(
                // `flex`, not a plain block: the portrait is a <button>, which
                // is inline-level, so a block wrapper builds a line box around
                // it and adds ~5px of descender space under the card. That made
                // the hero's portrait row 5px taller than the create slot's and
                // knocked the two out of line (ALE-99).
                'flex animate-in fade-in-0 zoom-in-95 duration-300 ease-out motion-reduce:animate-none',
                slideIn(),
              )}
            >
              <StagePortrait character={props.selected} hue={hue()} onOpen={props.onOpen} />
            </div>
          )}
        </Show>
        <PeekPortrait character={props.next} side="right" onClick={() => props.onStep(1)} />
      </div>
      {/* pedestal glow */}
      <div
        aria-hidden="true"
        class="pointer-events-none -mt-6 h-4 w-64 rounded-[100%] blur-md transition-colors duration-300 sm:w-80"
        style={{ background: `oklch(0.5 0.14 ${hue()} / 0.35)` }}
      />
      <Show when={props.selected.id} keyed>
        {(_id) => (
          <div class="flex flex-col items-center gap-4 animate-in fade-in-0 slide-in-from-bottom-2 fill-mode-backwards duration-300 [animation-delay:80ms] motion-reduce:animate-none">
            <Nameplate character={props.selected} hue={hue()} />
            <VitalsRow character={props.selected} defense={props.defense} />
            <SummaryLine character={props.selected} />
          </div>
        )}
      </Show>
      <div class="flex flex-wrap items-center justify-center gap-2">
        <Button size="lg" onClick={() => props.onOpen()}>
          Abrir ficha <Kbd>⏎</Kbd>
        </Button>
        <Button variant="outline" onClick={() => props.onDossier()} aria-pressed={props.dossierOpen}>
          Dossiê <Kbd>D</Kbd>
        </Button>
      </div>
    </div>
  )
}

/** The selected character's 3:4 portrait — hue gradient + giant monogram
 *  until real art lands (art will fill the same frame). */
function StagePortrait(props: { character: Character; hue: number; onOpen: () => void }) {
  return (
    <button
      type="button"
      onClick={() => props.onOpen()}
      aria-label={`Abrir ficha de ${props.character.name}`}
      class="relative aspect-[3/4] w-44 overflow-hidden rounded-sm border-2 border-grimorio-iron transition-transform hover:scale-[1.01] sm:w-56 lg:w-64"
      style={{
        background: portraitGradient(props.character.name),
        // Iron frame + a thin gold filete (the 1px spread), then a soft glow in
        // the character's own hue so the roster still reads authored.
        'box-shadow': `0 0 0 1px oklch(0.8 0.11 85 / 0.55), 0 0 44px oklch(0.55 0.15 ${props.hue} / 0.35)`,
      }}
    >
      <span class="absolute inset-0 flex select-none items-center justify-center font-display text-[7rem] leading-none text-white/20 sm:text-[9rem]">
        {initials(props.character.name)}
      </span>
      <span class="absolute inset-x-0 bottom-0 h-1/4 bg-gradient-to-t from-black/50 to-transparent" />
    </button>
  )
}

/** Engraved Cinzel nameplate + class/race kicker. */
function Nameplate(props: { character: Character; hue: number }) {
  const race = () => props.character.races[0]?.race
  return (
    <div class="max-w-2xl px-4 text-center">
      {/* Two lines are always reserved (`lh` = this element's own line-height,
          so it follows the responsive font size). The stage is a centred
          column: a name wrapping to a second line otherwise pushed the whole
          portrait row 20px up, and arrowing through the roster made the cards
          dance (ALE-99). */}
      <h2
        class="min-h-[2lh] font-display text-2xl uppercase tracking-[0.12em] sm:text-4xl"
        style={{
          'text-shadow': `0 1px 0 oklch(0.2 0 0), 0 0 24px oklch(0.55 0.15 ${props.hue} / 0.35)`,
        }}
      >
        {props.character.name}
      </h2>
      <p class="mt-1 text-[11px] font-semibold uppercase tracking-[0.25em] text-muted-foreground">
        {primaryRole(props.character)}
        {race() ? ` · ${race()}` : ''}
      </p>
    </div>
  )
}

function VitalsRow(props: { character: Character; defense: number | null }) {
  return (
    <div class="flex items-center gap-5 text-sm">
      {/* An em dash while the computed sheet is in flight — never a stale 0,
          which would read as a real (and wrong) defense. */}
      <Vital label="DEF" value={props.defense === null ? '—' : String(props.defense)} />
      <Vital
        label="PV"
        value={`${props.character.hpCurrent}/${props.character.hpMax}`}
        tone="text-[color:var(--hp-full)]"
      />
      <Vital
        label="PM"
        value={`${props.character.mpCurrent}/${props.character.mpMax}`}
        tone="text-[color:var(--mp-arcane)]"
        dim={props.character.mpMax === 0}
      />
    </div>
  )
}

function Vital(props: { label: string; value: string; tone?: string; dim?: boolean }) {
  return (
    <span class={cn('flex items-baseline gap-1.5', props.dim && 'opacity-50')}>
      <span class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">
        {props.label}
      </span>
      <span class={cn('font-mono text-lg font-semibold', props.tone)}>{props.value}</span>
    </span>
  )
}

function SummaryLine(props: { character: Character }) {
  const parts = () =>
    [
      props.character.god ? `Devoto de ${props.character.god}` : null,
      props.character.origin,
      props.character.size,
    ].filter(Boolean)
  return <p class="text-xs text-muted-foreground">{parts().join(' · ')}</p>
}
