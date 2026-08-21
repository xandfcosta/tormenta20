import { useNavigate } from '@tanstack/solid-router'
import { SceneShell } from '@/shared/layout/scene-shell'
import { createSfx } from '@/shared/lib/sfx'
import { useUi } from '@/shared/stores/ui-context'
import { cn } from '@/shared/lib/utils'
import { CorSection } from './cor-section'
import { FocoSection, MovimentoSection } from './foco-movimento-section'
import { PecasSection } from './pecas-section'
import { RaioSection } from './raio-section'
import { TipografiaSection } from './tipografia-section'

/**
 * O Grimório: a folha de especificação viva do sistema de desenho.
 *
 * Ela existiu antes, foi apagada, e dois comentários do `index.css` continuaram
 * mandando "validar live at /grimorio" por meses depois disso — apontando para
 * o nada (ALE-173). Volta como o lugar onde uma decisão de desenho se toma
 * OLHANDO, em vez de se imaginar: a escala do raio, o vizinho de matiz que faz
 * uma barra ferida ler como cheia, as três gramáticas de foco que ninguém
 * consegue comparar espalhadas por 22 arquivos.
 *
 * Duas regras sustentam a página, e as duas existem contra o mesmo apodrecer:
 *
 * 1. **Ela lê o valor do navegador, nunca o transcreve.** Cada legenda vem do
 *    `getComputedStyle` do nó ao lado, então o número e o desenho não têm como
 *    discordar.
 * 2. **Ela usa os componentes de verdade**, importados de `shared/ui`. Uma
 *    imitação com as mesmas classes mentiria no primeiro dia em que alguém
 *    mexesse no original.
 *
 * É cena como as outras — mesma casca, mesmos tokens — porque uma folha de
 * desenho que morasse fora do escopo `.scene-grimorio` mostraria cores que a
 * cena não usa.
 */

const TRILHA = [
  { id: 'cor', rotulo: 'Cor' },
  { id: 'raio', rotulo: 'Raio' },
  { id: 'tipografia', rotulo: 'Tipografia' },
  { id: 'pecas', rotulo: 'Peças' },
  { id: 'foco', rotulo: 'Foco' },
  { id: 'movimento', rotulo: 'Movimento' },
]

export function GrimorioPage() {
  const navigate = useNavigate()
  const ui = useUi()
  const sfx = createSfx(ui)

  return (
    <SceneShell
      dense
      bleed
      title="Grimório"
      backLabel="Hub"
      onBack={() => {
        sfx('back')
        navigate({ to: '/' })
      }}
      onEnter={() => sfx('transition')}
    >
      <div class="flex min-h-0 flex-1 flex-col gap-3 px-4 py-3 lg:flex-row lg:gap-4">
        <nav
          aria-label="Seções do sistema de desenho"
          data-nav-region="trilha"
          data-nav-layout="list"
          class={cn(
            // Uma árvore só, quebrando por classe: fila horizontal no estreito,
            // coluna ancorada no largo — a mesma regra da trilha do /gm.
            'flex shrink-0 gap-1 overflow-x-auto lg:w-40 lg:flex-col lg:overflow-visible',
          )}
        >
          {TRILHA.map((item) => (
            <a
              href={`#${item.id}`}
              class="shrink-0 rounded-none border border-grimorio-iron px-3 py-1.5 font-heading text-2xs uppercase tracking-[0.16em] text-muted-foreground transition-colors hover:border-grimorio-gold hover:text-grimorio-gold"
            >
              {item.rotulo}
            </a>
          ))}
        </nav>

        <div class="min-h-0 flex-1 space-y-8 overflow-y-auto pr-1 pb-8">
          <p class="max-w-prose text-xs text-muted-foreground">
            Tudo nesta folha é medido no navegador em tempo real e desenhado pelos componentes de
            verdade. Se um número aqui estiver errado, o errado é o código — não a legenda.
          </p>
          <CorSection />
          <RaioSection />
          <TipografiaSection />
          <PecasSection />
          <FocoSection />
          <MovimentoSection />
        </div>
      </div>
    </SceneShell>
  )
}
