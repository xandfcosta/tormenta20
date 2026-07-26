import { ATTRIBUTE_ABBR } from '@tormenta20/t20-data'
import type { ReactNode } from 'react'
import { Badge } from '@/shared/ui/badge'
import {
  classGrant,
  type GrantLine,
  originGrant,
  raceGrant,
  signed,
} from './grant-helpers'

/**
 * Inline "what this pick grants" preview. Rendered live under each picker in
 * the creation wizard so the player sees a race/class/origin's attribute
 * deltas and abilities before committing — no hidden bonuses.
 */
function GrantBox({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="space-y-2 rounded-md border border-border bg-muted/30 p-3 text-sm">
      <p className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
        {title}
      </p>
      {children}
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
      <ul className="space-y-1">
        {lines.map((line) => (
          <li key={line.id} className="leading-snug">
            <span className="font-medium">{line.name}</span>
            <span className="text-muted-foreground"> — {line.description}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

export function RaceGrantPanel({ raceId }: { raceId: string }) {
  const grant = raceGrant(raceId)
  if (!grant) return null
  return (
    <GrantBox title={grant.name}>
      {grant.deltas.length > 0 ? (
        <div className="flex flex-wrap gap-1.5">
          {grant.deltas.map(([key, amount]) => (
            <Badge key={key} variant={amount < 0 ? 'outline' : 'secondary'}>
              {signed(amount)} {ATTRIBUTE_ABBR[key]}
            </Badge>
          ))}
        </div>
      ) : (
        <p className="text-xs italic text-muted-foreground">
          Bônus de atributo à escolha — definido na ficha.
        </p>
      )}
      <GrantList label="Habilidades" lines={grant.abilities} />
    </GrantBox>
  )
}

export function ClassGrantPanel({ className }: { className: string }) {
  const grant = classGrant(className)
  return (
    <GrantBox title={className}>
      {grant.vitals && (
        <p className="text-xs text-muted-foreground">
          PV {grant.vitals.pvInicial} inicial (+{grant.vitals.pvPerLevel}/nível)
          {' · '}
          PM +{grant.vitals.mpPerLevel}/nível
        </p>
      )}
      <GrantList label="Habilidades de 1º nível" lines={grant.powers} />
    </GrantBox>
  )
}

export function OriginGrantPanel({ originId }: { originId: string }) {
  const grant = originGrant(originId)
  if (!grant) return null
  return (
    <GrantBox title={grant.name}>
      <p className="text-xs text-muted-foreground">Escolha 2 benefícios:</p>
      <GrantList label="Benefícios" lines={grant.benefits} />
      {grant.poderUnico && (
        <GrantList label="Poder único" lines={[grant.poderUnico]} />
      )}
    </GrantBox>
  )
}
