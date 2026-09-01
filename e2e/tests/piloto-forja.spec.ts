import { expect, test } from '@playwright/test'
import { medeOContraste } from './support/contraste'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

/**
 * A FORJA em Datastar (ALE-272, fatia 9) — a folha em branco e os atributos.
 *
 * Dois casos, e os dois se justificam com mecanismo que só um navegador tem.
 *
 * O primeiro é o REDESENHO PELO FORMULÁRIO. Esta cena não tem sinal nenhum: o
 * `@post` do Datastar manda o `<form>` inteiro (`contentType: 'form'`) e o
 * servidor devolve a folha com o equipamento da classe escolhida. O teste de
 * handler prova que o servidor responde certo ao formulário; ele NÃO prova que
 * o Datastar coleta os controles e aplica o remendo, que é a parte nova e a
 * única que pode quebrar sem ninguém ver.
 *
 * O segundo é o de sempre nesta casa: CONTRASTE e LEIAUTE reais. A ficha
 * atravessou duas fatias sem medição porque o medidor não era importável
 * (ALE-272); uma cena nova que não entra numa lista de visitas nasce sem
 * medição, em silêncio, que é a marca desta família.
 *
 * O que NÃO está aqui, de propósito: recusa de escolha que o kit não oferece,
 * nascimento com o kit de p140, e o limite da compra de pontos. Isso é regra de
 * SERVIDOR e está preso em `api/piloto_forja_test.go`, que é a camada mais
 * barata que a segura.
 */
test.use({ storageState: '.auth/user.json' })

/** O rádio é `sr-only`: quem recebe o clique é o rótulo inteiro, a carta. */
async function escolheACarta(page: import('@playwright/test').Page, grupo: string, valor: string) {
  await page.locator(`label:has(input[name="${grupo}"][value="${valor}"])`).click()
}

test('o equipamento aparece e segue a classe, redesenhado pelo servidor', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/piloto/personagens/nova')
  await expect(page.getByRole('heading', { name: 'A folha em branco' })).toBeVisible()

  // O CONTROLE do caso: antes da classe não há kit para oferecer, e é essa
  // ausência que o clique tem de desfazer. Sem ela, um "Equipamento inicial"
  // que já estivesse na página passaria como se o remendo tivesse chegado.
  await expect(page.getByText('Equipamento inicial')).toHaveCount(0)

  await escolheACarta(page, 'class', 'Guerreiro')
  await expect(page.getByText('Equipamento inicial')).toBeVisible()
  // Pelo `id` e não por `getByLabel`: o nome acessível de uma carta de raça
  // carrega as habilidades dela, e "Armadura de Allihanna" (dahllan) casa com
  // um rótulo de "Armadura" — três elementos, e o modo estrito reprova.
  await expect(page.locator('#weaponMartial')).toBeVisible()
  await expect(page.locator('#armor')).toContainText('Brunea')

  // O nome digitado ANTES do remendo sobrevive a ele: quem carrega o estado é o
  // formulário, e o servidor o devolve preenchido.
  await page.getByLabel('Nome').fill('Thessa de Valkaria')
  await escolheACarta(page, 'class', 'Arcanista')
  await expect(page.getByText('Arcanistas começam sem armadura')).toBeVisible()
  await expect(page.locator('#weaponMartial')).toHaveCount(0)
  await expect(page.getByLabel('Nome')).toHaveValue('Thessa de Valkaria')

  const contraste = await medeOContraste(page)
  expect(contraste.medidos, 'o medidor não achou texto: a folha não carregou').toBeGreaterThan(100)
  expect(contraste.falhas, 'texto abaixo do AA na folha da forja').toEqual([])
  await expectNoHorizontalOverflow(page, VIEWPORTS)
})

test('a distribuição de atributos anda pelo servidor', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/piloto/personagens/nova')
  await page.getByLabel('Nome').fill('Herói do Guarda da Forja')
  await escolheACarta(page, 'race', 'Humano')
  await escolheACarta(page, 'class', 'Guerreiro')
  await expect(page.getByText('Equipamento inicial')).toBeVisible()
  await page.selectOption('#origin', 'Acólito')
  await expect(page.getByText('Equipamento inicial')).toBeVisible()
  await page.selectOption('#weaponSimple', { label: 'Adaga' })
  await page.selectOption('#weaponMartial', { label: 'Espada longa' })
  await page.selectOption('#armor', { label: 'Couro batido' })
  await page.getByRole('button', { name: 'Forjar' }).click()

  await page.waitForURL(/\/atributos$/)
  await expect(page.getByText('10 de 10 pontos')).toBeVisible()

  await page.getByRole('button', { name: 'Aumentar Força' }).click()
  await expect(page.getByText('9 de 10 pontos')).toBeVisible()

  const contraste = await medeOContraste(page)
  expect(contraste.medidos, 'o medidor não achou texto: a cena não carregou').toBeGreaterThan(20)
  expect(contraste.falhas, 'texto abaixo do AA nos atributos da forja').toEqual([])
  await expectNoHorizontalOverflow(page, VIEWPORTS)

  // A GRAMÁTICA DE TECLADO, medida aqui e não na lista do
  // `piloto-gramatica-do-teclado.spec.ts`: aquele guarda enumera cenas de
  // endereço FIXO, e o desta tem o id de um herói que só existe depois de
  // alguém forjar. Sem esta asserção, a única cena do piloto com endereço
  // dinâmico nasceria fora do regime — que é a forma exata do defeito que
  // aquele guarda existe para prender.
  await page.setViewportSize({ width: 1400, height: 900 })
  const regiao = page.locator('[data-nav-region="content"]')
  await expect(regiao).toHaveCount(1)
  expect(
    await regiao.locator('button:not([disabled]),a[href]').count(),
    'a região da cena não tem item nenhum para as setas dirigirem',
  ).toBeGreaterThan(0)
})
