import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Field, FieldGroup, FieldLabel } from '@/shared/ui/field'
import { NumberField } from '@/features/character-build/number-field'
import { deriveDraftVitals } from '@/features/character-build/draft-vitals'
import { useCreationWizard } from '@/features/character-build/creation-wizard-context'
import type { CharacterFormValues } from '@/features/character-build/wizard-steps'

/**
 * PV/PM máximos are derived (classe + Constituição + nível + poderes passivos),
 * shown read-only — só os valores atuais são editáveis (começar ferido).
 */
export function VitalidadeStep() {
  const { form, raceChoices } = useCreationWizard()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Vitalidade</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        <form.Subscribe
          selector={(s: { values: CharacterFormValues }) => s.values}
        >
          {(values: CharacterFormValues) => {
            const { pvMax, pmMax } = deriveDraftVitals(values, raceChoices)
            return (
              <FieldGroup className="grid gap-4 sm:grid-cols-4 sm:gap-6">
                <DerivedStat label="PV máx" value={pvMax} />
                <NumberField
                  form={form}
                  name="hpCurrent"
                  label="PV atual"
                  min={0}
                  max={pvMax}
                />
                <DerivedStat label="PM máx" value={pmMax} />
                <NumberField
                  form={form}
                  name="mpCurrent"
                  label="PM atual"
                  min={0}
                  max={pmMax}
                />
              </FieldGroup>
            )
          }}
        </form.Subscribe>
        <p className="text-[11px] text-muted-foreground">
          PV/PM máximos derivam da classe, Constituição, nível e poderes — não
          são editáveis aqui.
        </p>
      </CardContent>
    </Card>
  )
}

function DerivedStat({ label, value }: { label: string; value: number }) {
  return (
    <Field>
      <FieldLabel>{label}</FieldLabel>
      <div className="flex h-9 w-20 items-center justify-center rounded-md border border-border bg-muted/40 sm:w-24">
        <span className="font-display text-xl font-semibold tabular-nums">
          {value}
        </span>
      </div>
    </Field>
  )
}
