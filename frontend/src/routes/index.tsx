import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { meQueryOptions } from '@/lib/queries'
/**
 * Root landing. Two modes:
 *   - logged in  → shortcut cards for the three main flows
 *   - logged out → feature summary + login/register CTAs
 */
export const Route = createFileRoute('/')({
  component: HomePage,
})

type Shortcut = {
  to: '/characters' | '/campaigns' | '/gm'
  title: string
  description: string
}

const SHORTCUTS: readonly Shortcut[] = [
  {
    to: '/characters',
    title: 'Personagens',
    description:
      'Ficha completa, com atributos, perícias, poderes de classe, equipamentos e efeitos ativos.',
  },
  {
    to: '/campaigns',
    title: 'Campanhas',
    description:
      'Gerencie campanhas, adicione personagens como membros e conduza sessões com notas e status.',
  },
  {
    to: '/gm',
    title: 'Ferramentas de mestre',
    description:
      'Bestiário, construtor de encontros, tabelas de mesa e gerador de masmorras — todos alimentados pelas regras do livro.',
  },
]

function HomePage() {
  const me = useQuery(meQueryOptions)
  const user = me.data

  if (user) {
    return (
      <div className="mx-auto h-full max-w-5xl space-y-6 overflow-y-auto p-6">
        <div>
          <h1 className="text-3xl font-semibold">Tormenta 20</h1>
          <p className="text-sm text-muted-foreground">
            Olá, {user.name ?? user.email} — por onde começamos?
          </p>
        </div>

        <div className="grid gap-4 sm:grid-cols-3">
          {SHORTCUTS.map((s) => (
            <Link key={s.to} to={s.to}>
              <Card className="h-full transition hover:border-primary/40 hover:shadow-md">
                <CardHeader>
                  <CardTitle>{s.title}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm text-muted-foreground">
                  {s.description}
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="mx-auto h-full max-w-3xl space-y-8 overflow-y-auto p-6">
      <div className="space-y-3 text-center">
        <h1 className="text-4xl font-bold">Tormenta 20</h1>
        <p className="text-muted-foreground">
          Gerenciador de sessões, fichas e ferramentas de mestre com as
          regras do livro básico.
        </p>
        <div className="flex justify-center gap-2 pt-3">
          <Link to="/login">
            <Button variant="outline">Login</Button>
          </Link>
          <Link to="/register">
            <Button>Criar conta</Button>
          </Link>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <FeatureCard
          title="Ficha automática"
          description="Atributos, perícias, defesa e ataques calculados a partir das regras — sem contar +1s à mão."
        />
        <FeatureCard
          title="Campanhas com sessões"
          description="Organize sua mesa em campanhas, adicione membros e mantenha um histórico de sessões com notas."
        />
        <FeatureCard
          title="Ferramentas de mestre"
          description="Bestiário, encontros, tabelas de mesa e masmorras — tudo pronto entre uma cena e outra."
        />
      </div>
    </div>
  )
}

function FeatureCard({
  title,
  description,
}: {
  title: string
  description: string
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent className="text-sm text-muted-foreground">
        {description}
      </CardContent>
    </Card>
  )
}
