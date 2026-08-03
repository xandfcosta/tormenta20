import { BookOpen, ChevronRight, Flame, Sparkles, Zap } from 'lucide-react'
import { useState } from 'react'
import type { Character } from '@/shared/api/api'
import { useAllConditionals } from '@/entities/character/derived'
import { dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { Badge } from '@/shared/ui/badge'
import type { ActivationKind } from '@tormenta20/t20-data'
import {
  activeTriggeredPassives,
  gatilhoLabel,
  groupPlayPowers,
  shortSourceLabel,
  type PlayPower,
} from './power-play-groups'
import { PowerActionSlot } from './power-action-slot'
import { ownedPowerSpec } from './power-spec-resolver'
import { ownedAbilities } from './sheet-search-index'

/** One owned power joined with its activation spec (undefined = no entry). */
/**
 * Play-mode Poderes list (Phase D): AÇÕES ordered for the table on top,
 * PASSIVAS collapsed behind a disclosure. Rows are one-line summaries whose
 * tap expands the full rule text; the action affordance stays inline.
 */
export function PowerPlayList({ character }: { character: Character }) {
  const conditionals = useAllConditionals(character)
  const activeFlags = new Set(
    conditionals.flatMap((c) => (c.active && c.effect.flag ? [c.effect.flag] : [])),
  )
  const powers = ownedAbilities(character).map((entry) => ({
    entry,
    spec: ownedPowerSpec(entry),
  }))
  const { acoes, passivas } = groupPlayPowers(powers, activeFlags)
  if (powers.length === 0) {
    return (
      <p className={cn('text-xs italic', dimText)}>
        Nenhum poder ou habilidade.
      </p>
    )
  }
  return (
    <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
      <section className="space-y-1.5">
        <GroupHeading>Ações</GroupHeading>
        {acoes.length === 0 ? (
          <p className={cn('text-xs italic', dimText)}>Nenhuma ação ativável.</p>
        ) : (
          <ul className="space-y-1.5">
            {acoes.map((p) => (
              <PlayPowerRow key={rowKey(p)} power={p} character={character} />
            ))}
          </ul>
        )}
      </section>
      <PassivasDisclosure
        passivas={passivas}
        activeFlags={activeFlags}
        character={character}
      />
    </div>
  )
}

function rowKey(power: PlayPower): string {
  return `${power.entry.source}-${power.entry.name}`
}

function GroupHeading({ children }: { children: string }) {
  return (
    <h4 className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
      {children}
    </h4>
  )
}

/**
 * PASSIVAS live behind `mostrar (N)` — they already flow through the sheet's
 * numbers, so at the table they're reference, not actions. While collapsed,
 * any triggered passive whose gatilho is up gets a live ● line.
 */
function PassivasDisclosure({
  passivas,
  activeFlags,
  character,
}: {
  passivas: readonly PlayPower[]
  activeFlags: ReadonlySet<string>
  character: Character
}) {
  const [open, setOpen] = useState(false)
  if (passivas.length === 0) return null
  return (
    <section className="space-y-1.5">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex items-center gap-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground hover:text-foreground"
      >
        <ChevronRight
          className={cn('size-3.5 transition-transform', open && 'rotate-90')}
        />
        Passivas · {open ? 'ocultar' : `mostrar (${passivas.length})`}
      </button>
      {!open &&
        activeTriggeredPassives(passivas, activeFlags).map((p) => (
          <p
            key={rowKey(p)}
            className="text-[11px] text-emerald-700 dark:text-emerald-300"
          >
            ● gatilho ativo: {p.entry.name} ({gatilhoLabel(p.spec)})
          </p>
        ))}
      {open && (
        <ul className="space-y-1.5">
          {passivas.map((p) => (
            <PlayPowerRow key={rowKey(p)} power={p} character={character} />
          ))}
        </ul>
      )}
    </section>
  )
}

/**
 * One power, one line: kind icon + name + short source badge, action slot
 * right-aligned, rule text clamped to a dim line. Tapping the text area (not
 * the action buttons) expands the full description + book page. On a 390px
 * phone the flex wraps into two visual lines; the slot keeps its 44px target.
 */
function PlayPowerRow({
  power,
  character,
}: {
  power: PlayPower
  character: Character
}) {
  const [open, setOpen] = useState(false)
  const { entry, spec } = power
  return (
    <li className="rounded border border-border p-2">
      <div className="flex flex-wrap items-start gap-1.5">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          aria-expanded={open}
          aria-label={`Detalhes de ${entry.name}`}
          className="flex min-w-0 flex-1 basis-40 flex-col gap-0.5 text-left"
        >
          <span className="flex flex-wrap items-center gap-1.5">
            <PowerKindIcon kind={spec?.kind} />
            <span className="text-xs font-semibold">{entry.name}</span>
            <Badge variant="secondary" className="px-1 py-0 text-[9px]">
              {shortSourceLabel(entry.source)}
            </Badge>
          </span>
          {!open && (
            <span
              className={cn('line-clamp-1 text-[11px] leading-snug', dimText)}
            >
              {entry.detail}
            </span>
          )}
        </button>
        <PowerActionSlot
          spec={spec}
          character={character}
          className="ml-auto shrink-0 justify-end"
        />
      </div>
      {open && (
        <p className={cn('mt-1 text-[11px] leading-snug', dimText)}>
          {entry.detail}
          {spec && <span className="ml-1 opacity-70">(p{spec.bookPage})</span>}
        </p>
      )}
    </li>
  )
}

function PowerKindIcon({ kind }: { kind: ActivationKind | undefined }) {
  const iconClass = 'size-3.5 shrink-0 text-muted-foreground'
  if (kind === 'instant') return <Zap className={iconClass} />
  if (kind === 'stance') return <Flame className={iconClass} />
  if (kind === 'triggered-passive') return <Sparkles className={iconClass} />
  return <BookOpen className={iconClass} />
}
