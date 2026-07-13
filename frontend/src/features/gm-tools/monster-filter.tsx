import { useMemo, useState } from 'react'
import {
  type ColumnFiltersState,
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { Badge } from '@/shared/ui/badge'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { fuzzyFilter } from '@/shared/lib/fuzzy-filter'
import { BESTIARY, type Monster, type MonsterTipo } from '@tormenta20/t20-data'
import {
  MONSTER_TIPOS as TIPOS,
  MONSTER_TIPO_LABEL as TIPO_LABEL,
} from './monster-format'

/**
 * Shared bestiary filter — fuzzy name + multi-select tipo + ND range, run
 * through a headless TanStack Table. `useMonsterFilter` owns the state and
 * returns the filtered/sorted list; `MonsterFilters` renders the controls.
 * Reused by the bestiary page and the encounter-builder monster picker so
 * both filter identically.
 */
const columnHelper = createColumnHelper<Monster>()
const columns = [
  columnHelper.accessor('name', { id: 'name', filterFn: fuzzyFilter<Monster>() }),
  columnHelper.accessor('tipo', {
    id: 'tipo',
    filterFn: (row, id, value: MonsterTipo[]) =>
      value.length === 0 || value.includes(row.getValue<MonsterTipo>(id)),
  }),
  columnHelper.accessor('nd', {
    id: 'nd',
    filterFn: (row, id, [min, max]: [number, number]) => {
      const nd = row.getValue<number>(id)
      return nd >= min && nd <= max
    },
  }),
]

export type MonsterFilterControls = {
  name: string
  setName: (v: string) => void
  ndMin: number
  setNdMin: (v: number) => void
  ndMax: number
  setNdMax: (v: number) => void
  tipos: Set<MonsterTipo>
  toggleTipo: (t: MonsterTipo) => void
}

export function useMonsterFilter(): {
  filtered: Monster[]
  controls: MonsterFilterControls
} {
  const [name, setName] = useState('')
  const [tipos, setTipos] = useState<Set<MonsterTipo>>(new Set())
  const [ndMin, setNdMin] = useState(0)
  const [ndMax, setNdMax] = useState(20)

  const columnFilters = useMemo<ColumnFiltersState>(
    () => [
      { id: 'name', value: name },
      { id: 'tipo', value: [...tipos] },
      { id: 'nd', value: [ndMin, ndMax] },
    ],
    [name, tipos, ndMin, ndMax],
  )

  const table = useReactTable({
    data: BESTIARY as Monster[],
    columns,
    state: { columnFilters },
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
  })

  const filtered = table
    .getRowModel()
    .rows.map((r) => r.original)
    .sort((a, b) => a.nd - b.nd || a.name.localeCompare(b.name))

  const toggleTipo = (t: MonsterTipo) =>
    setTipos((prev) => {
      const next = new Set(prev)
      if (next.has(t)) next.delete(t)
      else next.add(t)
      return next
    })

  return {
    filtered,
    controls: { name, setName, ndMin, setNdMin, ndMax, setNdMax, tipos, toggleTipo },
  }
}

export function MonsterFilters({
  name,
  setName,
  ndMin,
  setNdMin,
  ndMax,
  setNdMax,
  tipos,
  toggleTipo,
  idPrefix = 'mf',
}: MonsterFilterControls & { idPrefix?: string }) {
  return (
    <div className="space-y-3">
      <Input
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="Buscar monstro…"
        aria-label="Buscar monstro"
      />
      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="text-xs font-medium" htmlFor={`${idPrefix}-nd-min`}>
            ND mínimo
          </label>
          <NumberInput
            id={`${idPrefix}-nd-min`}
            min={0}
            max={20}
            step={0.25}
            value={ndMin}
            onChange={setNdMin}
          />
        </div>
        <div>
          <label className="text-xs font-medium" htmlFor={`${idPrefix}-nd-max`}>
            ND máximo
          </label>
          <NumberInput
            id={`${idPrefix}-nd-max`}
            min={0}
            max={20}
            step={0.25}
            value={ndMax}
            onChange={setNdMax}
          />
        </div>
      </div>
      <div className="flex flex-wrap gap-2">
        {TIPOS.map((t) => (
          <button key={t} type="button" onClick={() => toggleTipo(t)}>
            <Badge variant={tipos.has(t) ? 'default' : 'outline'}>
              {TIPO_LABEL[t]}
            </Badge>
          </button>
        ))}
      </div>
    </div>
  )
}
