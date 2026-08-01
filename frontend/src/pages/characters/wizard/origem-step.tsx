import { Check } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Combobox } from '@/shared/ui/combobox'
import { Field, FieldError } from '@/shared/ui/field'
import { cn } from '@/shared/lib/utils'
import { originGrant } from '@/features/character-build/grant-helpers'
import { powerPickOptions } from '@/features/character-build/class-power-helpers'
import {
  type FieldApi,
  useCreationWizard,
} from '@/features/character-build/creation-wizard-context'
import { toOptions } from '@/features/character-build/wizard-steps'

const ORIGIN_BENEFIT_CAP = 2

export function OrigemStep() {
  const { form, options } = useCreationWizard()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Origem</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <form.Field name="origin">
          {(f: FieldApi) => {
            const invalid = f.state.meta.isTouched && !f.state.meta.isValid
            return (
              <Field data-invalid={invalid} className="sm:max-w-xs">
                <Combobox
                  id={f.name}
                  options={toOptions(options.origins)}
                  value={f.state.value as string}
                  onChange={(val: string) => {
                    f.handleChange(val)
                    // reset benefit picks when the origin changes
                    form.setFieldValue('originChoices', [])
                  }}
                  placeholder="Selecionar origem"
                  searchPlaceholder="Buscar origens…"
                  emptyMessage="Nenhuma origem encontrada."
                />
                {invalid && <FieldError errors={f.state.meta.errors} />}
              </Field>
            )
          }}
        </form.Field>

        <form.Subscribe
          selector={(s: {
            values: {
              origin: string
              originChoices: string[]
              powerChoices: Record<string, string[]>
            }
          }) => ({
            origin: s.values.origin,
            picks: s.values.originChoices,
            powerChoices: s.values.powerChoices ?? {},
          })}
        >
          {({
            origin,
            picks,
            powerChoices,
          }: {
            origin: string
            picks: string[]
            powerChoices: Record<string, string[]>
          }) =>
            origin ? (
              <BenefitPicker
                originId={origin}
                picks={picks}
                powerChoices={powerChoices}
                onChange={(next) => form.setFieldValue('originChoices', next)}
                onPowerPick={(benefitId, powerId) =>
                  form.setFieldValue('powerChoices', {
                    ...powerChoices,
                    [benefitId]: powerId ? [powerId] : [],
                  })
                }
              />
            ) : null
          }
        </form.Subscribe>
      </CardContent>
    </Card>
  )
}

function BenefitPicker({
  originId,
  picks,
  powerChoices,
  onChange,
  onPowerPick,
}: {
  originId: string
  picks: string[]
  powerChoices: Record<string, string[]>
  onChange: (next: string[]) => void
  onPowerPick: (benefitId: string, powerId: string) => void
}) {
  const grant = originGrant(originId)
  if (!grant) return null
  const options = grant.poderUnico
    ? [...grant.benefits, grant.poderUnico]
    : grant.benefits
  const chosen = new Set(picks)
  const full = picks.length >= ORIGIN_BENEFIT_CAP
  const toggle = (id: string) =>
    onChange(
      chosen.has(id)
        ? picks.filter((x) => x !== id)
        : full
          ? picks
          : [...picks, id],
    )

  return (
    <div className="space-y-2">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Escolha 2 benefícios · {picks.length}/{ORIGIN_BENEFIT_CAP}
      </p>
      <ul className="space-y-1">
        {options.map((b) => {
          const selected = chosen.has(b.id)
          const isPoderUnico = b.id === grant.poderUnico?.id
          const locked = !selected && full
          return (
            <li key={b.id}>
              <button
                type="button"
                disabled={locked}
                onClick={() => toggle(b.id)}
                className={cn(
                  'flex w-full items-start gap-2 rounded-md border p-2 text-left transition-colors',
                  selected
                    ? 'border-primary bg-accent'
                    : locked
                      ? 'border-border opacity-50'
                      : 'border-border hover:bg-accent',
                )}
              >
                <span className="mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-sm border border-border">
                  {selected && <Check className="size-3 text-primary" />}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-1.5">
                    <p className="text-xs font-semibold">{b.name}</p>
                    {isPoderUnico && (
                      <Badge variant="secondary" className="px-1 py-0 text-[9px]">
                        poder único
                      </Badge>
                    )}
                  </div>
                  <p className="line-clamp-2 text-[11px] leading-snug text-muted-foreground">
                    {b.description}
                  </p>
                </div>
              </button>
              {selected && b.powerPick && (
                <FreePowerPicker
                  pool={b.powerPick}
                  value={powerChoices[b.id]?.[0] ?? ''}
                  onPick={(id) => onPowerPick(b.id, id)}
                />
              )}
            </li>
          )
        })}
      </ul>
      {picks.length < ORIGIN_BENEFIT_CAP && (
        <p className="text-[11px] text-[color:var(--hp-hurt)]">
          Escolha {ORIGIN_BENEFIT_CAP - picks.length} benefício(s) — ou termine
          depois na ficha.
        </p>
      )}
    </div>
  )
}

/**
 * Concrete power pick for a free-pick benefit ("um poder de combate/da
 * Tormenta a sua escolha"). Prereqs are advisory — GM arbitrates (the benefit
 * text says they apply).
 */
function FreePowerPicker({
  pool,
  value,
  onPick,
}: {
  pool: 'combate' | 'tormenta'
  value: string
  onPick: (powerId: string) => void
}) {
  const options = powerPickOptions(pool)
  const chosen = options.find((o) => o.value === value)
  return (
    <div className="ml-6 mt-1 space-y-1 rounded-md border border-dashed border-border p-2">
      <p className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
        Poder {pool === 'combate' ? 'de combate' : 'da Tormenta'} · escolha 1
      </p>
      <Combobox
        options={options.map((o) => ({ value: o.value, label: o.label }))}
        value={value}
        onChange={onPick}
        placeholder="Escolher poder"
        searchPlaceholder="Buscar poder…"
        emptyMessage="Nenhum."
        allowClear
        clearLabel="Nenhum"
      />
      {chosen && (
        <p className="text-[10px] leading-snug text-muted-foreground">
          {chosen.description}
        </p>
      )}
      {!chosen && (
        <p className="text-[10px] text-[color:var(--hp-hurt)]">
          Escolha o poder concedido pelo benefício.
        </p>
      )}
    </div>
  )
}
