import { ATTRIBUTE_ABBR, ATTRIBUTE_KEYS, type AttributeKey } from '@tormenta20/t20-data'
import { cn } from '@/shared/lib/utils'
import {
  type RaceChoice,
  type RaceChoiceMeta,
  raceChoiceMeta,
  resolveRaceDeltas,
} from './grant-helpers'
import { DeltaBadges } from './grant-panels'

/**
 * Inline attribute-choice capture for a selected race. Floating races (Humano,
 * Lefou, Osteon, Sereia) get +1 attribute pills with the excluded attribute
 * disabled and the guaranteed penalty shown; subrace races (Suraggel) get
 * ascendência cards. Fixed races render nothing.
 */
export function RaceChoiceControls({
  raceName,
  choice,
  onChange,
}: {
  raceName: string
  choice: RaceChoice
  onChange: (next: RaceChoice) => void
}) {
  const meta = raceChoiceMeta(raceName)
  if (meta.kind === 'none') return null
  if (meta.kind === 'floating') {
    return <FloatingPicker meta={meta} choice={choice} onChange={onChange} />
  }
  return (
    <SubracePicker
      raceName={raceName}
      options={meta.options}
      choice={choice}
      onChange={onChange}
    />
  )
}

function FloatingPicker({
  meta,
  choice,
  onChange,
}: {
  meta: Extract<RaceChoiceMeta, { kind: 'floating' }>
  choice: RaceChoice
  onChange: (next: RaceChoice) => void
}) {
  const picks = choice.floatingPicks ?? []
  const placed = picks.filter((a) => a !== meta.exclude).length
  const toggle = (attr: AttributeKey) => {
    if (attr === meta.exclude) return
    if (picks.includes(attr)) {
      onChange({ ...choice, floatingPicks: picks.filter((a) => a !== attr) })
    } else if (placed < meta.count) {
      onChange({ ...choice, floatingPicks: [...picks, attr] })
    }
  }
  return (
    <div className="space-y-1.5">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Distribua +{meta.value} em {meta.count} atributos · {placed}/{meta.count}
      </p>
      <div className="flex flex-wrap gap-1.5">
        {ATTRIBUTE_KEYS.map((attr) => {
          const excluded = attr === meta.exclude
          const selected = picks.includes(attr)
          const full = placed >= meta.count && !selected
          return (
            <button
              key={attr}
              type="button"
              disabled={excluded || full}
              onClick={() => toggle(attr)}
              title={
                excluded ? `Não pode aumentar ${ATTRIBUTE_ABBR[attr]}` : undefined
              }
              className={cn(
                'rounded-md border px-2 py-1 font-mono text-xs transition-colors',
                selected ? 'border-primary bg-accent' : 'border-border',
                (excluded || full) && 'opacity-40',
              )}
            >
              {ATTRIBUTE_ABBR[attr]}
            </button>
          )
        })}
      </div>
      {meta.penalty && (
        <p className="text-[11px] text-muted-foreground">
          Penalidade fixa:{' '}
          <span className="font-mono">
            −{Math.abs(meta.penalty.value)} {ATTRIBUTE_ABBR[meta.penalty.attribute]}
          </span>
        </p>
      )}
    </div>
  )
}

function SubracePicker({
  raceName,
  options,
  choice,
  onChange,
}: {
  raceName: string
  options: string[]
  choice: RaceChoice
  onChange: (next: RaceChoice) => void
}) {
  return (
    <div className="space-y-1.5">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Escolha a ascendência
      </p>
      <div className="grid gap-1.5 sm:grid-cols-2">
        {options.map((opt) => {
          const selected = choice.ascendencia === opt
          return (
            <button
              key={opt}
              type="button"
              onClick={() => onChange({ ...choice, ascendencia: opt })}
              className={cn(
                'space-y-1 rounded-md border p-2 text-left transition-colors',
                selected ? 'border-primary bg-accent' : 'border-border hover:bg-accent',
              )}
            >
              <p className="text-xs font-semibold capitalize">{opt}</p>
              <DeltaBadges deltas={resolveRaceDeltas(raceName, { ascendencia: opt })} />
            </button>
          )
        })}
      </div>
    </div>
  )
}
