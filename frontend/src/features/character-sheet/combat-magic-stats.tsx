import { Crosshair, Shield, ShieldCheck, Sparkles, Sword, Zap } from 'lucide-react'
import { SPELLCASTER_CLASSES } from '@tormenta20/t20-data'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import type { Character } from '@/shared/api/api'
import {
  requireExpertise,
  useComputedSheet,
} from '@/entities/character/computed-sheet'
import { useWeaponCards } from '@/entities/character/weapon-cards'
import type {
  ExpertiseBreakdown,
  WeaponCard,
} from '@/shared/lib/computed-sheet-v2'
import { ATTRIBUTE_ABBR } from '@/entities/character/expertise'
import { dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { signed } from './signed'

type StatRow = { label: string; amount: number; muted?: boolean; note?: string }

/**
 * One breakdown line: source + amount, with the modifier's note (the WHY —
 * "desbalanceada: -2 em ataque") as a dim sub-line so rows explain
 * themselves instead of showing a bare item name.
 */
function StatRowLine({ row }: { row: StatRow }) {
  return (
    <li
      className={cn(
        'border-b border-border pb-1',
        row.muted && dimText,
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="truncate">{row.label}</span>
        <span className="shrink-0 font-mono">{signed(row.amount)}</span>
      </div>
      {/* wrap, never truncate: a nowrap note becomes the grid's min-content
          and inflates the whole dialog past its max-width */}
      {row.note && (
        <p className={cn('text-[11px] leading-snug', dimText)}>{row.note}</p>
      )}
    </li>
  )
}

export function CombatStats({ character }: { character: Character }) {
  const sheet = useComputedSheet(character)
  const def = sheet.defense
  const luta = requireExpertise(sheet, 'Luta', 'strength')
  const pontaria = requireExpertise(sheet, 'Pontaria', 'dexterity')
  const rd = sheet.damageReduction

  const defenseRows: StatRow[] = [{ label: 'Base', amount: 10 }]
  if (def.dexApplied) {
    defenseRows.push({
      label: 'Destreza',
      amount: sheet.attributes.dexterity.total,
    })
  } else {
    defenseRows.push({
      label: 'Destreza (bloqueada por armadura pesada)',
      amount: 0,
      muted: true,
    })
  }
  for (const c of def.contributions) {
    defenseRows.push({ label: c.source, amount: c.amount, note: c.note })
  }

  // Global attack modifiers ({k:'attack', scope:'all'}) — buffs/conditionals like
  // Fúria that apply to every attack, regardless of the weapon. Weapon-specific
  // (scope:'this') mods are deliberately excluded here: the non-proficiency
  // penalty is already surfaced through the expertise path, so folding
  // scope:'this' on top would double-count it.
  const attackAll = sheet.attackAll

  const attackRows = (
    e: ExpertiseBreakdown,
    attrAbbr: string,
  ): StatRow[] => {
    const rows: StatRow[] = [
      { label: '½ nível', amount: e.halfLevel },
      { label: attrAbbr, amount: e.attrValue },
    ]
    if (e.training) rows.push({ label: 'Treino', amount: e.training })
    for (const c of e.itemContributions) {
      rows.push({ label: c.source, amount: c.amount, note: c.note })
    }
    for (const c of attackAll.contributions) {
      rows.push({ label: c.source, amount: c.amount, note: c.note })
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
        sub={rd.total > 0 ? `RD ${rd.total}` : undefined}
        extra={
          rd.total > 0
            ? {
                title: `Redução de dano ${rd.total}`,
                rows: rd.sources.map((s) => ({
                  label: s.source,
                  amount: s.amount,
                })),
              }
            : undefined
        }
      />
      <CombatBox
        label="Atq CaC"
        dialogTitle="Ataque Corpo a Corpo (Luta)"
        value={luta.total + attackAll.total}
        rows={attackRows(luta, ATTRIBUTE_ABBR[luta.attribute])}
        icon={<Sword className="size-3.5" />}
        signed
      />
      <CombatBox
        label="Atq Dist"
        dialogTitle="Ataque à Distância (Pontaria)"
        value={pontaria.total + attackAll.total}
        rows={attackRows(pontaria, ATTRIBUTE_ABBR[pontaria.attribute])}
        icon={<Crosshair className="size-3.5" />}
        signed
      />
    </div>
  )
}

function CombatBox({
  label,
  dialogTitle,
  value,
  rows,
  icon,
  signed: showSigned,
  sub,
  extra,
}: {
  label: string
  /** Full name for the breakdown dialog when the box label is abbreviated. */
  dialogTitle?: string
  value: number
  rows: StatRow[]
  icon: React.ReactNode
  signed?: boolean
  /** Small companion line under the value (e.g. "RD 4"). */
  sub?: string
  /** Titled section after the total — values that relate to the stat but
   *  don't sum into it (RD sources under Defesa). */
  extra?: { title: string; rows: StatRow[] }
}) {
  const display = showSigned ? signed(value) : String(value)
  return (
    <Dialog>
      <DialogTrigger asChild>
        <button
          type="button"
          className={cn(
            'relative flex cursor-pointer flex-col items-center rounded-lg border-2 p-2 text-center shadow-inner outline-none transition-colors',
            'border-red-800/50  from-red-100 to-red-50 text-red-900',
            'hover:from-red-200 hover:to-red-100',
            'dark:border-red-500/40 dark:from-red-950/40  dark:text-red-200 dark:hover:from-red-900/40',
            'focus-visible:ring-2 focus-visible:ring-red-500/60 hover:ring-1 hover:ring-red-500/50',
          )}
          aria-label={`${label} ${display}`}
        >
          <span className="flex items-center gap-1 text-[9px] font-bold uppercase tracking-widest text-red-800/80 dark:text-red-300/80">
            <span className="text-red-700 dark:text-red-300">{icon}</span>
            {label}
          </span>
          <span className="mt-0.5 text-2xl font-bold leading-none text-red-800 dark:text-red-100">
            {display}
          </span>
          {sub && (
            <span className="text-[10px] font-semibold uppercase tracking-widest text-red-800/70 dark:text-red-300/70">
              {sub}
            </span>
          )}
        </button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-red-700/40 bg-muted text-foreground dark:border-red-500/40  ',
        )}
      >
        <DialogHeader>
          <DialogTitle
            className={cn(
              'flex items-center gap-2 text-red-800 dark:text-red-200',
            )}
          >
            {icon}
            {dialogTitle ?? label}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          <ul className="space-y-1">
            {rows.map((r, i) => (
              <StatRowLine key={i} row={r} />
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
          {extra && (
            <div className="space-y-1">
              <p className="text-xs font-bold uppercase tracking-widest text-red-800/80 dark:text-red-300/80">
                {extra.title}
              </p>
              <ul className="space-y-1">
                {extra.rows.map((r, i) => (
                  <StatRowLine key={i} row={r} />
                ))}
              </ul>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

/** True when any of the character's classes casts spells (contextual HUD). */
export function isCasterCharacter(character: Character): boolean {
  return character.classes.some((c) =>
    (SPELLCASTER_CLASSES as readonly string[]).includes(c.className),
  )
}

/**
 * Fort/Ref/Von triple — the most reactive numbers in the game ("teste de
 * Reflexos CD 20!") promoted to the always-visible HUD. Same engine path as
 * the attack boxes, same breakdown dialogs.
 */
export function SavesStats({ character }: { character: Character }) {
  const sheet = useComputedSheet(character)
  const saves = [
    { name: 'Fortitude', attribute: 'constitution', abbr: 'CON' },
    { name: 'Reflexos', attribute: 'dexterity', abbr: 'DES' },
    { name: 'Vontade', attribute: 'wisdom', abbr: 'SAB' },
  ] as const
  return (
    <div className="grid grid-cols-3 gap-2">
      {saves.map((meta) => {
        const total = requireExpertise(sheet, meta.name, meta.attribute)
        const rows: StatRow[] = [
          { label: '½ nível', amount: total.halfLevel },
          { label: meta.abbr, amount: total.attrValue },
        ]
        if (total.training) rows.push({ label: 'Treino', amount: total.training })
        for (const c of total.itemContributions) {
          rows.push({ label: c.source, amount: c.amount })
        }
        return (
          <CombatBox
            key={meta.name}
            label={meta.name.slice(0, 4)}
            dialogTitle={meta.name}
            value={total.total}
            rows={rows}
            icon={<ShieldCheck className="size-3.5" />}
            signed
          />
        )
      })}
    </div>
  )
}

/**
 * Attack + damage breakdown rows for a weapon card — the Go engine owns the
 * numbers (`WeaponCard`), this applies the structural labels (½ nível, FOR/DES,
 * Treino) and folds in the global attack/damage contributions. The FOR damage
 * row shows for melee/thrown (Luta), never ranged (`strDamage` is 0 there).
 */
function weaponCardRows(card: WeaponCard): {
  attackRows: StatRow[]
  damageRows: StatRow[]
} {
  const attrAbbr = card.attribute === 'strength' ? 'FOR' : 'DES'
  const attackRows: StatRow[] = [
    { label: '½ nível', amount: card.expertise.halfLevel },
    { label: attrAbbr, amount: card.expertise.attrValue },
    ...(card.expertise.training
      ? [{ label: 'Treino', amount: card.expertise.training }]
      : []),
    ...card.expertise.itemContributions.map((c) => ({
      label: c.source,
      amount: c.amount,
      note: c.note,
    })),
    ...card.attackAll.contributions.map((c) => ({
      label: c.source,
      amount: c.amount,
      note: c.note,
    })),
  ]
  const damageRows: StatRow[] = [
    ...(card.skill === 'Luta' ? [{ label: 'FOR', amount: card.strDamage }] : []),
    ...card.damageAll.contributions.map((c) => ({
      label: c.source,
      amount: c.amount,
      note: c.note,
    })),
  ]
  return { attackRows, damageRows }
}

/**
 * Equipped-weapon formula cards: "Machado · +10 · 1d12+7 · 19/x3" so a hit
 * never costs a tab switch to roll damage. Attack = Luta/Pontaria + global
 * attack mods (same math as the Atq boxes); damage adds FOR for melee/thrown
 * (engine convention). The numbers come from the Go engine (`useWeaponCards`).
 */
export function WeaponFormulaCards({ character }: { character: Character }) {
  const cards = useWeaponCards(character)
  if (cards.length === 0) {
    return (
      <p className="self-center text-center text-xs italic text-muted-foreground">
        Nenhuma arma empunhada.
      </p>
    )
  }
  return (
    <div className="grid gap-2" style={{ gridTemplateColumns: `repeat(${cards.length}, 1fr)` }}>
      {cards.map((card) => {
        const { name, skill, attack, damage } = card
        const dmgBonus = card.damageBonus
        const crit = `${card.critRange < 20 ? `${card.critRange}-20` : '20'}/x${card.critMult}`
        const rows = weaponCardRows(card)
        const { attackRows, damageRows } = rows

        return (
          <Dialog key={name}>
            <DialogTrigger asChild>
              <button
                type="button"
                className="flex cursor-pointer flex-col items-center rounded-lg border-2 border-red-800/50 p-2 text-center transition-colors hover:bg-red-950/20 dark:border-red-500/40"
                aria-label={`Detalhamento de ${name}`}
                title={`${skill} ${signed(attack)} · dano ${damage}${dmgBonus ? signed(dmgBonus) : ''} · crítico ${crit}`}
              >
                <span className="max-w-full truncate text-[9px] font-bold uppercase tracking-widest text-red-800/80 dark:text-red-300/80">
                  {name}
                </span>
                <span className="mt-0.5 font-mono text-sm font-bold leading-tight text-red-800 dark:text-red-100">
                  {signed(attack)} · {damage}
                  {dmgBonus !== 0 ? signed(dmgBonus) : ''}
                </span>
                <span className="text-[10px] text-muted-foreground">{crit}</span>
              </button>
            </DialogTrigger>
            <DialogContent
              className={cn(
                'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
                'border-red-700/40 bg-muted text-foreground dark:border-red-500/40',
              )}
            >
              <DialogHeader>
                <DialogTitle className="flex items-center gap-2 text-red-800 dark:text-red-200">
                  <Sword className="size-3.5" />
                  {name}
                </DialogTitle>
              </DialogHeader>
              <div className="space-y-3 text-sm">
                <div>
                  <p className="text-xs font-bold uppercase tracking-widest text-red-800/80 dark:text-red-300/80">
                    Ataque ({skill}) {signed(attack)}
                  </p>
                  <ul className="mt-1 space-y-1">
                    {attackRows.map((r, i) => (
                      <StatRowLine key={i} row={r} />
                    ))}
                  </ul>
                </div>
                <div>
                  <p className="text-xs font-bold uppercase tracking-widest text-red-800/80 dark:text-red-300/80">
                    Dano {damage}
                    {dmgBonus !== 0 ? signed(dmgBonus) : ''} · crítico {crit}
                  </p>
                  <ul className="mt-1 space-y-1">
                    {damageRows.length === 0 ? (
                      <li className={cn('text-xs', dimText)}>
                        Só o dado da arma.
                      </li>
                    ) : (
                      damageRows.map((r, i) => <StatRowLine key={i} row={r} />)
                    )}
                  </ul>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        )
      })}
    </div>
  )
}

export function MagicStats({ character }: { character: Character }) {
  const sheet = useComputedSheet(character)
  const pmLimit = sheet.pmLimit
  // CD base from the engine so the key attribute is the FINAL value (racial/item
  // bonuses included) — the raw attribute understated Osteon casters by 1.
  const baseCd = sheet.bestBaseSpellCd ?? 0
  const dc = sheet.spellDCBonus
  const cost = sheet.pmCostMod

  const limitRows: StatRow[] = [
    { label: 'Nível de conjurador', amount: pmLimit.base },
    ...pmLimit.contributions.map((c) => ({
      label: c.source,
      amount: c.amount,
      note: c.note,
    })),
  ]
  const dcRows: StatRow[] = dc.contributions.map((c) => ({
    label: c.source,
    amount: c.amount,
    note: c.note,
  }))
  const costRows: StatRow[] = cost.contributions.map((c) => ({
    label: c.source,
    amount: c.amount,
    note: c.note,
  }))

  const showDC = dc.total !== 0
  const showCost = cost.total !== 0

  return (
    <div className="grid grid-cols-3 gap-2">
      <MagicBox
        label="Limite PM"
        dialogTitle="Limite de PM por magia"
        value={pmLimit.total}
        rows={limitRows}
        icon={<Zap className="size-3.5" />}
      />
      <MagicBox
        label="CD Magia"
        dialogTitle="CD dos testes de resistência das suas magias"
        value={baseCd + dc.total}
        rows={[
          {
            label: 'CD base (nível + atributo-chave)',
            amount: baseCd,
          },
          ...(showDC
            ? dcRows
            : [{ label: 'Sem bônus de itens', amount: 0, muted: true }]),
        ]}
        icon={<Sparkles className="size-3.5" />}
      />
      <MagicBox
        label="Custo PM"
        dialogTitle="Modificador de custo de PM"
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
  dialogTitle,
  value,
  rows,
  icon,
  signed: showSigned,
  prefix,
}: {
  label: string
  /** Full name for the breakdown dialog when the box label is abbreviated. */
  dialogTitle?: string
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
            'border-violet-800/50  from-violet-100 to-violet-50 text-violet-900',
            'hover:from-violet-200 hover:to-violet-100',
            'dark:border-violet-500/40 dark:from-violet-950/40  dark:text-violet-200 dark:hover:from-violet-900/40',
            'focus-visible:ring-2 focus-visible:ring-violet-500/60',
          )}
          aria-label={`${label} ${display}`}
        >
          <span className="flex items-center gap-1 text-[9px] font-bold uppercase tracking-widest text-violet-800/80 dark:text-violet-300/80">
            <span className="text-violet-700 dark:text-violet-300">{icon}</span>
            {label}
          </span>
          <span className="mt-0.5 text-2xl font-bold leading-none text-violet-800 dark:text-violet-100">
            {display}
          </span>
        </button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-violet-700/40 bg-muted text-foreground dark:border-violet-500/40  ',
        )}
      >
        <DialogHeader>
          <DialogTitle
            className={cn(
              'flex items-center gap-2 text-violet-800 dark:text-violet-200',
            )}
          >
            {icon}
            {dialogTitle ?? label}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          <ul className="space-y-1">
            {rows.map((r, i) => (
              <StatRowLine key={i} row={r} />
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
