import { expect, test } from '@playwright/test'

/**
 * A ROLAGEM POR TECLADO nas cenas em Datastar.
 *
 * A casca do app é `h-dvh` com `overflow-hidden`, então o DOCUMENTO não rola —
 * quem rola são caixas aninhadas. Uma dessas caixas cujo conteúdo é só TEXTO não
 * tem nenhum descendente focável, e sem `tabindex` o foco nunca entra nela: seta,
 * PageDown, Home e End não fazem nada, e o conteúdo escondido fica inalcançável
 * sem mouse.
 *
 * Por que E2E, que nesta casa precisa se justificar: a pergunta é "isto
 * transborda?", e transbordo é LEIAUTE REAL. Em jsdom `scrollHeight` e
 * `clientHeight` são zero, então a condição nunca dispara e o guarda passaria
 * verde sobre todas as telas, sempre — a pior forma de teste, o que afirma o
 * oposto do que mede.
 *
 * AMOSTRAGEM e não enumeração: o guarda percorre as cenas e, dentro de cada uma,
 * TODA caixa que rola — não há uma asserção por caixa conhecida. Cena nova entra
 * na lista de baixo e já nasce medida; caixa nova dentro de uma cena listada já
 * é medida sem tocar no arquivo.
 */

// O estado de login é o do MESTRE: quatro das seis cenas são da Mesa do Mestre e
// as outras duas ele também alcança.
test.use({ storageState: '.auth/user.json' })

const CENAS = [
  { nome: 'catálogos do mestre', url: '/mestre/condicoes' },
  { nome: 'bestiário do mestre', url: '/mestre/bestiario' },
  { nome: 'encontros', url: '/mestre/encontros' },
  { nome: 'improviso', url: '/mestre/improviso' },
  { nome: 'grimório', url: '/grimorio' },
  { nome: 'personagens', url: '/personagens' },
]

for (const scene of CENAS) {
  test(`o teclado alcança tudo que rola na cena de ${scene.nome}`, async ({ page }) => {
    await page.setViewportSize({ width: 1200, height: 700 })
    await page.goto(scene.url)
    await page.waitForLoadState('networkidle')

    const measure = await page.evaluate(() => {
      const scrolls = (e: Element) => {
        const cs = getComputedStyle(e)
        return /auto|scroll/.test(cs.overflowY) && e.scrollHeight > e.clientHeight + 4
      }
      const FOCUSABLE = 'a[href],button,input,select,textarea,[tabindex]:not([tabindex="-1"])'
      return {
        // O CONTROLE: a cena desenhou. Sem ele, "nenhuma caixa presa" seria
        // verdade também sobre uma página que não carregou ou deu 404 — que é
        // exatamente como esta família de guarda passa verde sobre nada.
        desenhou: document.querySelectorAll('[data-slot="scene-shell"]').length > 0,
        documentoRola:
          document.documentElement.scrollHeight > document.documentElement.clientHeight,
        presas: [...document.querySelectorAll('*')]
          .filter(scrolls)
          .filter((e) => (e as HTMLElement).tabIndex < 0)
          .filter((e) => e.querySelectorAll(FOCUSABLE).length === 0)
          .map((e) => ({
            classe: String((e as HTMLElement).className).slice(0, 60),
            escondido: e.scrollHeight - e.clientHeight,
          })),
      }
    })

    expect(measure.desenhou, `a cena de ${scene.nome} não desenhou`).toBe(true)

    // Se o documento inteiro rola, o teclado já alcança tudo pelo corpo — e a
    // regra abaixo não se aplica. Hoje nenhuma cena do app é assim, mas
    // afirmar a condição em vez de assumi-la é o que impede este guarda de
    // acusar uma cena que mudou de casca por um motivo legítimo.
    if (measure.documentoRola) return

    expect(
      measure.presas,
      `caixas que rolam sem foco nem descendente focável em ${scene.nome}: ` +
        `o teclado não alcança o conteúdo escondido (${measure.presas
          .map((p) => `${p.escondido}px em .${p.classe}`)
          .join('; ')})`,
    ).toEqual([])
  })
}
