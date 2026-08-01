import {
  caminhoSlotFor,
  type ClassChoices,
  devotoOptionsFor,
} from '@tormenta20/t20-data'
import { Check, Lock, Search } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Badge } from '@/shared/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Combobox } from '@/shared/ui/combobox'
import { Field, FieldLabel } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { cn } from '@/shared/lib/utils'
import { useCreationWizard } from '@/features/character-build/creation-wizard-context'
import {
  type ChoiceOption,
  type ClassEntry,
  classPowerCandidates,
  type PowerOption,
  powerBlockedReason,
  powerChoiceOptions,
  totalSlots,
  usedSlots,
} from '@/features/character-build/class-power-helpers'
import { toOptions } from '@/features/character-build/wizard-steps'

type Choices = ClassChoices

export function PoderesStep() {
  const { form } = useCreationWizard()

  // Reconcile over-cap: if the player lowered the class level after picking
  // powers, drop the excess so a level-N character never carries more elective
  // powers than the slots it earns. Runs on mount + when class/level changes.
  const primaryClass = form.state.values.classes[0]?.className as
    | string
    | undefined
  const primaryLevel = form.state.values.classes[0]?.level as number | undefined
  // biome-ignore lint/correctness/useExhaustiveDependencies: reads form.state imperatively; primaryClass/primaryLevel are the intended re-run triggers.
  useEffect(() => {
    const v = form.state.values
    const total = totalSlots(v.classes)
    const cp = (v.classPowers ?? []) as string[]
    if (cp.length > total) {
      form.setFieldValue('classPowers', cp.slice(0, total))
    }
  }, [form, primaryClass, primaryLevel])

  return (
    <form.Subscribe
      selector={(s: {
        values: {
          classes: ClassEntry[]
          classPowers: string[]
          classChoices: Choices
          powerChoices: Record<string, string[]>
        }
      }) => s.values}
    >
      {(v: {
        classes: ClassEntry[]
        classPowers: string[]
        classChoices: Choices
        powerChoices: Record<string, string[]>
      }) => {
        const primary = v.classes[0]
        const classPowers = v.classPowers ?? []
        const classChoices = v.classChoices ?? {}
        const powerChoices = v.powerChoices ?? {}
        const setPowerChoice = (powerId: string, ids: string[]) =>
          form.setFieldValue('powerChoices', { ...powerChoices, [powerId]: ids })
        const total = totalSlots(v.classes)
        const chosen = new Set(classPowers)
        const toggle = (id: string) =>
          form.setFieldValue(
            'classPowers',
            chosen.has(id)
              ? classPowers.filter((x) => x !== id)
              : [...classPowers, id],
          )
        const setChoice = (
          className: string,
          field: 'caminho' | 'devoto',
          value: string,
        ) =>
          form.setFieldValue('classChoices', {
            ...classChoices,
            [className]: { ...classChoices[className], [field]: value || undefined },
          })

        return (
          <Card>
            <CardHeader>
              <CardTitle className="font-display tracking-wide">Poderes</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {!primary?.className ? (
                <p className="text-sm text-muted-foreground">
                  Selecione uma classe primeiro (etapa Classe).
                </p>
              ) : (
                <>
                  {v.classes.map((c) => (
                    <ClassChoiceRow
                      key={c.className}
                      entry={c}
                      choices={classChoices}
                      onChoice={setChoice}
                    />
                  ))}
                  {total === 0 ? (
                    <p className="text-sm text-muted-foreground">
                      Personagem de 1º nível ainda não escolhe poderes de classe.
                    </p>
                  ) : (
                    <ElectiveSection
                      classes={v.classes}
                      chosen={chosen}
                      chosenIds={classPowers}
                      total={total}
                      classChoices={classChoices}
                      powerChoices={powerChoices}
                      onToggle={toggle}
                      onPowerChoice={setPowerChoice}
                    />
                  )}
                </>
              )}
            </CardContent>
          </Card>
        )
      }}
    </form.Subscribe>
  )
}

function ClassChoiceRow({
  entry,
  choices,
  onChoice,
}: {
  entry: ClassEntry
  choices: Choices
  onChoice: (cn: string, field: 'caminho' | 'devoto', v: string) => void
}) {
  const cam = caminhoSlotFor(entry.className)
  const showCam = cam && entry.level >= cam.minLevel
  const devoto = devotoOptionsFor(entry.className)
  if (!showCam && !devoto) return null
  const blob = choices[entry.className] ?? {}
  return (
    <div className="flex flex-wrap gap-3">
      {showCam && (
        <Field className="min-w-48">
          <FieldLabel>Caminho de {entry.className}</FieldLabel>
          <Combobox
            options={cam.options.map((o) => ({ value: o.id, label: o.name }))}
            value={blob.caminho ?? ''}
            onChange={(val) => onChoice(entry.className, 'caminho', val)}
            placeholder="Escolher caminho"
            searchPlaceholder="Buscar…"
            emptyMessage="Nenhum."
          />
        </Field>
      )}
      {devoto && (
        <Field className="min-w-48">
          <FieldLabel>Devoto ({entry.className})</FieldLabel>
          <Combobox
            options={toOptions(devoto.map((d) => d.name))}
            value={blob.devoto ?? ''}
            onChange={(val) => onChoice(entry.className, 'devoto', val)}
            placeholder="Escolher deus"
            searchPlaceholder="Buscar deus…"
            emptyMessage="Nenhum."
            allowClear
            clearLabel="Nenhum"
          />
        </Field>
      )}
    </div>
  )
}

function ElectiveSection({
  classes,
  chosen,
  chosenIds,
  total,
  classChoices,
  powerChoices,
  onToggle,
  onPowerChoice,
}: {
  classes: ClassEntry[]
  chosen: Set<string>
  chosenIds: string[]
  total: number
  classChoices: Choices
  powerChoices: Record<string, string[]>
  onToggle: (id: string) => void
  onPowerChoice: (powerId: string, ids: string[]) => void
}) {
  const [query, setQuery] = useState('')
  const primary = classes[0]
  const { classPowers, generalPowers } = classPowerCandidates(primary.className)
  const byId = new Map(
    [...classPowers, ...generalPowers].map((o) => [o.id, o] as const),
  )
  // Repeatable powers eat one slot per sub-choice; others one each.
  const used = usedSlots(chosenIds, powerChoices, byId)
  const remaining = Math.max(0, total - used)
  const q = query.trim().toLowerCase()
  const filterFn = (o: PowerOption) => !q || o.name.toLowerCase().includes(q)
  const ctx = { chosenIds: chosen, classChoices }
  const canAdd = remaining > 0

  return (
    <div className="space-y-3">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Poderes: {used} de {total} escolhidos · {remaining} restantes
      </p>

      <div className="space-y-1.5">
        <p className="text-xs font-semibold">Poderes de {primary.className}</p>
        <ul className="space-y-1">
          {classPowers.filter(filterFn).map((o) => (
            <PowerRow
              key={o.id}
              option={o}
              level={primary.level}
              selected={chosen.has(o.id)}
              canAdd={canAdd}
              ctx={ctx}
              onToggle={() => onToggle(o.id)}
              choiceValue={powerChoices[o.id] ?? []}
              onChoice={(ids) => onPowerChoice(o.id, ids)}
            />
          ))}
        </ul>
      </div>

      <div className="space-y-1.5">
        <div className="flex items-center gap-2">
          <p className="text-xs font-semibold">Poderes Gerais</p>
          <div className="relative ml-auto w-40">
            <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Buscar poder"
              aria-label="Buscar poder geral"
              className="h-8 pl-7 text-xs"
            />
          </div>
        </div>
        <ul className="max-h-[min(280px,34vh)] space-y-1 overflow-y-auto p-0.5">
          {generalPowers.filter(filterFn).map((o) => (
            <PowerRow
              key={o.id}
              option={o}
              level={primary.level}
              selected={chosen.has(o.id)}
              canAdd={canAdd}
              ctx={ctx}
              onToggle={() => onToggle(o.id)}
              choiceValue={powerChoices[o.id] ?? []}
              onChoice={(ids) => onPowerChoice(o.id, ids)}
            />
          ))}
        </ul>
      </div>

      {remaining > 0 && (
        <p className="text-[11px] text-[color:var(--hp-hurt)]">
          Faltam {remaining} poderes — você pode terminar depois na ficha.
        </p>
      )}
    </div>
  )
}

function PowerRow({
  option,
  level,
  selected,
  canAdd,
  ctx,
  onToggle,
  choiceValue,
  onChoice,
}: {
  option: PowerOption
  level: number
  selected: boolean
  canAdd: boolean
  ctx: { chosenIds: Set<string>; classChoices: Choices }
  onToggle: () => void
  choiceValue: string[]
  onChoice: (ids: string[]) => void
}) {
  const blocked = powerBlockedReason(option, level, ctx)
  const locked = !selected && (!!blocked || !canAdd)
  return (
    <li>
      <button
        type="button"
        disabled={locked}
        onClick={onToggle}
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
          {selected ? (
            <Check className="size-3 text-primary" />
          ) : locked && blocked ? (
            <Lock className="size-2.5 text-muted-foreground" />
          ) : null}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <p className="text-xs font-semibold">{option.name}</p>
            {blocked && (
              <Badge variant="outline" className="px-1 py-0 text-[9px]">
                {blocked}
              </Badge>
            )}
          </div>
          <p className="line-clamp-2 text-[11px] leading-snug text-muted-foreground">
            {option.description}
          </p>
        </div>
      </button>
      {selected && option.choice && (
        <PowerChoiceSelector
          choice={option.choice}
          value={choiceValue}
          onChange={onChoice}
          canAddMore={canAdd}
        />
      )}
    </li>
  )
}

function PowerChoiceSelector({
  choice,
  value,
  onChange,
  canAddMore,
}: {
  choice: NonNullable<PowerOption['choice']>
  value: string[]
  onChange: (ids: string[]) => void
  canAddMore: boolean
}) {
  const options = powerChoiceOptions(choice)
  const selected = new Set(value)
  const toggle = (id: string) => {
    if (choice.repeatable) {
      if (selected.has(id)) {
        onChange(value.filter((x) => x !== id))
      } else if (canAddMore) {
        // each extra pick on a repeatable power consumes a slot
        onChange([...value, id])
      }
    } else {
      onChange(selected.has(id) ? [] : [id])
    }
  }
  const pending = value.length === 0
  const chosenDescs = options.filter((o) => selected.has(o.id) && o.desc)
  return (
    <div className="ml-6 mt-1 space-y-1 rounded-md border border-dashed border-border p-2">
      <p className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
        {choice.label}
        {choice.repeatable ? ` (${value.length})` : ''}
      </p>
      <div className="flex flex-wrap gap-1">
        {options.map((o: ChoiceOption) => {
          const on = selected.has(o.id)
          const lockedOpt = choice.repeatable && !on && !canAddMore
          return (
            <button
              key={o.id}
              type="button"
              disabled={lockedOpt}
              onClick={() => toggle(o.id)}
              title={o.desc ?? o.note}
              className={cn(
                'rounded-md border px-2 py-0.5 text-[11px] transition-colors',
                on
                  ? 'border-primary bg-accent'
                  : lockedOpt
                    ? 'border-border opacity-40'
                    : 'border-border hover:bg-accent',
              )}
            >
              {o.name}
              {o.note ? (
                <span className="ml-1 text-muted-foreground">· {o.note}</span>
              ) : null}
            </button>
          )
        })}
      </div>
      {chosenDescs.map((o) => (
        <div
          key={o.id}
          className="rounded-md border-l-2 border-primary/50 bg-accent/40 px-2 py-1"
        >
          <p className="text-[11px] font-semibold text-foreground">
            <span className="mr-1 text-[color:var(--primary)]">✦</span>
            {o.note ?? o.name}
          </p>
          <p className="text-[10px] leading-snug text-muted-foreground">
            {o.desc}
          </p>
        </div>
      ))}
      {pending && (
        <p className="text-[10px] text-[color:var(--hp-hurt)]">
          Escolha {choice.label.toLowerCase()}.
        </p>
      )}
    </div>
  )
}
