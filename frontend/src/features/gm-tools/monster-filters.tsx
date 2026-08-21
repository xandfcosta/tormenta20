import { Search } from 'lucide-solid'
import { For } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { ND_MAX, ND_MIN, ND_STEP, type MonsterFilterStore } from './monster-filter'
import { MONSTER_TIPOS, MONSTER_TIPO_LABEL } from './monster-format'

/**
 * The bestiary's filter controls. Shared by the Bestiário and by the
 * in-session monster add so both narrow identically — the GM who learned the
 * filter at the table already knows it mid-combat.
 */
export function MonsterFilters(props: {
  filter: MonsterFilterStore
  /** Prefix for the field ids — two mounted copies must not collide. */
  idPrefix: string
}) {
  const criteria = () => props.filter.criteria()

  return (
    <div class="space-y-2">
      <div class="relative">
        <Search
          class="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden="true"
        />
        <Input
          type="search"
          value={criteria().name}
          onInput={(event) => props.filter.setName(event.currentTarget.value)}
          placeholder="Buscar criatura"
          class="pl-8"
          aria-label="Buscar criatura"
        />
      </div>

      <div class="flex flex-wrap items-end gap-2">
        <NdField
          id={`${props.idPrefix}-nd-min`}
          label="ND mínimo"
          value={criteria().ndMin}
          onChange={props.filter.setNdMin}
        />
        <NdField
          id={`${props.idPrefix}-nd-max`}
          label="ND máximo"
          value={criteria().ndMax}
          onChange={props.filter.setNdMax}
        />
      </div>

      <div class="flex flex-wrap gap-1.5">
        <For each={MONSTER_TIPOS}>
          {(tipo) => {
            const on = () => criteria().tipos.has(tipo)
            return (
              <button
                type="button"
                aria-pressed={on()}
                onClick={() => props.filter.toggleTipo(tipo)}
                class={cn(
                  'rounded-sm border px-2 py-0.5 text-[11px] transition-colors',
                  on()
                    ? 'border-grimorio-gold bg-accent font-medium text-grimorio-gold'
                    : 'border-grimorio-iron text-muted-foreground hover:bg-accent',
                )}
              >
                {MONSTER_TIPO_LABEL[tipo]}
              </button>
            )
          }}
        </For>
      </div>
    </div>
  )
}

function NdField(props: {
  id: string
  label: string
  value: number
  onChange: (nd: number) => void
}) {
  return (
    <div class="space-y-1">
      <label
        for={props.id}
        class="block font-heading text-[10px] uppercase tracking-[0.14em] text-muted-foreground"
      >
        {props.label}
      </label>
      <NumberInput
        id={props.id}
        min={ND_MIN}
        max={ND_MAX}
        step={ND_STEP}
        value={props.value}
        onChange={props.onChange}
        class="w-24"
        aria-label={props.label}
        spinnerLabel={props.label.toLowerCase()}
      />
    </div>
  )
}
