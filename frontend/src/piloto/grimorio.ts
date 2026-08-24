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

import { piscarVital, pulsarVez } from '@/features/session-tracker/turn-juice'

/** Onde a legenda vai buscar o que medir: a amostra da mesma figura. */
function amostraDe(legenda: Element): HTMLElement | null {
  return legenda.closest('figure, [data-par]')?.querySelector<HTMLElement>('[data-amostra]') ?? null
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

/** Preenche cada legenda com a propriedade computada da amostra ao lado. */
function medeAsPropriedades(): void {
  for (const legenda of document.querySelectorAll<HTMLElement>('[data-medir]')) {
    const amostra = amostraDe(legenda)
    const propriedade = legenda.dataset.medir
    if (!amostra || !propriedade) continue
    legenda.textContent = getComputedStyle(amostra).getPropertyValue(propriedade).trim() || '—'
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
  for (const legenda of document.querySelectorAll<HTMLElement>('[data-contraste]')) {
    const amostra = amostraDe(legenda)
    if (!amostra) continue
    const razao = contrasteNoPainel(getComputedStyle(amostra).backgroundColor)
    if (razao === null) continue
    const serveDeTexto = razao >= 4.5
    legenda.textContent = serveDeTexto ? `${razao}:1 no painel` : `${razao}:1 — só bloco, não texto`
    // A cor do aviso é do TEMA e não inventada aqui: o dourado é o que a casa
    // usa para "olhe para isto".
    legenda.classList.toggle('text-grimorio-gold', !serveDeTexto)
    legenda.classList.toggle('font-bold', !serveDeTexto)
    legenda.classList.toggle('text-muted-foreground', serveDeTexto)
  }
}

/**
 * A medida de uma CÉLULA da comparação: altura, largura e raio da peça que ela
 * contém.
 *
 * Ela existe porque duas peças parecidas passam por iguais aos olhos — e foi
 * assim que a primeira versão da seção de Peças deixou passar 12px de diferença
 * no botão `xs`. O número é o que separa "parece igual" de "é igual".
 *
 * Ela espera pelo FATO e não pelo relógio. A coluna da SPA é montada por
 * elementos customizados, e a primeira versão disto usava dois
 * `requestAnimationFrame` — que é chute: mediu antes de os elementos existirem
 * e escreveu a coluna inteira vazia. `customElements.whenDefined` resolve
 * quando o elemento REALMENTE está registrado, e o `requestAnimationFrame`
 * depois dele é só para o layout assentar.
 *
 * Genérica de propósito: ela descobre as tags a esperar olhando o documento
 * (nome com hífen é elemento customizado), então esta ilha não precisa saber
 * que existe uma segunda chamada `spa-alguma-coisa`.
 */
async function medeAsCelulas(): Promise<void> {
  const customizadas = new Set(
    [...document.querySelectorAll('[data-amostra-cela] *')]
      .map((e) => e.tagName.toLowerCase())
      .filter((tag) => tag.includes('-')),
  )
  await Promise.all([...customizadas].map((tag) => customElements.whenDefined(tag)))
  await new Promise((pronto) => requestAnimationFrame(() => pronto(null)))
  escreveAsMedidas()
}

function escreveAsMedidas(): void {
  for (const legenda of document.querySelectorAll<HTMLElement>('[data-medir-cela]')) {
    const cela = legenda.previousElementSibling
    const peca = cela?.querySelector<HTMLElement>('button, input, [role="progressbar"]')
    if (!peca) continue
    const caixa = peca.getBoundingClientRect()
    const raio = getComputedStyle(peca).borderRadius
    legenda.textContent = `h ${Math.round(caixa.height)} · w ${Math.round(caixa.width)} · r ${raio}`
  }
}

/**
 * Uma passada só, no carregamento. Não há observador nem repetição de propósito:
 * a folha é estática depois de desenhada, e um `MutationObserver` aqui seria
 * maquinaria para um caso que não existe.
 */
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
  const linha = document.querySelector<HTMLElement>('[data-linha-iniciativa]')
  const parado = window.matchMedia('(prefers-reduced-motion: reduce)')

  const mostraOGate = () => {
    const rotulo = document.querySelector<HTMLElement>('[data-movimento-reduzido]')
    if (rotulo) rotulo.textContent = parado.matches ? 'LIGADO' : 'desligado'
  }
  mostraOGate()
  parado.addEventListener('change', mostraOGate)

  for (const botao of document.querySelectorAll<HTMLElement>('[data-disparar]')) {
    botao.addEventListener('click', () => {
      // A guarda é aqui e não dentro de cada animação, igual à sessão: um
      // ponto só decide, e é ele que a folha demonstra.
      if (parado.matches || !linha) return
      const qual = botao.dataset.disparar
      if (qual === 'ferir') piscarVital(linha, { curou: false })
      if (qual === 'curar') piscarVital(linha, { curou: true })
      if (qual === 'vez') pulsarVez(linha)
    })
  }
}

export function medeAFolha(): void {
  medeAsPropriedades()
  medeOsContrastes()
  // As células esperam os elementos customizados existirem — ver `medeAsCelulas`.
  void medeAsCelulas()
  ligaOsDisparos()
}

medeAFolha()
