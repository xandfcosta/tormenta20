import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { FieldGroup } from '@/shared/ui/field'
import { NumberField } from '@/features/character-build/number-field'
import {
  anyRacePending,
  raceAttributeDeltas,
} from '@/features/character-build/grant-helpers'
import { useCreationWizard } from '@/features/character-build/creation-wizard-context'

export function AtributosStep() {
  const { form, raceChoices } = useCreationWizard()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Atributos</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-xs text-muted-foreground">
          Edite a base (preset da classe); o total à direita já inclui os bônus
          de raça.
        </p>
        <form.Subscribe selector={(s: { values: { races: string[] } }) => s.values.races}>
          {(races: string[]) => {
            const d = raceAttributeDeltas(races, raceChoices)
            return (
              <>
                {anyRacePending(races, raceChoices) && (
                  <p className="mb-3 text-[11px] text-[color:var(--hp-hurt)]">
                    Há escolhas de atributo de raça pendentes — os +1 não
                    definidos não serão aplicados ao salvar.
                  </p>
                )}
                <FieldGroup className="grid gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3">
                  <NumberField form={form} name="strength" label="Força" min={-5} max={10} raceDelta={d.strength} />
                  <NumberField form={form} name="dexterity" label="Destreza" min={-5} max={10} raceDelta={d.dexterity} />
                  <NumberField form={form} name="constitution" label="Constituição" min={-5} max={10} raceDelta={d.constitution} />
                  <NumberField form={form} name="intelligence" label="Inteligência" min={-5} max={10} raceDelta={d.intelligence} />
                  <NumberField form={form} name="wisdom" label="Sabedoria" min={-5} max={10} raceDelta={d.wisdom} />
                  <NumberField form={form} name="charisma" label="Carisma" min={-5} max={10} raceDelta={d.charisma} />
                </FieldGroup>
              </>
            )
          }}
        </form.Subscribe>
      </CardContent>
    </Card>
  )
}
