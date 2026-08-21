import { EXPERTISE_NAMES } from '@/shared/api/expertise-names'
import { DEFORMIDADE_PERICIA_BONUS, DEFORMIDADE_SLOTS, deformidadeAvailablePowers } from '@/shared/rules/deformidade'
import { Show } from 'solid-js'
import { raceWithDeformidade } from '@/shared/lib/abilities-cache'
import { tormentaPowersRecord } from '@/shared/lib/rules-catalog-cache'
import { cn } from '@/shared/lib/utils'
import { Select, type SelectOption } from '@/shared/ui/select'
import type { DeformidadeDraft, RaceChoice } from './grant-helpers'

const NONE = ''

export type DeformidadeControlsProps = {
  raceName: string
  choice: RaceChoice
  onChange: (next: RaceChoice) => void
}

/**
 * Deformidade capture (Lefou, p23): two slots of +2 em perícia, the second
 * swappable for a real poder da Tormenta — book-strict, only ONE bônus may be
 * swapped, and the real power costs −1 Carisma (p136). Under-filling is allowed;
 * the sheet finishes what the Forja leaves open.
 */
export function DeformidadeControls(props: DeformidadeControlsProps) {
  const draft = (): DeformidadeDraft => props.choice.deformidade ?? { pericias: [] }
  const swapOn = () => draft().tormentaPower !== undefined

  const set = (next: DeformidadeDraft) =>
    props.onChange({ ...props.choice, deformidade: next })

  const setPericia = (slot: 0 | 1, name: string) => {
    const pericias = [...draft().pericias]
    pericias[slot] = name
    // Drop the holes a cleared first slot would leave: the payload counts
    // perícias by length, and an empty string would buy a bonus of nothing.
    set({ ...draft(), pericias: pericias.filter(Boolean) })
  }

  return (
    <Show when={raceWithDeformidade([props.raceName])}>
      <div class="space-y-1.5">
        <p class="text-2xs font-semibold uppercase tracking-wide text-muted-foreground">
          Deformidade · {DEFORMIDADE_SLOTS} bônus de +{DEFORMIDADE_PERICIA_BONUS}
        </p>

        <PericiaSlot
          slot={0}
          value={draft().pericias[0] ?? NONE}
          exclude={draft().pericias[1]}
          onPick={setPericia}
        />

        <Show
          when={swapOn()}
          fallback={
            <PericiaSlot
              slot={1}
              value={draft().pericias[1] ?? NONE}
              exclude={draft().pericias[0]}
              onPick={setPericia}
            />
          }
        >
          <TormentaPowerSlot draft={draft()} onSet={set} />
        </Show>

        <SwapToggle
          on={swapOn()}
          onToggle={() =>
            swapOn()
              ? set({ pericias: draft().pericias })
              : set({ pericias: draft().pericias.slice(0, 1), tormentaPower: NONE })
          }
        />
        <p class="text-3xs text-muted-foreground">
          Cada bônus conta como poder da Tormenta (exceto para perda de Carisma).
        </p>
      </div>
    </Show>
  )
}

function SlotRow(props: { index: number; children: import('solid-js').JSX.Element }) {
  return (
    <div class="flex items-center gap-2">
      <span
        aria-hidden="true"
        class="w-4 shrink-0 text-center font-mono text-2xs text-muted-foreground"
      >
        {props.index + 1}
      </span>
      {props.children}
    </div>
  )
}

function PericiaSlot(props: {
  slot: 0 | 1
  value: string
  exclude?: string
  onPick: (slot: 0 | 1, name: string) => void
}) {
  const options = (): SelectOption<string>[] => [
    { value: NONE, label: 'Nenhuma' },
    ...EXPERTISE_NAMES.filter((name) => name !== props.exclude).map((name) => ({
      value: name,
      label: name,
    })),
  ]
  const selected = () => options().find((o) => o.value === props.value) ?? null

  return (
    <SlotRow index={props.slot}>
      <Select
        options={options()}
        value={selected()}
        onChange={(option) => props.onPick(props.slot, option?.value ?? NONE)}
        placeholder={`Perícia (+${DEFORMIDADE_PERICIA_BONUS})`}
        size="sm"
        aria-label={`Bônus de perícia ${props.slot + 1}`}
        class="flex-1"
      />
    </SlotRow>
  )
}

function TormentaPowerSlot(props: {
  draft: DeformidadeDraft
  onSet: (next: DeformidadeDraft) => void
}) {
  // Perícia bonuses count as owned powers for prerequisites (p23), so the
  // pickable pool grows with each placed perícia.
  const options = (): SelectOption<string>[] =>
    deformidadeAvailablePowers(tormentaPowersRecord(), props.draft.pericias.length).map(
      (power) => ({ value: power.id, label: power.name }),
    )
  const selected = () => options().find((o) => o.value === props.draft.tormentaPower) ?? null

  return (
    <div class="space-y-1">
      <SlotRow index={1}>
        <Select
          options={options()}
          value={selected()}
          onChange={(option) =>
            props.onSet({ ...props.draft, tormentaPower: option?.value ?? NONE })
          }
          placeholder="Poder da Tormenta"
          size="sm"
          aria-label="Poder da Tormenta"
          class="flex-1"
        />
      </SlotRow>
      <Show when={props.draft.tormentaPower}>
        <p class="pl-6 text-3xs text-[color:var(--hp-hurt)]">
          −1 Carisma (poder da Tormenta, p136)
        </p>
      </Show>
    </div>
  )
}

function SwapToggle(props: { on: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      aria-pressed={props.on}
      onClick={() => props.onToggle()}
      class={cn(
        'flex w-full items-center gap-2 rounded-sm border px-2 py-1 text-left text-2xs transition-colors',
        props.on
          ? 'border-grimorio-gold bg-accent'
          : 'border-grimorio-iron text-muted-foreground hover:bg-accent',
      )}
    >
      <span class="flex size-3.5 shrink-0 items-center justify-center rounded-none border border-grimorio-iron">
        <Show when={props.on}>
          <span class="size-2 rounded-none bg-grimorio-gold" />
        </Show>
      </span>
      Trocar o 2º bônus por um poder da Tormenta
    </button>
  )
}
