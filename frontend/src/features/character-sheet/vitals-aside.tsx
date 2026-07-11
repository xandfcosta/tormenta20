import { useQueryClient } from '@tanstack/react-query'
import { useRef } from 'react'
import { useDebouncedCallback } from '@tanstack/react-pacer'
import type { Character, UpdateVitalsInput } from '@/shared/api/api'
import { api } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'
import { accentTitle, panelBg, surface } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { CombatStats, MagicStats } from './combat-magic-stats'
import { ResourceBar } from './resource-bar'

// Guards vitals mutations against races: we snapshot the pre-burst state,
// let onSetCurrent apply optimistic updates, then debounce the network
// send. On failure we roll back to the snapshot (cleared after success).
export function VitalsAside({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const rollbackSnapshot = useRef<Character | undefined>(undefined)

  const sendVitals = useDebouncedCallback(
    async (input: UpdateVitalsInput) => {
      try {
        const updated = await api.characters.updateVitals(character.id, input)
        qc.setQueryData(queryKey, updated)
        invalidateCharacterDependents(qc, character.id)
      } catch {
        if (rollbackSnapshot.current) {
          qc.setQueryData(queryKey, rollbackSnapshot.current)
        }
      } finally {
        rollbackSnapshot.current = undefined
      }
    },
    { wait: 350 },
  )

  const setVital = (
    field: 'hpCurrent' | 'mpCurrent',
    max: number,
    next: number,
  ) => {
    const clamped = Math.max(0, Math.min(max, next))
    if (clamped === character[field]) return
    qc.cancelQueries({ queryKey })
    if (!rollbackSnapshot.current) {
      rollbackSnapshot.current = qc.getQueryData<Character>(queryKey)
    }
    qc.setQueryData<Character>(queryKey, (prev) =>
      prev ? { ...prev, [field]: clamped } : prev,
    )
    sendVitals({ [field]: clamped })
  }

  const setHp = (next: number) => setVital('hpCurrent', character.hpMax, next)
  const setMp = (next: number) => setVital('mpCurrent', character.mpMax, next)

  return (
    <aside
      className={cn(
        'flex min-h-0 flex-col gap-3 overflow-y-auto rounded-xl p-3 sm:p-4',
        surface,
        panelBg,
      )}
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:gap-3 lg:flex-col">
        <ResourceBar
          label="Vida"
          current={character.hpCurrent}
          max={character.hpMax}
          fromColor="from-red-700"
          toColor="to-red-500"
          accent="text-red-700 dark:text-red-300"
          className="sm:flex-1 lg:flex-none"
          onSetCurrent={setHp}
        />
        <ResourceBar
          label="Mana"
          current={character.mpCurrent}
          max={character.mpMax}
          fromColor="from-blue-700"
          toColor="to-blue-500"
          accent="text-blue-700 dark:text-blue-300"
          className="sm:flex-1 lg:flex-none"
          onSetCurrent={setMp}
        />
      </div>

      <div className="grid grid-cols-3 gap-2 sm:grid-cols-6 lg:grid-cols-3">
        <AttributeBox label="FOR" value={character.strength} />
        <AttributeBox label="DES" value={character.dexterity} />
        <AttributeBox label="CON" value={character.constitution} />
        <AttributeBox label="INT" value={character.intelligence} />
        <AttributeBox label="SAB" value={character.wisdom} />
        <AttributeBox label="CAR" value={character.charisma} />
      </div>

      <CombatStats character={character} />
      <MagicStats character={character} />
    </aside>
  )
}

function AttributeBox({ label, value }: { label: string; value: number }) {
  const sign = value >= 0 ? '+' : ''
  return (
    <div
      className={cn(
        'relative rounded-lg border-2 p-2 text-center shadow-inner',
        'border-amber-700/30 bg-gradient-to-b from-amber-100 to-amber-50 dark:border-amber-500/30 dark:from-zinc-900 dark:to-zinc-950',
      )}
    >
      <p className="text-[9px] font-bold uppercase tracking-widest text-amber-800/80 dark:text-amber-400/70">
        {label}
      </p>
      <p
        className={cn(
          'mt-0.5 font-serif text-2xl font-bold leading-none',
          accentTitle,
        )}
      >
        {sign}
        {value}
      </p>
    </div>
  )
}
