import { Backpack, BadgeCheck, BookOpen, ScrollText, Swords, ToggleRight, Zap } from 'lucide-solid'
import type { Component } from 'solid-js'
import { grantedSpells } from '@/entities/character/granted-spells'
import type { Character } from '@/shared/api/api'
import { AbilitiesPanel } from './abilities-panel'
import { AbilitiesPendingBadge } from './abilities-pending-badge'
import { BagPanel } from './bag-panel'
import { CombatPanel } from './combat-panel'
import { EffectsCountBadge } from './effects-count-badge'
import { EffectsPanel } from './effects-panel'
import { isCasterCharacter } from './is-caster'
import { ExpertisesPanel } from './expertises-panel'
import { ProficienciesPanel } from './proficiencies-panel'
import { SpellbookPanel } from './spellbook-panel'

/**
 * One switchable block of the sheet: tab value, short label, icon, optional
 * live badge, and the component that renders it. Shared by the desktop rail and
 * the phone bottom bar so the two layouts stay in lockstep — add a block here
 * and both pick it up.
 *
 * `component` (not a `render(character)` function like the React version): a
 * function called with a value captures that value, so the panel would never
 * see a later character. Rendered through `<Dynamic component={...}
 * character={...} />`, the prop stays a getter and the block tracks edits.
 */
/**
 * O que todo bloco da ficha recebe.
 *
 * `glance` é a ficha aberta pelo MESTRE no painel do combatente: ali ela é para
 * CONSULTAR um número no meio do turno, não para ler a matemática dele. Quem
 * usa hoje é o bloco Perícias, que sem isso gasta duas linhas por perícia
 * repetindo o que o diálogo de decomposição já diz palavra por palavra.
 */
export type SheetPanelProps = {
  character: Character
  glance?: boolean
  /**
   * A ficha desenhada DENTRO de uma sessão ao vivo. Quem lê hoje é o bloco
   * Efeitos: encerrar cena e encerrar dia somem, porque na mesa o descanso é
   * decisão do mestre (ALE-216). Esconder é só UX — quem recusa é o handler.
   */
  inSession?: boolean
}

export type SheetSection = {
  value: string
  label: string
  icon: Component<{ class?: string }>
  badge?: Component<{ character: Character }>
  component: Component<SheetPanelProps>
  /** Dim (but keep) the tab when irrelevant for this character. */
  dim?: (character: Character) => boolean
}

/**
 * Old deep-link tab values → their current home. Equipado + Inventário were
 * merged into the Mochila bag; `?tab=inventory` bookmarks must keep landing on
 * items, not fall back to the first block.
 */
export function resolveSheetTab(tab: string): string {
  if (tab === 'inventory' || tab === 'equipment') return 'bag'
  return tab
}

/**
 * The non-vitals blocks: PV/PM live in the HUD (or, no painel do combatente, na
 * `CombatantBand`), never here. Campaign membership is managed on the campaign
 * screens (ALE-32), so there is no "Campanhas" block.
 *
 * The React file wrapped every panel in `memo()` — its comment explains that
 * without it a tab switch re-ran EVERY mounted panel, costing 250–540ms. There
 * is no such wrapper here: a switch touches only what actually reads the signal
 * that changed. Whether the blocks should nonetheless stay MOUNTED between
 * switches is a separate, measurable question — answered with numbers in
 * ALE-83, not assumed here.
 */
export const SHEET_PANELS: SheetSection[] = [
  {
    value: 'expertises',
    label: 'Perícias',
    icon: ScrollText,
    component: ExpertisesPanel,
  },
  // Defesa, ataques, resistências, atributos e arma. Não tinham bloco nenhum
  // até a ALE-145: viviam só no `CharacterHud`, que os esconde abaixo de `md`.
  {
    value: 'combat',
    label: 'Combate',
    icon: Swords,
    component: CombatPanel,
  },
  {
    value: 'bag',
    label: 'Mochila',
    icon: Backpack,
    component: BagPanel,
  },
  {
    value: 'proficiencies',
    label: 'Proficiências',
    icon: BadgeCheck,
    component: ProficienciesPanel,
  },
  {
    value: 'conditionals',
    label: 'Efeitos',
    icon: ToggleRight,
    badge: EffectsCountBadge,
    component: EffectsPanel,
  },
  // Value stays 'abilities' — ?tab=abilities deep links must survive the
  // Habilidades→Poderes rename.
  {
    value: 'abilities',
    label: 'Poderes',
    icon: Zap,
    badge: AbilitiesPendingBadge,
    component: AbilitiesPanel,
  },
  {
    value: 'spells',
    label: 'Magias',
    icon: BookOpen,
    // Dim only when there is truly nothing castable — a Bárbaro with Totem
    // Espiritual owns a granted spell and must see the tab lit.
    dim: (c) => !isCasterCharacter(c) && grantedSpells(c).length === 0,
    component: SpellbookPanel,
  },
]
