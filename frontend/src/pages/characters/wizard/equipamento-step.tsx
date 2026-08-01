import {
  STARTING_KIT_BASE_ITEMS,
  STARTING_TIBARES_DICE,
  type StartingKit,
} from '@tormenta20/t20-data'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Combobox } from '@/shared/ui/combobox'
import { Field, FieldLabel } from '@/shared/ui/field'
import { NumberInput } from '@/shared/ui/number-input'
import { cn } from '@/shared/lib/utils'
import { useCreationWizard } from '@/features/character-build/creation-wizard-context'
import {
  lightArmorOptions,
  originStartingItems,
  purchasesTotal,
  startingLoadout,
  weaponOptions,
} from '@/features/character-build/starting-equipment'
import { StartingShop } from '@/features/character-build/starting-shop'

type EquipValues = {
  classes: { className: string; level: number }[]
  origin: string
  tibar: number
  startingWeaponSimple: string
  startingWeaponMartial: string
  startingArmor: string
  startingShield: boolean
  startingPurchases: Record<string, number>
}

/**
 * Equipamento inicial (book p140): unified L1 kit narrowed by the class's
 * proficiências + origem itens + Tabela 3-1 money. Homebrew: kit pickers stay
 * available at any level; money defaults to the level's table value and stays
 * editable. Under-filling is soft — the sheet inventory finishes the job.
 */
export function EquipamentoStep() {
  const { form } = useCreationWizard()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">
          Equipamento inicial
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <form.Subscribe selector={(s: { values: EquipValues }) => s.values}>
          {(v: EquipValues) => {
            const primary = v.classes[0]
            if (!primary?.className) {
              return (
                <p className="text-sm text-muted-foreground">
                  Selecione uma classe primeiro (etapa Classe).
                </p>
              )
            }
            const level = v.classes.reduce((n, c) => n + (c.level || 0), 0) || 1
            const { kit, tableMoney } = startingLoadout(
              primary.className,
              level,
            )
            return (
              <>
                <KitBaseLine />
                <WeaponPickers form={form} values={v} kit={kit} />
                <ArmorPicker form={form} values={v} kit={kit} />
                <OriginItems originName={v.origin} />
                <ExtrasNote kit={kit} />
                <MoneyField
                  form={form}
                  tibar={v.tibar}
                  level={level}
                  tableMoney={tableMoney}
                  spent={purchasesTotal(v.startingPurchases ?? {})}
                />
                <StartingShop
                  purchases={v.startingPurchases ?? {}}
                  remaining={Math.max(
                    0,
                    (v.tibar ?? 0) - purchasesTotal(v.startingPurchases ?? {}),
                  )}
                  onChange={(next) =>
                    form.setFieldValue('startingPurchases', next)
                  }
                />
              </>
            )
          }}
        </form.Subscribe>
      </CardContent>
    </Card>
  )
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
      {children}
    </p>
  )
}

function KitBaseLine() {
  return (
    <div className="space-y-1">
      <SectionLabel>Kit · automático (p140)</SectionLabel>
      <p className="text-sm">{STARTING_KIT_BASE_ITEMS.join(' · ')}</p>
    </div>
  )
}

type StepForm = ReturnType<typeof useCreationWizard>['form']

function WeaponPickers({
  form,
  values,
  kit,
}: {
  form: StepForm
  values: EquipValues
  kit: StartingKit
}) {
  const simple = weaponOptions('weapon-simple')
  const martial = weaponOptions('weapon-martial')
  return (
    <div className="flex flex-wrap gap-3">
      <Field className="min-w-52">
        <FieldLabel>Arma simples · escolha 1</FieldLabel>
        <Combobox
          options={simple.map((w) => ({ value: w.id, label: w.name }))}
          value={values.startingWeaponSimple}
          onChange={(id) => form.setFieldValue('startingWeaponSimple', id)}
          placeholder="Escolher arma"
          searchPlaceholder="Buscar arma…"
          emptyMessage="Nenhuma."
          allowClear
          clearLabel="Nenhuma"
        />
      </Field>
      {kit.weapons === 'simples+marcial' && (
        <Field className="min-w-52">
          <FieldLabel>Arma marcial · escolha 1 (proficiente)</FieldLabel>
          <Combobox
            options={martial.map((w) => ({ value: w.id, label: w.name }))}
            value={values.startingWeaponMartial}
            onChange={(id) => form.setFieldValue('startingWeaponMartial', id)}
            placeholder="Escolher arma"
            searchPlaceholder="Buscar arma…"
            emptyMessage="Nenhuma."
            allowClear
            clearLabel="Nenhuma"
          />
        </Field>
      )}
    </div>
  )
}

function ArmorPicker({
  form,
  values,
  kit,
}: {
  form: StepForm
  values: EquipValues
  kit: StartingKit
}) {
  if (kit.armor === 'nenhuma') {
    return (
      <div className="space-y-1">
        <SectionLabel>Armadura</SectionLabel>
        <p className="text-sm text-muted-foreground">
          Arcanistas começam sem armadura (p140).
        </p>
      </div>
    )
  }
  const options =
    kit.armor === 'brunea'
      ? [{ value: 'brunea', label: 'Brunea (proficiência em pesadas)' }]
      : lightArmorOptions().map((a) => ({ value: a.id, label: a.name }))
  return (
    <div className="space-y-1.5">
      <SectionLabel>
        Armadura{kit.armor === 'brunea' ? ' · brunea' : ' · leve a escolha'}
      </SectionLabel>
      <div className="flex flex-wrap gap-1.5">
        {options.map((o) => {
          const active = values.startingArmor === o.value
          return (
            <button
              key={o.value}
              type="button"
              aria-pressed={active}
              onClick={() =>
                form.setFieldValue('startingArmor', active ? '' : o.value)
              }
              className={cn(
                'rounded-md border px-2.5 py-1 text-xs transition-colors',
                active
                  ? 'border-primary bg-accent font-medium'
                  : 'border-border text-muted-foreground hover:bg-accent',
              )}
            >
              {o.label}
            </button>
          )
        })}
        {kit.shieldLeve && (
          <button
            type="button"
            aria-pressed={values.startingShield}
            onClick={() =>
              form.setFieldValue('startingShield', !values.startingShield)
            }
            className={cn(
              'rounded-md border px-2.5 py-1 text-xs transition-colors',
              values.startingShield
                ? 'border-primary bg-accent font-medium'
                : 'border-border text-muted-foreground hover:bg-accent',
            )}
          >
            + Escudo leve
          </button>
        )}
      </div>
    </div>
  )
}

function OriginItems({ originName }: { originName: string }) {
  const items = originStartingItems(originName)
  if (!originName) return null
  return (
    <div className="space-y-1">
      <SectionLabel>Origem · {originName} · automático</SectionLabel>
      {items.length === 0 ? (
        <p className="text-sm text-muted-foreground">Sem itens de origem.</p>
      ) : (
        <p className="text-sm">{items.join(' · ')}</p>
      )}
    </div>
  )
}

function ExtrasNote({ kit }: { kit: StartingKit }) {
  if (kit.extras.length === 0) return null
  return (
    <div className="space-y-1">
      <SectionLabel>Extras da classe</SectionLabel>
      {kit.extras.map((e) => (
        <p key={e.source} className="text-sm">
          {e.description}
          <span className="block text-[11px] text-muted-foreground">
            {e.source} — adicione o item escolhido na ficha.
          </span>
        </p>
      ))}
    </div>
  )
}

function rollDice(count: number, sides: number): number {
  let total = 0
  for (let i = 0; i < count; i++) {
    total += 1 + Math.floor(Math.random() * sides)
  }
  return total
}

const tibarFmt = (v: number) =>
  v.toLocaleString('pt-BR', { maximumFractionDigits: 2 })

function MoneyField({
  form,
  tibar,
  level,
  tableMoney,
  spent,
}: {
  form: StepForm
  tibar: number
  level: number
  tableMoney: number | null
  spent: number
}) {
  const remaining = (tibar ?? 0) - spent
  return (
    <div className="space-y-1.5">
      <SectionLabel>Dinheiro inicial (Tabela 3-1, p140)</SectionLabel>
      {spent > 0 && (
        <p
          className={cn(
            'text-xs',
            remaining < 0
              ? 'text-[color:var(--hp-hurt)]'
              : 'text-muted-foreground',
          )}
        >
          Gasto T$ {tibarFmt(spent)} · Restante T$ {tibarFmt(Math.max(0, remaining))}
        </p>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm">T$</span>
        <NumberInput
          min={0}
          max={1_000_000}
          step={1}
          value={tibar}
          onChange={(v) => form.setFieldValue('tibar', v)}
          aria-label="Tibares iniciais"
        />
        {tableMoney === null ? (
          <button
            type="button"
            onClick={() =>
              form.setFieldValue('tibar', rollDice(4, 6))
            }
            className="rounded-md border border-border px-2.5 py-1 text-xs hover:bg-accent"
          >
            🎲 Rolar {STARTING_TIBARES_DICE} (Nv 1)
          </button>
        ) : (
          <button
            type="button"
            onClick={() => form.setFieldValue('tibar', tableMoney)}
            className="rounded-md border border-border px-2.5 py-1 text-xs hover:bg-accent"
          >
            Usar Tabela 3-1 (Nv {level}): T$ {tableMoney.toLocaleString('pt-BR')}
          </button>
        )}
      </div>
    </div>
  )
}
