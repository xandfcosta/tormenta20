import { Crosshair, Shield, ShieldCheck, Sparkles, Sword, Zap } from 'lucide-react'
import {
  ATTRIBUTE_KEYS,
  CLASS_SPELLCASTING_ATTRIBUTE,
  SPELLCASTER_CLASSES,
  spellSaveDc,
  statFor,
} from '@tormenta20/t20-data'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import type { Character } from '@/shared/api/api'
import { getCatalogItem } from '@tormenta20/t20-data'
import {
  attributeTotal,
  defenseTotal,
  expertiseTotalWithItems,
  pmCostMod,
  pmLimitTotal,
  spellDCBonus,
  useCharacterEffects,
} from '@/entities/character/derived'
import { ATTRIBUTE_ABBR, expertiseStateFor } from '@/entities/character/expertise'
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
    defenseRows.push({
      label: 'Destreza',
      amount: attributeTotal(character, 'dexterity', effects),
    })
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

  // Global attack modifiers ({k:'attack', scope:'all'}) — buffs/conditionals like
  // Fúria that apply to every attack, regardless of the weapon. Weapon-specific
  // (scope:'this') mods are deliberately excluded here: the non-proficiency
  // penalty is already surfaced through the expertise path, so folding
  // scope:'this' on top would double-count it.
  const attackAll = statFor(effects, { k: 'attack', scope: 'all' })

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
    for (const c of attackAll.contributions) {
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
        value={luta.total + attackAll.total}
        rows={attackRows(luta, ATTRIBUTE_ABBR[lutaState.attribute])}
        icon={<Sword className="size-3.5" />}
        signed
      />
      <CombatBox
        label="Atq Dist"
        value={pontaria.total + attackAll.total}
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
            'border-red-800/50  from-red-100 to-red-50 text-red-900',
            'hover:from-red-200 hover:to-red-100',
            'dark:border-red-500/40 dark:from-red-950/40  dark:text-red-200 dark:hover:from-red-900/40',
            'focus-visible:ring-2 focus-visible:ring-red-500/60',
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
            {label}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          <ul className="space-y-1">
            {rows.map((r, i) => (
              <li
                key={i}
                className={cn(
                  'flex items-center justify-between gap-2 border-b border-border pb-1 ',
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
  const effects = useCharacterEffects(character)
  const saves = [
    { name: 'Fortitude', attribute: 'constitution', abbr: 'CON' },
    { name: 'Reflexos', attribute: 'dexterity', abbr: 'DES' },
    { name: 'Vontade', attribute: 'wisdom', abbr: 'SAB' },
  ] as const
  return (
    <div className="grid grid-cols-3 gap-2">
      {saves.map((meta) => {
        const state = expertiseStateFor(character, meta)
        const total = expertiseTotalWithItems(character, state, effects)
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
 * Equipped-weapon formula cards: "Machado · +10 · 1d12+7 · 19/x3" so a hit
 * never costs a tab switch to roll damage. Attack = Luta/Pontaria + global
 * attack mods (same math as the Atq boxes); damage adds FOR for melee/thrown
 * (engine convention).
 */
export function WeaponFormulaCards({ character }: { character: Character }) {
  const effects = useCharacterEffects(character)
  const attackAll = statFor(effects, { k: 'attack', scope: 'all' })
  const forTotal = attributeTotal(character, 'strength', effects)
  const weapons = character.items
    .filter((i) => i.equipped === 'wielded' || i.equipped === 'wielded2')
    .flatMap((i) => {
      const catalog = i.catalogId ? getCatalogItem(i.catalogId) : undefined
      return catalog?.weapon ? [{ name: i.name, weapon: catalog.weapon }] : []
    })
    .slice(0, 2)
  if (weapons.length === 0) {
    return (
      <p className="self-center text-center text-xs italic text-muted-foreground">
        Nenhuma arma empunhada.
      </p>
    )
  }
  return (
    <div className="grid gap-2" style={{ gridTemplateColumns: `repeat(${weapons.length}, 1fr)` }}>
      {weapons.map(({ name, weapon }) => {
        const skill = weapon.purpose === 'ranged' ? 'Pontaria' : 'Luta'
        const state = expertiseStateFor(character, {
          name: skill,
          attribute: skill === 'Luta' ? 'strength' : 'dexterity',
          abbr: skill === 'Luta' ? 'FOR' : 'DES',
        })
        const attack =
          expertiseTotalWithItems(character, state, effects).total +
          attackAll.total
        const dmgBonus = weapon.purpose === 'ranged' ? 0 : forTotal
        const crit = `${weapon.critRange < 20 ? `${weapon.critRange}-20` : '20'}/x${weapon.critMult}`
        return (
          <div
            key={name}
            className="flex flex-col items-center rounded-lg border-2 border-red-800/50 p-2 text-center dark:border-red-500/40"
            title={`${skill} ${signed(attack)} · dano ${weapon.damage}${dmgBonus ? signed(dmgBonus) : ''} · crítico ${crit}`}
          >
            <span className="max-w-full truncate text-[9px] font-bold uppercase tracking-widest text-red-800/80 dark:text-red-300/80">
              {name}
            </span>
            <span className="mt-0.5 font-mono text-sm font-bold leading-tight text-red-800 dark:text-red-100">
              {signed(attack)} · {weapon.damage}
              {dmgBonus !== 0 ? signed(dmgBonus) : ''}
            </span>
            <span className="text-[10px] text-muted-foreground">{crit}</span>
          </div>
        )
      })}
    </div>
  )
}

/** Best spell save CD across the character's caster classes (absolute, not
 *  the item bonus): CD = spellSaveDc(nível, atributo-chave) + bônus de itens. */
function bestBaseSpellCd(character: Character): number | null {
  let best = -Infinity
  for (const entry of character.classes) {
    const attr =
      CLASS_SPELLCASTING_ATTRIBUTE[
        entry.className as keyof typeof CLASS_SPELLCASTING_ATTRIBUTE
      ]
    if (!attr) continue
    const key = ATTRIBUTE_KEYS.find((k) => k === attr)
    if (!key) continue
    const dc = spellSaveDc(character.level, character[key])
    if (dc > best) best = dc
  }
  return best === -Infinity ? null : best
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
        value={(bestBaseSpellCd(character) ?? 0) + dc.total}
        rows={[
          {
            label: 'CD base (nível + atributo-chave)',
            amount: bestBaseSpellCd(character) ?? 0,
          },
          ...(showDC
            ? dcRows
            : [{ label: 'Sem bônus de itens', amount: 0, muted: true }]),
        ]}
        icon={<Sparkles className="size-3.5" />}
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
            {label}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          <ul className="space-y-1">
            {rows.map((r, i) => (
              <li
                key={i}
                className={cn(
                  'flex items-center justify-between gap-2 border-b border-border pb-1 ',
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
