import { Link } from '@tanstack/react-router'
import { BookMarked, Dices, Scroll, Skull, Swords } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { PageChrome } from '@/shared/ui/page-chrome'

/** GM tools hub — a launchpad linking to each generator / lookup. */
type Tool = {
  to:
    | '/gm/random-tables'
    | '/gm/bestiary'
    | '/gm/encounters'
    | '/gm/dungeon-generator'
    | '/gm/catalogs'
  title: string
  description: string
  Icon: LucideIcon
}

const TOOLS: Tool[] = [
  {
    to: '/gm/catalogs',
    title: 'Catálogos',
    description:
      'Consulta rápida de condições, magias, poderes e itens — busca única entre todos os catálogos.',
    Icon: BookMarked,
  },
  {
    to: '/gm/random-tables',
    title: 'Tabelas de mesa',
    description:
      'Rola d6/d20/2d12 nas tabelas Cap 6 (ruína, perseguições, buscas, consequências, ideias de masmorra).',
    Icon: Dices,
  },
  {
    to: '/gm/bestiary',
    title: 'Bestiário',
    description:
      'Consulta rápida de criaturas por ND, tipo e tamanho, com ataques e habilidades especiais.',
    Icon: Skull,
  },
  {
    to: '/gm/encounters',
    title: 'Construtor de encontros',
    description:
      'Calcula ND total do combate combinando criaturas e distribui XP entre o grupo.',
    Icon: Swords,
  },
  {
    to: '/gm/dungeon-generator',
    title: 'Gerador de masmorras',
    description:
      'Estrutura salas, ameaças e objetivos por tamanho seguindo o Cap 6.',
    Icon: Scroll,
  },
]

export function GMHubPage() {
  return (
    <PageChrome className="space-y-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">
          Ferramentas de mestre
        </h1>
        <p className="text-sm text-muted-foreground">
          Utilitários apoiando o mestre entre as sessões e durante o jogo.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {TOOLS.map((tool) => (
          <ToolCard key={tool.to} tool={tool} />
        ))}
      </div>
    </PageChrome>
  )
}

function ToolCard({ tool }: { tool: Tool }) {
  const { Icon } = tool
  return (
    <Link to={tool.to} className="group">
      <Card className="h-full transition-colors group-hover:border-primary">
        <CardHeader className="flex flex-row items-center gap-3">
          <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted">
            <Icon className="size-5" />
          </span>
          <CardTitle>{tool.title}</CardTitle>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">
          {tool.description}
        </CardContent>
      </Card>
    </Link>
  )
}
