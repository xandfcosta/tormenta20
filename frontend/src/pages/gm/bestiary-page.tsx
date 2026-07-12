import { Link } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
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
import { SectionHeading } from '@/shared/ui/section-heading'
import { WindowVirtualList } from '@/shared/ui/virtual-list'
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

/**
 * Bestiary lookup. All 20 monsters currently in BESTIARY; filters:
 *   - free-text name (accent-insensitive)
 *   - MonsterTipo multi-select
 *   - ND min/max
 * Sort ND ascending by default. Click opens Dialog with full stats.
 */

export function BestiaryPage() {
  const [name, setName] = useState('')
  const [tipos, setTipos] = useState<Set<MonsterTipo>>(new Set())
  const [ndMin, setNdMin] = useState(0)
  const [ndMax, setNdMax] = useState(20)
  const [selected, setSelected] = useState<Monster | null>(null)

  const filtered = useMemo(() => {
    const nameNorm = normalize(name)
    return BESTIARY.filter((m) => {
      if (nameNorm && !normalize(m.name).includes(nameNorm)) return false
      if (tipos.size > 0 && !tipos.has(m.tipo)) return false
      if (m.nd < ndMin || m.nd > ndMax) return false
      return true
    })
      .slice()
      .sort((a, b) => a.nd - b.nd || a.name.localeCompare(b.name))
  }, [name, tipos, ndMin, ndMax])

  const toggleTipo = (t: MonsterTipo) => {
    setTipos((prev) => {
      const next = new Set(prev)
      if (next.has(t)) next.delete(t)
      else next.add(t)
      return next
    })
  }

  return (
    <PageChrome width="wide" className="space-y-4">
      <div className="flex items-center gap-3">
        <Link to="/gm">
          <Button variant="outline" size="sm">
            ←
          </Button>
        </Link>
        <SectionHeading variant="aharadak" as="h1">
          Bestiário
        </SectionHeading>
        <span className="text-sm text-muted-foreground">
          {filtered.length} / {BESTIARY.length}
        </span>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Filtros</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <div>
              <label className="text-xs font-medium" htmlFor="mob-name">
                Nome
              </label>
              <Input
                id="mob-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Buscar…"
              />
            </div>
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
                onChange={setNdMin}
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
                onChange={setNdMax}
              />
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            {TIPOS.map((t) => {
              const active = tipos.has(t)
              return (
                <button
                  key={t}
                  type="button"
                  onClick={() => toggleTipo(t)}
                >
                  <Badge variant={active ? 'default' : 'outline'}>
                    {TIPO_LABEL[t]}
                  </Badge>
                </button>
              )
            })}
          </div>
        </CardContent>
      </Card>

      {filtered.length === 0 ? (
        <p className="p-6 text-center text-sm text-muted-foreground">
          Nenhum monstro casa com os filtros.
        </p>
      ) : (
        <WindowVirtualList
          items={filtered}
          estimateSize={76}
          gap={8}
          getKey={(m) => m.id}
          renderItem={(m) => <MonsterRow monster={m} onOpen={setSelected} />}
        />
      )}

      <MonsterDialog
        monster={selected}
        onClose={() => setSelected(null)}
      />
    </PageChrome>
  )
}

// ─── Row ────────────────────────────────────────────────────────

function MonsterRow({
  monster,
  onOpen,
}: {
  monster: Monster
  onOpen: (m: Monster) => void
}) {
  return (
    <button
      type="button"
      onClick={() => onOpen(monster)}
      className="w-full rounded-md border p-3 text-left transition hover:border-[color:var(--primary)]/50 hover:shadow-sm"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex-1 space-y-1">
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
        </div>
      </div>
    </button>
  )
}

// ─── Dialog ─────────────────────────────────────────────────────

function MonsterDialog({
  monster,
  onClose,
}: {
  monster: Monster | null
  onClose: () => void
}) {
  return (
    <Dialog
      open={monster !== null}
      onOpenChange={(open) => !open && onClose()}
    >
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        {monster && (
          <>
            <DialogHeader>
              <DialogTitle>
                {monster.name}{' '}
                <Badge variant="secondary">ND {formatNd(monster.nd)}</Badge>
              </DialogTitle>
              <DialogDescription>
                {TIPO_LABEL[monster.tipo]} · {monster.size} · p
                {monster.bookPage} · XP {xpForNd(monster.nd)}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4 text-sm">
              <div className="grid grid-cols-3 gap-2">
                <Stat label="HP" value={monster.hp} />
                <Stat label="Defesa" value={monster.defesa} />
                <Stat label="Deslocamento" value={monster.deslocamento} />
              </div>

              <section>
                <h3 className="mb-1 text-xs font-semibold uppercase text-muted-foreground">
                  Atributos
                </h3>
                <div className="grid grid-cols-3 gap-1 sm:grid-cols-6">
                  <Stat label="For" value={signed(monster.forca)} />
                  <Stat label="Des" value={signed(monster.destreza)} />
                  <Stat label="Con" value={signed(monster.constituicao)} />
                  <Stat label="Int" value={signed(monster.inteligencia)} />
                  <Stat label="Sab" value={signed(monster.sabedoria)} />
                  <Stat label="Car" value={signed(monster.carisma)} />
                </div>
              </section>

              <section>
                <h3 className="mb-1 text-xs font-semibold uppercase text-muted-foreground">
                  Perícias de resistência
                </h3>
                <div className="grid grid-cols-3 gap-1">
                  <Stat label="Fortitude" value={signed(monster.fortitude)} />
                  <Stat label="Reflexos" value={signed(monster.reflexos)} />
                  <Stat label="Vontade" value={signed(monster.vontade)} />
                </div>
              </section>

              {monster.attacks.length > 0 && (
                <section>
                  <h3 className="mb-1 text-xs font-semibold uppercase text-muted-foreground">
                    Ataques
                  </h3>
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
                          <p className="text-xs text-muted-foreground">
                            {a.special}
                          </p>
                        )}
                      </div>
                    ))}
                  </div>
                </section>
              )}

              {monster.specialAbilities.length > 0 && (
                <section>
                  <h3 className="mb-1 text-xs font-semibold uppercase text-muted-foreground">
                    Habilidades especiais
                  </h3>
                  <ul className="list-disc space-y-1 pl-5">
                    {monster.specialAbilities.map((ability, i) => (
                      <li key={i}>{ability}</li>
                    ))}
                  </ul>
                </section>
              )}
            </div>

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
