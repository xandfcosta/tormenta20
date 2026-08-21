import { type JSX, Show, createSignal, onMount } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { SectionLabel, SectionTitle } from '@/shared/ui/section-label'

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

/**
 * Contraste de uma cor contra o painel da cena, medido de verdade.
 *
 * Passa pelo canvas de propósito: `getComputedStyle` devolve `oklch(...)` sem
 * converter, e ler aqueles três números como se fossem RGB dá razão de
 * contraste inventada — foi o que aconteceu na primeira versão desta medição, e
 * ela jurou 2,02 onde o valor é 8,86. Pintar um pixel e ler de volta é o único
 * jeito de sair do espaço de cor e chegar em sRGB.
 */
function contrasteNoPainel(cor: string): number | null {
  const tela = document.createElement('canvas')
  tela.width = 1
  tela.height = 1
  const ctx = tela.getContext('2d')
  const cena = document.querySelector('.scene-grimorio')
  if (!ctx || !cena) return null

  const paraRgb = (css: string): [number, number, number] => {
    ctx.clearRect(0, 0, 1, 1)
    ctx.fillStyle = css
    ctx.fillRect(0, 0, 1, 1)
    const [r, g, b] = ctx.getImageData(0, 0, 1, 1).data
    return [r ?? 0, g ?? 0, b ?? 0]
  }
  const luminancia = ([r, g, b]: [number, number, number]) => {
    const [lr, lg, lb] = [r, g, b].map((v) => {
      const c = v / 255
      return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
    })
    return 0.2126 * (lr ?? 0) + 0.7152 * (lg ?? 0) + 0.0722 * (lb ?? 0)
  }

  const painel = getComputedStyle(cena).getPropertyValue('--grimorio-panel').trim()
  const [claro, escuro] = [luminancia(paraRgb(cor)), luminancia(paraRgb(painel))].sort(
    (a, b) => b - a,
  )
  return Number((((claro ?? 0) + 0.05) / ((escuro ?? 0) + 0.05)).toFixed(2))
}

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
      <SectionTitle id={`${props.id}-titulo`} class="text-sm">
        {props.titulo}
      </SectionTitle>
      {props.children}
    </section>
  )
}

/** Um bloco de amostras com um título curto e a explicação do que ele prova. */
export function SpecBlock(props: { titulo: string; nota?: string; children: JSX.Element }) {
  return (
    <div class="space-y-2">
      <SectionLabel>{props.titulo}</SectionLabel>
      {props.nota && <p class="max-w-prose text-xs text-muted-foreground">{props.nota}</p>}
      <div class="flex flex-wrap gap-3">{props.children}</div>
    </div>
  )
}

/**
 * Uma amostra de cor: o quadrado é pintado pelo utilitário REAL e a legenda é
 * o valor que o navegador resolveu para ele.
 */
export function ColorSwatch(props: {
  classe: string
  token: string
  nota?: string
  /** Superfície não vira texto: para ela a razão contra o painel não diz nada. */
  superficie?: boolean
}) {
  const [cor, refCor] = createComputedValue('background-color')
  // O contraste contra o PAINEL, que é o fundo em que quase todo texto da cena
  // é lido. Ele responde o que o olho não responde sozinho: esta cor serve de
  // TEXTO ou só de bloco? Medido, `--hp-full` dá 4,66 e o `emerald-400` que a
  // ficha usa a centímetros dele dá 8,86 — dois verdes que parecem o mesmo
  // papel e não são (ALE-173, P3).
  const contraste = () => (props.superficie || cor() === '—' ? null : contrasteNoPainel(cor()))
  return (
    <figure class="w-40 space-y-1">
      <div
        ref={refCor}
        class={cn('h-12 w-full rounded-sm border border-grimorio-iron', props.classe)}
      />
      <figcaption class="space-y-0.5">
        <p class="font-mono text-2xs text-foreground">{props.token}</p>
        <p class="font-mono text-3xs text-muted-foreground">{cor()}</p>
        <Show when={contraste()}>
          {(razao) => (
            <p
              class={cn(
                'font-mono text-3xs',
                razao() >= 4.5 ? 'text-muted-foreground' : 'font-bold text-grimorio-gold',
              )}
            >
              {razao()}:1 {razao() >= 4.5 ? 'no painel' : '— só bloco, não texto'}
            </p>
          )}
        </Show>
        {props.nota && <p class="text-3xs text-muted-foreground">{props.nota}</p>}
      </figcaption>
    </figure>
  )
}

/**
 * Duas grafias do MESMO rótulo, lado a lado, com o valor que cada uma resolve.
 *
 * Existe para a decisão que não se toma lendo: quando duas partes do app
 * escrevem a mesma coisa de dois jeitos, a pergunta não é qual código está
 * certo — é qual das duas o olho prefere. A legenda traz o pixel e onde cada
 * uma vive hoje.
 */
export function ComparaRotulo(props: {
  texto: string
  a: { classe: string; nome: string; onde: string }
  b: { classe: string; nome: string; onde: string }
}) {
  const [espA, refA] = createComputedValue('letter-spacing')
  const [espB, refB] = createComputedValue('letter-spacing')
  return (
    <div class="w-full space-y-3">
      <div class="space-y-0.5">
        <p ref={refA} class={cn('font-heading uppercase text-grimorio-gold', props.a.classe)}>
          {props.texto}
        </p>
        <p class="font-mono text-3xs text-muted-foreground">
          {props.a.nome} → {espA()} · {props.a.onde}
        </p>
      </div>
      <div class="space-y-0.5">
        <p ref={refB} class={cn('font-heading uppercase text-grimorio-gold', props.b.classe)}>
          {props.texto}
        </p>
        <p class="font-mono text-3xs text-muted-foreground">
          {props.b.nome} → {espB()} · {props.b.onde}
        </p>
      </div>
    </div>
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
