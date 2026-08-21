import { STARTING_KIT_BASE_ITEMS, STARTING_TIBARES_DICE } from '@/shared/rules/class-starting-kits'
import type { StartingKit } from '@/shared/api/catalog-types'
import { For, type JSX, Show } from 'solid-js'
import { BagagemPanel } from '@/features/character-build/bagagem-panel'
import { useForge } from '@/features/character-build/forge-context'
import { appliedRaceDeltas } from '@/features/character-build/grant-helpers'
import { OrigemItemsSection } from '@/features/character-build/origem-items-section'
import { rollDice } from '@/shared/lib/dice'
import { StartingShop } from '@/features/character-build/starting-shop'
import {
  bagagemGroups,
  lightArmorOptions,
  origemRolledMoneySum,
  purchasesTotal,
  startingLoadout,
  startingSlots,
  weaponOptions,
} from '@/features/character-build/starting-equipment'
import { totalClassLevel } from '@/features/character-build/class-entries'
import { cn } from '@/shared/lib/utils'
import { NumberInput } from '@/shared/ui/number-input'
import { Select } from '@/shared/ui/select'

/**
 * Seventh step: equipamento inicial (p140). Choosers on the left, the bag they
 * fill on the right — the bag is derived, never mirrored, so a pick and its
 * consequence (espaços, T$) are on screen together. Under-filling is soft: a
 * ghost line marks what is missing and the sheet can finish it.
 */
export function EquipamentoStep() {
  const { draft } = useForge()

  const primary = () => draft.values.classes[0]?.className ?? ''
  const level = () => totalClassLevel(draft.values.classes) || 1
  const loadout = () => (primary() ? startingLoadout(primary(), level()) : null)

  const kitDraft = () => ({
    weaponSimple: draft.values.startingWeaponSimple ?? '',
    weaponMartial: draft.values.startingWeaponMartial ?? '',
    armor: draft.values.startingArmor ?? '',
    shield: draft.values.startingShield ?? false,
  })

  const forTotal = () =>
    draft.values.strength +
    (appliedRaceDeltas(draft.values.races, draft.raceChoices).strength ?? 0)

  const purchases = () => draft.values.startingPurchases ?? {}
  const originPicks = () => draft.values.originItemPicks ?? {}

  const slotsOf = (kit: StartingKit) =>
    startingSlots(
      kitDraft(),
      kit,
      draft.values.origin,
      originPicks(),
      purchases(),
      forTotal(),
    )

  const setPurchaseQty = (id: string, qty: number) => {
    const next = { ...purchases() }
    if (qty <= 0) delete next[id]
    else next[id] = qty
    draft.setValue('startingPurchases', next)
  }

  return (
    <section class="flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="forge-step-equipamento">
      <h2
        id="forge-step-equipamento"
        class="font-heading text-lg uppercase tracking-[0.16em] text-grimorio-gold"
      >
        O que você carrega
      </h2>

      <Show
        when={loadout()}
        fallback={
          <p class="flex flex-1 items-center justify-center text-center text-sm text-muted-foreground">
            Escolha uma classe primeiro — é ela que define o kit inicial.
          </p>
        }
      >
        {(loaded) => (
          <div class="grid gap-4 lg:min-h-0 lg:flex-1 lg:grid-cols-[1fr_20rem] lg:items-start">
            <div class="space-y-3 lg:min-h-0 lg:max-h-full lg:overflow-y-auto lg:pr-1">
              <KitChoosers kit={loaded().kit} />
              <OrigemItemsSection
                originName={draft.values.origin}
                picks={originPicks()}
                onPick={(label, value) =>
                  draft.setValue('originItemPicks', { ...originPicks(), [label]: value })
                }
                onMoneyRoll={(amount) => draft.setValue('tibar', (draft.values.tibar ?? 0) + amount)}
              />
              <ExtrasNote kit={loaded().kit} />
              <MoneyField
                level={level()}
                tableMoney={loaded().tableMoney}
                origemRolled={origemRolledMoneySum(draft.values.origin, originPicks())}
              />
              <StartingShop
                purchases={purchases()}
                remaining={(draft.values.tibar ?? 0) - purchasesTotal(purchases())}
                onChange={(next) => draft.setValue('startingPurchases', next)}
              />
            </div>

            <BagagemPanel
              groups={bagagemGroups(
                kitDraft(),
                loaded().kit,
                draft.values.origin,
                originPicks(),
                purchases(),
              )}
              slotsUsed={slotsOf(loaded().kit).used}
              slotsCapacity={slotsOf(loaded().kit).capacity}
              tibar={draft.values.tibar ?? 0}
              purchases={purchases()}
              onPurchaseQty={setPurchaseQty}
            />
          </div>
        )}
      </Show>
    </section>
  )
}

/** The class kit's picks, in one block — the base items come automatically. */
function KitChoosers(props: { kit: StartingKit }) {
  const { draft } = useForge()

  return (
    <div class="space-y-3 rounded-sm border border-grimorio-iron p-3">
      <p class="font-heading text-[11px] uppercase tracking-[0.16em] text-muted-foreground">
        Kit da classe{' '}
        <span class="normal-case tracking-normal">
          · {STARTING_KIT_BASE_ITEMS.join(' · ')} (automático)
        </span>
      </p>

      <div class="flex flex-wrap gap-3">
        <WeaponChooser
          anchor="chooser-arma-simples"
          label="Arma simples · escolha 1"
          category="weapon-simple"
          value={draft.values.startingWeaponSimple ?? ''}
          onPick={(id) => draft.setValue('startingWeaponSimple', id)}
        />
        <Show when={props.kit.weapons === 'simples+marcial'}>
          <WeaponChooser
            anchor="chooser-arma-marcial"
            label="Arma marcial · escolha 1 (proficiente)"
            category="weapon-martial"
            value={draft.values.startingWeaponMartial ?? ''}
            onPick={(id) => draft.setValue('startingWeaponMartial', id)}
          />
        </Show>
      </div>

      <ArmorChooser kit={props.kit} />
    </div>
  )
}

function WeaponChooser(props: {
  anchor: string
  label: string
  category: 'weapon-simple' | 'weapon-martial'
  value: string
  onPick: (id: string) => void
}) {
  const options = () =>
    weaponOptions(props.category).map((weapon) => ({ value: weapon.id, label: weapon.name }))
  const selected = () => options().find((option) => option.value === props.value) ?? null

  return (
    <div class="min-w-52 space-y-1" id={props.anchor}>
      <p class="font-heading text-[10px] uppercase tracking-[0.14em] text-muted-foreground">
        {props.label}
      </p>
      <Select
        options={options()}
        value={selected()}
        onChange={(option) => props.onPick(option?.value ?? '')}
        placeholder="Escolher arma"
        size="sm"
        aria-label={props.label}
      />
    </div>
  )
}

function ArmorChooser(props: { kit: StartingKit }) {
  const { draft } = useForge()
  const options = () =>
    props.kit.armor === 'brunea'
      ? [{ value: 'brunea', label: 'Brunea (proficiência em pesadas)' }]
      : lightArmorOptions().map((armor) => ({ value: armor.id, label: armor.name }))

  return (
    <Show
      when={props.kit.armor !== 'nenhuma'}
      fallback={
        <p class="text-xs text-muted-foreground">Arcanistas começam sem armadura (p140).</p>
      }
    >
      <div class="space-y-1.5" id="chooser-armadura">
        <p class="font-heading text-[10px] uppercase tracking-[0.14em] text-muted-foreground">
          Armadura{props.kit.armor === 'brunea' ? ' · brunea' : ' · leve a escolha'}
        </p>
        <div class="flex flex-wrap gap-1.5">
          <For each={options()}>
            {(option) => {
              const active = () => draft.values.startingArmor === option.value
              return (
                <Chip
                  pressed={active()}
                  onClick={() => draft.setValue('startingArmor', active() ? '' : option.value)}
                >
                  {active() ? '✓ ' : ''}
                  {option.label}
                </Chip>
              )
            }}
          </For>
          <Show when={props.kit.shieldLeve}>
            <Chip
              pressed={draft.values.startingShield ?? false}
              onClick={() => draft.setValue('startingShield', !draft.values.startingShield)}
            >
              {draft.values.startingShield ? '✓ Escudo leve' : '+ Escudo leve'}
            </Chip>
          </Show>
        </div>
      </div>
    </Show>
  )
}

function Chip(props: { pressed: boolean; onClick: () => void; children: JSX.Element }) {
  return (
    <button
      type="button"
      aria-pressed={props.pressed}
      onClick={() => props.onClick()}
      class={cn(
        'rounded-sm border px-2.5 py-1.5 text-xs transition-colors sm:py-1',
        props.pressed
          ? 'border-grimorio-gold bg-accent font-medium text-grimorio-gold'
          : 'border-grimorio-iron text-muted-foreground hover:bg-accent',
      )}
    >
      {props.children}
    </button>
  )
}

function ExtrasNote(props: { kit: StartingKit }) {
  return (
    <Show when={props.kit.extras.length > 0}>
      <div class="space-y-1 rounded-sm border border-grimorio-iron p-3">
        <p class="font-heading text-[11px] uppercase tracking-[0.16em] text-muted-foreground">
          Extras da classe
        </p>
        <For each={props.kit.extras}>
          {(extra) => (
            <p class="text-xs">
              {extra.description}
              <span class="block text-[11px] text-muted-foreground">
                {extra.source} — adicione o item escolhido na ficha.
              </span>
            </p>
          )}
        </For>
      </div>
    </Show>
  )
}

/**
 * Starting money (Tabela 3-1). Level 1 rolls 4d6; above it the table gives a
 * flat amount. Either way the T$ an origin already rolled is preserved — it was
 * granted by the past, not by the table.
 */
function MoneyField(props: { level: number; tableMoney: number | null; origemRolled: number }) {
  const { draft } = useForge()
  const rolled = () => draft.values.startingMoneyRolled ?? false

  return (
    <div class="space-y-1.5 rounded-sm border border-grimorio-iron p-3">
      <p class="font-heading text-[11px] uppercase tracking-[0.16em] text-muted-foreground">
        Dinheiro inicial · Tabela 3-1
      </p>
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-sm">T$</span>
        <NumberInput
          min={0}
          max={1_000_000}
          value={draft.values.tibar ?? 0}
          onChange={(value) => draft.setValue('tibar', value)}
          class="w-32"
          aria-label="Tibares iniciais"
          spinnerLabel="tibares"
        />
        <Show
          when={props.tableMoney === null}
          fallback={
            <button
              type="button"
              onClick={() => draft.setValue('tibar', (props.tableMoney ?? 0) + props.origemRolled)}
              class="rounded-sm border border-grimorio-iron px-2.5 py-1.5 text-xs hover:bg-accent"
            >
              Usar Tabela 3-1 (Nv {props.level}): T$ {(props.tableMoney ?? 0).toLocaleString('pt-BR')}
            </button>
          }
        >
          <button
            type="button"
            disabled={rolled()}
            onClick={() => {
              draft.patchValues({
                tibar: rollDice(4, 6) + props.origemRolled,
                startingMoneyRolled: true,
              })
            }}
            class={cn(
              'rounded-sm border border-grimorio-iron px-2.5 py-1.5 text-xs',
              rolled() ? 'opacity-60' : 'hover:bg-accent',
            )}
          >
            {rolled() ? `Rolado (${STARTING_TIBARES_DICE})` : `🎲 Rolar ${STARTING_TIBARES_DICE} (Nv 1)`}
          </button>
        </Show>
      </div>
    </div>
  )
}
