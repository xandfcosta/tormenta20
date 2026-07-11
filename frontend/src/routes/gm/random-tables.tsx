import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useState } from 'react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { DicePill } from '@/shared/ui/dice-pill'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'
import {
  BUSCA_CHALLENGE_TABLE,
  BUSCA_OUTCOME_TABLE,
  buscaChallengeFromRoll,
  buscaOutcomeFromSuccesses,
  buscaTestCd,
  chaseEventFromRoll,
  dungeonIdeaFromRoll,
  rewardCastigoFromRoll,
  ruinaFromRoll,
  type BuscaChallengeRow,
  type ChaseEventRow,
  type DungeonIdea,
  type RewardCastigoRow,
  type RuinaRow,
} from '@tormenta20/t20-data'
import { meQueryOptions } from '@/shared/lib/queries'

/**
 * Rolls the Cap 6 mesa tables client-side (no API round-trip — all
 * catalogs live in t20-data). Each table is independent + carries its
 * own roll history so the mestre can compare recent draws.
 */
export const Route = createFileRoute('/gm/random-tables')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  component: RandomTablesPage,
})

function RandomTablesPage() {
  return (
    <PageChrome className="space-y-4">
      <div className="flex items-center gap-3">
        <Link to="/gm">
          <Button variant="outline" size="sm">
            ←
          </Button>
        </Link>
        <SectionHeading variant="aharadak" as="h1">
          Tabelas de mesa
        </SectionHeading>
      </div>

      <RuinaCard />
      <ChaseEventCard />
      <BuscaCard />
      <ConsequenciasCard />
      <DungeonIdeaCard />
    </PageChrome>
  )
}

// ─── Helpers ────────────────────────────────────────────────────

function rollDie(sides: number): number {
  return Math.floor(Math.random() * sides) + 1
}

function useRollHistory<T>() {
  const [history, setHistory] = useState<{ roll: number; result: T }[]>([])
  const push = (roll: number, result: T) => {
    setHistory((prev) => [{ roll, result }, ...prev].slice(0, 5))
  }
  const clear = () => setHistory([])
  return { history, push, clear }
}

// ─── Ermos: Ruína d6 ────────────────────────────────────────────

function RuinaCard() {
  const { history, push, clear } = useRollHistory<RuinaRow>()
  const roll = () => {
    const d = rollDie(6)
    push(d, ruinaFromRoll(d))
  }
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="font-display tracking-wide">
          Ermos — Ruína (d6, p269)
        </CardTitle>
        <div className="flex gap-2">
          <Button onClick={roll}>Rolar d6</Button>
          <Button variant="outline" onClick={clear}>
            Limpar
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <History history={history}>
          {(entry) => (
            <>
              <DicePill sides={6} />
              <span className="font-hud tabular-nums text-[color:var(--primary)]">
                = {entry.roll}
              </span>
              <span className="font-medium">{entry.result.label}</span>
            </>
          )}
        </History>
      </CardContent>
    </Card>
  )
}

// ─── Tabela 6-5: Perseguições d20 ───────────────────────────────

function ChaseEventCard() {
  const { history, push, clear } = useRollHistory<ChaseEventRow>()
  const roll = () => {
    const d = rollDie(20)
    push(d, chaseEventFromRoll(d))
  }
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="font-display tracking-wide">
          Perseguições (d20, p274)
        </CardTitle>
        <div className="flex gap-2">
          <Button onClick={roll}>Rolar d20</Button>
          <Button variant="outline" onClick={clear}>
            Limpar
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <History history={history}>
          {(entry) => (
            <>
              <DicePill sides={20} />
              <span className="font-hud tabular-nums text-[color:var(--primary)]">
                = {entry.roll}
              </span>
              <Badge
                variant={
                  entry.result.kind === 'nenhum'
                    ? 'secondary'
                    : entry.result.kind === 'atalho'
                      ? 'default'
                      : 'outline'
                }
              >
                {entry.result.kind}
              </Badge>
              {entry.result.test && entry.result.cd && (
                <span className="text-sm text-muted-foreground">
                  {entry.result.test} CD {entry.result.cd}
                </span>
              )}
              <span className="text-sm">{entry.result.example}</span>
            </>
          )}
        </History>
      </CardContent>
    </Card>
  )
}

// ─── Tabela 6-6: Buscas 2d12 ────────────────────────────────────

function BuscaCard() {
  const { history, push, clear } = useRollHistory<BuscaChallengeRow>()
  const [level, setLevel] = useState<number>(1)
  const roll = () => {
    const d = rollDie(12) + rollDie(12)
    push(d, buscaChallengeFromRoll(d))
  }
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="font-display tracking-wide">
          Buscas (2d12, p279)
        </CardTitle>
        <div className="flex items-center gap-2">
          <label className="text-xs" htmlFor="busca-level">
            Nível
          </label>
          <input
            id="busca-level"
            type="number"
            min={1}
            max={20}
            value={level}
            onChange={(e) => setLevel(Number(e.target.value) || 1)}
            className="h-8 w-16 rounded-md border bg-transparent px-2 text-sm"
          />
          <Button onClick={roll}>Rolar 2d12</Button>
          <Button variant="outline" onClick={clear}>
            Limpar
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        <p className="text-xs text-muted-foreground">
          Total no catálogo: {BUSCA_CHALLENGE_TABLE.length} entradas ·{' '}
          CD do teste = <b>{buscaTestCd(level)}</b> (20 + ½ nível)
        </p>
        <History history={history}>
          {(entry) => (
            <>
              <DicePill count={2} sides={12} />
              <span className="font-hud tabular-nums text-[color:var(--primary)]">
                = {entry.roll}
              </span>
              <Badge>{entry.result.skill}</Badge>
              <span className="text-sm text-muted-foreground">
                {entry.result.example}
              </span>
            </>
          )}
        </History>
      </CardContent>
    </Card>
  )
}

// ─── Tabela 6-7: Consequências ─────────────────────────────────

function ConsequenciasCard() {
  const [successes, setSuccesses] = useState<0 | 1 | 2 | 3>(2)
  const outcome = buscaOutcomeFromSuccesses(successes)
  const { history, push, clear } = useRollHistory<RewardCastigoRow>()
  const roll = () => {
    const d = rollDie(6)
    push(d, rewardCastigoFromRoll(d))
  }
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="font-display tracking-wide">
          Consequências de Buscas (p279)
        </CardTitle>
        <div className="flex gap-2">
          <Button onClick={roll}>Rolar d6</Button>
          <Button variant="outline" onClick={clear}>
            Limpar
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center gap-2">
          <label className="text-xs" htmlFor="busca-succ">
            Sucessos (0-3):
          </label>
          <select
            id="busca-succ"
            value={successes}
            onChange={(e) =>
              setSuccesses(Number(e.target.value) as 0 | 1 | 2 | 3)
            }
            className="h-8 rounded-md border bg-transparent px-2 text-sm"
          >
            {BUSCA_OUTCOME_TABLE.map((r) => (
              <option key={r.successes} value={r.successes}>
                {r.successes}
              </option>
            ))}
          </select>
          <span className="text-sm">
            → <b>{outcome.result}</b>
          </span>
        </div>
        <History history={history}>
          {(entry) => (
            <>
              <DicePill sides={6} />
              <span className="font-hud tabular-nums text-[color:var(--primary)]">
                = {entry.roll}
              </span>
              <Badge>Recompensa: {entry.result.reward}</Badge>
              <Badge variant="destructive">
                Castigo: {entry.result.castigo}
              </Badge>
            </>
          )}
        </History>
      </CardContent>
    </Card>
  )
}

// ─── Tabela 6-2: Ideias de Masmorra d20 ─────────────────────────

function DungeonIdeaCard() {
  const { history, push, clear } = useRollHistory<DungeonIdea>()
  const roll = () => {
    const d = rollDie(20)
    push(d, dungeonIdeaFromRoll(d))
  }
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="font-display tracking-wide">
          Ideias de masmorra (d20, p263)
        </CardTitle>
        <div className="flex gap-2">
          <Button onClick={roll}>Rolar d20</Button>
          <Button variant="outline" onClick={clear}>
            Limpar
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <History history={history}>
          {(entry) => (
            <>
              <DicePill sides={20} />
              <span className="font-hud tabular-nums text-[color:var(--primary)]">
                = {entry.roll}
              </span>
              <span className="font-medium">{entry.result.label}</span>
            </>
          )}
        </History>
      </CardContent>
    </Card>
  )
}

// ─── Shared history renderer ────────────────────────────────────

function History<T>({
  history,
  children,
}: {
  history: { roll: number; result: T }[]
  children: (entry: { roll: number; result: T }) => React.ReactNode
}) {
  if (history.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        Ainda não rolou. Clique no botão acima.
      </p>
    )
  }
  return (
    <div className="space-y-1">
      {history.map((entry, i) => (
        <div
          key={i}
          className="flex flex-wrap items-center gap-2 rounded-md border p-2 text-sm"
        >
          {children(entry)}
        </div>
      ))}
    </div>
  )
}
