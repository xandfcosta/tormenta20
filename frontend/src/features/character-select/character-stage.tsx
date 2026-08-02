import { useMemo } from 'react'
import type { Character } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'
import {
  characterEffects,
  defenseTotal,
} from '@/entities/character/derived'
import { primaryRole } from './select-helpers'

/**
 * Center stage of the character selector (design "palco + dossiê"): the
 * selected character's portrait on a spotlit pedestal, prev/next peeking
 * dimmed from the sides, engraved Cinzel nameplate + vitals beneath, and the
 * page's single primary CTA. Peeks are restrained coverflow — ±1 at reduced
 * scale, no 3D.
 */
export function CharacterStage({
  selected,
  prev,
  next,
  onStep,
  onOpen,
  onDossier,
  dossierOpen,
}: {
  selected: Character
  prev: Character | null
  next: Character | null
  onStep: (delta: 1 | -1) => void
  onOpen: () => void
  onDossier: () => void
  dossierOpen: boolean
}) {
  const hue = hueFromName(selected.name)
  const defense = useMemo(
    () => defenseTotal(selected, characterEffects(selected)).total,
    [selected],
  )
  return (
    <div className="relative flex min-h-0 flex-1 flex-col items-center justify-center gap-4 py-2">
      {/* spotlight wash in the character's hue */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10"
        style={{
          background: `radial-gradient(ellipse 60% 50% at 50% 42%, oklch(0.55 0.15 ${hue} / 0.14), transparent 70%)`,
        }}
      />
      <div className="flex items-center justify-center gap-4 sm:gap-8">
        <PeekPortrait character={prev} side="left" onClick={() => onStep(-1)} />
        <StagePortrait character={selected} hue={hue} onOpen={onOpen} />
        <PeekPortrait character={next} side="right" onClick={() => onStep(1)} />
      </div>
      {/* pedestal glow */}
      <div
        aria-hidden
        className="pointer-events-none -mt-6 h-4 w-64 rounded-[100%] blur-md sm:w-80"
        style={{ background: `oklch(0.5 0.14 ${hue} / 0.35)` }}
      />
      <Nameplate character={selected} hue={hue} />
      <VitalsRow character={selected} defense={defense} />
      <SummaryLine character={selected} />
      <div className="flex flex-wrap items-center justify-center gap-2">
        <Button size="lg" onClick={onOpen}>
          Abrir ficha <kbd className="ml-1 text-[10px] opacity-70">⏎</kbd>
        </Button>
        <Button variant="outline" onClick={onDossier} aria-pressed={dossierOpen}>
          Dossiê <kbd className="ml-1 text-[10px] opacity-70">D</kbd>
        </Button>
      </div>
    </div>
  )
}

function portraitGradient(name: string): string {
  const hue = hueFromName(name)
  return `linear-gradient(155deg, oklch(0.55 0.15 ${hue}) 0%, oklch(0.30 0.09 ${hue}) 70%, oklch(0.22 0.06 ${hue}) 100%)`
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  return parts
    .slice(0, 2)
    .map((w) => w[0])
    .join('')
    .toUpperCase()
}

/** The selected character's 3:4 portrait — hue gradient + giant monogram
 *  until real art lands (art will fill the same frame). */
function StagePortrait({
  character,
  hue,
  onOpen,
}: {
  character: Character
  hue: number
  onOpen: () => void
}) {
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label={`Abrir ficha de ${character.name}`}
      className="relative aspect-[3/4] w-44 overflow-hidden rounded-lg outline outline-2 outline-offset-4 transition-transform hover:scale-[1.01] sm:w-56 lg:w-64"
      style={{
        background: portraitGradient(character.name),
        outlineColor: `oklch(0.6 0.16 ${hue} / 0.6)`,
      }}
    >
      <span className="absolute inset-0 flex select-none items-center justify-center font-display text-[7rem] leading-none text-white/20 sm:text-[9rem]">
        {initials(character.name)}
      </span>
      <span className="absolute inset-x-0 bottom-0 h-1/4 bg-gradient-to-t from-black/50 to-transparent" />
    </button>
  )
}

/** Dimmed side peek (±1). Click steps the selection toward it. */
function PeekPortrait({
  character,
  side,
  onClick,
}: {
  character: Character | null
  side: 'left' | 'right'
  onClick: () => void
}) {
  if (!character) return <div className="w-20 sm:w-28 lg:w-32" aria-hidden />
  return (
    <button
      type="button"
      onClick={onClick}
      title={character.name}
      aria-label={`${side === 'left' ? 'Anterior' : 'Próximo'}: ${character.name}`}
      className="group relative aspect-[3/4] w-20 overflow-hidden rounded-md opacity-50 transition-all hover:opacity-80 sm:w-28 lg:w-32"
      style={{ background: portraitGradient(character.name) }}
    >
      <span className="absolute inset-0 flex select-none items-center justify-center font-display text-4xl text-white/20 sm:text-5xl">
        {initials(character.name)}
      </span>
      <span className="absolute inset-x-0 bottom-0 truncate bg-black/50 px-1 py-0.5 text-center text-[10px] text-white/80 opacity-0 transition-opacity group-hover:opacity-100">
        {character.name}
      </span>
    </button>
  )
}

/** Engraved Cinzel nameplate + class/race kicker. */
function Nameplate({ character, hue }: { character: Character; hue: number }) {
  const race = character.races[0]?.race
  return (
    <div className="max-w-2xl px-4 text-center">
      <h2
        className="font-display text-2xl uppercase tracking-[0.12em] sm:text-4xl"
        style={{ textShadow: `0 1px 0 oklch(0.2 0 0), 0 0 24px oklch(0.55 0.15 ${hue} / 0.35)` }}
      >
        {character.name}
      </h2>
      <p className="mt-1 text-[11px] font-semibold uppercase tracking-[0.25em] text-muted-foreground">
        {primaryRole(character)}
        {race ? ` · ${race}` : ''}
      </p>
    </div>
  )
}

function VitalsRow({
  character,
  defense,
}: {
  character: Character
  defense: number
}) {
  return (
    <div className="flex items-center gap-5 text-sm">
      <Vital label="DEF" value={String(defense)} />
      <Vital
        label="PV"
        value={`${character.hpCurrent}/${character.hpMax}`}
        tone="text-[color:var(--hp-full)]"
      />
      <Vital
        label="PM"
        value={`${character.mpCurrent}/${character.mpMax}`}
        tone="text-[color:var(--primary)]"
        dim={character.mpMax === 0}
      />
    </div>
  )
}

function Vital({
  label,
  value,
  tone,
  dim = false,
}: {
  label: string
  value: string
  tone?: string
  dim?: boolean
}) {
  return (
    <span className={cn('flex items-baseline gap-1.5', dim && 'opacity-50')}>
      <span className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">
        {label}
      </span>
      <span className={cn('font-mono text-lg font-semibold', tone)}>{value}</span>
    </span>
  )
}

function SummaryLine({ character }: { character: Character }) {
  const parts = [
    character.god ? `Devoto de ${character.god}` : null,
    character.origin,
    character.size,
  ].filter(Boolean)
  return (
    <p className="text-xs text-muted-foreground">{parts.join(' · ')}</p>
  )
}
