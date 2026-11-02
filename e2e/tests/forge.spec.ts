import { expect, test } from '@playwright/test'
import { medeOContraste } from './support/contrast'
import { expectOneFocusRing } from './support/focus'
import { measureDsText, measureDsTouchTargets } from './support/ds-floors'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

/**
 * A FORJA — a folha em branco e os atributos.
 *
 * Dois casos, e os dois se justificam com mecanismo que só um navegador tem.
 *
 * O primeiro é o REDESENHO PELO FORMULÁRIO. O `@post` do Datastar manda o
 * `<form>` inteiro (`contentType: 'form'`, achado pelo `selector`) e o servidor
 * devolve a folha com o equipamento da classe escolhida. As escolhas moram em
 * folhas FORA do formulário, e os rádios se ligam a ele pelo atributo `form`: o
 * teste de handler prova o HTML, e só o navegador prova que o `FormData` leva
 * esses rádios e que o remendo chega.
 *
 * O segundo é o de sempre nesta casa: CONTRASTE e LEIAUTE reais — uma cena nova
 * que não entra numa lista de visitas nasce sem medição, em silêncio, que é a
 * marca desta família.
 *
 * O que NÃO está aqui, de propósito: recusa de escolha que o kit não oferece,
 * nascimento com o kit de p140, e a escolha livre dos atributos. Isso é regra de
 * SERVIDOR e está preso em `api/forge_test.go`, que é a camada mais barata que
 * a segura.
 */
// COMO CELULAR (`hasTouch`), que é como o piso de toque se mede: com ponteiro
// fino o kit encolhe os controles, e o piso de 44px não vale.
test.use({ storageState: '.auth/user.json', hasTouch: true, isMobile: true })

/** Os pisos do Tempestade (12px de texto, 44px de alvo), com o denominador. */
async function expectDsFloors(page: import('@playwright/test').Page, where: string) {
  for (const measure of [measureDsText, measureDsTouchTargets]) {
    const report = await measure(page)
    expect(report.medidos, `${where}: o medidor não mediu nada`).toBeGreaterThan(0)
    expect(report.falhas, `pisos do Tempestade ${where}`).toEqual([])
  }
}

/**
 * Abre a folha da escolha pela linha e toca a opção. A escolha fecha a folha e
 * redesenha a forja, e a linha passa a dizer o que foi escolhido.
 *
 * Pelo RÓTULO e não pelo `value` do rádio: a opção submete o slug (`humano`) e
 * mostra o nome (`Humano`), e um spec que casa pelo `value` mede o
 * identificador, que é a metade que uma pessoa nunca vê.
 */
async function escolhe(page: import('@playwright/test').Page, group: string, rotulo: string) {
  await page.locator(`#forge-pick-${group}`).click()
  const sheet = page.locator('dialog[open]')
  await expect(sheet).toBeVisible()
  await sheet
    .locator(`label:has(input[name="${group}"])`)
    .filter({ has: page.getByText(rotulo, { exact: true }) })
    .click()
  // A classe não tem linha que a diga: ela entra na lista, com o nível no formulário.
  if (group === 'class') {
    await expect(page.locator(`input[name="classes"][value^="${rotulo}:"]`)).toHaveCount(1)
    return
  }
  await expect(page.locator(`#forge-pick-${group}`)).toContainText(rotulo)
}

test('o equipamento aparece e segue a classe, redesenhado pelo servidor', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/personagens/nova')
  await expect(page.getByRole('heading', { name: 'Forjar um herói' })).toBeVisible()

  // O CONTROLE do caso: antes da classe não há kit para oferecer, e é essa
  // ausência que o clique tem de desfazer. Sem ela, um seletor de arma que já
  // estivesse na página passaria como se o remendo tivesse chegado.
  await expect(page.locator('#forge-pick-weaponSimple')).toHaveCount(0)

  await escolhe(page, 'class', 'Guerreiro')
  // Pelo `id` e não por `getByLabel`: o nome acessível de uma opção de raça
  // carrega as habilidades dela, e "Armadura de Allihanna" (dahllan) casa com
  // um rótulo de "Armadura" — três elementos, e o modo estrito reprova.
  await expect(page.locator('#forge-pick-weaponMartial')).toBeVisible()
  await expect(page.locator('#forge-armor')).toContainText('Brunea')

  // A BUSCA da folha é do navegador (`data-show` sobre o texto dobrado em
  // `data-find`): com "LONGA" maiúsculo, só a espada longa fica.
  await page.locator('#forge-pick-weaponMartial').click()
  const sheet = page.locator('dialog[open]')
  await sheet.getByRole('searchbox').fill('LONGA')
  await expect(sheet.getByText('Espada longa', { exact: true })).toBeVisible()
  await expect(sheet.getByText('Alabarda', { exact: true })).toBeHidden()
  await sheet.getByRole('button', { name: 'Fechar' }).click()
  await expect(page.locator('dialog[open]')).toHaveCount(0)

  // O nome digitado ANTES do remendo sobrevive a ele: quem carrega o estado é o
  // formulário, e o servidor o devolve preenchido.
  await page.getByLabel('Nome').fill('Thessa de Valkaria')
  await escolhe(page, 'class', 'Arcanista')
  await page.getByRole('button', { name: 'Tirar Guerreiro' }).click()
  await expect(page.getByText('Arcanistas começam sem armadura')).toBeVisible()
  await expect(page.locator('#forge-pick-weaponMartial')).toHaveCount(0)
  await expect(page.getByLabel('Nome')).toHaveValue('Thessa de Valkaria')

  const ratio = await medeOContraste(page)
  expect(ratio.medidos, 'o medidor não achou texto: a folha não carregou').toBeGreaterThan(100)
  expect(ratio.falhas, 'texto abaixo do AA na folha da forja').toEqual([])
  await expectDsFloors(page, 'na folha da forja')
  await expectOneFocusRing(page, 'na folha da forja', 5)
  await expectNoHorizontalOverflow(page, VIEWPORTS)

  // O ANEL DE FOCO das OPÇÕES é medido com a folha ABERTA: o rádio fica
  // invisível e quem acende é o rótulo (`.ds-pick:has(:focus-visible)`), que a
  // regra global não alcança. Fechada, a folha não tem nada focável.
  await page.locator('#forge-pick-race').click()
  await expect(page.locator('dialog[open]')).toBeVisible()
  await expectOneFocusRing(page, 'na folha de raça da forja', 10)
})

// O mecanismo só existe no navegador: o rádio marcado sai da folha, e um remendo que o reaproveitasse
// para a carta seguinte manteria o `checked` da PROPRIEDADE, que o HTML servido não mostra.
test('a classe que entra não traz outra junto no gesto seguinte', async ({ page }) => {
  await page.goto('/personagens/nova')
  await escolhe(page, 'class', 'Guerreiro')
  await escolhe(page, 'class', 'Arcanista')
  await page.getByRole('button', { name: 'Aumentar Guerreiro' }).click()
  const classes = page.locator('input[name="classes"]')
  await expect(page.locator('input[name="classes"][value="Guerreiro:2"]')).toHaveCount(1)
  await expect.poll(() => classes.evaluateAll((all) => all.map((el) => (el as HTMLInputElement).value))).toEqual([
    'Guerreiro:2',
    'Arcanista:1',
  ])
  await expect(page.getByText('Nível 3', { exact: true })).toBeVisible()
})

test('a distribuição de atributos anda pelo servidor', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/personagens/nova')
  await page.getByLabel('Nome').fill('Herói do Guarda da Forja')
  await escolhe(page, 'race', 'Humano')
  await escolhe(page, 'class', 'Guerreiro')
  await escolhe(page, 'origin', 'Acólito')
  await escolhe(page, 'weaponSimple', 'Adaga')
  await escolhe(page, 'weaponMartial', 'Espada longa')
  await escolhe(page, 'armor', 'Couro batido')
  await page.getByRole('button', { name: 'Forjar' }).click()

  await page.waitForURL(/\/atributos$/)
  await expect(page.getByText('10 de 10 pontos')).toBeVisible()

  await page.getByRole('button', { name: 'Aumentar Força' }).click()
  await expect(page.getByText('9 de 10 pontos')).toBeVisible()

  const contrast = await medeOContraste(page)
  expect(contrast.medidos, 'o medidor não achou texto: a cena não carregou').toBeGreaterThan(20)
  expect(contrast.falhas, 'texto abaixo do AA nos atributos da forja').toEqual([])
  await expectDsFloors(page, 'nos atributos da forja')
  await expectNoHorizontalOverflow(page, VIEWPORTS)

  // A GRAMÁTICA DE TECLADO, medida aqui e não na lista do
  // `keyboard-grammar.spec.ts`: aquele guarda enumera cenas de
  // endereço FIXO, e o desta tem o id de um herói que só existe depois de
  // alguém forjar. Sem esta asserção, a única cena do app com endereço
  // dinâmico nasceria fora do regime — que é a forma exata do defeito que
  // aquele guarda existe para prender.
  await page.setViewportSize({ width: 1400, height: 900 })
  const region = page.locator('[data-nav-region="content"]')
  await expect(region).toHaveCount(1)
  expect(
    await region.locator('button:not([disabled]),a[href]').count(),
    'a região da cena não tem item nenhum para as setas dirigirem',
  ).toBeGreaterThan(0)
})
