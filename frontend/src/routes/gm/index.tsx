import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { meQueryOptions } from '@/lib/queries'

/**
 * GM tools hub. Landing page with links to each generator/lookup.
 * Sub-tools ship progressively as Fase E1-E4.
 */
export const Route = createFileRoute('/gm/')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  component: GMHubPage,
})

const TOOLS = [
  {
    to: '/gm/random-tables',
    title: 'Tabelas de mesa',
    description:
      'Rola d6/d20/2d12 nas tabelas Cap 6 (ruína, perseguições, buscas, consequências, ideias de masmorra).',
    ready: true,
  },
  {
    to: '/gm/bestiary',
    title: 'Bestiário',
    description:
      'Consulta rápida de criaturas por ND, tipo e tamanho, com ataques e habilidades especiais.',
    ready: true,
  },
  {
    to: '/gm/encounters',
    title: 'Construtor de encontros',
    description:
      'Calcula ND total do combate combinando criaturas e distribui XP entre o grupo.',
    ready: true,
  },
  {
    to: '/gm/dungeon-generator',
    title: 'Gerador de masmorras',
    description:
      'Estrutura salas, ameaças e objetivos por tamanho seguindo o Cap 6.',
    ready: false,
  },
] as const

function GMHubPage() {
  return (
    <div className="mx-auto h-full max-w-5xl space-y-6 overflow-y-auto p-6">
      <div>
        <h1 className="text-3xl font-semibold">Ferramentas de mestre</h1>
        <p className="text-sm text-muted-foreground">
          Utilitários apoiando o mestre entre as sessões e durante o jogo.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        {TOOLS.map((tool) =>
          tool.ready ? (
            <Link key={tool.to} to={tool.to}>
              <Card className="transition hover:border-primary/40 hover:shadow-md">
                <CardHeader>
                  <CardTitle>{tool.title}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm text-muted-foreground">
                  {tool.description}
                </CardContent>
              </Card>
            </Link>
          ) : (
            <Card
              key={tool.to}
              className="opacity-60"
            >
              <CardHeader className="flex flex-row items-center justify-between">
                <CardTitle>{tool.title}</CardTitle>
                <Button variant="outline" size="sm" disabled>
                  Em breve
                </Button>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                {tool.description}
              </CardContent>
            </Card>
          ),
        )}
      </div>
    </div>
  )
}
