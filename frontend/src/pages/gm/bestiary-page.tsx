import { useMemo, useState } from 'react'
import {
  type ColumnFiltersState,
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent } from '@/shared/ui/card'
import { VirtualList } from '@/shared/ui/virtual-list'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { PageChrome } from '@/shared/ui/page-chrome'
import { cn } from '@/shared/lib/utils'
import { useMediaQuery } from '@/shared/lib/use-media-query'
import {
  BESTIARY,
  type Monster,
  type MonsterTipo,
  xpForNd,
} from '@tormenta20/t20-data'
import {
  MONSTER_TIPOS as TIPOS,
  MONSTER_TIPO_LABEL as TIPO_LABEL,
  formatNd,
  normalizeMonsterName as normalize,
} from '@/features/gm-tools/monster-format'
import { GmPageHeader } from '@/features/gm-tools/gm-page-header'

// Headless table columns drive filtering: an accent-insensitive name filter,
// a multi-select tipo filter, and an ND range filter. Rows are read back out
// and rendered as a virtualized list — the table itself never renders.
const columnHelper = createColumnHelper<Monster>()
const columns = [
  columnHelper.accessor('name', {
    id: 'name',
    filterFn: (row, id, value: string) => {
      const q = normalize(value)
      return !q || normalize(row.getValue<string>(id)).includes(q)
    },
  }),
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

/**
 * Bestiary lookup — a master/detail: filters + monster list on the left,
 * the selected monster's full stat block on the right. Desktop shows the
 * detail in a side panel; phone (no room for it) opens a dialog on tap.
 */
export function BestiaryPage() {
  const [name, setName] = useState('')
  const [tipos, setTipos] = useState<Set<MonsterTipo>>(new Set())
  const [ndMin, setNdMin] = useState(0)
  const [ndMax, setNdMax] = useState(20)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [dialogId, setDialogId] = useState<string | null>(null)
  const isDesktop = useMediaQuery('(min-width: 1024px)')

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

  const selected = filtered.find((m) => m.id === selectedId) ?? filtered[0]
  const dialogMonster = filtered.find((m) => m.id === dialogId) ?? null

  const openMonster = (m: Monster) => {
    setSelectedId(m.id)
    if (!isDesktop) setDialogId(m.id)
  }

  const toggleTipo = (t: MonsterTipo) => {
    setTipos((prev) => {
      const next = new Set(prev)
      if (next.has(t)) next.delete(t)
      else next.add(t)
      return next
    })
  }

  return (
    <PageChrome width="wide" className="flex min-h-0 flex-1 flex-col gap-4">
      <GmPageHeader
        title="Bestiário"
        aside={
          <span className="text-sm text-muted-foreground">
            {filtered.length} / {BESTIARY.length}
          </span>
        }
      />

      <div className="grid h-[calc(100dvh-13rem)] min-h-0 flex-1 gap-4 lg:h-[calc(100dvh-8rem)] lg:grid-cols-[22rem_1fr]">
        <div className="flex min-h-0 min-w-0 flex-col gap-3">
          <BestiaryFilters
            name={name}
            onName={setName}
            ndMin={ndMin}
            onNdMin={setNdMin}
            ndMax={ndMax}
            onNdMax={setNdMax}
            tipos={tipos}
            onToggleTipo={toggleTipo}
          />
          {filtered.length === 0 ? (
            <p className="p-6 text-center text-sm text-muted-foreground">
              Nenhum monstro casa com os filtros.
            </p>
          ) : (
            <VirtualList
              items={filtered}
              estimateSize={76}
              gap={8}
              className="min-h-0 flex-1"
              getKey={(m) => m.id}
              renderItem={(m) => (
                <MonsterRow
                  monster={m}
                  selected={selected?.id === m.id}
                  onOpen={openMonster}
                />
              )}
            />
          )}
        </div>

        <Card className="hidden min-w-0 overflow-y-auto lg:block">
          <CardContent>
            {selected ? (
              <MonsterDetail monster={selected} />
            ) : (
              <p className="text-sm text-muted-foreground">
                Selecione um monstro.
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      <MonsterDialog monster={dialogMonster} onClose={() => setDialogId(null)} />
    </PageChrome>
  )
}

// ─── Filters ────────────────────────────────────────────────────

function BestiaryFilters({
  name,
  onName,
  ndMin,
  onNdMin,
  ndMax,
  onNdMax,
  tipos,
  onToggleTipo,
}: {
  name: string
  onName: (v: string) => void
  ndMin: number
  onNdMin: (v: number) => void
  ndMax: number
  onNdMax: (v: number) => void
  tipos: Set<MonsterTipo>
  onToggleTipo: (t: MonsterTipo) => void
}) {
  return (
    <Card className="shrink-0">
      <CardContent className="space-y-3">
        <Input
          value={name}
          onChange={(e) => onName(e.target.value)}
          placeholder="Buscar monstro…"
          aria-label="Buscar monstro"
        />
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-xs font-medium" htmlFor="nd-min">
              ND mínimo
            </label>
            <NumberInput
              id="nd-min"
              min={0}
              max={20}
              step={0.25}
              value={ndMin}
              onChange={onNdMin}
            />
          </div>
          <div>
            <label className="text-xs font-medium" htmlFor="nd-max">
              ND máximo
            </label>
            <NumberInput
              id="nd-max"
              min={0}
              max={20}
              step={0.25}
              value={ndMax}
              onChange={onNdMax}
            />
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {TIPOS.map((t) => (
            <button key={t} type="button" onClick={() => onToggleTipo(t)}>
              <Badge variant={tipos.has(t) ? 'default' : 'outline'}>
                {TIPO_LABEL[t]}
              </Badge>
            </button>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

// ─── Row ────────────────────────────────────────────────────────

function MonsterRow({
  monster,
  selected,
  onOpen,
}: {
  monster: Monster
  selected: boolean
  onOpen: (m: Monster) => void
}) {
  return (
    <button
      type="button"
      onClick={() => onOpen(monster)}
      aria-pressed={selected}
      className={cn(
        'w-full rounded-md border p-3 text-left transition-colors',
        selected ? 'border-primary bg-accent' : 'hover:border-primary',
      )}
    >
      <p className="font-medium">
        {monster.name}{' '}
        <Badge variant="secondary">ND {formatNd(monster.nd)}</Badge>{' '}
        <Badge variant="outline">{TIPO_LABEL[monster.tipo]}</Badge>{' '}
        <Badge variant="outline">{monster.size}</Badge>
      </p>
      <p className="text-xs text-muted-foreground">
        HP {monster.hp} · Defesa {monster.defesa} · Deslocamento{' '}
        {monster.deslocamento} · p{monster.bookPage}
      </p>
    </button>
  )
}

// ─── Detail (shared by side panel + dialog) ─────────────────────

function MonsterDetail({ monster }: { monster: Monster }) {
  return (
    <div className="space-y-4 text-sm">
      <div>
        <h2 className="text-lg font-semibold">
          {monster.name}{' '}
          <Badge variant="secondary">ND {formatNd(monster.nd)}</Badge>
        </h2>
        <p className="text-xs text-muted-foreground">
          {TIPO_LABEL[monster.tipo]} · {monster.size} · p{monster.bookPage} · XP{' '}
          {xpForNd(monster.nd)}
        </p>
      </div>

      <div className="grid grid-cols-3 gap-2">
        <Stat label="HP" value={monster.hp} />
        <Stat label="Defesa" value={monster.defesa} />
        <Stat label="Deslocamento" value={monster.deslocamento} />
      </div>

      <DetailSection title="Atributos">
        <div className="grid grid-cols-3 gap-1 sm:grid-cols-6">
          <Stat label="For" value={signed(monster.forca)} />
          <Stat label="Des" value={signed(monster.destreza)} />
          <Stat label="Con" value={signed(monster.constituicao)} />
          <Stat label="Int" value={signed(monster.inteligencia)} />
          <Stat label="Sab" value={signed(monster.sabedoria)} />
          <Stat label="Car" value={signed(monster.carisma)} />
        </div>
      </DetailSection>

      <DetailSection title="Perícias de resistência">
        <div className="grid grid-cols-3 gap-1">
          <Stat label="Fortitude" value={signed(monster.fortitude)} />
          <Stat label="Reflexos" value={signed(monster.reflexos)} />
          <Stat label="Vontade" value={signed(monster.vontade)} />
        </div>
      </DetailSection>

      {monster.attacks.length > 0 && (
        <DetailSection title="Ataques">
          <div className="space-y-1">
            {monster.attacks.map((a, i) => (
              <div key={i} className="rounded-md border p-2">
                <p className="font-medium">
                  {a.name}{' '}
                  <span className="text-muted-foreground">
                    {signed(a.attackBonus)} · {a.damage}
                  </span>
                </p>
                {a.special && (
                  <p className="text-xs text-muted-foreground">{a.special}</p>
                )}
              </div>
            ))}
          </div>
        </DetailSection>
      )}

      {monster.specialAbilities.length > 0 && (
        <DetailSection title="Habilidades especiais">
          <ul className="list-disc space-y-1 pl-5">
            {monster.specialAbilities.map((ability, i) => (
              <li key={i}>{ability}</li>
            ))}
          </ul>
        </DetailSection>
      )}
    </div>
  )
}

function MonsterDialog({
  monster,
  onClose,
}: {
  monster: Monster | null
  onClose: () => void
}) {
  return (
    <Dialog open={monster !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        {monster && (
          <>
            <DialogHeader className="sr-only">
              <DialogTitle>{monster.name}</DialogTitle>
              <DialogDescription>
                Estatísticas de {monster.name}
              </DialogDescription>
            </DialogHeader>
            <MonsterDetail monster={monster} />
            <DialogFooter>
              <Button variant="outline" onClick={onClose}>
                Fechar
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}

// ─── Helpers ────────────────────────────────────────────────────

function DetailSection({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <section>
      <h3 className="mb-1 text-xs font-semibold uppercase text-muted-foreground">
        {title}
      </h3>
      {children}
    </section>
  )
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-md border p-2 text-center">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-sm font-semibold">{value}</p>
    </div>
  )
}

function signed(n: number): string {
  return n >= 0 ? `+${n}` : `${n}`
}
