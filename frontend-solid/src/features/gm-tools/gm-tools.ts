/**
 * The Mesa do Mestre's tools, in rail order. One scene with a rail instead of
 * the React app's hub-of-cards: the GM switches tools MID-COMBAT, and a hub
 * charges a round trip for every switch.
 *
 * "Improviso" is the Cap 6 tables and the dungeon generator together — the d20
 * dungeon idea is a Cap 6 table that appeared in both, and one home for it is
 * one place to look.
 */
export const GM_TOOLS = [
  {
    slug: 'bestiario',
    label: 'Bestiário',
    hint: 'Criaturas por ND, tipo e tamanho, com ataques e habilidades.',
  },
  {
    slug: 'encontros',
    label: 'Encontros',
    hint: 'ND do combate e XP do grupo, combinando criaturas.',
  },
  {
    slug: 'improviso',
    label: 'Improviso',
    hint: 'Tabelas do Cap 6 e estrutura de masmorra na hora.',
  },
  {
    slug: 'catalogos',
    label: 'Catálogos',
    hint: 'Condições, magias, poderes e itens numa busca só.',
  },
] as const

export type ToolSlug = (typeof GM_TOOLS)[number]['slug']

/** The first tool — where `/gm` lands and where an unknown slug is sent. */
export const DEFAULT_TOOL: ToolSlug = GM_TOOLS[0].slug

/** Whether a string names a tool — for a slug arriving from the URL. */
export function isToolSlug(value: string): value is ToolSlug {
  return GM_TOOLS.some((tool) => tool.slug === value)
}

/** The tool's label, for the header and the document title. */
export function toolLabel(slug: ToolSlug): string {
  return GM_TOOLS.find((tool) => tool.slug === slug)?.label ?? ''
}
