import { useMemo, useState } from 'react'
import type { GeneralPower } from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { VirtualList } from '@/shared/ui/virtual-list'
import { GeneralPowerRow } from './class-power-row'

/**
 * The general-power browse pool: a search box + an "owned only" toggle over a
 * virtualized list. High-level characters can have hundreds of eligible powers,
 * so it's collapsed by its host card and virtualized here.
 */
export function GeneralPowersPool({
  powers,
  isOwned,
  isLocked,
  onToggle,
  disabled,
}: {
  powers: GeneralPower[]
  isOwned: (id: string) => boolean
  isLocked: (power: GeneralPower) => boolean
  onToggle: (id: string) => void
  disabled: boolean
}) {
  const [query, setQuery] = useState('')
  const [ownedOnly, setOwnedOnly] = useState(false)

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    return powers.filter((p) => {
      if (ownedOnly && !isOwned(p.id)) return false
      if (q && !p.name.toLowerCase().includes(q)) return false
      return true
    })
  }, [powers, query, ownedOnly, isOwned])

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Buscar poder…"
          aria-label="Buscar poder geral"
          className="h-8 text-xs"
        />
        <Button
          type="button"
          variant={ownedOnly ? 'default' : 'outline'}
          size="sm"
          className="h-8 shrink-0 text-xs"
          aria-pressed={ownedOnly}
          onClick={() => setOwnedOnly((v) => !v)}
        >
          Meus
        </Button>
      </div>
      {filtered.length === 0 ? (
        <p className="text-xs italic text-muted-foreground">Nenhum poder.</p>
      ) : (
        <VirtualList
          items={filtered}
          estimateSize={64}
          gap={6}
          getKey={(p) => p.id}
          className="max-h-80"
          renderItem={(p) => (
            <GeneralPowerRow
              power={p}
              owned={isOwned(p.id)}
              locked={isLocked(p)}
              disabled={disabled}
              onToggle={() => onToggle(p.id)}
            />
          )}
        />
      )}
    </div>
  )
}
