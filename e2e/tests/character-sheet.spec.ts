import { type Locator, type Page, expect, test } from '@playwright/test'
import { openSheetFromRoster } from './support/roster'

// A hero no other spec asserts on, so the vitals edit can't disturb them.
const HERO = 'Necromante Nv12 Magias'

/**
 * Waits for a vitals edit to actually reach the API. The two directions take
 * different routes — reduzir goes through the damage pipeline (temp HP pool),
 * curar patches the vitals directly — so match either.
 */
function vitalsWrite(page: Page) {
  return page.waitForResponse(
    (res) =>
      /\/api\/characters\/\d+\/(damage|vitals)$/.test(res.url()) &&
      res.request().method() !== 'GET' &&
      res.ok(),
  )
}

async function currentHp(vida: Locator): Promise<number> {
  const value = await vida.getAttribute('aria-valuenow')
  if (value === null) throw new Error('barra de Vida sem aria-valuenow (esperado um inteiro)')
  return Number(value)
}

/**
 * Login → Hub → abrir personagem → editar bloco da ficha (vitals).
 *
 * Reads the current PV instead of hardcoding it and puts it back at the end:
 * the edit is a real server mutation, so the test must leave the seed as it
 * found it (F.I.R.S.T — repeatable).
 */
test('Hub → herói → editar Vida no bloco de vitals (persiste no servidor)', async ({ page }) => {
  await page.goto('/')
  await page.getByText('Meus Heróis').click()
  await expect(page).toHaveURL(/\/piloto\/personagens$/)
  await openSheetFromRoster(page, HERO)
  await expect(page).toHaveURL(/\/characters\/\d+$/)

  const vida = page.getByRole('progressbar', { name: 'Vida' })
  const before = await currentHp(vida)

  const decremented = vitalsWrite(page)
  await page.getByRole('button', { name: /^Reduzir Vida/ }).click()
  await expect(vida).toHaveAttribute('aria-valuenow', String(before - 1))
  await decremented

  // Reload proves the write reached the API — not just the optimistic cache.
  await page.reload()
  await expect(vida).toHaveAttribute('aria-valuenow', String(before - 1))

  const restored = vitalsWrite(page)
  await page.getByRole('button', { name: /^Aumentar Vida/ }).click()
  await expect(vida).toHaveAttribute('aria-valuenow', String(before))
  await restored
})

/**
 * A linha de PV/PM cabe na coluna quando a barra de rolagem come espaço
 * (ALE-196).
 *
 * A linha é quase toda `shrink-0` — rótulo, os dois botões e o número — e só a
 * barra encolhe; sem `min-w-0` no item de grade, e com o piso de 32px da barra,
 * ela pedia 252px numa coluna de 243 e pintava para fora. Medido a 375px, que
 * são os 390 do menor formato da casa MENOS os ~15px da barra de rolagem
 * CLÁSSICA: Linux e Windows a desenham dentro da caixa, e é por isso que a tela
 * passava no desktop deste repositório e vazava no runner.
 *
 * Quem cedeu espaço foi a barra, e não os botões: eles são alvo de toque (a
 * ALE-177 já mede que 56% dos alvos da ficha estão abaixo do mínimo) e a barra
 * é indicador, com o número exato ao lado dela.
 *
 * Por que e2e: largura real, barra de rolagem real. Em jsdom todo elemento mede
 * zero e a asserção passaria verde sobre a tela quebrada.
 */
test('a linha de PV/PM não vaza a coluna numa tela apertada', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 844 })
  await page.goto('/')
  await page.getByText('Meus Heróis').click()
  await expect(page).toHaveURL(/\/piloto\/personagens$/)
  await openSheetFromRoster(page, HERO)
  await expect(page.getByRole('progressbar', { name: 'Vida' })).toBeVisible()

  // Mede TUDO dentro do bloco de vitais, e não só a borda da linha: com o item
  // de grade encolhendo mas a barra presa no piso antigo, quem vazava passava a
  // ser o CONTEÚDO da linha — o defeito só mudava de lugar, e uma asserção na
  // borda da linha diria que estava tudo bem.
  const escapando = await page.evaluate(() => {
    const barra = document.querySelector('[role="progressbar"][aria-label="Vida"]')
    const bloco = barra?.parentElement?.parentElement
    if (!bloco) return null
    return [...bloco.querySelectorAll('*')]
      .filter((node) => {
        const pai = node.parentElement
        if (!pai) return false
        const estilo = getComputedStyle(node)
        if (estilo.position === 'absolute' || estilo.position === 'fixed') return false
        if (getComputedStyle(pai).overflowX !== 'visible') return false
        const r = node.getBoundingClientRect()
        return r.width > 0 && r.right > pai.getBoundingClientRect().right + 1
      })
      .map((node) => (node.textContent ?? '').trim().slice(0, 20) || node.tagName)
      .slice(0, 5)
  })

  expect(escapando, 'o bloco de PV/PM não foi encontrado').not.toBeNull()
  expect(escapando, 'a linha de PV/PM pintou para fora da coluna').toEqual([])
})

/**
 * A navegação da ficha continua na tela no CELULAR DEITADO (ALE-162).
 *
 * O defeito: a 844×390 a barra de abas nascia em y=401 numa tela de 390, e a
 * página não rola — a navegação inteira da ficha ficava inalcançável, com o
 * jogador vendo só o crachá. Não dava para trocar de bloco.
 *
 * A causa é que o gate do bloco de derivados (Defesa, ataques, resistências,
 * atributos) era só de LARGURA: 844 passa do `md`, então o bloco aparecia — e,
 * abaixo do `lg`, empilhava. Media 134px somados aos 152 do crachá, numa caixa
 * de 313. O celular deitado tem largura de tablet e altura de telefone.
 *
 * Esconder ali não esconde o dado: os mesmos números moram na aba Combate
 * desde a ALE-145, que é como o telefone em pé sempre os alcançou.
 *
 * Por que e2e: é geometria contra a janela real. Em jsdom todo elemento mede
 * zero e a barra "cabe" em qualquer formato.
 */
test('no celular deitado, a barra de abas da ficha fica na tela', async ({ page }) => {
  await page.setViewportSize({ width: 844, height: 390 })
  await page.goto('/')
  await page.getByText('Meus Heróis').click()
  await expect(page).toHaveURL(/\/piloto\/personagens$/)
  await openSheetFromRoster(page, HERO)
  await expect(page.getByRole('progressbar', { name: 'Vida' })).toBeVisible()

  const barra = page.getByRole('tablist')
  await expect(barra).toBeVisible()
  await expect(barra, 'a barra de abas saiu da tela').toBeInViewport()

  // E ela FUNCIONA: alcançável não é o mesmo que clicável, e é a troca de bloco
  // que o defeito impedia.
  await page.getByRole('tab', { name: 'Mochila' }).click()
  await expect(page.getByRole('tab', { name: 'Mochila' })).toHaveAttribute(
    'data-selected',
    '',
  )
})

/**
 * No celular, o nome do herói cabe E o nível continua alterável (ALE-183).
 *
 * O defeito: a 390px o chip de Defesa e o stepper de nível dividiam a faixa com
 * o nome, que truncava para "Tanqu…". A issue propunha tirar o stepper nesse
 * formato — e isso seria tirar uma CAPACIDADE, não mover um controle: ele é o
 * único lugar do app inteiro que muda o nível de um personagem, porque não há
 * tela de edição e a Forja só cria. Ele desceu uma fileira, para junto dos
 * selos de classe, onde havia folga.
 *
 * As duas asserções andam juntas de propósito: a primeira sozinha passaria
 * verde com o stepper apagado, que é exatamente o conserto errado.
 *
 * Por que e2e: largura real e fonte real decidem se o nome trunca. Em jsdom
 * todo elemento mede zero e nada nunca corta.
 */
test('no celular o nome do herói cabe, e o nível continua alterável', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/')
  await page.getByText('Meus Heróis').click()
  await expect(page).toHaveURL(/\/piloto\/personagens$/)
  await openSheetFromRoster(page, HERO)
  await expect(page.getByRole('progressbar', { name: 'Vida' })).toBeVisible()

  const cortado = await page.evaluate((nome) => {
    const el = [...document.querySelectorAll<HTMLElement>('*')].find(
      (n) => n.children.length === 0 && (n.textContent ?? '').trim().startsWith(nome),
    )
    return el ? { texto: (el.textContent ?? '').trim(), cortado: el.scrollWidth > el.clientWidth + 1 } : null
  }, HERO)

  expect(cortado, `não achei "${HERO}" no crachá`).not.toBeNull()
  expect(cortado?.cortado, 'o nome do herói está truncado no crachá').toBe(false)

  // E o nível continua na tela e clicável: alcançável não é o mesmo que
  // presente, e subir de nível não tem outro caminho no app.
  const nivel = page.getByRole('group', { name: 'Nível' })
  await expect(nivel).toBeVisible()
  await expect(nivel).toBeInViewport()
  await expect(nivel.getByRole('button', { name: 'Aumentar nível' })).toBeEnabled()
})
