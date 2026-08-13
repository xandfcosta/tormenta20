import { Crosshair, Shield, ShieldCheck, Sword } from 'lucide-solid'
import { For, createMemo } from 'solid-js'
import { computedSheetFor, requireExpertise } from '@/entities/character/computed-sheet'
import { ATTRIBUTE_ABBR } from '@/entities/character/expertise'
import type { Character } from '@/shared/api/api'
import { StatBox } from './stat-box'
import { SAVES, defenseRows, expertiseRows } from './stat-rows'

export type StatsProps = {
  character: Character
  activeConditionals: ReadonlySet<string>
}

/**
 * Defesa + the two attack numbers, each opening its own breakdown. These are
 * the tiles a player looks at mid-turn, so they live in the always-visible HUD
 * rather than behind a block.
 */
export function CombatStats(props: StatsProps) {
  const sheet = createMemo(() => computedSheetFor(props.character, props.activeConditionals))
  const luta = () => requireExpertise(sheet(), 'Luta', 'strength')
  const pontaria = () => requireExpertise(sheet(), 'Pontaria', 'dexterity')
  const rd = () => sheet().damageReduction

  return (
    <div class="grid grid-cols-3 gap-2">
      <StatBox
        label="Defesa"
        value={sheet().defense.total}
        rows={defenseRows(sheet())}
        icon={Shield}
        sub={rd().total > 0 ? `RD ${rd().total}` : undefined}
        extra={
          rd().total > 0
            ? {
                title: `Redução de dano ${rd().total}`,
                rows: rd().sources.map((s) => ({ label: s.source, amount: s.amount })),
              }
            : undefined
        }
      />
      <StatBox
        label="Atq CaC"
        dialogTitle="Ataque Corpo a Corpo (Luta)"
        value={luta().total + sheet().attackAll.total}
        rows={expertiseRows(luta(), ATTRIBUTE_ABBR[luta().attribute], sheet().attackAll)}
        icon={Sword}
        signedValue
      />
      <StatBox
        label="Atq Dist"
        dialogTitle="Ataque à Distância (Pontaria)"
        value={pontaria().total + sheet().attackAll.total}
        rows={expertiseRows(pontaria(), ATTRIBUTE_ABBR[pontaria().attribute], sheet().attackAll)}
        icon={Crosshair}
        signedValue
      />
    </div>
  )
}

/**
 * Fortitude / Reflexos / Vontade — "teste de Reflexos CD 20!" is the hottest
 * call at the table, so the triple is promoted to the HUD with the same
 * breakdown dialogs as the attack tiles.
 */
export function SavesStats(props: StatsProps) {
  const sheet = createMemo(() => computedSheetFor(props.character, props.activeConditionals))
  return (
    <div class="grid grid-cols-3 gap-2">
      <For each={SAVES}>
        {(save) => {
          const total = () => requireExpertise(sheet(), save.name, save.attribute)
          return (
            <StatBox
              label={save.name.slice(0, 4)}
              dialogTitle={save.name}
              value={total().total}
              rows={expertiseRows(total(), save.abbr)}
              icon={ShieldCheck}
              signedValue
            />
          )
        }}
      </For>
    </div>
  )
}
