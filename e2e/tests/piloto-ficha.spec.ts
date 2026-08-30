import { expect, test } from '@playwright/test'
import { expectDentroDaJanela, expectNadaRolaDeLado } from './support/geometry'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

/**
 * A FICHA em Datastar (ALE-272, fatia 1) — a casca, as abas e o crachá.
 *
 * O que o servidor escreve está preso em Go, que é mais barato: o endereço das
 * abas, a posse, o degrau de nível e a faixa dos vitais. O que sobra para cá é o
 * que só o navegador mede — LEIAUTE REAL nos seis formatos.
 *
 * E leiaute aqui não é enfeite: o crachá é uma fileira com retrato, identidade,
 * nível e oito botões de vital, e a barra de abas tem sete itens. As duas são
 * exatamente a forma que já transbordou nesta casa (ALE-162, ALE-178) — um
 * controle que sai da janela não é feio, é inalcançável.
 */
test.use({ storageState: '.auth/user.json' })

/** O primeiro herói do elenco, pelo endereço da ficha nova. */
async function aFichaDoPrimeiro(page: import('@playwright/test').Page) {
  await page.goto('/piloto/personagens')
  const href = await page.locator('a[aria-label^="Abrir ficha de"]').first().getAttribute('href')
  const id = href?.match(/\d+/)?.[0]
  expect(id, 'não achei um herói no elenco: o resto do caso não mediria nada').toBeTruthy()
  await page.goto(`/piloto/personagens/${id}`)
  return id as string
}

test('a ficha cabe nos seis formatos, e nenhum botão do crachá sai da janela', async ({ page }) => {
  await aFichaDoPrimeiro(page)
  await expect(page.getByRole('navigation', { name: 'Seções da ficha' })).toBeVisible()

  await expectNoHorizontalOverflow(page, VIEWPORTS)

  // O CRACHÁ é o que mais aperta: oito passos de vital, o degrau de nível e o
  // retrato na mesma fileira. No telefone em pé é onde a conta estoura.
  await page.setViewportSize({ width: 390, height: 844 })
  await expectDentroDaJanela(page)
})

/**
 * O MESMO OLHAR, EM TODA ABA — e ele caminha pela barra em vez de ter uma lista.
 *
 * A primeira versão deste arquivo media só a aba padrão, que na fatia 1 era um
 * aviso de duas linhas. Quando o painel de Proficiências chegou (fatia 2), o
 * guarda continuou verde sem nunca ter aberto o painel novo: é exatamente a
 * forma da ALE-237 e da ALE-252, onde a cobertura é função de onde o teste
 * NAVEGA e não de quantas asserções ele tem.
 *
 * Ele lê os `href` da barra em vez de trazer uma lista escrita, e essa é a
 * diferença entre AMOSTRAGEM e ENUMERAÇÃO: o painel da fatia 6 vai ser medido
 * sem ninguém lembrar de vir aqui. Uma lista escrita à mão nasce incompleta na
 * primeira vez que alguém esquece.
 */
test('nenhum painel da ficha transborda o telefone', async ({ page }) => {
  await aFichaDoPrimeiro(page)
  const enderecos = await page
    .getByRole('navigation', { name: 'Seções da ficha' })
    .getByRole('link')
    .evaluateAll((links) => links.map((l) => (l as HTMLAnchorElement).href))
  expect(enderecos, 'a barra de abas veio vazia: este caso não mediria nada').toHaveLength(7)

  await page.setViewportSize({ width: 390, height: 844 })
  for (const endereco of enderecos) {
    await page.goto(endereco)
    await expect(page.getByRole('navigation', { name: 'Seções da ficha' })).toBeVisible()
    await expectDentroDaJanela(page)
    // AS DUAS, e a segunda não é redundância — foi medida.
    //
    // O `expectDentroDaJanela` ignora, DE PROPÓSITO, quem tem um eixo rolável
    // acima: pela definição dele, há como chegar lá. O painel da ficha rola na
    // vertical, e `overflow-y: auto` faz o navegador computar o `overflow-x`
    // como `auto` junto — então um bloco de 500px numa janela de 390 passa
    // por ele em silêncio. Provei sabotando: o caso ficou VERDE com o bloco
    // largo no ar, e só o `expectNadaRolaDeLado` o viu. É a mesma lacuna que
    // a ALE-178 nomeou.
    await expectNadaRolaDeLado(page)
  }
})

test('as sete abas são endereços, e a ativa se anuncia', async ({ page }) => {
  const id = await aFichaDoPrimeiro(page)

  const abas = page.getByRole('navigation', { name: 'Seções da ficha' }).getByRole('link')
  await expect(abas).toHaveCount(7)

  // A ABA É UM ENDEREÇO: recarregar tem de cair na mesma seção. É o contrato que
  // a SPA tinha e que um sinal de cliente não daria — e é por isso que elas são
  // links e não botões.
  await page.goto(`/piloto/personagens/${id}?tab=spells`)
  await expect(page.getByRole('link', { name: 'Magias', exact: true })).toHaveAttribute('aria-current', 'page')
  await page.reload()
  await expect(page.getByRole('link', { name: 'Magias', exact: true })).toHaveAttribute('aria-current', 'page')

  // E o endereço ANTIGO da Mochila continua chegando nela: `inventory` é
  // favorito de quando a aba se chamava assim.
  await page.goto(`/piloto/personagens/${id}?tab=inventory`)
  await expect(page.getByRole('link', { name: 'Mochila', exact: true })).toHaveAttribute('aria-current', 'page')
})
