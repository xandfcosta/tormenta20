import { createSignal } from 'solid-js'
import { createPrefersReducedMotion } from '@/shared/lib/media-query'
import { Button } from '@/shared/ui/button'
import { piscarVital, pulsarVez } from '@/features/session-tracker/turn-juice'
import { SpecBlock, SpecSection } from './spec-primitives'

/**
 * As duas coisas que uma folha impressa não consegue mostrar: o que acontece
 * quando o dedo chega (foco) e o que acontece quando o estado muda (movimento).
 *
 * A seção de movimento DISPARA a animação de verdade, importada de onde a
 * sessão a usa. Se alguém quebrar o flash, esta página para de piscar — que é
 * mais barato de notar do que abrir uma sessão ao vivo e ferir um combatente.
 */

/** As três gramáticas de foco que a auditoria contou, lado a lado (ALE-173, P4). */
export function FocoSection() {
  return (
    <SpecSection id="foco" titulo="Foco">
      <SpecBlock
        titulo="As três gramáticas"
        nota="Navegue com Tab. Hoje convivem três tratamentos, e 75 dos 82 botões crus caem num quarto, acidental."
      >
        <Button variant="outline">anel do shadcn</Button>
        <button
          type="button"
          class="rounded-none border border-grimorio-iron px-3 py-1.5 text-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-grimorio-gold"
        >
          contorno do grimório
        </button>
        <div data-nav-region="amostra">
          <button
            type="button"
            class="rounded-none border border-grimorio-iron px-3 py-1.5 text-sm"
          >
            regra global (só dentro de data-nav-region)
          </button>
        </div>
      </SpecBlock>
    </SpecSection>
  )
}

/**
 * O movimento que EXPLICA o que chegou pelo socket (ALE-174).
 *
 * O gate aparece na tela de propósito: `prefers-reduced-motion` NÃO cobre
 * WAAPI, então todo `el.animate` da casa tem de perguntar — e aqui dá para ver
 * se ele está perguntando, ligando a preferência no sistema e clicando de novo.
 */
export function MovimentoSection() {
  const parado = createPrefersReducedMotion()
  const [linha, setLinha] = createSignal<HTMLDivElement>()

  const disparar = (acao: () => void) => {
    if (parado()) return
    acao()
  }

  return (
    <SpecSection id="movimento" titulo="Movimento">
      <SpecBlock
        titulo="A linha da iniciativa"
        nota="Os mesmos disparos que a sessão usa. Eles vão por diferença de VALOR e não por montagem: a lista remonta a cada sync, e animação de entrada tocaria em todas as linhas de uma vez."
      >
        <div class="w-full max-w-md space-y-2">
          <div
            ref={setLinha}
            class="relative flex items-center gap-3 rounded-none border border-grimorio-gold/60 bg-[color-mix(in_oklch,var(--grimorio-gold)_6%,transparent)] p-2.5 text-sm"
          >
            <span class="font-mono text-xs">18</span>
            <span class="font-medium">Ogro das Cavernas</span>
          </div>
          <div class="flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={() => disparar(() => piscarVital(linha(), { curou: false }))}
            >
              Ferir
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => disparar(() => piscarVital(linha(), { curou: true }))}
            >
              Curar
            </Button>
            <Button size="sm" variant="outline" onClick={() => disparar(() => pulsarVez(linha()))}>
              Passar a vez
            </Button>
          </div>
          <p class="text-xs text-muted-foreground">
            Movimento reduzido:{' '}
            <span class="font-mono text-grimorio-gold">{parado() ? 'LIGADO' : 'desligado'}</span> —
            com ele ligado, os três botões não animam nada.
          </p>
        </div>
      </SpecBlock>
    </SpecSection>
  )
}
