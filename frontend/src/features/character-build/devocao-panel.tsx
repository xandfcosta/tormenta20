import {
  type Deus,
  devotoEligible,
  type GrantedPowerOption,
} from '@tormenta20/t20-data'
import { deuses } from '@/shared/lib/abilities-cache'
import { grantedPowerOptionsFor } from '@/shared/lib/divine-powers-cache'
import { cn } from '@/shared/lib/utils'

const ACTION_LABEL: Record<string, string> = {
  padrao: 'ação padrão',
  movimento: 'ação de movimento',
  livre: 'ação livre',
  reacao: 'reação',
  gratuita: 'ação gratuita',
  completa: 'ação completa',
  passivo: 'passivo',
  varia: 'varia',
}

/**
 * Devoção panel (Identidade step). Becoming devoto grants ONE poder concedido
 * of the player's choice from the god's list (book p96) — this captures that
 * pick. Eligibility (race/class in the god's Devotos line; Humano/Clérigo any)
 * is ADVISORY: incompatible picks warn "negociado com o mestre", not block.
 */
export function DevocaoPanel({
  godName,
  value,
  onChange,
  raceNames,
  classNames,
}: {
  godName: string
  value: string
  onChange: (powerName: string) => void
  raceNames: string[]
  classNames: string[]
}) {
  // Read the primed catalog (warm by loader gate); ~20 deuses, linear find is
  // cheap and avoids a module-level Map that would evaluate before priming.
  const deus = deuses().find((d) => d.name === godName)
  if (!deus) return null
  const powers = grantedPowerOptionsFor(deus.id)
  if (powers.length === 0) return null
  const eligible = devotoEligible(deus, raceNames, classNames)
  return (
    <div className="space-y-2 rounded-lg border border-border p-3 sm:col-span-2">
      <p className="text-xs font-semibold">Devoção a {deus.name}</p>
      <DeusFacts deus={deus} />
      {!eligible && (
        <p className="text-[11px] text-[color:var(--hp-hurt)]">
          Raça/classe fora da lista de devotos de {deus.name} (p96) — negociado
          com o mestre.
        </p>
      )}
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Poder concedido · escolha 1 (p96)
      </p>
      <div className="grid gap-1.5">
        {powers.map((p) => (
          <GrantedPowerRow
            key={p.name}
            power={p}
            selected={value === p.name}
            onPick={() => onChange(value === p.name ? '' : p.name)}
          />
        ))}
      </div>
      <p className="text-[10px] text-muted-foreground">
        Devoto segue as Obrigações &amp; Restrições do deus; violá-las custa
        todos os PM (p96).
      </p>
    </div>
  )
}

function DeusFacts({ deus }: { deus: Deus }) {
  const facts = [
    deus.energia && `Energia: ${deus.energia}`,
    deus.armaPreferida && `Arma: ${deus.armaPreferida}`,
    deus.portfolio,
  ].filter(Boolean)
  if (facts.length === 0) return null
  return (
    <p className="text-[11px] text-muted-foreground">{facts.join(' · ')}</p>
  )
}

function GrantedPowerRow({
  power,
  selected,
  onPick,
}: {
  power: GrantedPowerOption
  selected: boolean
  onPick: () => void
}) {
  const meta = [
    ACTION_LABEL[power.action] ?? power.action,
    power.pmCost === 'variavel'
      ? 'PM variável'
      : power.pmCost > 0
        ? `${power.pmCost} PM`
        : null,
  ]
    .filter(Boolean)
    .join(' · ')
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onPick}
      className={cn(
        'flex items-start gap-2 rounded-md border p-2 text-left transition-colors',
        selected ? 'border-primary bg-accent' : 'border-border hover:bg-accent',
      )}
    >
      <span className="mt-1 flex size-3.5 shrink-0 items-center justify-center rounded-full border border-border">
        {selected && <span className="size-2 rounded-full bg-primary" />}
      </span>
      <span className="min-w-0 flex-1">
        <span className="flex items-baseline gap-1.5">
          <span className="text-xs font-semibold">{power.name}</span>
          <span className="text-[10px] text-muted-foreground">· {meta}</span>
        </span>
        <span className="block text-[11px] leading-snug text-muted-foreground">
          {power.description}
        </span>
      </span>
    </button>
  )
}
