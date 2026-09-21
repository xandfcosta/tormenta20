/**
 * A ILHA QUE MEDE a folha de especificação (ALE-251).
 *
 * A regra fundadora do grimório é: **ele lê o valor do navegador, nunca o
 * transcreve.** Uma folha que crava "2px" ao lado do quadrado apodrece no dia
 * em que o token muda, e apodrece CALADA — o número continua ali, bonito e
 * errado, e quem consulta acredita nele.
 *
 * Isso é `getComputedStyle`, que só existe no navegador. Então esta é a
 * primeira ilha de JS da migração que é NECESSÁRIA e não conveniente: sem ela a
 * página vira exatamente o que ela existe para combater.
 *
 * O contrato com o templ é um só: dentro de uma `<figure>`, o nó desenhado leva
 * `data-amostra` e cada legenda leva `data-medir="<propriedade>"`. A ilha
 * preenche as legendas com o que o navegador resolveu para a amostra ao lado.
 */

import { piscarVital, pulsarVez } from '@/lib/turn-juice'

/** Onde a legenda vai buscar o que medir: a amostra da mesma figura. */
function amostraDe(caption: Element): HTMLElement | null {
  return caption.closest('figure, [data-par]')?.querySelector<HTMLElement>('[data-amostra]') ?? null
}

/**
 * Contraste de uma cor contra o painel da cena, medido de VERDADE.
 *
 * Passa pelo canvas de propósito, e isto é lição cara: `getComputedStyle`
 * devolve `oklch(...)` sem converter, e ler aqueles três números como se fossem
 * RGB dá razão inventada — a primeira versão desta medição jurou 2,02 onde o
 * valor é 8,86. Pintar um pixel e ler de volta é o único jeito de sair do
 * espaço de cor e chegar em sRGB.
 */
function contrasteNoPainel(color: string): number | null {
  const page = document.createElement('canvas')
  page.width = 1
  page.height = 1
  const ctx = page.getContext('2d')
  const scene = document.querySelector('.scene-grimorio')
  if (!ctx || !scene) return null

  const toRgb = (css: string): [number, number, number] => {
    ctx.clearRect(0, 0, 1, 1)
    ctx.fillStyle = css
    ctx.fillRect(0, 0, 1, 1)
    const [r, g, b] = ctx.getImageData(0, 0, 1, 1).data
    return [r ?? 0, g ?? 0, b ?? 0]
  }
  const luminance = ([r, g, b]: [number, number, number]) => {
    const [lr, lg, lb] = [r, g, b].map((v) => {
      const c = v / 255
      return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
    })
    return 0.2126 * (lr ?? 0) + 0.7152 * (lg ?? 0) + 0.0722 * (lb ?? 0)
  }

  const panel = getComputedStyle(scene).getPropertyValue('--grimorio-panel').trim()
  const [light, dark] = [luminance(toRgb(color)), luminance(toRgb(panel))].sort(
    (a, b) => b - a,
  )
  return Number((((light ?? 0) + 0.05) / ((dark ?? 0) + 0.05)).toFixed(2))
}

/** Preenche cada legenda com a propriedade computada da amostra ao lado. */
function medeAsPropriedades(): void {
  for (const caption of document.querySelectorAll<HTMLElement>('[data-medir]')) {
    const sample = amostraDe(caption)
    const property = caption.dataset.medir
    if (!sample || !property) continue
    caption.textContent = getComputedStyle(sample).getPropertyValue(property).trim() || '—'
  }
}

/**
 * Escreve a razão de contraste, e a FRASE que ela significa.
 *
 * O número sozinho não responde a pergunta que a folha existe para responder —
 * "esta cor serve de TEXTO ou só de bloco?". Medido, o `--hp-full` dá 4,66 e o
 * `emerald-400` que a ficha usa a centímetros dele dá 8,86: dois verdes que
 * parecem o mesmo papel e não são (ALE-173, P3).
 */
function medeOsContrastes(): void {
  for (const caption of document.querySelectorAll<HTMLElement>('[data-contraste]')) {
    const sample = amostraDe(caption)
    if (!sample) continue
    const ratio = contrasteNoPainel(getComputedStyle(sample).backgroundColor)
    if (ratio === null) continue
    const servesAsText = ratio >= 4.5
    caption.textContent = servesAsText ? `${ratio}:1 no painel` : `${ratio}:1 — só bloco, não texto`
    // A cor do aviso é do TEMA e não inventada aqui: o dourado é o que a casa
    // usa para "olhe para isto".
    caption.classList.toggle('text-grimorio-gold', !servesAsText)
    caption.classList.toggle('font-bold', !servesAsText)
    caption.classList.toggle('text-muted-foreground', servesAsText)
  }
}

/**
 * A medida de uma CÉLULA da seção de Peças: altura, largura e raio da peça que
 * ela contém.
 *
 * Ela existe porque duas peças parecidas passam por iguais aos olhos — e foi
 * assim que a primeira versão da seção de Peças deixou passar 12px de diferença
 * no botão `xs`. O número é o que separa "parece igual" de "é igual".
 *
 * # A ESPERA encolheu para um quadro (ALE-314)
 *
 * Aqui havia uma espera por `customElements.whenDefined`, e ela era o conserto
 * de um defeito real: a coluna da SPA era montada por elementos customizados, e
 * a versão anterior usava dois `requestAnimationFrame` — que é chute. Mediu
 * antes de os elementos existirem e escreveu a coluna inteira VAZIA.
 *
 * Com o Solid fora, não há elemento customizado nenhum nesta folha: tudo é
 * `templ` que chega pronto no HTML. O que sobra a esperar é o LAYOUT assentar,
 * que é um quadro. Uma passada só, no carregamento, e sem observador de
 * propósito — a folha é estática depois de desenhada.
 */
async function medeAsCelulas(): Promise<void> {
  await new Promise((pronto) => requestAnimationFrame(() => pronto(null)))
  for (const caption of document.querySelectorAll<HTMLElement>('[data-medir-cela]')) {
    const cell = caption.previousElementSibling
    const token = cell?.querySelector<HTMLElement>('button, input, [role="progressbar"]')
    if (!token) continue
    const box = token.getBoundingClientRect()
    const radius = getComputedStyle(token).borderRadius
    caption.textContent = `h ${Math.round(box.height)} · w ${Math.round(box.width)} · r ${radius}`
  }
}

/**
 * A seção de MOVIMENTO: os mesmos disparos que a sessão usa, importados do
 * MESMO módulo (`turn-juice`). Uma cópia com os mesmos keyframes mentiria no
 * primeiro dia em que alguém mexesse no original — é a regra nº 2 da folha
 * aplicada a animação em vez de a componente.
 *
 * O gate de `prefers-reduced-motion` aparece NA TELA de propósito: a preferência
 * não cobre WAAPI, então todo `el.animate` da casa tem de perguntar por conta
 * própria — e aqui dá para ver se ele está perguntando, ligando a preferência
 * no sistema e clicando de novo.
 */
function ligaOsDisparos(): void {
  const row = document.querySelector<HTMLElement>('[data-linha-iniciativa]')
  const still = window.matchMedia('(prefers-reduced-motion: reduce)')

  const showsGate = () => {
    const label = document.querySelector<HTMLElement>('[data-movimento-reduzido]')
    if (label) label.textContent = still.matches ? 'LIGADO' : 'desligado'
  }
  showsGate()
  still.addEventListener('change', showsGate)

  for (const button of document.querySelectorAll<HTMLElement>('[data-disparar]')) {
    button.addEventListener('click', () => {
      // A guarda é aqui e não dentro de cada animação, igual à sessão: um
      // ponto só decide, e é ele que a folha demonstra.
      if (still.matches || !row) return
      const which = button.dataset.disparar
      if (which === 'ferir') piscarVital(row, { curou: false })
      if (which === 'curar') piscarVital(row, { curou: true })
      if (which === 'vez') pulsarVez(row)
    })
  }
}

export function medeAFolha(): void {
  medeAsPropriedades()
  medeOsContrastes()
  // As células esperam o layout assentar — ver `medeAsCelulas`.
  void medeAsCelulas()
  ligaOsDisparos()
}

medeAFolha()
