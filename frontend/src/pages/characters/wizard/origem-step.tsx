import { Check } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Combobox } from '@/shared/ui/combobox'
import { Field, FieldError } from '@/shared/ui/field'
import { cn } from '@/shared/lib/utils'
import { originGrant } from '@/features/character-build/grant-helpers'
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
          selector={(s: { values: { origin: string; originChoices: string[] } }) => ({
            origin: s.values.origin,
            picks: s.values.originChoices,
          })}
        >
          {({ origin, picks }: { origin: string; picks: string[] }) =>
            origin ? (
              <BenefitPicker
                originId={origin}
                picks={picks}
                onChange={(next) => form.setFieldValue('originChoices', next)}
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
  onChange,
}: {
  originId: string
  picks: string[]
  onChange: (next: string[]) => void
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
