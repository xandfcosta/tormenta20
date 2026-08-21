import type { ActivationKind } from '@/shared/api/catalog-types'
import { BookOpen, ChevronRight, Flame, Sparkles, Zap } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import { Dynamic } from 'solid-js/web'
import { allConditionals } from '@/entities/character/derived'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { cn } from '@/shared/lib/utils'
import { PowerActionSlot } from './power-action-slot'
import {
  type PlayPower,
  activeTriggeredPassives,
  gatilhoLabel,
  groupPlayPowers,
  shortSourceLabel,
} from './power-play-groups'
import { ownedPowerSpec } from './power-spec-resolver'
import { ownedAbilities } from './sheet-search-index'

/**
 * Play-mode Poderes list: AÇÕES ordered for the table on top, PASSIVAS
 * collapsed behind a disclosure. Rows are one-line summaries whose tap expands
 * the full rule text; the action affordance stays inline.
 */
export function PowerPlayList(props: { character: Character }) {
  const conditionals = useConditionals()

  const activeFlags = createMemo(
    () =>
      new Set(
        allConditionals(props.character, conditionals.active(props.character.id)).flatMap(
          (entry) => (entry.active && entry.effect.flag ? [entry.effect.flag] : []),
        ),
      ),
  )
  const powers = createMemo(() =>
    ownedAbilities(props.character).map((entry) => ({ entry, spec: ownedPowerSpec(entry) })),
  )
  const groups = createMemo(() => groupPlayPowers(powers(), activeFlags()))

  return (
    <Show
      when={powers().length > 0}
      fallback={<p class="text-xs italic text-muted-foreground">Nenhum poder ou habilidade.</p>}
    >
      <div class="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
        <section class="space-y-1.5">
          <GroupHeading>Ações</GroupHeading>
          <Show
            when={groups().acoes.length > 0}
            fallback={<p class="text-xs italic text-muted-foreground">Nenhuma ação ativável.</p>}
          >
            <ul class="space-y-1.5">
              <For each={groups().acoes}>
                {(power) => <PlayPowerRow power={power} character={props.character} />}
              </For>
            </ul>
          </Show>
        </section>
        <PassivasDisclosure
          passivas={groups().passivas}
          activeFlags={activeFlags()}
          character={props.character}
        />
      </div>
    </Show>
  )
}

function GroupHeading(props: { children: string }) {
  return (
    <h4 class="text-2xs font-semibold uppercase tracking-wide text-muted-foreground">
      {props.children}
    </h4>
  )
}

/**
 * PASSIVAS live behind `mostrar (N)` — they already flow through the sheet's
 * numbers, so at the table they're reference, not actions. While collapsed, a
 * triggered passive whose gatilho is up still gets a live ● line.
 */
function PassivasDisclosure(props: {
  passivas: readonly PlayPower[]
  activeFlags: ReadonlySet<string>
  character: Character
}) {
  const [open, setOpen] = createSignal(false)
  return (
    <Show when={props.passivas.length > 0}>
      <section class="space-y-1.5">
        <button
          type="button"
          onClick={() => setOpen(!open())}
          aria-expanded={open()}
          class="flex items-center gap-1 text-2xs font-semibold uppercase tracking-wide text-muted-foreground hover:text-foreground"
        >
          <ChevronRight
            aria-hidden="true"
            class={cn('size-3.5 transition-transform', open() && 'rotate-90')}
          />
          Passivas · {open() ? 'ocultar' : `mostrar (${props.passivas.length})`}
        </button>
        <Show when={!open()}>
          <For each={activeTriggeredPassives(props.passivas, props.activeFlags)}>
            {(power) => (
              <p class="text-2xs text-bonus-ink">
                ● gatilho ativo: {power.entry.name} ({gatilhoLabel(power.spec)})
              </p>
            )}
          </For>
        </Show>
        <Show when={open()}>
          <ul class="space-y-1.5">
            <For each={props.passivas}>
              {(power) => <PlayPowerRow power={power} character={props.character} />}
            </For>
          </ul>
        </Show>
      </section>
    </Show>
  )
}

const KIND_ICON: Partial<Record<ActivationKind, typeof Zap>> = {
  instant: Zap,
  stance: Flame,
  'triggered-passive': Sparkles,
}

/**
 * One power, one line: kind icon + name + short source badge, action slot
 * right-aligned, rule text clamped to a dim line. Tapping the text area (not
 * the action buttons) expands the full description + book page. At phone width
 * the flex wraps into two visual lines and the slot keeps its 44px target.
 */
function PlayPowerRow(props: { power: PlayPower; character: Character }) {
  const [open, setOpen] = createSignal(false)
  const icon = () => (props.power.spec ? KIND_ICON[props.power.spec.kind] : undefined) ?? BookOpen

  return (
    <li class="rounded-none border border-border p-2">
      <div class="flex flex-wrap items-start gap-1.5">
        <button
          type="button"
          onClick={() => setOpen(!open())}
          aria-expanded={open()}
          aria-label={`Detalhes de ${props.power.entry.name}`}
          class="flex min-w-0 flex-1 basis-40 flex-col gap-0.5 text-left"
        >
          <span class="flex flex-wrap items-center gap-1.5">
            <Dynamic
              component={icon()}
              aria-hidden="true"
              class="size-3.5 shrink-0 text-muted-foreground"
            />
            <span class="text-xs font-semibold">{props.power.entry.name}</span>
            <span class="rounded-none bg-muted px-1 py-0 text-4xs text-muted-foreground">
              {shortSourceLabel(props.power.entry.source)}
            </span>
          </span>
          <Show when={!open()}>
            <span class="line-clamp-1 text-2xs leading-snug text-muted-foreground">
              {props.power.entry.detail}
            </span>
          </Show>
        </button>
        <PowerActionSlot
          spec={props.power.spec}
          character={props.character}
          class="ml-auto shrink-0 justify-end"
        />
      </div>
      <Show when={open()}>
        <p class="mt-1 text-2xs leading-snug text-muted-foreground">
          {props.power.entry.detail}
          <Show when={props.power.spec}>
            {(spec) => <span class="ml-1 opacity-70">(p{spec().bookPage})</span>}
          </Show>
        </p>
      </Show>
    </li>
  )
}
