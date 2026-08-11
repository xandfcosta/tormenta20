import type { JSX, ParentProps } from 'solid-js'

/**
 * The illuminated head of a tome leaf: a small eyebrow, the title in display
 * caps, and a gilt rule under it. Shared by the campaign leaves so "abrir nova
 * crônica" and "entrar na mesa" read as pages of one book.
 *
 * On a short viewport (a phone held sideways is ~390px tall) the eyebrow goes
 * and the title shrinks — otherwise the heading alone eats half the screen and
 * the form starts below the fold.
 *
 * @example <TomeHeading eyebrow="Folha em branco">Abrir nova crônica</TomeHeading>
 */
export function TomeHeading(props: ParentProps<{ eyebrow: string; icon?: JSX.Element }>) {
  return (
    <header class="space-y-3 text-center [@media(max-height:520px)]:space-y-1.5">
      <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground [@media(max-height:520px)]:hidden">
        {props.eyebrow}
      </p>
      <h1 class="font-display text-3xl uppercase leading-tight tracking-wide text-grimorio-gold sm:text-4xl [@media(max-height:520px)]:text-2xl">
        {props.icon}
        {props.children}
      </h1>
      <div
        aria-hidden="true"
        class="h-px w-full bg-gradient-to-r from-transparent via-grimorio-gold/40 to-transparent"
      />
    </header>
  )
}
