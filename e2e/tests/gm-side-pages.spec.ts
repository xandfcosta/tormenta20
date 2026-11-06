import { expect, test } from '@playwright/test'

/**
 * AS PÁGINAS À PARTE DO MESTRE (ALE-608). O navegador é a única testemunha de
 * duas coisas:
 *
 *  1. **Qual das duas formas do cabeçalho aparece** — os botões soltos ou o
 *     menu "Ferramentas". Quem decide é a folha compilada pela largura, e o
 *     teste de handler vê as duas no HTML.
 *  2. **A página se atualizar sozinha** quando OUTRO cliente muda o PV de
 *     alguém: é o fluxo com a marca da página remendando a região dela.
 */

test.use({ storageState: '.auth/user.json' })

const MESA = '/campanhas/1/sessoes/4'
const PAGES = ['Notas', 'Grupo', 'Elenco', 'Bestiário']

test('no desktop as páginas ficam soltas no cabeçalho, e no celular num menu Ferramentas', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 })
  await page.goto(MESA)
  const header = page.locator('#table-header')
  const tools = header.getByRole('button', { name: 'Ferramentas' })
  for (const name of PAGES) {
    await expect(header.getByRole('link', { name, exact: true }), `o desktop não mostra ${name}`).toBeVisible()
  }
  await expect(tools, 'o desktop mostra o menu das telas estreitas').toBeHidden()

  await page.setViewportSize({ width: 390, height: 844 })
  await expect(tools).toBeVisible()
  await tools.click()
  const menu = page.locator('#gm-side-menu')
  for (const name of PAGES) {
    await expect(menu.getByRole('menuitem', { name: new RegExp(`^${name}`) }), `o menu não tem ${name}`).toBeVisible()
  }
})

test('a página do Grupo se atualiza quando outro cliente fere alguém', async ({ page, browser }) => {
  // O CONTROLE: o fluxo da página está aberto antes do golpe, senão a mudança
  // chegaria pela carga e não pelo remendo. A RESPOSTA, e não o pedido: o
  // cabeçalho do SSE só sai depois de o servidor assinar o barramento.
  const streaming = page.waitForResponse((r) => r.url().includes('/fluxo?pagina=grupo'))
  await page.goto(`${MESA}/grupo`)
  await streaming
  const row = page.locator('.ds-gm-member', { hasText: 'Guerreiro Veterano Nv8' })
  const vital = row.locator('.ds-gm-vital').first()
  await expect(vital).toContainText('PV')
  const before = await vital.innerText()

  // O endereço do passo de 1 PV sai do próprio botão, e quem o chama é OUTRO
  // cliente — a mesa noutra aba, que não é esta página.
  const minus = await row.getByRole('button', { name: /^Tirar 1 PV de/ }).getAttribute('data-on:click')
  const wound = minus?.match(/: '([^']+)'\)/)?.[1]
  expect(wound, `o − não escreve o passo de 1 PV: ${minus}`).toBeTruthy()
  const other = await browser.newContext({ storageState: '.auth/user.json' })
  try {
    await other.request.post(wound as string)
    await expect(vital, 'a página do Grupo não se atualizou').not.toHaveText(before)
  } finally {
    await other.request.post((wound as string).replace('/ferir/', '/curar/'))
    await other.close()
  }
  await expect(vital).toHaveText(before)
})
