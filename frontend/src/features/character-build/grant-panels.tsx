import { ATTRIBUTE_ABBR, type AttributeKey } from '@tormenta20/t20-data'
import { ChevronRight } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { AbilityLine } from '@/shared/ui/ability-line'
import { Badge } from '@/shared/ui/badge'
import { cn } from '@/shared/lib/utils'
import { classGrant, type GrantLine, originGrant, signed } from './grant-helpers'

/**
 * Inline "what this pick grants" preview. Rendered live under each picker in
 * the creation wizard so the player sees a race/class/origin's attribute
 * deltas and abilities before committing — no hidden bonuses.
 */
export function GrantBox({
  title,
  children,
}: {
  title: string
  children: ReactNode
}) {
  return (
    <div className="space-y-2 rounded-md border border-border bg-muted/30 p-3 text-sm">
      <p className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
        {title}
      </p>
      {children}
    </div>
  )
}

/**
 * Collapsed ability/benefit list — shows `▸ N label` and expands the full
 * AbilityLine list on click. Keeps constrained wizard steps within the
 * viewport (the prose is reference, not a decision input at commit time).
 */
export function AbilityDisclosure({
  label,
  singular,
  lines,
  defaultOpen = false,
}: {
  /** Plural noun, e.g. "habilidades". */
  label: string
  /** Singular noun for count === 1, e.g. "habilidade". Falls back to label. */
  singular?: string
  lines: GrantLine[]
  defaultOpen?: boolean
}) {
  const [open, setOpen] = useState(defaultOpen)
  if (lines.length === 0) return null
  const noun = lines.length === 1 ? (singular ?? label) : label
  return (
    <div className="space-y-1.5">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        className="flex items-center gap-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground hover:text-foreground"
      >
        <ChevronRight
          className={cn('size-3 transition-transform', open && 'rotate-90')}
          aria-hidden
        />
        {lines.length} {noun}
      </button>
      {open && (
        <ul className="space-y-1.5">
          {lines.map((line) => (
            <AbilityLine
              key={line.id}
              name={line.name}
              description={line.description}
            />
          ))}
        </ul>
      )}
    </div>
  )
}

function GrantList({ label, lines }: { label: string; lines: GrantLine[] }) {
  if (lines.length === 0) return null
  return (
    <div className="space-y-1">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        {label}
      </p>
      <ul className="space-y-1.5">
        {lines.map((line) => (
          <AbilityLine
            key={line.id}
            name={line.name}
            description={line.description}
          />
        ))}
      </ul>
    </div>
  )
}

/**
 * Signed attribute-delta chips (`+2 CON`, `−1 DES`). Negatives use the outline
 * variant so a penalty reads distinctly from a bonus. Shared by the race
 * picker's selected-race detail and its subrace option cards.
 */
export function DeltaBadges({
  deltas,
}: {
  deltas: Partial<Record<AttributeKey, number>>
}) {
  const entries = (Object.entries(deltas) as [AttributeKey, number][]).filter(
    ([, v]) => v !== 0,
  )
  if (entries.length === 0) return null
  return (
    <div className="flex flex-wrap gap-1.5">
      {entries.map(([key, amount]) => (
        <Badge key={key} variant={amount < 0 ? 'outline' : 'secondary'}>
          {signed(amount)} {ATTRIBUTE_ABBR[key]}
        </Badge>
      ))}
    </div>
  )
}

export function ClassGrantPanel({
  className,
  level = 1,
}: {
  className: string
  level?: number
}) {
  const grant = classGrant(className, level)
  return (
    <GrantBox title={className}>
      {grant.vitals && (
        <p className="text-xs text-muted-foreground">
          PV {grant.vitals.pvInicial} inicial (+{grant.vitals.pvPerLevel}/nível)
          {' · '}
          PM +{grant.vitals.mpPerLevel}/nível
        </p>
      )}
      <AbilityDisclosure
        label="habilidades automáticas"
        singular="habilidade automática"
        lines={grant.powers}
      />
    </GrantBox>
  )
}

export function OriginGrantPanel({
  originId,
  collapsible = false,
}: {
  originId: string
  collapsible?: boolean
}) {
  const grant = originGrant(originId)
  if (!grant) return null
  const poder = grant.poderUnico ? [grant.poderUnico] : []
  return (
    <GrantBox title={grant.name}>
      <p className="text-xs text-muted-foreground">Escolha 2 benefícios:</p>
      {collapsible ? (
        <>
          <AbilityDisclosure
            label="benefícios"
            singular="benefício"
            lines={grant.benefits}
          />
          <AbilityDisclosure label="poder único" lines={poder} />
        </>
      ) : (
        <>
          <GrantList label="Benefícios" lines={grant.benefits} />
          {grant.poderUnico && (
            <GrantList label="Poder único" lines={[grant.poderUnico]} />
          )}
        </>
      )}
    </GrantBox>
  )
}
