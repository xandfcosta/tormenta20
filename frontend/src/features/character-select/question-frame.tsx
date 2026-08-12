/**
 * The "?" that stands in for a portrait that doesn't exist yet — shared by the
 * empty roster and by the trailing create slot (ALE-98), so the two states
 * speak with one voice instead of drifting into two different question marks.
 *
 * Draws only the glyph: the caller owns the frame, because the empty stage uses
 * a plain div and the create slot needs a real button around it.
 *
 * @example <div class="aspect-[3/4] border-2 border-dashed"><QuestionFrame /></div>
 */
export function QuestionFrame() {
  return (
    <span class="flex size-full select-none items-center justify-center font-heading text-7xl text-grimorio-gold/30">
      ?
    </span>
  )
}
