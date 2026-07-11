import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { DicePill } from '@/shared/ui/dice-pill'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'
import {
  DUNGEON_SIZE_TABLE,
  classifyDungeonSize,
  dungeonIdeaFromRoll,
  dungeonSizeRow,
  plannedThreats,
  ROOMS_PER_THREAT,
  type DungeonIdea,
  type DungeonSize,
} from '@tormenta20/t20-data'
import { meQueryOptions } from '@/entities/user/queries'
const SIZE_LABEL: Record<DungeonSize, string> = {
  pequena: 'Pequena',
  media: 'Média',
  grande: 'Grande',
}

const PACING_LABEL: Record<
  'parte-de-sessao' | 'sessao-inteira' | 'aventura-inteira',
  string
> = {
  'parte-de-sessao': 'Parte de uma sessão',
  'sessao-inteira': 'Sessão inteira',
  'aventura-inteira': 'Aventura inteira',
}

/**
 * Dungeon generator — parametric structure (rooms → size → threats +
 * objectives) plus Tabela 6-2 d20 idea roller. All catalogs are pure
 * so this page runs entirely client-side.
 */
export const Route = createFileRoute('/gm/dungeon-generator')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  component: DungeonGeneratorPage,
})

function rollDie(sides: number): number {
  return Math.floor(Math.random() * sides) + 1
}

function DungeonGeneratorPage() {
  const [numRooms, setNumRooms] = useState<number>(6)
  const [objective, setObjective] = useState<string>('')
  const [ideas, setIdeas] = useState<
    { roll: number; result: DungeonIdea }[]
  >([])

  const size = useMemo<DungeonSize | null>(
    () => classifyDungeonSize(numRooms || 1),
    [numRooms],
  )
  const sizeRow = size ? dungeonSizeRow(size) : null
  const threats = useMemo(
    () => plannedThreats(numRooms || 1),
    [numRooms],
  )

  const rollIdea = () => {
    const d = rollDie(20)
    setIdeas((prev) => [{ roll: d, result: dungeonIdeaFromRoll(d) }, ...prev].slice(0, 3))
  }

  /**
   * "Gerar" combina uma ideia da Tabela 6-2 com um tamanho aleatório
   * (uniforme entre pequena/média/grande), sorteando um numRooms dentro
   * do range daquele tier. Bom pra kickstart quando o GM não tem plano.
   */
  const generate = () => {
    const d = rollDie(20)
    setIdeas((prev) => [{ roll: d, result: dungeonIdeaFromRoll(d) }, ...prev].slice(0, 3))
    const randomTier =
      DUNGEON_SIZE_TABLE[Math.floor(Math.random() * DUNGEON_SIZE_TABLE.length)]!
    const roomsInTier =
      randomTier.minRooms +
      Math.floor(
        Math.random() * (randomTier.maxRooms - randomTier.minRooms + 1),
      )
    setNumRooms(roomsInTier)
  }

  const reset = () => {
    setNumRooms(6)
    setObjective('')
    setIdeas([])
  }

  return (
    <PageChrome className="space-y-4">
      <div className="flex items-center gap-3">
        <Link to="/gm">
          <Button variant="outline" size="sm">
            ←
          </Button>
        </Link>
        <SectionHeading variant="aharadak" as="h1">
          Gerador de masmorras
        </SectionHeading>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="font-display text-base tracking-wide">
            Estrutura
          </CardTitle>
          <div className="flex gap-2">
            <Button variant="outline" onClick={generate}>
              Gerar
            </Button>
            <Button variant="outline" onClick={reset}>
              Reset
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs font-medium" htmlFor="num-rooms">
                Número de salas
              </label>
              <NumberInput
                id="num-rooms"
                min={1}
                max={50}
                value={numRooms}
                onChange={setNumRooms}
              />
            </div>
            <div>
              <label className="text-xs font-medium" htmlFor="objective">
                Objetivo principal (opcional)
              </label>
              <Input
                id="objective"
                value={objective}
                onChange={(e) => setObjective(e.target.value)}
                placeholder="Resgatar o príncipe capturado…"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <Stat
              label="Tamanho"
              value={size ? SIZE_LABEL[size] : 'Fora do range'}
            />
            <Stat
              label="Pacing"
              value={sizeRow ? PACING_LABEL[sizeRow.pacing] : '—'}
            />
            <Stat label="Ameaças" value={threats} />
            <Stat
              label="Ameaças / 3 salas"
              value={`1 (regra fixa)`}
            />
          </div>

          {sizeRow && (
            <div className="rounded-md border p-3 text-sm">
              <p className="mb-2 font-medium">Objetivos ({SIZE_LABEL[size!]})</p>
              <ul className="ml-4 list-disc space-y-1 text-muted-foreground">
                <li>Principal: 1 (obrigatório)</li>
                <li>
                  Secundários (opcionais): até{' '}
                  {sizeRow.maxSecondaryObjectives}
                </li>
                <li>Opcionais: {sizeRow.optionalObjectives}</li>
              </ul>
            </div>
          )}

          {size === null && numRooms > 50 && (
            <p className="text-sm text-amber-700 dark:text-amber-400">
              Livro p263 recomenda não passar de 50 salas — masmorras
              maiores viram tediosas.
            </p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="font-display text-base tracking-wide">
            Ideias (Tabela 6-2 p263)
          </CardTitle>
          <Button onClick={rollIdea}>Rolar ideia (d20)</Button>
        </CardHeader>
        <CardContent>
          {ideas.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Ainda não rolou. Clique no botão acima ou "Gerar".
            </p>
          ) : (
            <div className="space-y-1">
              {ideas.map((entry, i) => (
                <div
                  key={i}
                  className="flex items-center gap-2 rounded-md border p-2 text-sm"
                >
                  <DicePill sides={20} />
                  <span className="font-hud tabular-nums text-[color:var(--primary)]">
                    = {entry.roll}
                  </span>
                  <span className={i === 0 ? 'font-medium' : ''}>
                    {entry.result.label}
                  </span>
                  {i === 0 && <Badge>Atual</Badge>}
                </div>
              ))}
            </div>
          )}
          <p className="mt-2 text-xs text-muted-foreground">
            Regra pacing: 1 ameaça a cada {ROOMS_PER_THREAT} salas
            (arredondando pra cima).
          </p>
        </CardContent>
      </Card>
    </PageChrome>
  )
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-lg font-semibold">{value}</p>
    </div>
  )
}
