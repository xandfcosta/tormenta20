import {
  DEFORMIDADE_PERICIA_BONUS,
  DEFORMIDADE_SLOTS,
  deformidadeAvailablePowers,
  EXPERTISE_NAMES,
} from '@tormenta20/t20-data'
import { raceWithDeformidade } from '@/shared/lib/abilities-cache'
import { Combobox } from '@/shared/ui/combobox'
import { cn } from '@/shared/lib/utils'
import type { DeformidadeDraft, RaceChoice } from './grant-helpers'

/**
 * Deformidade choice capture (Lefou, book p23) — 2 slots of +2 em perícia,
 * with the second slot swappable for a real poder da Tormenta (book-strict:
 * only UM bônus can be swapped; the real power costs −1 Carisma, p136).
 * Under-filling is allowed — GM-negotiated homebrew grants fewer picks.
 * Rendered inside the race detail box, next to the floating attribute picks.
 */
export function DeformidadeControls({
  raceName,
  choice,
  onChange,
}: {
  raceName: string
  choice: RaceChoice
  onChange: (next: RaceChoice) => void
}) {
  if (!raceWithDeformidade([raceName])) return null
  const draft: DeformidadeDraft = choice.deformidade ?? { pericias: [] }
  const swapOn = 'tormentaPower' in draft && draft.tormentaPower !== undefined
  const set = (next: DeformidadeDraft) =>
    onChange({ ...choice, deformidade: next })
  const setPericia = (slot: 0 | 1, name: string) => {
    const pericias = [...draft.pericias]
    pericias[slot] = name
    set({ ...draft, pericias: pericias.filter(Boolean) })
  }
  return (
    <div className="space-y-1.5">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Deformidade · {DEFORMIDADE_SLOTS} bônus de +{DEFORMIDADE_PERICIA_BONUS}
      </p>
      <PericiaSlot
        slot={0}
        value={draft.pericias[0] ?? ''}
        exclude={draft.pericias[1]}
        onPick={setPericia}
      />
      {swapOn ? (
        <TormentaPowerSlot draft={draft} onSet={set} />
      ) : (
        <PericiaSlot
          slot={1}
          value={draft.pericias[1] ?? ''}
          exclude={draft.pericias[0]}
          onPick={setPericia}
        />
      )}
      <SwapToggle
        on={swapOn}
        onToggle={() =>
          swapOn
            ? set({ pericias: draft.pericias })
            : set({ pericias: draft.pericias.slice(0, 1), tormentaPower: '' })
        }
      />
      <p className="text-[10px] text-muted-foreground">
        Cada bônus conta como poder da Tormenta (exceto para perda de Carisma).
      </p>
    </div>
  )
}

function PericiaSlot({
  slot,
  value,
  exclude,
  onPick,
}: {
  slot: 0 | 1
  value: string
  exclude?: string
  onPick: (slot: 0 | 1, name: string) => void
}) {
  const options = EXPERTISE_NAMES.filter((n) => n !== exclude).map((n) => ({
    value: n,
    label: n,
  }))
  return (
    <div className="flex items-center gap-2">
      <span className="w-4 shrink-0 text-center font-mono text-[11px] text-muted-foreground">
        {slot + 1}
      </span>
      <Combobox
        options={options}
        value={value}
        onChange={(v) => onPick(slot, v)}
        placeholder={`Perícia (+${DEFORMIDADE_PERICIA_BONUS})`}
        searchPlaceholder="Buscar perícia…"
        emptyMessage="Nenhuma."
        allowClear
        clearLabel="Nenhuma"
      />
    </div>
  )
}

function TormentaPowerSlot({
  draft,
  onSet,
}: {
  draft: DeformidadeDraft
  onSet: (next: DeformidadeDraft) => void
}) {
  // Perícia bonuses count as owned powers for prerequisites (p23), so the
  // pickable pool grows with each placed perícia.
  const options = deformidadeAvailablePowers(draft.pericias.length).map((p) => ({
    value: p.id,
    label: p.name,
  }))
  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <span className="w-4 shrink-0 text-center font-mono text-[11px] text-muted-foreground">
          2
        </span>
        <Combobox
          options={options}
          value={draft.tormentaPower ?? ''}
          onChange={(v) => onSet({ ...draft, tormentaPower: v || '' })}
          placeholder="Poder da Tormenta"
          searchPlaceholder="Buscar poder…"
          emptyMessage="Nenhum."
        />
      </div>
      {draft.tormentaPower && (
        <p className="pl-6 text-[10px] text-[color:var(--hp-hurt)]">
          −1 Carisma (poder da Tormenta, p136)
        </p>
      )}
    </div>
  )
}

function SwapToggle({ on, onToggle }: { on: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      aria-pressed={on}
      onClick={onToggle}
      className={cn(
        'flex w-full items-center gap-2 rounded-md border px-2 py-1 text-left text-[11px] transition-colors',
        on
          ? 'border-primary bg-accent'
          : 'border-border text-muted-foreground hover:bg-accent',
      )}
    >
      <span className="flex size-3.5 shrink-0 items-center justify-center rounded-sm border border-border">
        {on && <span className="size-2 rounded-sm bg-primary" />}
      </span>
      Trocar o 2º bônus por um poder da Tormenta
    </button>
  )
}
