import { useQueryClient } from '@tanstack/solid-query'
import { caminhoSlotFor } from '@/shared/rules/abilities-caminhos'
import type { ClassChoiceBlob, ClassChoices, GeneralPower } from '@/shared/api/catalog-types'
import { For, type JSX, Show, createMemo, createSignal } from 'solid-js'
import { electiveSlotUsage } from '@/entities/character/class-powers'
import { evaluatePrerequisite, parseClassChoices } from '@/entities/character/derived'
import { allGeneralPowers, classPowersFor, devotoOptionsFor } from '@/shared/lib/abilities-cache'
import type { Character } from '@/shared/api/api'
import { Select, type SelectOption } from '@/shared/ui/select'
import { choiceActions } from './choice-mutations'
import { type CardFocus, CollapsibleAbilityCard } from './collapsible-ability-card'
import { GeneralPowersPool } from './general-powers-pool'
import { parseChoices } from './parse-choices'
import { ClassPowerRow } from './power-rows'

/** Value the Select uses for "no pick" — an empty option clears the choice. */
const NO_CHOICE = ''

/**
 * Per-class subpath picks: devoto (Clérigo/Paladino/Druida) and caminho
 * (Arcanista L1, Paladino/Cavaleiro L5). Renders nothing when the class has no
 * slot or the character hasn't reached the caminho's minLevel.
 *
 * A `Select` rather than the searchable picker: both lists are short and the
 * control has to SHOW the current pick, which `PickerCombobox` deliberately
 * never does.
 */
function ClassChoicesPicker(props: {
  character: Character
  className: string
  level: number
  classChoices: ClassChoices
}) {
  const queryClient = useQueryClient()
  const [pending, setPending] = createSignal(false)

  const devotoOptions = () => devotoOptionsFor(props.className)
  const caminhoSlot = () => caminhoSlotFor(props.className)
  const showCaminho = () => {
    const slot = caminhoSlot()
    return slot !== null && props.level >= slot.minLevel
  }
  const blob = (): ClassChoiceBlob => props.classChoices[props.className] ?? {}

  /** A class with neither pick left is dropped from the blob entirely, so the
   *  stored JSON never grows keys that mean "nothing chosen". */
  const commit = async (nextBlob: ClassChoiceBlob) => {
    const next: ClassChoices = { ...props.classChoices }
    if (nextBlob.devoto || nextBlob.caminho) next[props.className] = nextBlob
    else delete next[props.className]
    setPending(true)
    try {
      await choiceActions(queryClient, props.character.id).setClassChoices(next)
    } catch {
      // choiceActions already rolled back and told the player.
    } finally {
      setPending(false)
    }
  }

  return (
    <Show when={devotoOptions() !== null || showCaminho()}>
      <div class="mb-3 space-y-2">
        <p class="text-3xs font-semibold uppercase tracking-wide text-muted-foreground">
          Escolhas
        </p>
        <Show when={devotoOptions()}>
          {(deuses) => (
            <ChoiceSelect
              label="Devoto"
              ariaLabel={`Devoto de ${props.className}`}
              emptyLabel="Sem devoto"
              options={deuses().map((deus) => ({ value: deus.id, label: deus.name }))}
              value={blob().devoto ?? NO_CHOICE}
              disabled={pending()}
              onChange={(value) =>
                void commit({ ...blob(), devoto: value || undefined })
              }
            />
          )}
        </Show>
        <Show when={showCaminho() && caminhoSlot()}>
          {(slot) => (
            <ChoiceSelect
              label="Caminho"
              ariaLabel={`Caminho de ${props.className}`}
              emptyLabel="Não escolhido"
              options={slot().options.map((option) => ({
                value: option.id,
                label: option.name,
              }))}
              value={blob().caminho ?? NO_CHOICE}
              disabled={pending()}
              onChange={(value) =>
                void commit({ ...blob(), caminho: value || undefined })
              }
            />
          )}
        </Show>
      </div>
    </Show>
  )
}

function ChoiceSelect(props: {
  label: string
  ariaLabel: string
  emptyLabel: string
  options: SelectOption<string>[]
  value: string
  disabled: boolean
  onChange: (value: string) => void
}) {
  const options = createMemo<SelectOption<string>[]>(() => [
    { value: NO_CHOICE, label: props.emptyLabel },
    ...props.options,
  ])
  const selected = () => options().find((option) => option.value === props.value) ?? null

  return (
    <div>
      <p class="mb-1 text-2xs text-muted-foreground">{props.label}</p>
      <Select
        aria-label={props.ariaLabel}
        options={options()}
        value={selected()}
        disabled={props.disabled}
        placeholder={props.emptyLabel}
        size="sm"
        class="w-full"
        onChange={(option) => props.onChange(option?.value ?? NO_CHOICE)}
      />
    </div>
  )
}

/**
 * One class card from `character.classes`: the powers it grants automatically,
 * its elective pool, the general-power pool any slot may be spent on (p33), and
 * the devoto/caminho picker when the class has one.
 */
export function ClassesSection(props: {
  entry: { className: string; level: number }
  character: Character
  focus: CardFocus
  pending: number
}) {
  const queryClient = useQueryClient()
  const [saving, setSaving] = createSignal(false)

  const chosen = createMemo(() => parseChoices(props.character.classPowers))
  const classChoices = createMemo(() => parseClassChoices(props.character.classChoices))
  const pool = createMemo(() => classPowersFor(props.entry.className))

  const granted = createMemo(() =>
    pool()
      .filter((p) => p.grantedAtLevel !== undefined && p.grantedAtLevel <= props.entry.level)
      .sort((a, b) => (a.grantedAtLevel ?? 0) - (b.grantedAtLevel ?? 0)),
  )
  const electives = createMemo(() =>
    pool()
      .filter((p) => p.grantedAtLevel === undefined)
      .sort((a, b) => (a.minLevel ?? 1) - (b.minLevel ?? 1)),
  )
  const slots = createMemo(() =>
    electiveSlotUsage(props.entry.className, props.entry.level, chosen()),
  )
  const generalPool = createMemo(() => allGeneralPowers())

  const toggleElective = async (powerId: string) => {
    const owned = chosen().includes(powerId)
    if (!owned && slots().remaining <= 0) return
    const next = owned
      ? chosen().filter((id) => id !== powerId)
      : [...chosen(), powerId]
    setSaving(true)
    try {
      await choiceActions(queryClient, props.character.id).setClassPowers(next)
    } catch {
      // choiceActions already rolled back and told the player.
    } finally {
      setSaving(false)
    }
  }

  const generalLabel = () =>
    [...new Set(generalPool().map((p) => p.kind))].join(', ') || 'sem pools'

  return (
    <CollapsibleAbilityCard
      id={`classe:${props.entry.className}`}
      title={`${props.entry.className} ${props.entry.level}`}
      count={
        slots().total > 0
          ? `${slots().used}/${slots().total} poderes · ${slots().remaining} restantes`
          : undefined
      }
      pending={props.pending}
      defaultOpen={props.pending > 0}
      focus={props.focus}
    >
      <Show
        when={pool().length > 0}
        fallback={<p class="text-xs italic text-muted-foreground">Classe não está no catálogo.</p>}
      >
        <div class="space-y-3">
          <ClassChoicesPicker
            character={props.character}
            className={props.entry.className}
            level={props.entry.level}
            classChoices={classChoices()}
          />

          <Show when={granted().length > 0}>
            <PowerGroup title="Concedidos">
              <For each={granted()}>{(power) => <ClassPowerRow power={power} owned />}</For>
            </PowerGroup>
          </Show>

          <Show when={electives().length > 0}>
            <PowerGroup title={`Poderes de ${props.entry.className}`}>
              <For each={electives()}>
                {(power) => {
                  const owned = () => chosen().includes(power.id)
                  const checks = () =>
                    (power.prerequisites ?? []).map((prereq) =>
                      evaluatePrerequisite(
                        prereq,
                        props.character,
                        new Set(chosen()),
                        classChoices(),
                      ),
                    )
                  const locked = () =>
                    (power.minLevel ?? 1) > props.entry.level ||
                    checks().some((check) => !check.met) ||
                    (slots().remaining <= 0 && !owned())
                  return (
                    <ClassPowerRow
                      power={power}
                      owned={owned()}
                      locked={locked()}
                      prereqChecks={checks()}
                      disabled={saving()}
                      onToggle={() => void toggleElective(power.id)}
                    />
                  )
                }}
              </For>
            </PowerGroup>
          </Show>

          <Show when={generalPool().length > 0}>
            <div>
              <GroupTitle>Poderes Gerais ({generalLabel()})</GroupTitle>
              <GeneralPowersPool
                powers={generalPool()}
                isOwned={(id) => chosen().includes(id)}
                isLocked={(power: GeneralPower) =>
                  (power.minLevel ?? 1) > props.entry.level ||
                  (slots().remaining <= 0 && !chosen().includes(power.id))
                }
                disabled={saving()}
                onToggle={(id) => void toggleElective(id)}
              />
            </div>
          </Show>
        </div>
      </Show>
    </CollapsibleAbilityCard>
  )
}

function GroupTitle(props: { children: JSX.Element }) {
  return (
    <p class="mb-1 text-3xs font-semibold uppercase tracking-wide text-muted-foreground">
      {props.children}
    </p>
  )
}

function PowerGroup(props: { title: string; children: JSX.Element }) {
  return (
    <div>
      <GroupTitle>{props.title}</GroupTitle>
      <ul class="space-y-1.5">{props.children}</ul>
    </div>
  )
}
