import { Check, Info } from 'lucide-react'
import type { ClassPower, GeneralPower, PowerKind } from '@tormenta20/t20-data'
import { Badge } from '@/shared/ui/badge'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/shared/ui/popover'
import type { PrerequisiteCheck } from '@/entities/character/derived'
import { accentTitle, subtleText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'

export function PowerKindBadge({ kind }: { kind: PowerKind }) {
  return (
    <Badge variant="secondary" className="text-[9px] uppercase tracking-wide">
      {kind}
    </Badge>
  )
}

/** Owned/available toggle; disabled + dimmed when locked and not owned. */
function PowerCheckbox({
  owned,
  disabled,
  onToggle,
}: {
  owned: boolean
  disabled?: boolean
  onToggle: () => void
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      disabled={disabled}
      className={cn(
        'mt-0.5 flex size-4 shrink-0 items-center justify-center rounded border',
        owned
          ? 'border-primary bg-primary text-primary-foreground'
          : 'border-border hover:bg-muted',
        disabled && 'cursor-not-allowed opacity-40',
      )}
      aria-pressed={owned}
      aria-label={owned ? 'Remover poder' : 'Selecionar poder'}
    >
      {owned ? <Check className="size-3" /> : null}
    </button>
  )
}

/** Info affordance carrying a description off the collapsed row (touch-safe). */
function PowerInfo({ text }: { text: string }) {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="Ver descrição"
          className="flex size-6 shrink-0 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          <Info className="size-3.5" />
        </button>
      </PopoverTrigger>
      {/* No forced side: radix collision-flips, so it never clips off a
          full-width phone panel (the old side="left" did). */}
      <PopoverContent
        align="end"
        className="w-[min(18rem,calc(100vw-2rem))] text-xs leading-snug"
      >
        {text}
      </PopoverContent>
    </Popover>
  )
}

/**
 * A general-power row (browse pool). Description stays inline when owned,
 * otherwise it moves behind the info popover to keep the pool scannable.
 * Rendered inside a virtualized list, so it's a `div`, not an `li`.
 */
export function GeneralPowerRow({
  power,
  owned,
  locked,
  onToggle,
  disabled,
}: {
  power: GeneralPower
  owned: boolean
  locked?: boolean
  onToggle: () => void
  disabled?: boolean
}) {
  return (
    <div
      className={cn(
        'flex items-start gap-2 rounded border border-border p-2',
        owned && 'bg-muted',
      )}
    >
      <PowerCheckbox
        owned={owned}
        disabled={disabled || (locked && !owned)}
        onToggle={onToggle}
      />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1">
          <p className={cn('text-xs font-semibold', accentTitle)}>
            {power.name}
          </p>
          <PowerKindBadge kind={power.kind} />
          {power.minLevel !== undefined && power.minLevel > 1 && (
            <Badge variant="outline" className="text-[9px]">
              ≥L{power.minLevel}
            </Badge>
          )}
          {locked && !owned && (
            <Badge variant="destructive" className="text-[9px] uppercase">
              Bloqueado
            </Badge>
          )}
          {!owned && <PowerInfo text={power.description} />}
        </div>
        {owned && (
          <p className={cn('mt-0.5 text-[11px] leading-snug', subtleText)}>
            {power.description}
          </p>
        )}
      </div>
    </div>
  )
}

/**
 * A class-power row: auto-granted powers show their description inline; unowned
 * electives keep the prereq line inline (it's the actionable "why locked") but
 * push the long description into the info popover.
 */
export function ClassPowerRow({
  power,
  owned,
  locked,
  prereqChecks,
  onToggle,
  disabled,
}: {
  power: ClassPower
  owned: boolean
  locked?: boolean
  prereqChecks?: PrerequisiteCheck[]
  onToggle?: () => void
  disabled?: boolean
}) {
  const isAuto = power.grantedAtLevel !== undefined
  const inlineDescription = owned || isAuto
  return (
    <li
      className={cn(
        'flex items-start gap-2 rounded border border-border p-2',
        owned && 'bg-muted',
      )}
    >
      {!isAuto && onToggle && (
        <PowerCheckbox
          owned={owned}
          disabled={disabled || (locked && !owned)}
          onToggle={onToggle}
        />
      )}
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1">
          <p className={cn('text-xs font-semibold', accentTitle)}>
            {power.name}
          </p>
          {isAuto && (
            <Badge variant="secondary" className="text-[9px]">
              L{power.grantedAtLevel}
            </Badge>
          )}
          {!isAuto && power.minLevel !== undefined && power.minLevel > 1 && (
            <Badge variant="outline" className="text-[9px]">
              ≥L{power.minLevel}
            </Badge>
          )}
          {locked && !owned && (
            <Badge variant="destructive" className="text-[9px] uppercase">
              Bloqueado
            </Badge>
          )}
          {!inlineDescription && <PowerInfo text={power.description} />}
        </div>
        {!owned && prereqChecks && prereqChecks.length > 0 && (
          <p className="mt-0.5 text-[10px] leading-snug">
            <span className={cn('font-semibold', subtleText)}>Requer: </span>
            {prereqChecks.map((c, i) => (
              <span key={`${c.reason}:${i}`}>
                {i > 0 && <span className={subtleText}>, </span>}
                <span
                  className={cn(
                    c.prereq.kind === 'note'
                      ? 'text-foreground'
                      : c.met
                        ? 'text-emerald-700 dark:text-emerald-300'
                        : 'text-red-700 dark:text-red-300',
                  )}
                >
                  {c.reason}
                </span>
              </span>
            ))}
          </p>
        )}
        {inlineDescription && (
          <p className={cn('mt-0.5 text-[11px] leading-snug', subtleText)}>
            {power.description}
          </p>
        )}
      </div>
    </li>
  )
}
