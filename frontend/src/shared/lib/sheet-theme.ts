/**
 * Sheet theme tokens — now plain shadcn class strings. The custom "Controlled
 * Decay" amber palette was stripped (functional focus); these map onto default
 * shadcn semantic classes so the ~24 importing files render plain without a
 * per-file rewrite. Being inlined + removed progressively.
 */

export const surface = 'border'

export const panelBg = 'bg-card'

export const sheetBg = 'bg-background'

export const hoverRow = 'hover:bg-accent'

export const subtleText = 'text-muted-foreground'

export const dimText = 'text-muted-foreground'

export const accentStrong = 'text-foreground font-semibold'

export const accentTitle = 'text-foreground'

export const accentBadge = ''

export const selectClass =
  'cursor-pointer rounded-md border border-input bg-transparent outline-none focus:ring-2 focus:ring-ring'
