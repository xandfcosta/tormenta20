/**
 * Character sheet theme tokens — Tailwind class strings shared across
 * sheet panels. Extracted from `routes/characters/$id.tsx` (D-1) so the
 * upcoming D3 split into subcomponents can share the same visual
 * language without each file re-declaring the magic strings.
 *
 * Do not add new visual variants here. When the D0 design-system pass
 * lands (fonts, tokens, semantic vars), these will be replaced by
 * component primitives + CSS variables and this module will go away.
 */

export const surface =
  'border-2 border-amber-700/40 dark:border-amber-500/40'

export const panelBg = 'bg-amber-50/70 dark:bg-zinc-900/40'

export const sheetBg =
  'bg-gradient-to-br from-amber-50 via-orange-50 to-amber-100 text-zinc-900 dark:from-zinc-900 dark:via-zinc-950 dark:to-black dark:text-zinc-100'

export const hoverRow =
  'hover:bg-amber-100/60 dark:hover:bg-zinc-900/60'

export const subtleText = 'text-zinc-600 dark:text-zinc-400'

export const dimText = 'text-zinc-500 dark:text-zinc-500'

export const accentStrong = 'text-amber-800 dark:text-amber-200'

export const accentTitle = 'text-amber-900 dark:text-amber-50'

export const accentBadge =
  'border-amber-700/50 bg-amber-200/60 text-amber-900 hover:bg-amber-200 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-200 dark:hover:bg-amber-500/20'
