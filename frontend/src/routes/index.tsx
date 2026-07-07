import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { PageChrome } from '@/components/ui/page-chrome'
import { SectionHeading, type SectionHeadingVariant } from '@/components/ui/section-heading'
import { meQueryOptions } from '@/lib/queries'

/**
 * Root landing. Two modes:
 *   - logged in  → shortcut cards for the three main flows
 *   - logged out → hero + feature summary
 *
 * Both variants sit inside `PageChrome` so the design-system tokens
 * (background, noise layer, container width) apply uniformly. Section
 * glyphs — Aharadak for GM-arcane, Kallyadranoch for combat — mark the
 * matching shortcut cards without decoration for its own sake.
 */
export const Route = createFileRoute('/')({
  component: HomePage,
})

type Shortcut = {
  to: '/characters' | '/campaigns' | '/gm'
  title: string
  description: string
  glyph: SectionHeadingVariant
}

const SHORTCUTS: readonly Shortcut[] = [
  {
    to: '/characters',
    title: 'Personagens',
    description:
      'Ficha completa, com atributos, perícias, poderes de classe, equipamentos e efeitos ativos.',
    glyph: 'default',
  },
  {
    to: '/campaigns',
    title: 'Campanhas',
    description:
      'Gerencie campanhas, adicione personagens como membros e conduza sessões com notas e status.',
    glyph: 'kallyadranoch',
  },
  {
    to: '/gm',
    title: 'Ferramentas de mestre',
    description:
      'Bestiário, construtor de encontros, tabelas de mesa e gerador de masmorras — todos alimentados pelas regras do livro.',
    glyph: 'aharadak',
  },
]

function HomePage() {
  const me = useQuery(meQueryOptions)
  const user = me.data

  if (user) {
    return (
      <PageChrome width="wide" className="space-y-8">
        <header className="space-y-2">
          <SectionHeading as="h1" variant="kallyadranoch">
            Tormenta 20
          </SectionHeading>
          <p className="text-sm text-muted-foreground">
            Olá, {user.name ?? user.email} — por onde começamos?
          </p>
        </header>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {SHORTCUTS.map((s) => (
            <Link key={s.to} to={s.to} className="group">
              <Card className="h-full border-border/60 bg-card/70 transition group-hover:border-[color:var(--primary)]/60 group-hover:shadow-lg">
                <CardHeader>
                  <SectionHeading as="h3" variant={s.glyph}>
                    {s.title}
                  </SectionHeading>
                </CardHeader>
                <CardContent className="text-sm text-muted-foreground">
                  {s.description}
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      </PageChrome>
    )
  }

  return (
    <PageChrome width="compact" className="space-y-10">
      <section className="space-y-4 text-center">
        <SectionHeading
          as="h1"
          variant="kallyadranoch"
          className="justify-center"
        >
          Tormenta 20
        </SectionHeading>
        <p className="text-base text-muted-foreground">
          Gerenciador de sessões, fichas e ferramentas de mestre com as
          regras do livro básico.
        </p>
        <div className="flex flex-wrap justify-center gap-3 pt-2">
          <Link to="/login">
            <Button variant="outline">Entrar</Button>
          </Link>
          <Link to="/register">
            <Button>Criar conta</Button>
          </Link>
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-3">
        <FeatureCard
          title="Ficha automática"
          description="Atributos, perícias, defesa e ataques calculados a partir das regras — sem contar +1s à mão."
        />
        <FeatureCard
          title="Campanhas com sessões"
          description="Organize sua mesa em campanhas, adicione membros e mantenha um histórico de sessões com notas."
          glyph="kallyadranoch"
        />
        <FeatureCard
          title="Ferramentas de mestre"
          description="Bestiário, encontros, tabelas de mesa e masmorras — tudo pronto entre uma cena e outra."
          glyph="aharadak"
        />
      </section>
    </PageChrome>
  )
}

function FeatureCard({
  title,
  description,
  glyph = 'default',
}: {
  title: string
  description: string
  glyph?: SectionHeadingVariant
}) {
  return (
    <Card className="h-full border-border/60 bg-card/60">
      <CardHeader>
        <SectionHeading as="h3" variant={glyph}>
          {title}
        </SectionHeading>
      </CardHeader>
      <CardContent className="text-sm text-muted-foreground">
        {description}
      </CardContent>
    </Card>
  )
}
