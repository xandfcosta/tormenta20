import { ATTRIBUTE_ABBR, ATTRIBUTE_KEYS, type AttributeKey } from '@/shared/api/attribute-keys'
import { TriangleAlert } from 'lucide-solid'
import { For, type JSX, Show } from 'solid-js'
import { totalClassLevel } from '@/features/character-build/class-entries'
import {
  type Pendencia,
  creationBlockers,
  creationPendencias,
} from '@/features/character-build/creation-pendencias'
import { deriveDraftDefense } from '@/features/character-build/draft-defense'
import { deriveDraftVitals } from '@/features/character-build/draft-vitals'
import { useForge } from '@/features/character-build/forge-context'
import {
  type RaceChoiceState,
  appliedRaceDeltas,
  originGrant,
  signed,
} from '@/features/character-build/grant-helpers'
import { draftPowerPool } from '@/features/character-build/power-pool'
import { bagagemGroups, startingLoadout } from '@/features/character-build/starting-equipment'
import type { CharacterFormValues } from '@/features/character-build/wizard-steps'

/**
 * Last step: the character as a sheet, before it is one. Whatever is still
 * missing is said at the top in amber — none of it blocks the forge, because a
 * half-finished character is a legitimate thing to save and finish at the
 * table; but the Resumo is the last screen where the draft still exists.
 */
export function ResumoStep() {
  const { draft } = useForge()

  const values = () => draft.values
  const blockers = () => creationBlockers(values(), draft.raceChoices)
  const pendencias = () => creationPendencias(values(), draft.raceChoices)
  const vitals = () => deriveDraftVitals(values(), draft.raceChoices)
  const defense = () => deriveDraftDefense(values(), draft.raceChoices)

  const lineage = () => {
    const classes = values().classes.filter((entry) => entry.className)
    return [
      values().races.join(' · '),
      classes.length > 0
        ? `${classes.map((c) => c.className).join(' · ')} Nv ${totalClassLevel(classes)}`
        : '',
      values().origin,
      values().size,
    ]
      .filter(Boolean)
      .join(' · ')
  }

  const totals = () => {
    const deltas = appliedRaceDeltas(values().races, draft.raceChoices)
    return ATTRIBUTE_KEYS.map((key: AttributeKey) => ({
      key,
      abbr: ATTRIBUTE_ABBR[key],
      total: values()[key] + (deltas[key] ?? 0),
    }))
  }

  return (
    <section class="flex min-h-0 flex-1 flex-col gap-4" aria-labelledby="forge-step-resumo">
      <h2 id="forge-step-resumo" class="sr-only">
        Resumo
      </h2>

      <BlockersBanner items={blockers()} />
      <PendenciasBanner items={pendencias()} />

      <div class="space-y-1 text-center">
        <p class="font-heading text-2xl uppercase tracking-[0.12em] text-grimorio-gold sm:text-3xl">
          {values().name.trim() || 'Sem nome'}
        </p>
        <p class="text-xs text-muted-foreground">{lineage()}</p>
      </div>

      <div class="flex flex-wrap items-center justify-center gap-x-6 gap-y-2 border-y border-grimorio-iron py-2 font-mono text-sm">
        <Stat label="Defesa" abbr="DEF" value={defense()} />
        <Stat label="Pontos de vida" abbr="PV" value={vitals().pvMax} />
        <Stat label="Pontos de mana" abbr="PM" value={vitals().pmMax} />
        <Stat label="Deslocamento em metros" abbr="Desl" value={values().displacement} suffix="m" />
      </div>

      <div class="flex flex-wrap justify-center gap-x-5 gap-y-1 font-mono text-sm">
        <For each={totals()}>
          {(attribute) => (
            <span class="flex items-baseline gap-1">
              <span aria-hidden="true" class="text-[10px] uppercase tracking-widest text-muted-foreground">
                {attribute.abbr}
              </span>
              <span aria-hidden="true" class="tabular-nums text-foreground">
                {signed(attribute.total)}
              </span>
              <span class="sr-only">{`${attribute.abbr} ${attribute.total}`}</span>
            </span>
          )}
        </For>
      </div>

      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <SummaryCard title="Perícias" lines={values().trainedExpertises} empty="Nenhuma treinada." />
        <SummaryCard
          title="Poderes"
          lines={chosenPowerNames(values(), draft.raceChoices)}
          empty="Nenhum poder escolhido."
        />
        <SummaryCard
          title="Origem"
          lines={originBenefitNames(values())}
          empty="Nenhum benefício."
        />
        <SummaryCard title="Bagagem" lines={bagNames(values())} empty="Nada na mochila." />
      </div>
    </section>
  )
}

/** Names of the elective powers taken, resolved off the same pool the Poderes
 *  step drew from — an id in the summary would be a leak of the data model. */
function chosenPowerNames(
  values: CharacterFormValues,
  raceChoices: RaceChoiceState,
): string[] {
  const byId = new Map(
    draftPowerPool(values, raceChoices).map((option) => [option.id, option.name]),
  )
  return values.classPowers.map((id) => byId.get(id) ?? id)
}

function originBenefitNames(values: CharacterFormValues): string[] {
  const grant = values.origin ? originGrant(values.origin) : null
  if (!grant) return []
  const pool = grant.poderUnico ? [...grant.benefits, grant.poderUnico] : grant.benefits
  return values.originChoices.map((id) => pool.find((benefit) => benefit.id === id)?.name ?? id)
}

function bagNames(values: CharacterFormValues): string[] {
  const primary = values.classes[0]?.className
  if (!primary) return []
  const level = totalClassLevel(values.classes) || 1
  const groups = bagagemGroups(
    {
      weaponSimple: values.startingWeaponSimple ?? '',
      weaponMartial: values.startingWeaponMartial ?? '',
      armor: values.startingArmor ?? '',
      shield: values.startingShield ?? false,
    },
    startingLoadout(primary, level).kit,
    values.origin,
    values.originItemPicks ?? {},
    values.startingPurchases ?? {},
  )
  return groups.flatMap((group) =>
    group.lines.map((line) =>
      line.kind === 'item'
        ? `${line.name}${line.qty > 1 ? ` ×${line.qty}` : ''}`
        : `${line.label} (pendente)`,
    ),
  )
}

/**
 * What is stopping the forge. Red, not amber, and above the soft pendências:
 * the "Criar personagem" button is disabled while any of these stand, and a
 * dead button whose reason is off screen is the worst last screen there is.
 */
function BlockersBanner(props: { items: Pendencia[] }) {
  return (
    <Show when={props.items.length > 0}>
      <div class="space-y-1 rounded-md border border-[color:var(--hp-critical)]/60 bg-[color:var(--hp-critical)]/10 p-3">
        <p class="flex items-center gap-1.5 text-xs font-semibold text-[color:var(--hp-critical)]">
          <TriangleAlert aria-hidden="true" class="size-3.5" />
          Falta o essencial para forjar
        </p>
        <ul class="space-y-0.5">
          <For each={props.items}>
            {(item) => <li class="text-[11px] text-foreground">· {item.label}</li>}
          </For>
        </ul>
      </div>
    </Show>
  )
}

function PendenciasBanner(props: { items: Pendencia[] }) {
  return (
    <Show when={props.items.length > 0}>
      <div class="space-y-1 rounded-md border border-[color:var(--hp-hurt)]/50 bg-[color:var(--hp-hurt)]/10 p-3">
        <p class="flex items-center gap-1.5 text-xs font-semibold text-[color:var(--hp-hurt)]">
          <TriangleAlert aria-hidden="true" class="size-3.5" />
          {props.items.length === 1
            ? '1 pendência'
            : `${props.items.length} pendências`}{' '}
          — dá para criar assim e terminar na ficha
        </p>
        <ul class="space-y-0.5">
          <For each={props.items}>
            {(item) => <li class="text-[11px] text-muted-foreground">· {item.label}</li>}
          </For>
        </ul>
      </div>
    </Show>
  )
}

function Stat(props: { label: string; abbr: string; value: number; suffix?: string }) {
  return (
    <span class="flex items-baseline gap-1">
      <span aria-hidden="true" class="text-[10px] uppercase tracking-widest text-muted-foreground">
        {props.abbr}
      </span>
      <span aria-hidden="true" class="text-lg tabular-nums text-grimorio-gold">
        {props.value}
        {props.suffix}
      </span>
      {/* One interpolation, not `{label} {value}`: two expressions render two
          text nodes and the announced string comes out split. */}
      <span class="sr-only">{`${props.label} ${props.value}`}</span>
    </span>
  )
}

function SummaryCard(props: { title: string; lines: string[]; empty: string }): JSX.Element {
  return (
    <section
      aria-label={props.title}
      class="space-y-1 rounded-md border border-grimorio-iron bg-muted/10 p-3"
    >
      <p class="font-heading text-[11px] uppercase tracking-[0.16em] text-grimorio-gold">
        {props.title}
      </p>
      <Show
        when={props.lines.length > 0}
        fallback={<p class="text-[11px] text-muted-foreground">{props.empty}</p>}
      >
        <ul class="space-y-0.5">
          <For each={props.lines}>
            {(line) => <li class="text-[11px] text-foreground">· {line}</li>}
          </For>
        </ul>
      </Show>
    </section>
  )
}
