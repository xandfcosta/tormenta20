import { useEffect, useMemo, useState } from 'react'
import type { Character } from '@/shared/api/api'
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/shared/ui/command'
import {
  buildSheetSearchIndex,
  type SheetSearchEntry,
} from './sheet-search-index'

/**
 * Sheet-global search palette (audit P1: "3-second lookup"). `/` or Ctrl+K
 * opens it; every row shows the ANSWER inline (perícia total, item quantity,
 * rule text) so most lookups end without navigation — selecting a row still
 * jumps to the owning tab for the full context.
 */
export function SheetSearch({
  character,
  onNavigate,
}: {
  character: Character
  onNavigate: (tab: string) => void
}) {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement
      const typing = target.tagName === 'INPUT' || target.tagName === 'TEXTAREA'
      const isPalette =
        e.key === '/' || (e.key.toLowerCase() === 'k' && (e.ctrlKey || e.metaKey))
      if (!isPalette || (typing && e.key === '/')) return
      e.preventDefault()
      setOpen((v) => !v)
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  const index = useMemo(
    () => (open ? buildSheetSearchIndex(character) : []),
    [open, character],
  )
  const groups = useMemo(() => groupBySource(index), [index])

  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      title="Buscar na ficha"
      description="Perícias, poderes, itens, magias e condições"
    >
      <CommandInput placeholder="Buscar na ficha… (perícia, poder, item, magia)" />
      <CommandList>
        <CommandEmpty>Nada encontrado.</CommandEmpty>
        {groups.map(([source, entries]) => (
          <CommandGroup key={source} heading={source}>
            {entries.map((entry) => (
              <CommandItem
                key={`${source}-${entry.name}`}
                value={`${entry.name} ${entry.source}`}
                onSelect={() => {
                  onNavigate(entry.tab)
                  setOpen(false)
                }}
              >
                <span className="min-w-0 flex-1 truncate font-medium">
                  {entry.name}
                </span>
                <span className="ml-2 max-w-[55%] truncate text-xs text-muted-foreground">
                  {entry.detail}
                </span>
              </CommandItem>
            ))}
          </CommandGroup>
        ))}
      </CommandList>
    </CommandDialog>
  )
}

function groupBySource(
  index: SheetSearchEntry[],
): [string, SheetSearchEntry[]][] {
  const order = ['Perícia', 'Item', 'Magia', 'Condição']
  const map = new Map<string, SheetSearchEntry[]>()
  for (const entry of index) {
    const key = order.includes(entry.source)
      ? entry.source
      : 'Poderes & habilidades'
    const list = map.get(key) ?? []
    list.push(entry)
    map.set(key, list)
  }
  const rank = (k: string) =>
    ['Perícia', 'Poderes & habilidades', 'Item', 'Magia', 'Condição'].indexOf(k)
  return [...map.entries()].sort((a, b) => rank(a[0]) - rank(b[0]))
}
