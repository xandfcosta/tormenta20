import { expect, test } from '@playwright/test'
import { touchTargets } from './support/touch-targets'

/**
 * O PISO DE TOQUE DA FICHA NO TELEFONE — WCAG 2.5.8, AA.
 *
 * E2E porque a pergunta é sobre GEOMETRIA RENDERIZADA: quanto um alvo mede
 * depois de o texto quebrar, e a que distância ele ficou do vizinho. Em jsdom
 * todo elemento mede zero e a resposta seria "conforme" para qualquer código —
 * que é a pior forma de verde.
 *
 * **Contar `< 24px` NÃO é medir a norma**: isso responde "quantos são pequenos"
 * quando a pergunta é "quantos reprovam", e infla o resultado em ordens de
 * grandeza. A norma tem duas exceções — espaçamento e equivalente —, e quem as
 * mede é o `support/touch-targets.ts`.
 *
 * Quem impede a receita de crachá de ser reescrita à mão é o
 * `TestNoHandwrittenBadgeRecipe`, em Go, que é mais barato. **Este guarda
 * responde a outra pergunta**: que o piso de fato DESENHA. Um `min-h-6` que não
 * chegasse à folha compilada passaria naquele e reprovaria neste.
 */
test.use({ storageState: '.auth/user.json' })

const ABAS = [
  'expertises',
  'combat',
  'bag',
  'proficiencies',
  'conditionals',
  'abilities',
  'spells',
] as const

// DOIS heróis, e não um: a ficha RAMIFICA pelo dado. Metade do painel de
// Combate só existe para quem conjura, e foram as abas do arcanista — Mochila e
// Magias — que continham as reprovações. Medir só o guerreiro dá verde sobre a
// tela que está errada.
const HEROES = [
  { id: 1, quem: 'o guerreiro' },
  { id: 3, quem: 'quem conjura' },
]

/**
 * O CONTROLE DO CANAL, e ele não é cerimônia.
 *
 * "Nenhum alvo reprovou" e "o seletor não casou com nada" se parecem no
 * terminal. Este caso põe na página dois botões de 20×20 encostados — que a
 * norma reprova sem discussão, porque os círculos de 24px deles se cruzam — e
 * exige que o medidor os ache. Sem ele, todo o verde abaixo seria sobre nada.
 */
test('o medidor acusa um alvo que reprova de verdade', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/personagens/1?tab=expertises')
  await expect(page.getByRole('heading', { name: 'Perícias' }).first()).toBeVisible()

  await page.evaluate(() => {
    const crate = document.createElement('div')
    crate.style.cssText = 'position:fixed;top:400px;left:8px;z-index:9999'
    crate.innerHTML =
      '<button style="width:20px;height:20px">a</button><button style="width:20px;height:20px">b</button>'
    document.body.append(crate)
  })

  const { alvos: targets } = await touchTargets(page)
  const planted = targets.filter((a) => a.nome === 'a' || a.nome === 'b')
  expect(planted.map((a) => a.reprova), 'o medidor não achou dois alvos 20×20 colados').toEqual([
    true,
    true,
  ])
})

// UM caso por herói, e não um por aba: são catorze navegações contra duas, e
// e2e é a faixa mais cara do repositório. A aba ofensora entra na MENSAGEM, que
// é o que a divisão em catorze casos comprava.
for (const hero of HEROES) {
  test(`a ficha de ${hero.quem} cumpre o piso de toque a 390px, nas sete abas`, async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    const rejects: string[] = []

    for (const aba of ABAS) {
      await page.goto(`/personagens/${hero.id}?tab=${aba}`)
      // A ESPERA É PELA ABA PEDIDA, e não por `networkidle`: um parâmetro com o
      // nome errado desenha a primeira aba em SILÊNCIO, e sete medições da mesma
      // página passam com cara de sete abas. O `networkidle` também não serve na
      // Mesa, onde o SSE nunca fecha.
      await expect(page.locator('[aria-current="page"]')).toBeVisible()

      const { medidos: measured, alvos: targets } = await touchTargets(page)

      // O DENOMINADOR: sem alvo medido, a asserção final é verde sobre uma
      // página que não carregou — uma cena fora de cena devolve `medidos=4` e
      // passa.
      expect(measured, `a aba ${aba} mediu ${measured} alvos: ela não desenhou`).toBeGreaterThan(15)

      for (const a of targets.filter((x) => x.reprova)) {
        rejects.push(`${aba}: "${a.nome}" ${a.larg}x${a.alt} — ${a.familia}`)
      }
    }

    expect(
      rejects,
      'alvos abaixo de 24px sem a folga que a exceção de espaçamento pede, e sem equivalente que passe',
    ).toEqual([])
  })
}
