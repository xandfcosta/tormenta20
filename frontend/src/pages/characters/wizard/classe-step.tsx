import { ATTRIBUTE_KEYS, attributePresetForClass } from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { Field, FieldDescription, FieldError, FieldLabel } from '@/shared/ui/field'
import { NumberInput } from '@/shared/ui/number-input'
import { ClassEntryRow } from '@/features/character-build/class-entry-row'
import { ClassTileGrid } from '@/features/character-build/class-picker'
import { ClassGrantPanel } from '@/features/character-build/grant-panels'
import {
  type FieldApi,
  useCreationWizard,
} from '@/features/character-build/creation-wizard-context'

export function ClasseStep() {
  const { form, options } = useCreationWizard()

  const applyPreset = (className: string) => {
    const preset = attributePresetForClass(className)
    if (!preset) return
    for (const attr of ATTRIBUTE_KEYS) form.setFieldValue(attr, preset[attr])
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Classe</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <form.Field name="classes" mode="array">
          {(classesField: FieldApi) => {
            const items = classesField.state.value as {
              className: string
              level: number
            }[]
            const primary = items[0]?.className ?? ''
            const arrayInvalid =
              classesField.state.meta.isTouched &&
              !classesField.state.meta.isValid
            const pickPrimary = (className: string) => {
              if (items.length === 0) {
                classesField.pushValue({ className, level: 1 })
              } else {
                // Replace via the array API (not setFieldValue on a nested
                // path) so the array Field re-renders and the tile selection
                // updates. Read the level from the live form state — the
                // captured `items` is stale w.r.t. the level input's edits.
                const level =
                  (form.state.values.classes[0]?.level as number) ?? 1
                classesField.replaceValue(0, { className, level })
              }
              applyPreset(className)
            }
            const addMulticlass = () => {
              const next =
                options.classes.find(
                  (c) => !items.some((it) => it.className === c),
                ) ?? options.classes[0]
              classesField.pushValue({ className: next, level: 1 })
            }
            return (
              <div className="lg:grid lg:grid-cols-[1.1fr_1fr] lg:gap-4">
                {/* Left: scrollable class catalog + level + multiclass */}
                <div className="space-y-3">
                  <div className="max-h-[min(320px,40vh)] overflow-y-auto p-1.5">
                    <ClassTileGrid
                      options={options.classes}
                      selected={primary}
                      onSelect={pickPrimary}
                    />
                  </div>
                  {items.length > 0 && (
                    <form.Field name="classes[0].level">
                      {(f: FieldApi) => (
                        <Field className="w-24">
                          <FieldLabel htmlFor={f.name}>Nível</FieldLabel>
                          <NumberInput
                            id={f.name}
                            min={1}
                            max={20}
                            value={f.state.value as number}
                            onChange={(v) => f.handleChange(v)}
                            onBlur={f.handleBlur}
                          />
                        </Field>
                      )}
                    </form.Field>
                  )}

                  {items.slice(1).map((_, j) => {
                    const i = j + 1
                    return (
                      <div
                        key={i}
                        className="space-y-2 rounded-md border border-dashed border-border p-2"
                      >
                        <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
                          Classe adicional (avançado)
                        </p>
                        <ClassEntryRow
                          index={i}
                          classOptions={options.classes}
                          form={form}
                          onRemove={() => classesField.removeValue(i)}
                        />
                        <form.Subscribe
                          selector={(s: {
                            values: {
                              classes: { className: string; level: number }[]
                            }
                          }) =>
                            [
                              s.values.classes[i]?.className,
                              s.values.classes[i]?.level,
                            ] as const
                          }
                        >
                          {([className, level]: readonly [
                            string | undefined,
                            number | undefined,
                          ]) =>
                            className ? (
                              <ClassGrantPanel
                                className={className}
                                level={level ?? 1}
                              />
                            ) : null
                          }
                        </form.Subscribe>
                      </div>
                    )
                  })}

                  {items.length >= 1 && (
                    <ConfirmDialog
                      destructive={false}
                      title="Adicionar multiclasse?"
                      description="Multiclasse é adquirida em níveis mais altos via Poder de Multiclasse — não é padrão no nível 1. O preset de atributos usa apenas a primeira classe."
                      confirmLabel="Adicionar mesmo assim"
                      onConfirm={addMulticlass}
                      trigger={
                        <Button type="button" variant="outline" size="sm">
                          + Adicionar classe (multiclasse)
                        </Button>
                      }
                    />
                  )}

                  {arrayInvalid ? (
                    <FieldError errors={classesField.state.meta.errors} />
                  ) : !items.length ? (
                    <FieldDescription>Selecione uma classe.</FieldDescription>
                  ) : null}
                </div>

                {/* Right: pinned primary-class grant */}
                <div className="mt-3 lg:mt-0 lg:max-h-[min(420px,52vh)] lg:overflow-y-auto lg:pr-1">
                  <form.Subscribe
                    selector={(s: {
                      values: { classes: { className: string; level: number }[] }
                    }) =>
                      [
                        s.values.classes[0]?.className,
                        s.values.classes[0]?.level,
                      ] as const
                    }
                  >
                    {([className, level]: readonly [
                      string | undefined,
                      number | undefined,
                    ]) =>
                      className ? (
                        <ClassGrantPanel className={className} level={level ?? 1} />
                      ) : (
                        <p className="rounded-lg border border-dashed border-border p-4 text-center text-xs text-muted-foreground">
                          Selecione uma classe para ver os detalhes.
                        </p>
                      )
                    }
                  </form.Subscribe>
                </div>
              </div>
            )
          }}
        </form.Field>
      </CardContent>
    </Card>
  )
}
