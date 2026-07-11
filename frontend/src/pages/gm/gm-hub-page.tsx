import { Link } from '@tanstack/react-router'
import { Dices, Scroll, Skull, Swords } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'

/**
 * GM tools hub. Landing page with links to each generator/lookup.
 * Sub-tools ship progressively as Fase E1-E4.
 */
type Tool = {
  to: '/gm/random-tables' | '/gm/bestiary' | '/gm/encounters' | '/gm/dungeon-generator'
  title: string
  description: string
  icon: LucideIcon
  ready: boolean
}

const TOOLS: Tool[] = [
  {
    to: '/gm/random-tables',
    title: 'Tabelas de mesa',
    description:
      'Rola d6/d20/2d12 nas tabelas Cap 6 (ruína, perseguições, buscas, consequências, ideias de masmorra).',
    icon: Dices,
    ready: true,
  },
  {
    to: '/gm/bestiary',
    title: 'Bestiário',
    description:
      'Consulta rápida de criaturas por ND, tipo e tamanho, com ataques e habilidades especiais.',
    icon: Skull,
    ready: true,
  },
  {
    to: '/gm/encounters',
    title: 'Construtor de encontros',
    description:
      'Calcula ND total do combate combinando criaturas e distribui XP entre o grupo.',
    icon: Swords,
    ready: true,
  },
  {
    to: '/gm/dungeon-generator',
    title: 'Gerador de masmorras',
    description:
      'Estrutura salas, ameaças e objetivos por tamanho seguindo o Cap 6.',
    icon: Scroll,
    ready: true,
  },
]

export function GMHubPage() {
  return (
    <PageChrome className="space-y-6">
      <div className="space-y-1">
        <SectionHeading variant="kallyadranoch" as="h1">
          Ferramentas de mestre
        </SectionHeading>
        <p className="text-sm text-muted-foreground">
          Utilitários apoiando o mestre entre as sessões e durante o jogo.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        {TOOLS.map((tool) => {
          const Icon = tool.icon
          const inner = (
            <>
              <CardHeader className="flex flex-row items-center justify-between">
                <CardTitle className="flex items-center gap-2 font-display tracking-wide">
                  <Icon className="size-5 text-[color:var(--primary)]" />
                  {tool.title}
                </CardTitle>
                {!tool.ready && (
                  <Button variant="outline" size="sm" disabled>
                    Em breve
                  </Button>
                )}
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground">
                {tool.description}
              </CardContent>
            </>
          )
          return tool.ready ? (
            <Link key={tool.to} to={tool.to}>
              <Card className="h-full transition hover:border-[color:var(--primary)]/50 hover:shadow-[0_1px_0_rgba(0,0,0,0.04),0_18px_50px_-30px_var(--primary)]">
                {inner}
              </Card>
            </Link>
          ) : (
            <Card key={tool.to} className="opacity-60">
              {inner}
            </Card>
          )
        })}
      </div>
    </PageChrome>
  )
}
