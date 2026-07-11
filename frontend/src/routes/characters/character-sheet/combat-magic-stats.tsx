import { Crosshair, Shield, Sparkles, Sword, Zap } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import type { Character } from '@/shared/api/api'
import {
  defenseTotal,
  expertiseTotalWithItems,
  pmCostMod,
  pmLimitTotal,
  spellDCBonus,
  useCharacterEffects,
} from '@/shared/lib/derived'
import { ATTRIBUTE_ABBR, expertiseStateFor } from '@/shared/lib/expertise'
import { dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { signed } from './signed'

type StatRow = { label: string; amount: number; muted?: boolean }

export function CombatStats({ character }: { character: Character }) {
  const effects = useCharacterEffects(character)
  const def = defenseTotal(character, effects)
  const lutaState = expertiseStateFor(character, {
    name: 'Luta',
    attribute: 'strength',
    abbr: 'FOR',
  })
  const pontariaState = expertiseStateFor(character, {
    name: 'Pontaria',
    attribute: 'dexterity',
    abbr: 'DES',
  })
  const luta = expertiseTotalWithItems(character, lutaState, effects)
  const pontaria = expertiseTotalWithItems(character, pontariaState, effects)

  const defenseRows: StatRow[] = [{ label: 'Base', amount: 10 }]
  if (def.dexApplied) {
    defenseRows.push({ label: 'Destreza', amount: character.dexterity })
  } else {
    defenseRows.push({
      label: 'Destreza (bloqueada por armadura pesada)',
      amount: 0,
      muted: true,
    })
  }
  for (const c of def.contributions) {
    defenseRows.push({ label: c.source, amount: c.amount })
  }

  const attackRows = (
    e: ReturnType<typeof expertiseTotalWithItems>,
    attrAbbr: string,
  ): StatRow[] => {
    const rows: StatRow[] = [
      { label: '½ nível', amount: e.halfLevel },
      { label: attrAbbr, amount: e.attrValue },
    ]
    if (e.training) rows.push({ label: 'Treino', amount: e.training })
    for (const c of e.itemContributions) {
      rows.push({ label: c.source, amount: c.amount })
    }
    return rows
  }

  return (
    <div className="grid grid-cols-3 gap-2">
      <CombatBox
        label="Defesa"
        value={def.total}
        rows={defenseRows}
        icon={<Shield className="size-3.5" />}
      />
      <CombatBox
        label="Atq CaC"
        value={luta.total}
        rows={attackRows(luta, ATTRIBUTE_ABBR[lutaState.attribute])}
        icon={<Sword className="size-3.5" />}
        signed
      />
      <CombatBox
        label="Atq Dist"
        value={pontaria.total}
        rows={attackRows(pontaria, ATTRIBUTE_ABBR[pontariaState.attribute])}
        icon={<Crosshair className="size-3.5" />}
        signed
      />
    </div>
  )
}

function CombatBox({
  label,
  value,
  rows,
  icon,
  signed: showSigned,
}: {
  label: string
  value: number
  rows: StatRow[]
  icon: React.ReactNode
  signed?: boolean
}) {
  const display = showSigned ? signed(value) : String(value)
  return (
    <Dialog>
      <DialogTrigger asChild>
        <button
          type="button"
          className={cn(
            'relative flex cursor-pointer flex-col items-center rounded-lg border-2 p-2 text-center shadow-inner outline-none transition-colors',
            'border-red-800/50 bg-gradient-to-b from-red-100 to-red-50 text-red-900',
            'hover:from-red-200 hover:to-red-100',
            'dark:border-red-500/40 dark:from-red-950/40 dark:to-zinc-950 dark:text-red-200 dark:hover:from-red-900/40',
            'focus-visible:ring-2 focus-visible:ring-red-500/60',
          )}
          aria-label={`${label} ${display}`}
        >
          <span className="absolute left-1.5 top-1.5 text-red-700 dark:text-red-300">
            {icon}
          </span>
          <span className="text-[9px] font-bold uppercase tracking-widest text-red-800/80 dark:text-red-300/80">
            {label}
          </span>
          <span className="mt-0.5 font-serif text-2xl font-bold leading-none text-red-800 dark:text-red-100">
            {display}
          </span>
        </button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-red-700/40 bg-amber-50 text-zinc-900 dark:border-red-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle
            className={cn(
              'flex items-center gap-2 font-serif text-red-800 dark:text-red-200',
            )}
          >
            {icon}
            {label}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          <ul className="space-y-1">
            {rows.map((r, i) => (
              <li
                key={i}
                className={cn(
                  'flex items-center justify-between gap-2 border-b border-amber-700/15 pb-1 dark:border-amber-500/10',
                  r.muted && dimText,
                )}
              >
                <span className="truncate">{r.label}</span>
                <span className="shrink-0 font-mono">{signed(r.amount)}</span>
              </li>
            ))}
          </ul>
          <div
            className={cn(
              'flex items-center justify-between rounded-lg border px-3 py-2',
              'border-red-700/40 bg-red-100/60 dark:border-red-500/40 dark:bg-red-950/30',
            )}
          >
            <span className="text-xs uppercase tracking-widest text-red-800/80 dark:text-red-300/80">
              Total
            </span>
            <span className="font-mono text-2xl font-bold text-red-800 dark:text-red-100">
              {display}
            </span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

export function MagicStats({ character }: { character: Character }) {
  const effects = useCharacterEffects(character)
  const pmLimit = pmLimitTotal(character, effects)
  const dc = spellDCBonus(effects)
  const cost = pmCostMod(effects)

  const limitRows: StatRow[] = [
    { label: '½ nível (mín 1)', amount: pmLimit.base },
    ...pmLimit.contributions.map((c) => ({
      label: c.source,
      amount: c.amount,
    })),
  ]
  const dcRows: StatRow[] = dc.contributions.map((c) => ({
    label: c.source,
    amount: c.amount,
  }))
  const costRows: StatRow[] = cost.contributions.map((c) => ({
    label: c.source,
    amount: c.amount,
  }))

  const showDC = dc.total !== 0
  const showCost = cost.total !== 0

  return (
    <div className="grid grid-cols-3 gap-2">
      <MagicBox
        label="Limite PM"
        value={pmLimit.total}
        rows={limitRows}
        icon={<Zap className="size-3.5" />}
      />
      <MagicBox
        label="CD Magia"
        value={dc.total}
        rows={
          showDC
            ? dcRows
            : [{ label: 'Sem bônus de itens', amount: 0, muted: true }]
        }
        icon={<Sparkles className="size-3.5" />}
        signed
        prefix="+"
      />
      <MagicBox
        label="Custo PM"
        value={cost.total}
        rows={
          showCost
            ? costRows
            : [{ label: 'Sem mod de itens', amount: 0, muted: true }]
        }
        icon={<Sparkles className="size-3.5" />}
        signed
      />
    </div>
  )
}

function MagicBox({
  label,
  value,
  rows,
  icon,
  signed: showSigned,
  prefix,
}: {
  label: string
  value: number
  rows: StatRow[]
  icon: React.ReactNode
  signed?: boolean
  prefix?: string
}) {
  const display = showSigned ? signed(value) : `${prefix ?? ''}${value}`
  return (
    <Dialog>
      <DialogTrigger asChild>
        <button
          type="button"
          className={cn(
            'relative flex cursor-pointer flex-col items-center rounded-lg border-2 p-2 text-center shadow-inner outline-none transition-colors',
            'border-violet-800/50 bg-gradient-to-b from-violet-100 to-violet-50 text-violet-900',
            'hover:from-violet-200 hover:to-violet-100',
            'dark:border-violet-500/40 dark:from-violet-950/40 dark:to-zinc-950 dark:text-violet-200 dark:hover:from-violet-900/40',
            'focus-visible:ring-2 focus-visible:ring-violet-500/60',
          )}
          aria-label={`${label} ${display}`}
        >
          <span className="absolute left-1.5 top-1.5 text-violet-700 dark:text-violet-300">
            {icon}
          </span>
          <span className="text-[9px] font-bold uppercase tracking-widest text-violet-800/80 dark:text-violet-300/80">
            {label}
          </span>
          <span className="mt-0.5 font-serif text-2xl font-bold leading-none text-violet-800 dark:text-violet-100">
            {display}
          </span>
        </button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-violet-700/40 bg-amber-50 text-zinc-900 dark:border-violet-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle
            className={cn(
              'flex items-center gap-2 font-serif text-violet-800 dark:text-violet-200',
            )}
          >
            {icon}
            {label}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          <ul className="space-y-1">
            {rows.map((r, i) => (
              <li
                key={i}
                className={cn(
                  'flex items-center justify-between gap-2 border-b border-amber-700/15 pb-1 dark:border-amber-500/10',
                  r.muted && dimText,
                )}
              >
                <span className="truncate">{r.label}</span>
                <span className="shrink-0 font-mono">{signed(r.amount)}</span>
              </li>
            ))}
          </ul>
          <div
            className={cn(
              'flex items-center justify-between rounded-lg border px-3 py-2',
              'border-violet-700/40 bg-violet-100/60 dark:border-violet-500/40 dark:bg-violet-950/30',
            )}
          >
            <span className="text-xs uppercase tracking-widest text-violet-800/80 dark:text-violet-300/80">
              Total
            </span>
            <span className="font-mono text-2xl font-bold text-violet-800 dark:text-violet-100">
              {display}
            </span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
