import {
  STARTING_KIT_BASE_ITEMS,
  STARTING_TIBARES_DICE,
  type StartingKit,
} from '@tormenta20/t20-data'
import { Link } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Combobox } from '@/shared/ui/combobox'
import { Field, FieldLabel } from '@/shared/ui/field'
import { NumberInput } from '@/shared/ui/number-input'
import { cn } from '@/shared/lib/utils'
import { useCreationWizard } from '@/features/character-build/creation-wizard-context'
import {
  bagagemGroups,
  lightArmorOptions,
  origemRolledMoneySum,
  purchasesTotal,
  startingLoadout,
  startingSlots,
  weaponOptions,
} from '@/features/character-build/starting-equipment'
import { appliedRaceDeltas } from '@/features/character-build/grant-helpers'
import { BagagemPanel } from '@/features/character-build/bagagem-panel'
import { OrigemItemsSection } from '@/features/character-build/origem-items-section'
import { StartingShop } from '@/features/character-build/starting-shop'

type EquipValues = {
  classes: { className: string; level: number }[]
  races: string[]
  origin: string
  strength: number
  tibar: number
  startingWeaponSimple: string
  startingWeaponMartial: string
  startingArmor: string
  startingShield: boolean
  startingPurchases: Record<string, number>
  originItemPicks: Record<string, string>
  startingMoneyRolled: boolean
}

/**
 * Equipamento inicial (book p140) — design "Sua bagagem": choosers on the
 * left, a sticky derived inventory preview on the right (kit + picks + origem
 * + compras) with the slots bar and wallet chip as the step's only gauges.
 * Ghost lines in the bag scroll-focus their chooser. Under-filling is soft.
 */
export function EquipamentoStep() {
  const { form, raceChoices } = useCreationWizard()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">
          Equipamento inicial
        </CardTitle>
      </CardHeader>
      <CardContent>
        <form.Subscribe selector={(s: { values: EquipValues }) => s.values}>
          {(v: EquipValues) => {
            const primary = v.classes[0]
            if (!primary?.className) {
              return (
                <p className="text-sm text-muted-foreground">
                  Selecione uma classe primeiro —{' '}
                  <Link
                    to="/characters/new/classe"
                    className="underline underline-offset-2 hover:text-foreground"
                  >
                    ir para a etapa Classe ›
                  </Link>
                </p>
              )
            }
            const level = v.classes.reduce((n, c) => n + (c.level || 0), 0) || 1
            const { kit, tableMoney } = startingLoadout(primary.className, level)
            const draft = {
              weaponSimple: v.startingWeaponSimple ?? '',
              weaponMartial: v.startingWeaponMartial ?? '',
              armor: v.startingArmor ?? '',
              shield: v.startingShield ?? false,
            }
            const forTotal =
              (v.strength ?? 0) +
              (appliedRaceDeltas(v.races ?? [], raceChoices).strength ?? 0)
            const slots = startingSlots(
              draft,
              kit,
              v.origin,
              v.originItemPicks ?? {},
              v.startingPurchases ?? {},
              forTotal,
            )
            const setPurchaseQty = (id: string, qty: number) => {
              const next = { ...(v.startingPurchases ?? {}) }
              if (qty <= 0) delete next[id]
              else next[id] = qty
              form.setFieldValue('startingPurchases', next)
            }
            return (
              <div className="space-y-4 lg:grid lg:grid-cols-[1fr_16.5rem] lg:items-start lg:gap-4 lg:space-y-0">
                <div className="lg:order-2">
                  <BagagemPanel
                    groups={bagagemGroups(
                      draft,
                      kit,
                      v.origin,
                      v.originItemPicks ?? {},
                      v.startingPurchases ?? {},
                    )}
                    slotsUsed={slots.used}
                    slotsCapacity={slots.capacity}
                    tibar={v.tibar ?? 0}
                    purchases={v.startingPurchases ?? {}}
                    onPurchaseQty={setPurchaseQty}
                  />
                </div>
                <div className="space-y-4 lg:order-1">
                  <ClasseChoosers form={form} values={v} kit={kit} />
                  <OrigemItemsSection
                    originName={v.origin}
                    picks={v.originItemPicks ?? {}}
                    onPick={(label, val) =>
                      form.setFieldValue('originItemPicks', {
                        ...(v.originItemPicks ?? {}),
                        [label]: val,
                      })
                    }
                    onMoneyRoll={(_label, amount) =>
                      form.setFieldValue('tibar', (v.tibar ?? 0) + amount)
                    }
                  />
                  <ExtrasNote kit={kit} />
                  <MoneyField
                    form={form}
                    tibar={v.tibar}
                    level={level}
                    tableMoney={tableMoney}
                    rolled={v.startingMoneyRolled ?? false}
                    origemRolled={origemRolledMoneySum(
                      v.origin,
                      v.originItemPicks ?? {},
                    )}
                  />
                  <StartingShop
                    purchases={v.startingPurchases ?? {}}
                    remaining={
                      (v.tibar ?? 0) - purchasesTotal(v.startingPurchases ?? {})
                    }
                    onChange={(next) =>
                      form.setFieldValue('startingPurchases', next)
                    }
                  />
                </div>
              </div>
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

type StepForm = ReturnType<typeof useCreationWizard>['form']

/** Kit picks grouped under one bordered "Kit da classe" block. */
function ClasseChoosers({
  form,
  values,
  kit,
}: {
  form: StepForm
  values: EquipValues
  kit: StartingKit
}) {
  return (
    <div className="space-y-3 rounded-lg border border-border p-3">
      <SectionLabel>
        Kit da classe{' '}
        <span className="normal-case tracking-normal">
          · {STARTING_KIT_BASE_ITEMS.join(' · ')} (automático)
        </span>
      </SectionLabel>
      <WeaponPickers form={form} values={values} kit={kit} />
      <ArmorPicker form={form} values={values} kit={kit} />
    </div>
  )
}

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
      <Field className="min-w-52" id="chooser-arma-simples">
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
        <Field className="min-w-52" id="chooser-arma-marcial">
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
      <p className="text-sm text-muted-foreground">
        Arcanistas começam sem armadura (p140).
      </p>
    )
  }
  const options =
    kit.armor === 'brunea'
      ? [{ value: 'brunea', label: 'Brunea (proficiência em pesadas)' }]
      : lightArmorOptions().map((a) => ({ value: a.id, label: a.name }))
  return (
    <div className="space-y-1.5" id="chooser-armadura">
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
                'rounded-md border px-2.5 py-1.5 text-xs transition-colors sm:py-1',
                active
                  ? 'border-primary bg-accent font-medium'
                  : 'border-border text-muted-foreground hover:bg-accent',
              )}
            >
              {active ? '✓ ' : ''}
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
              'rounded-md border px-2.5 py-1.5 text-xs transition-colors sm:py-1',
              values.startingShield
                ? 'border-primary bg-accent font-medium'
                : 'border-border text-muted-foreground hover:bg-accent',
            )}
          >
            {values.startingShield ? '✓ Escudo leve' : '+ Escudo leve'}
          </button>
        )}
      </div>
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

function MoneyField({
  form,
  tibar,
  level,
  tableMoney,
  rolled,
  origemRolled,
}: {
  form: StepForm
  tibar: number
  level: number
  tableMoney: number | null
  rolled: boolean
  origemRolled: number
}) {
  return (
    <div className="space-y-1.5">
      <SectionLabel>Dinheiro inicial · Tabela 3-1</SectionLabel>
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
            disabled={rolled}
            onClick={() => {
              // Preserva o T$ já rolado por concessão de origem (2d6 etc.).
              form.setFieldValue('tibar', rollDice(4, 6) + origemRolled)
              form.setFieldValue('startingMoneyRolled', true)
            }}
            className={cn(
              'rounded-md border border-border px-2.5 py-1.5 text-xs',
              rolled ? 'opacity-60' : 'hover:bg-accent',
            )}
          >
            {rolled
              ? `Rolado (${STARTING_TIBARES_DICE})`
              : `🎲 Rolar ${STARTING_TIBARES_DICE} (Nv 1)`}
          </button>
        ) : (
          <button
            type="button"
            onClick={() =>
              form.setFieldValue('tibar', tableMoney + origemRolled)
            }
            className="rounded-md border border-border px-2.5 py-1.5 text-xs hover:bg-accent"
          >
            Usar Tabela 3-1 (Nv {level}): T$ {tableMoney.toLocaleString('pt-BR')}
          </button>
        )}
      </div>
    </div>
  )
}
