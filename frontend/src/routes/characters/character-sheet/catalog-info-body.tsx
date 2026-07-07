import type { CatalogItem } from '@tormenta20/t20-data'
import { accentStrong, dimText } from '@/lib/sheet-theme'
import { cn } from '@/lib/utils'
import {
  describeCondition,
  describeModifierTarget,
  formatLoad,
} from './item-describe'

/**
 * Rendered inside two dialogs — the "Adicionar do catálogo" preview
 * and the per-item `ItemInfoDialog`. Extracted so both consumers stay
 * in sync when a new field is added to `CatalogItem`.
 */
export function CatalogInfoBody({ catalog }: { catalog: CatalogItem }) {
  return (
    <div className="space-y-3 text-xs">
      <div>
        <p className={cn('font-semibold', accentStrong)}>{catalog.name}</p>
        <p className={dimText}>
          {catalog.category} • esp {formatLoad(catalog.slots)} • T${' '}
          {catalog.price} •{' '}
          {catalog.equip === 'either' ? 'qualquer equipar' : catalog.equip}
          {catalog.hands ? ` • ${catalog.hands} mão(s)` : ''}
        </p>
      </div>
      {catalog.weapon && (
        <div className="space-y-0.5">
          <p className={cn('text-[10px] uppercase tracking-widest', dimText)}>
            arma
          </p>
          <p>
            dano <span className="font-mono">{catalog.weapon.damage}</span> •
            crítico{' '}
            <span className="font-mono">
              {catalog.weapon.critRange}/×{catalog.weapon.critMult}
            </span>{' '}
            • {catalog.weapon.type} • {catalog.weapon.purpose}
            {catalog.weapon.range ? ` (${catalog.weapon.range})` : ''}
          </p>
          {catalog.weapon.traits.length > 0 && (
            <p className={dimText}>
              propriedades: {catalog.weapon.traits.join(', ')}
            </p>
          )}
        </div>
      )}
      {catalog.armor && (
        <div className="space-y-0.5">
          <p className={cn('text-[10px] uppercase tracking-widest', dimText)}>
            armadura
          </p>
          <p>
            Defesa +{catalog.armor.defense} • penalidade{' '}
            {catalog.armor.penalty} •{' '}
            {catalog.armor.heavy ? 'pesada' : 'leve'}
          </p>
        </div>
      )}
      {catalog.shield && (
        <div className="space-y-0.5">
          <p className={cn('text-[10px] uppercase tracking-widest', dimText)}>
            escudo
          </p>
          <p>
            Defesa +{catalog.shield.defense} • penalidade{' '}
            {catalog.shield.penalty}
          </p>
        </div>
      )}
      <div className="space-y-1">
        <p className={cn('text-[10px] uppercase tracking-widest', dimText)}>
          modificadores
        </p>
        {catalog.modifiers.length === 0 ? (
          <p className={dimText}>Nenhum.</p>
        ) : (
          <ul className="space-y-0.5">
            {catalog.modifiers.map((m, i) => {
              const cond = describeCondition(m)
              const sign = m.amount >= 0 ? '+' : ''
              return (
                <li key={i} className="flex flex-wrap gap-x-1">
                  <span className="font-mono">
                    {sign}
                    {m.amount}
                  </span>
                  <span>{describeModifierTarget(m.target)}</span>
                  <span className={cn('text-[10px]', dimText)}>
                    [{m.bonusType}]
                  </span>
                  {cond && (
                    <span className={cn('text-[10px]', dimText)}>— {cond}</span>
                  )}
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}
