import { expect, test } from '@playwright/test'

/**
 * TODA cena do piloto declara a gramática de teclado da casa.
 *
 * O defeito que isto prende foi contado antes de ser consertado: 6 das 20 cenas
 * declaravam alguma `data-nav-region` e 14 não declaravam nenhuma. O motor
 * estava no ar em todas, sem nada para dirigir. E ninguém viu porque o guarda
 * que prova a tese exercita as setas em `/` e não entra em mais nenhuma
 * tela — "um guarda só mede o que ele VISITA" (ALE-237, ALE-252).
 *
 * Este é o guarda que troca "visita o Hub" por "percorre a lista": cena nova
 * entra aqui e nasce medida. Continua sendo ENUMERAÇÃO de cenas, e isso é
 * honesto de dizer — o que restauraria amostragem de verdade seria a casca
 * declarar a região, e ela só pode declarar a que é dela (o `rail`). O miolo
 * cada cena tem de nomear, porque só ela sabe a própria forma.
 *
 * E2E porque `data-nav-region` sozinho não prova nada: o driver só liga em `≥xl`
 * com ponteiro fino, e a região tem de ter ÁREA (o `hasArea` do driver) — em
 * jsdom todo elemento mede zero e a região existiria no papel e não na tela.
 */
test.use({ storageState: '.auth/user.json' })

const CENAS = [
  { nome: 'hub', url: '/' },
  { nome: 'campanhas', url: '/campanhas' },
  { nome: 'personagens', url: '/personagens' },
  { nome: 'grimório', url: '/grimorio' },
  { nome: 'bestiário', url: '/mestre/bestiario' },
    // Os catálogos viraram NOVE cenas na ALE-264 — cada uma é uma parada do
  // trilho. Duas amostram as nove: elas passam pelos mesmos componentes, e a
  // que fugir disso é a que este guarda existe para pegar.
  { nome: 'condições', url: '/mestre/condicoes' },
  { nome: 'deuses', url: '/mestre/deuses' },
  { nome: 'encontros', url: '/mestre/encontros' },
  { nome: 'improviso', url: '/mestre/improviso' },
  { nome: 'admin', url: '/admin' },
  { nome: 'campanha nova', url: '/campanhas/nova' },
  { nome: 'forja', url: '/personagens/nova' },
]

// A segunda cena da forja NÃO cabe nesta lista: o endereço dela tem o id do
// herói, e o herói só existe depois de alguém forjar. Ela é medida em
// `piloto-forja.spec.ts`, que a alcança pelo caminho de verdade — a lista aqui é
// de cenas com endereço fixo, e fingir um id faria o guarda medir um 403.


for (const cena of CENAS) {
  test(`a cena de ${cena.nome} declara a gramática de teclado`, async ({ page }) => {
    await page.setViewportSize({ width: 1400, height: 900 })
    await page.goto(cena.url)
    await page.waitForLoadState('networkidle')

    const medida = await page.evaluate(() => {
      const comArea = (e: Element) => {
        const r = e.getBoundingClientRect()
        return r.width > 0 && r.height > 0
      }
      const regioes = [...document.querySelectorAll('[data-nav-region]')]
      return {
        desenhou: !!document.querySelector('[data-slot="scene-shell"]'),
        temVoltar: document.querySelector('[data-slot="scene-shell"]')?.hasAttribute('data-voltar'),
        regioes: regioes.filter(comArea).map((e) => ({
          nome: e.getAttribute('data-nav-region'),
          itens: e.querySelectorAll(
            'a[href],button:not([disabled]),[tabindex]:not([tabindex="-1"]),[data-nav-item]',
          ).length,
        })),
      }
    })

    // O CONTROLE: a cena desenhou. Sem ele "não declara região" seria verdade
    // sobre um 404 ou um redirecionamento, e a mensagem mandaria procurar no
    // arquivo errado.
    expect(medida.desenhou, `a cena de ${cena.nome} não desenhou`).toBe(true)

    expect(
      medida.regioes.length,
      `a cena de ${cena.nome} não declara nenhuma região: o driver de teclado carrega e não tem o que dirigir`,
    ).toBeGreaterThan(0)

    // Região SEM item é pior que região nenhuma: o driver tenta entrar e o foco
    // some. Cada uma tem de ter ao menos um alvo.
    for (const r of medida.regioes) {
      expect(r.itens, `a região "${r.nome}" de ${cena.nome} não tem item focável`).toBeGreaterThan(0)
    }
  })
}
