import { expect, test } from '@playwright/test'

/**
 * As listas virtualizadas da FORJA: os poderes de classe e a loja inicial.
 *
 * `VirtualList` mede as linhas para saber quais existem, e em jsdom todo
 * elemento mede zero — a lista não renderiza linha nenhuma e um teste de
 * componente passa verde sem olhar item algum. Tentei cobrir "poder travado por
 * nível" em vitest e bati exatamente nisso: a asserção não achava botão nenhum
 * porque nenhum existia (a regra ficou no unitário `class-powers.test.ts`).
 *
 * Anda o caminho de verdade porque a loja PRECISA da classe: é ela que define o
 * kit inicial. Não cria personagem — o rascunho fica no localStorage do
 * contexto, que morre com o teste, e a seed sai como entrou.
 */
test.describe('Forja — listas virtualizadas', () => {
  test('poderes e loja pintam linhas, e a busca troca o conjunto', async ({ page }) => {
    await page.goto('/characters/new/raca')
    // O card da raça é `role="option"` (padrão listbox), não botão.
    await page.getByRole('option', { name: /Humano/ }).first().click()

    // O losango do trilho é `aria-hidden`: o nome acessível é só o rótulo.
    await page.getByRole('button', { name: 'Classe', exact: true }).click()
    await page.getByRole('option', { name: /Guerreiro/ }).first().click()
    // Nível 3: no 1º não existe vaga de poder, e sem vaga a lista nem aparece.
    const nivel = page.getByRole('spinbutton', { name: 'Nível de Guerreiro' })
    await nivel.fill('3')
    await nivel.blur()

    // ── poderes: a lista virtualizada da classe ──
    await page.getByRole('button', { name: 'Poderes', exact: true }).click()
    // Ancora em algo VISÍVEL antes de contar: `count()` não espera, e contar
    // antes de a lista pintar dá zero sem nada estar quebrado.
    const busca = page.getByRole('searchbox', { name: 'Buscar poder' })
    await expect(busca).toBeVisible()
    await expect(page.getByRole('button', { name: /^Ataque Reflexo/ })).toBeVisible()

    await busca.fill('golpe')
    await expect(page.getByRole('button', { name: /Golpe/ }).first()).toBeVisible()
    // O outro modo de falha da virtualização: guardar o que pintou primeiro.
    await expect(page.getByRole('button', { name: /^Ataque Reflexo/ })).toHaveCount(0)

    // ── loja inicial: a outra lista virtualizada ──
    // Por URL: o trilho só habilita o passo depois dos anteriores, e o que
    // interessa aqui é a LISTA, não a ordem do assistente. O rascunho vive no
    // contexto, então a classe escolhida acima continua valendo.
    await page.goto('/characters/new/equipamento')
    // A loja só abre com dinheiro: sem tibares ela diz para rolar antes.
    await page.getByRole('spinbutton', { name: 'Tibares iniciais' }).fill('500')
    const buscaLoja = page.getByRole('searchbox', { name: 'Buscar item' })
    await expect(buscaLoja).toBeVisible()

    await buscaLoja.fill('adaga')
    await expect(page.getByText(/Adaga/).first()).toBeVisible()

    await buscaLoja.fill('zzzzzz')
    await expect(page.getByText(/^Adaga$/)).toHaveCount(0)
  })
})

/**
 * O catálogo fica com o palco enquanto NADA foi escolhido (ALE-171).
 *
 * O painel de detalhe reservava 871px de 1920 — 46% da tela — para dizer
 * "escolha uma raça para ver o que ela concede", enquanto os ladrilhos se
 * espremiam ao lado. É a mesma regra que a ALE-161 aplicou ao tabuleiro e a
 * ALE-171 à sessão: uma cena preenche o espaço que recebe.
 *
 * As DUAS metades são afirmadas. Só a primeira passaria verde com o painel
 * apagado, que é o conserto errado: escolhida a raça, é o detalhe dela que
 * precisa da largura — e o convite ENCOLHE mas não some, senão o passo deixa
 * de dizer o que vem depois do clique.
 *
 * Por que e2e: é largura de grade real respondendo a estado. Em jsdom todo
 * elemento mede zero e as duas medidas dariam iguais.
 */
test('na forja, o catálogo tem o palco até a primeira escolha', async ({ page }) => {
  await page.setViewportSize({ width: 1920, height: 1080 })
  await page.goto('/characters/new/raca')
  await expect(page.getByRole('option', { name: /Humano/ }).first()).toBeVisible()

  const colunas = () =>
    page.evaluate(() => {
      const grade = [...document.querySelectorAll<HTMLElement>('div')].find((n) =>
        (n.className || '').toString().includes('grid gap-4 lg:min-h-0'),
      )
      const cols = grade ? getComputedStyle(grade).gridTemplateColumns : ''
      return cols.split(' ').map((c) => Math.round(Number.parseFloat(c)))
    })

  const vazio = await colunas()
  expect(vazio.length, 'a grade da forja não tem duas colunas a 1920').toBe(2)
  expect(vazio[0], 'o catálogo não recebeu o palco com o painel vazio').toBeGreaterThan(vazio[1] * 2)
  // O convite continua na tela: ele é o que explica para que serve aquele lado.
  await expect(page.getByText(/Escolha uma raça para ver/)).toBeVisible()

  await page.getByRole('option', { name: /Humano/ }).first().click()
  await expect(page.getByText(/Escolha uma raça para ver/)).toBeHidden()

  const escolhido = await colunas()
  expect(
    escolhido[1],
    'o painel não retomou a largura com a raça escolhida',
  ).toBeGreaterThan(vazio[1])
})
