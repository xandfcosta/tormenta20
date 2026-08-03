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

/**
 * Inventory row action cluster: 5 icon slots (info, overlays, usar, editar,
 * remover) of size-7 (1.75rem) + 4 gap-2 (0.5rem) = 10.75rem. Rows and the
 * column header both reserve this width so the qtd/esp/total/equipar columns
 * stay aligned no matter how many buttons a given row actually renders
 * ("Usar" only exists on consumables). Fixed from sm: up — the phone layout
 * hides the numeric columns, so there alignment doesn't apply.
 */
export const INVENTORY_ACTIONS_WIDTH = 'sm:w-[10.75rem] shrink-0'
