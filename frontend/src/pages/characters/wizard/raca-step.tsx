import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Field, FieldDescription, FieldError } from '@/shared/ui/field'
import { RacePicker } from '@/features/character-build/race-picker'
import {
  type FieldApi,
  useCreationWizard,
} from '@/features/character-build/creation-wizard-context'

export function RacaStep() {
  const { form, options, raceChoices, setRaceChoices } = useCreationWizard()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Raça</CardTitle>
      </CardHeader>
      <CardContent>
        <form.Field name="races" mode="array">
          {(racesField: FieldApi) => {
            const value = racesField.state.value as string[]
            const invalid =
              racesField.state.meta.isTouched && !racesField.state.meta.isValid
            return (
              <Field data-invalid={invalid}>
                <RacePicker
                  options={options.races}
                  value={value}
                  choices={raceChoices}
                  onChange={(next) => racesField.handleChange(next)}
                  onChoicesChange={setRaceChoices}
                />
                {invalid ? (
                  <FieldError errors={racesField.state.meta.errors} />
                ) : !value.length ? (
                  <FieldDescription>
                    Selecione ao menos uma raça.
                  </FieldDescription>
                ) : null}
              </Field>
            )
          }}
        </form.Field>
      </CardContent>
    </Card>
  )
}
