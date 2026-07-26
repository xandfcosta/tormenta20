import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { FieldGroup } from '@/shared/ui/field'
import { NumberField } from '@/features/character-build/number-field'
import { useCreationWizard } from '@/features/character-build/creation-wizard-context'

export function VitalidadeStep() {
  const { form } = useCreationWizard()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Vitalidade</CardTitle>
      </CardHeader>
      <CardContent>
        <FieldGroup className="grid gap-4 sm:grid-cols-4 sm:gap-6">
          <NumberField form={form} name="hpMax" label="PV máx" min={1} />
          <NumberField form={form} name="hpCurrent" label="PV atual" min={0} />
          <NumberField form={form} name="mpMax" label="PM máx" min={0} />
          <NumberField form={form} name="mpCurrent" label="PM atual" min={0} />
        </FieldGroup>
      </CardContent>
    </Card>
  )
}
