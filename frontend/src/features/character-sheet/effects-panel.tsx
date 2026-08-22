import type { SheetPanelProps } from './sheet-sections'
import { ActiveEffectsSection } from './active-effects-section'
import { ConditionsSection } from './conditions-section'
import { SituationalSection } from './situational-section'
import { StancesSection } from './stances-section'
import { SectionTitle } from '@/shared/ui/section-label'
import { Panel } from '@/shared/ui/panel'

/**
 * The Efeitos block — everything currently changing this character's numbers,
 * in three categories that differ by WHO owns the state:
 *
 *  1. `ConditionsSection` — book conditions (p394-395). Server state, and they
 *     move the numbers: a condition that is only a badge was the ALE-28 bug.
 *  2. `ActiveEffectsSection` — used consumables and applied spell buffs, with
 *     a scene/day scope the `/active-effects` endpoints own.
 *  3. `StancesSection` — the power stances currently running, with what each
 *     one cost. Their ON switch lives in the Poderes block, where the PM is
 *     charged; here they can only be read and ended.
 *  4. `SituationalSection` — opt-in context modifiers (terrain, target type,
 *     homebrew item toggles). Client state, per character, in localStorage.
 */
export function EffectsPanel(props: SheetPanelProps) {
  return (
    <Panel as="section" fillHeight>
      <div class="shrink-0 border-b border-grimorio-iron px-3 py-2 sm:px-4">
        <SectionTitle contexto="painel">Efeitos</SectionTitle>
      </div>
      <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-3 sm:p-4">
        <ConditionsSection character={props.character} />
        <StancesSection character={props.character} />
        <ActiveEffectsSection character={props.character} inSession={props.inSession} />
        <SituationalSection character={props.character} />
      </div>
    </Panel>
  )
}
