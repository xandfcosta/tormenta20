import { type JSX, createSignal, onMount } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * As peças de que a folha de especificação é feita.
 *
 * A regra desta página inteira: **ela lê o valor do navegador, nunca o
 * transcreve**. Uma folha de sistema de desenho que crava "2px" ao lado do
 * quadrado apodrece no dia em que o token muda, e apodrece calada — o número
 * continua ali, bonito e errado, e quem consulta acredita nele. Aqui o quadrado
 * é desenhado pela classe de verdade e a legenda vem do `getComputedStyle` do
 * mesmo nó, então as duas metades não têm como discordar.
 *
 * A leitura é feita no `onMount` porque só existe caixa depois de montar.
 */

/** Lê uma propriedade computada do próprio nó, depois que ele existe. */
export function createComputedValue(
  propriedade: string,
): [() => string, (node: HTMLElement) => void] {
  const [valor, setValor] = createSignal('—')
  let node: HTMLElement | undefined
  onMount(() => {
    if (!node) return
    const estilo = getComputedStyle(node)
    setValor(estilo.getPropertyValue(propriedade).trim() || '—')
  })
  return [valor, (n: HTMLElement) => { node = n }]
}

/** Cabeçalho de uma seção da folha, com âncora para a trilha lateral. */
export function SpecSection(props: { id: string; titulo: string; children: JSX.Element }) {
  return (
    <section id={props.id} aria-labelledby={`${props.id}-titulo`} class="scroll-mt-4 space-y-3">
      <h2
        id={`${props.id}-titulo`}
        class="font-heading text-sm uppercase tracking-[0.18em] text-grimorio-gold"
      >
        {props.titulo}
      </h2>
      {props.children}
    </section>
  )
}

/** Um bloco de amostras com um título curto e a explicação do que ele prova. */
export function SpecBlock(props: { titulo: string; nota?: string; children: JSX.Element }) {
  return (
    <div class="space-y-2">
      <p class="font-heading text-2xs uppercase tracking-[0.16em] text-muted-foreground">
        {props.titulo}
      </p>
      {props.nota && <p class="max-w-prose text-xs text-muted-foreground">{props.nota}</p>}
      <div class="flex flex-wrap gap-3">{props.children}</div>
    </div>
  )
}

/**
 * Uma amostra de cor: o quadrado é pintado pelo utilitário REAL e a legenda é
 * o valor que o navegador resolveu para ele.
 */
export function ColorSwatch(props: { classe: string; token: string; nota?: string }) {
  const [cor, refCor] = createComputedValue('background-color')
  return (
    <figure class="w-40 space-y-1">
      <div
        ref={refCor}
        class={cn('h-12 w-full rounded-sm border border-grimorio-iron', props.classe)}
      />
      <figcaption class="space-y-0.5">
        <p class="font-mono text-2xs text-foreground">{props.token}</p>
        <p class="font-mono text-3xs text-muted-foreground">{cor()}</p>
        {props.nota && <p class="text-3xs text-muted-foreground">{props.nota}</p>}
      </figcaption>
    </figure>
  )
}

/**
 * Uma amostra de raio: o canto desenhado pela classe, e ao lado o pixel que
 * ela virou DENTRO desta cena — que é a informação que ninguém consegue prever
 * lendo o TSX, porque a cena redefine o `--radius` (ALE-173).
 */
export function RadiusSwatch(props: { classe: string; nome: string }) {
  const [raio, refRaio] = createComputedValue('border-radius')
  return (
    <figure class="w-28 space-y-1">
      <div
        ref={refRaio}
        class={cn(
          'h-14 w-full border border-grimorio-gold/60 bg-grimorio-panel-raised',
          props.classe,
        )}
      />
      <figcaption>
        <p class="font-mono text-2xs text-foreground">{props.nome}</p>
        <p class="font-mono text-3xs text-grimorio-gold">{raio()}</p>
      </figcaption>
    </figure>
  )
}
