import { Check } from 'lucide-react'
import { useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { cn } from '@/shared/lib/utils'
import { raceAttributeDeltas } from '@/features/character-build/grant-helpers'
import {
  bandPicksRemaining,
  type PericiaPlan,
  periciaPlan,
} from '@/features/character-build/pericia-helpers'
import { useCreationWizard } from '@/features/character-build/creation-wizard-context'

export function PericiasStep() {
  const { form, raceChoices } = useCreationWizard()
  const primaryName = form.state.values.classes[0]?.className as
    | string
    | undefined

  // Seed the class's fixed (auto-trained) perícias into the trained set once
  // the primary class is known, so they persist without a manual toggle.
  useEffect(() => {
    if (!primaryName) return
    const plan = periciaPlan(primaryName, 0)
    if (!plan) return
    const cur = form.state.values.trainedExpertises as string[]
    const missing = plan.fixed.filter((f) => !cur.includes(f))
    if (missing.length > 0) {
      form.setFieldValue('trainedExpertises', [...cur, ...missing])
    }
  }, [primaryName, form])

  return (
    <form.Subscribe
      selector={(s: {
        values: {
          classes: { className: string; level: number }[]
          races: string[]
          intelligence: number
          trainedExpertises: string[]
        }
      }) => s.values}
    >
      {(v: {
        classes: { className: string; level: number }[]
        races: string[]
        intelligence: number
        trainedExpertises: string[]
      }) => {
        const primary = v.classes[0]
        const intMod =
          v.intelligence + (raceAttributeDeltas(v.races, raceChoices).intelligence ?? 0)
        const plan = primary?.className
          ? periciaPlan(primary.className, intMod)
          : null
        const trained = v.trainedExpertises
        const set = (next: string[]) =>
          form.setFieldValue('trainedExpertises', next)
        const toggle = (name: string) =>
          set(
            trained.includes(name)
              ? trained.filter((x) => x !== name)
              : [...trained, name],
          )

        return (
          <Card>
            <CardHeader>
              <CardTitle className="font-display tracking-wide">
                Perícias
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {!plan ? (
                <p className="text-sm text-muted-foreground">
                  Selecione uma classe primeiro (etapa Classe).
                </p>
              ) : (
                <PericiaPicker
                  plan={plan}
                  trained={trained}
                  onToggle={toggle}
                  onSet={set}
                />
              )}
            </CardContent>
          </Card>
        )
      }}
    </form.Subscribe>
  )
}

function PericiaPicker({
  plan,
  trained,
  onToggle,
  onSet,
}: {
  plan: PericiaPlan
  trained: string[]
  onToggle: (name: string) => void
  onSet: (next: string[]) => void
}) {
  const trainedSet = new Set(trained)

  const pickEitherOr = (chosen: string, other: string) =>
    onSet([...trained.filter((x) => x !== other && x !== chosen), chosen])

  return (
    <div className="space-y-4">
      <div className="space-y-1.5">
        <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
          Treinadas (fixas)
        </p>
        <div className="flex flex-wrap gap-1.5">
          {plan.fixed.map((name) => (
            <span
              key={name}
              className="rounded-md border border-primary bg-accent px-2 py-1 text-xs"
            >
              {name}
            </span>
          ))}
        </div>
      </div>

      {plan.eitherOr && (
        <div className="space-y-1.5">
          <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
            Escolha uma
          </p>
          <div className="flex gap-2">
            {plan.eitherOr.map((name, i) => {
              const other = plan.eitherOr?.[i === 0 ? 1 : 0] ?? ''
              const selected = trainedSet.has(name)
              return (
                <button
                  key={name}
                  type="button"
                  onClick={() => pickEitherOr(name, other)}
                  className={cn(
                    'rounded-md border px-3 py-1.5 text-xs transition-colors',
                    selected
                      ? 'border-primary bg-accent'
                      : 'border-border hover:bg-accent',
                  )}
                >
                  {name}
                </button>
              )
            })}
          </div>
        </div>
      )}

      <PericiaBand
        label={`Da classe · escolha ${plan.classCount}`}
        pool={plan.classPool}
        count={plan.classCount}
        trainedSet={trainedSet}
        onToggle={onToggle}
      />

      {plan.intCount > 0 && (
        <PericiaBand
          label={`Por Inteligência · escolha ${plan.intCount}`}
          subtitle="Qualquer perícia fora da lista da classe."
          accent
          pool={plan.intPool}
          count={plan.intCount}
          trainedSet={trainedSet}
          onToggle={onToggle}
        />
      )}
    </div>
  )
}

/** One capped pick-band: a header count + a scrollable checkbox grid over its
 *  own pool. Reused for the class list and the +INT bonus. */
function PericiaBand({
  label,
  subtitle,
  accent,
  pool,
  count,
  trainedSet,
  onToggle,
}: {
  label: string
  subtitle?: string
  accent?: boolean
  pool: string[]
  count: number
  trainedSet: Set<string>
  onToggle: (name: string) => void
}) {
  const picked = pool.filter((p) => trainedSet.has(p)).length
  const remaining = bandPicksRemaining(pool, count, [...trainedSet])

  return (
    <div
      className={cn(
        'space-y-1.5',
        accent && 'border-l-2 border-primary/50 pl-3',
      )}
    >
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        {accent && <span className="mr-1 text-[color:var(--primary)]">✦</span>}
        {label} ({picked}/{count})
      </p>
      {subtitle && (
        <p className="text-[11px] normal-case text-muted-foreground/80">
          {subtitle}
        </p>
      )}
      <div className="grid max-h-[min(220px,28vh)] grid-cols-2 gap-1.5 overflow-y-auto p-0.5 sm:grid-cols-3">
        {pool.map((name) => {
          const selected = trainedSet.has(name)
          const locked = !selected && remaining === 0
          return (
            <button
              key={name}
              type="button"
              disabled={locked}
              onClick={() => onToggle(name)}
              className={cn(
                'flex items-center gap-1.5 rounded-md border px-2 py-1 text-left text-xs transition-colors',
                selected
                  ? 'border-primary bg-accent'
                  : locked
                    ? 'border-border opacity-40'
                    : 'border-border hover:bg-accent',
              )}
            >
              <span className="flex size-3.5 shrink-0 items-center justify-center rounded-sm border border-border">
                {selected && <Check className="size-2.5 text-primary" />}
              </span>
              <span className="truncate">{name}</span>
            </button>
          )
        })}
      </div>
      {remaining > 0 && (
        <p className="text-[11px] text-[color:var(--hp-hurt)]">
          Faltam {remaining} perícias — pode terminar depois na ficha.
        </p>
      )}
    </div>
  )
}
