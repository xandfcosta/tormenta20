import { useMemo, useState } from 'react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { DicePill } from '@/shared/ui/dice-pill'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { PageChrome } from '@/shared/ui/page-chrome'
import { clampToRange } from '@/shared/lib/bounded-number'
import { GmPageHeader } from '@/features/gm-tools/gm-page-header'
import { DUNGEON_SIZE_TABLE, classifyDungeonSize, dungeonIdeaFromRoll, dungeonSizeRow, plannedThreats, ROOMS_PER_THREAT, type DungeonIdea, type DungeonSize } from '@tormenta20/t20-data'

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

function rollDie(sides: number): number {
  return Math.floor(Math.random() * sides) + 1
}

export function DungeonGeneratorPage() {
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
      <GmPageHeader title="Gerador de masmorras" />

      <div className="grid gap-4 lg:grid-cols-[1fr_20rem] lg:items-start">
        <div className="space-y-4">
          <Card>
            <CardHeader className="flex flex-wrap items-center justify-between gap-2">
              <CardTitle className="text-base">Parâmetros</CardTitle>
              <div className="flex gap-2">
                <Button variant="outline" onClick={generate}>
                  Gerar
                </Button>
                <Button variant="outline" onClick={reset}>
                  Reset
                </Button>
              </div>
            </CardHeader>
            <CardContent className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="text-xs font-medium" htmlFor="num-rooms">
                  Número de salas
                </label>
                <NumberInput
                  id="num-rooms"
                  min={1}
                  max={50}
                  value={numRooms}
                  onChange={(v) => setNumRooms(clampToRange(v, { min: 1, max: 50 }))}
                />
              </div>
              <div>
                <label className="text-xs font-medium" htmlFor="objective">
                  Objetivo principal (opcional)
                </label>
                <Input
                  id="objective"
                  value={objective}
                  onChange={(e) => setObjective(e.target.value.slice(0, 200))}
                  maxLength={200}
                  placeholder="Resgatar o príncipe capturado…"
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-wrap items-center justify-between gap-2">
              <CardTitle className="text-base">
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
                      <span className="font-semibold tabular-nums">
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
        </div>

        <Card className="sticky top-0 z-10 order-first self-start lg:order-none lg:top-4">
          <CardHeader>
            <CardTitle className="text-base">Estrutura</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <Stat
                label="Tamanho"
                value={size ? SIZE_LABEL[size] : 'Fora do range'}
              />
              <Stat
                label="Pacing"
                value={sizeRow ? PACING_LABEL[sizeRow.pacing] : '—'}
              />
              <Stat label="Ameaças" value={threats} />
              <Stat label="Ameaças / 3 salas" value={`1 (regra fixa)`} />
            </div>

            {sizeRow && (
              <div className="rounded-md border p-3 text-sm">
                <p className="mb-2 font-medium">
                  Objetivos ({SIZE_LABEL[size!]})
                </p>
                <ul className="ml-4 list-disc space-y-1 text-muted-foreground">
                  <li>Principal: 1 (obrigatório)</li>
                  <li>
                    Secundários (opcionais): até {sizeRow.maxSecondaryObjectives}
                  </li>
                  <li>Opcionais: {sizeRow.optionalObjectives}</li>
                </ul>
              </div>
            )}

            {size === null && numRooms > 50 && (
              <p className="text-sm text-foreground">
                Livro p263 recomenda não passar de 50 salas — masmorras
                maiores viram tediosas.
              </p>
            )}
          </CardContent>
        </Card>
      </div>
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
