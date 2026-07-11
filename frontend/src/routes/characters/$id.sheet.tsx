import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'
import { Skeleton } from '@/shared/ui/skeleton'
import { characterSheetQueryOptions } from '@/entities/character/queries'
import { meQueryOptions } from '@/entities/user/queries'
import type { CharacterWithComputed, ComputedSheet } from '@/shared/api/api'

/**
 * Server-computed sheet view. Renders the ComputedSheet payload from
 * `GET /characters/:id/sheet` — the same fields the orchestrator now
 * produces (attrs+race+vitals+Defesa full+saves+skills+attacks+movement).
 *
 * Separate from the main editor page: this one is read-only + a
 * consistency check that the mapper + orchestrator are talking correctly.
 */
export const Route = createFileRoute('/characters/$id/sheet')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(
      characterSheetQueryOptions(Number(params.id)),
    ),
  component: CharacterSheetPage,
})

function CharacterSheetPage() {
  const { id } = Route.useParams()
  const query = useQuery(characterSheetQueryOptions(Number(id)))

  if (query.isLoading)
    return (
      <PageChrome className="space-y-3">
        <Skeleton className="h-8 w-72" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-40 w-full" />
      </PageChrome>
    )
  if (query.isError)
    return (
      <PageChrome>
        <p className="text-destructive">
          {(query.error as Error).message}
        </p>
      </PageChrome>
    )
  if (!query.data) return null

  const { computed } = query.data as CharacterWithComputed

  return (
    <PageChrome className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <Link to="/characters/$id" params={{ id }}>
          <Button variant="outline" size="sm">
            ← Voltar
          </Button>
        </Link>
        <SectionHeading variant="kallyadranoch" as="h1" className="text-2xl">
          <span>
            {query.data.name}{' '}
            <Badge variant="secondary">Nv {computed.level}</Badge>
          </span>
        </SectionHeading>
      </div>

      {computed.warnings.length > 0 && (
        <div className="rounded-md border border-amber-400/60 bg-amber-50 p-3 text-sm text-amber-900 dark:bg-amber-950/40 dark:text-amber-100">
          <p className="font-medium">Avisos do orchestrator:</p>
          <ul className="ml-4 mt-1 list-disc">
            {computed.warnings.map((w, i) => (
              <li key={i}>{w}</li>
            ))}
          </ul>
        </div>
      )}

      <AttributesCard computed={computed} />
      <VitalsAndMovementCard computed={computed} />
      <DefenseCard computed={computed} />
      <SavesCard computed={computed} />
      <AttacksCard computed={computed} />
      <SkillsCard computed={computed} />
    </PageChrome>
  )
}

// ─── Attributes ─────────────────────────────────────────────────

const ATTR_LABELS: Record<string, string> = {
  strength: 'For',
  dexterity: 'Des',
  constitution: 'Con',
  intelligence: 'Int',
  wisdom: 'Sab',
  charisma: 'Car',
}

function AttributesCard({ computed }: { computed: ComputedSheet }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Atributos</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-3 gap-3 sm:grid-cols-6">
          {Object.entries(computed.attributes).map(([key, a]) => (
            <div
              key={key}
              className="rounded-md border p-2 text-center"
            >
              <p className="text-xs font-medium text-muted-foreground">
                {ATTR_LABELS[key] ?? key}
              </p>
              <p className="text-2xl font-semibold">{signed(a.total)}</p>
              <p className="text-xs text-muted-foreground">
                base {signed(a.base)}
                {a.raceMod !== 0 && (
                  <>
                    {' · '}raça {signed(a.raceMod)}
                  </>
                )}
              </p>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

// ─── Vitals + Movement ──────────────────────────────────────────

function VitalsAndMovementCard({ computed }: { computed: ComputedSheet }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">
          Vitais + Deslocamento
        </CardTitle>
      </CardHeader>
      <CardContent className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
        <Stat label="PV" value={`${computed.vitals.pvCurrent}/${computed.vitals.pvMax}`} />
        <Stat label="PM" value={`${computed.vitals.pmCurrent}/${computed.vitals.pmMax}`} />
        <Stat label="Deslocamento" value={`${computed.deslocamento}m`} />
        <Stat label="Tamanho" value={computed.tamanho} />
      </CardContent>
    </Card>
  )
}

// ─── Defense ────────────────────────────────────────────────────

function DefenseCard({ computed }: { computed: ComputedSheet }) {
  const { base, attribute, armor, shield, total } = computed.defense
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Defesa</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-sm">
        <p className="text-3xl font-semibold">{total}</p>
        <p className="text-muted-foreground">
          {base} base
          {' + '}
          {signed(attribute)} Des
          {armor > 0 && <> {' + '}{armor} armadura</>}
          {shield > 0 && <> {' + '}{shield} escudo</>}
        </p>
      </CardContent>
    </Card>
  )
}

// ─── Saves ──────────────────────────────────────────────────────

function SavesCard({ computed }: { computed: ComputedSheet }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Resistências</CardTitle>
      </CardHeader>
      <CardContent className="grid grid-cols-3 gap-3 text-sm">
        <Stat label="Fortitude" value={signed(computed.saves.fortitude)} />
        <Stat label="Reflexos" value={signed(computed.saves.reflexos)} />
        <Stat label="Vontade" value={signed(computed.saves.vontade)} />
      </CardContent>
    </Card>
  )
}

// ─── Attacks ────────────────────────────────────────────────────

function AttacksCard({ computed }: { computed: ComputedSheet }) {
  const { mainHand, offHand } = computed.attacks
  if (!mainHand && !offHand) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="font-display tracking-wide">Ataques</CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">
          Sem arma equipada.
        </CardContent>
      </Card>
    )
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Ataques</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        {mainHand && (
          <div className="rounded-md border p-3">
            <p className="font-medium">
              {mainHand.weaponName ?? 'Mão principal'}{' '}
              <Badge variant="outline">{mainHand.skill}</Badge>
            </p>
            <p className="text-muted-foreground">
              Ataque {signed(mainHand.attackTotal)} · Dano {mainHand.damageDice}
              {mainHand.damageAttributeBonus !== 0 && (
                <> {signed(mainHand.damageAttributeBonus)}</>
              )}{' '}
              · Crítico {mainHand.critRange}+/×{mainHand.critMult} ·{' '}
              {mainHand.damageType}
            </p>
          </div>
        )}
        {offHand && (
          <div className="rounded-md border p-3">
            <p className="font-medium">
              {offHand.weaponName ?? 'Mão secundária'}{' '}
              <Badge variant="outline">{offHand.skill}</Badge>
            </p>
            <p className="text-muted-foreground">
              Ataque {signed(offHand.attackTotal)} · Dano {offHand.damageDice}
              {offHand.damageAttributeBonus !== 0 && (
                <> {signed(offHand.damageAttributeBonus)}</>
              )}
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ─── Skills ─────────────────────────────────────────────────────

function SkillsCard({ computed }: { computed: ComputedSheet }) {
  const entries = Object.entries(computed.skills).sort(([a], [b]) =>
    a.localeCompare(b),
  )
  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-display tracking-wide">Perícias</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 gap-1 text-sm sm:grid-cols-3">
          {entries.map(([id, s]) => (
            <div
              key={id}
              className="flex items-center justify-between rounded-md px-2 py-1 hover:bg-muted"
            >
              <span
                className={
                  s.trained ? 'font-medium' : 'text-muted-foreground'
                }
              >
                {capitalizeSkillId(id)}
                {s.trained && ' *'}
              </span>
              <span
                className={
                  s.cannotUse
                    ? 'text-muted-foreground line-through'
                    : 'font-mono'
                }
              >
                {s.cannotUse ? '—' : signed(s.total)}
              </span>
            </div>
          ))}
        </div>
        <p className="mt-2 text-xs text-muted-foreground">
          * = treinada. Perícias treinadas-apenas sem treino aparecem como
          "—".
        </p>
      </CardContent>
    </Card>
  )
}

// ─── Helpers ────────────────────────────────────────────────────

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-lg font-semibold">{value}</p>
    </div>
  )
}

function signed(n: number): string {
  return n >= 0 ? `+${n}` : `${n}`
}

function capitalizeSkillId(id: string): string {
  return id.charAt(0).toUpperCase() + id.slice(1)
}
