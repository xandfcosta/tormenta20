import { expect, type Page, test } from '@playwright/test'

/**
 * A MESA DO MESTRE (ALE-522). Cada caso está aqui porque o navegador é a ÚNICA
 * testemunha:
 *
 *  1. **O nome do sinal.** Nome de ATRIBUTO é minusculado pelo analisador de
 *     HTML, e nenhuma outra camada faz isso. O teste de handler manda um corpo
 *     escrito à mão e passa verde sobre o defeito; jsdom não tem Datastar.
 *  2. **O leiaute da linha da fila** a 390px — jsdom mede zero.
 *  3. **O remendo sem o stream**, que só um navegador com o canal cortado mede.
 *  4. **A centralização do `<dialog>`**, regra do agente do usuário combinada
 *     com a folha compilada.
 */

test.use({ storageState: '.auth/user.json' })

const MESA = '/campanhas/1/sessoes/4'

/**
 * Traz o grupo da campanha para a fila pelo Adicionar. É idempotente — clicar de
 * novo só traz quem faltou —, então não depende do que outro spec deixou.
 */
async function bringTheParty(page: Page): Promise<void> {
  await page.getByRole('button', { name: 'Adicionar à fila' }).first().click()
  const dialog = page.locator('#gm-add')
  await dialog.getByRole('button', { name: 'Grupo', exact: true }).click()
  await dialog.getByRole('button', { name: 'Adicionar o grupo' }).click()
  await expect(dialog, 'o Adicionar não fechou').not.toHaveAttribute('open', '')
}

test.describe('A mesa do mestre', () => {
  /**
   * O DEFEITO QUE SÓ O NAVEGADOR MOSTRA.
   *
   * Uma chave de atributo escrita em camelCase chega ao Datastar MINÚSCULA,
   * porque nome de atributo é minusculado pelo analisador — e ele então liga um
   * sinal NOVO com esse nome. O fio passa a levar os DOIS, e o servidor lê o
   * declarado: um número plausível no lugar do certo.
   *
   * A requisição é interceptada e ABORTADA de propósito: o que se mede é o que
   * a página manda, e descansar de verdade mexeria nos PV de fichas que outros
   * specs leem.
   */
  test('a qualidade que o mestre escolhe é a que vai no fio', async ({ page }) => {
    await page.goto(MESA)

    let body: string | null = null
    await page.route('**/descanso/dia', async (route) => {
      body = route.request().postData()
      await route.abort()
    })

    await page.getByRole('button', { name: 'Cena, descanso e sessão' }).click()
    await page.getByRole('button', { name: 'Recuperar · dia' }).click()
    const quality = page.getByRole('group', { name: 'Qualidade do descanso' })
    await expect(quality).toBeVisible()
    // O CONTROLE de que o sinal declarado chega ao elemento: sem ele nenhuma
    // opção nasceria marcada e a tela mentiria antes de qualquer escolha.
    await expect(quality.getByRole('button', { name: 'Normal' })).toHaveAttribute('aria-pressed', 'true')

    await quality.getByRole('button', { name: 'Luxuosa' }).click()
    await page.getByRole('button', { name: 'Descansar' }).click()

    await expect.poll(() => body, { message: 'o clique não postou nada' }).not.toBeNull()
    const signals = JSON.parse(body ?? '{}') as Record<string, unknown>
    expect(signals.rest_quality, `o fio levou ${body}`).toBe('luxuosa')
    // E leva UM nome só: dois é exatamente a forma do defeito.
    expect(Object.keys(signals).filter((k) => k.toLowerCase().includes('quality'))).toHaveLength(1)
  })

  /**
   * A LINHA DA FILA com o − e o + do PV, numa tela de 390: o formato onde a
   * fileira estoura, e estourar aqui é o + saindo pela borda. Quem cede é o
   * nome — e é isso que a asserção prende: os passos INTEIROS dentro da linha.
   */
  test('os passos da linha cabem na fila a 390px', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto(MESA)
    await bringTheParty(page)

    // A linha medida TEM o crachá "Ficha", que é o que empurra os passos: uma
    // linha sem ele passaria verde sobre o caso que importa.
    const row = page.locator('#table-tracker .ds-gm-row').filter({ hasText: 'Ficha' }).first()
    await expect(row).toBeVisible()
    await expect(row.getByRole('button', { name: /^Tirar 1 PV de / })).toBeVisible()

    const measure = await row.evaluate((el) => {
      const box = el.getBoundingClientRect()
      const steps = (el.querySelector('.ds-gm-steps') as HTMLElement).getBoundingClientRect()
      return {
        cut: el.scrollWidth - el.clientWidth,
        outRight: steps.right - box.right,
        outLeft: box.left - steps.left,
      }
    })
    expect(measure.cut, 'a linha recortou o próprio conteúdo').toBeLessThanOrEqual(1)
    expect(measure.outRight, 'os passos saíram pela direita da linha').toBeLessThanOrEqual(1)
    expect(measure.outLeft, 'os passos saíram pela esquerda da linha').toBeLessThanOrEqual(1)
  })

  /**
   * O COMANDO REMENDA A CENA SOZINHO, sem depender do stream.
   *
   * Medir "o remendo chega antes do batimento" seria VÁCUO: o stream acorda a
   * cada mutação e entregaria a mesma tela igualmente rápido. O que o remendo
   * compra é INDEPENDÊNCIA — então a forma honesta de medi-lo é cortar o
   * stream: com o canal fechado, o único caminho para o DOM mudar é a resposta
   * do POST.
   *
   * O verbo é o − e o + de um PV: os dois devolvem a linha ao que era, e a
   * sessão é compartilhada por outros specs.
   */
  test('o comando redesenha a cena mesmo com o stream cortado', async ({ page }) => {
    // REGEX e não glob: o `@get` do Datastar anexa os sinais como query string,
    // e um glob terminado em `/fluxo` não casa — o teste passaria verde com o
    // stream ABERTO. Quem denuncia isso é o controle abaixo.
    let triedOpening = 0
    await page.route(/\/campanhas\/\d+\/sessoes\/\d+\/fluxo(\?|$)/, async (route) => {
      triedOpening++
      await route.abort()
    })

    await page.goto(MESA)
    await bringTheParty(page)

    const row = page.locator('#table-tracker .ds-gm-row').filter({ hasText: 'Ficha' }).first()
    const pv = row.locator('.ds-gm-vital .ds-row-meta').first()
    await expect(pv).toBeVisible()
    // O CONTROLE de que o canal está mesmo fechado: a página TENTOU abri-lo e
    // foi barrada.
    expect(triedOpening, 'a página nem tentou abrir o stream — o corte não prova nada').toBeGreaterThan(0)

    const before = (await pv.textContent()) ?? ''
    await row.getByRole('button', { name: /^Tirar 1 PV de / }).click()
    await expect(pv, 'o − não redesenhou a linha com o stream cortado').not.toHaveText(before)

    // Devolve a linha ao que era: a sessão é compartilhada.
    await row.getByRole('button', { name: /^Devolver 1 PV a / }).click()
    await expect(pv).toHaveText(before)
  })

  /**
   * NENHUM DIÁLOGO FECHADO ROUBA O CLIQUE DA CENA.
   *
   * O navegador dá `display:none` a `<dialog>` fechado, e um `display` da folha
   * SOBREPÕE isso: o diálogo fechado ocupa a tela inteira e engole o clique de
   * todo botão da Mesa. Ele varre `dialog:not([open])` em vez de citar ids —
   * amostragem e não enumeração, então vale para o diálogo que nascer amanhã.
   */
  test('nenhum diálogo fechado rouba o clique da cena', async ({ page }) => {
    await page.goto(MESA)

    const shut = await page.evaluate(() =>
      [...document.querySelectorAll('dialog:not([open])')].map((d) => ({
        id: d.id,
        display: getComputedStyle(d).display,
      })),
    )
    // O CONTROLE: a cena do mestre TEM diálogos.
    expect(shut.length, 'a cena não tem diálogo nenhum para medir').toBeGreaterThan(0)
    for (const d of shut) {
      expect(d.display, `o diálogo #${d.id} está fechado e ocupando a tela`).toBe('none')
    }
  })

  /**
   * O `<dialog>` modal centralizado no desktop: a centralização é a `margin:
   * auto` que o navegador aplica sobre `inset: 0`, e o `preflight` do Tailwind
   * a zera. Só a camada de topo do navegador responde.
   */
  test('o diálogo do descanso nasce centralizado', async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 768 })
    await page.goto(MESA)
    await page.getByRole('button', { name: 'Cena, descanso e sessão' }).click()
    await page.getByRole('button', { name: 'Recuperar · dia' }).click()

    const box = page.locator('dialog#gm-day-rest')
    await expect(box).toBeVisible()
    const center = await box.evaluate((el) => {
      const r = el.getBoundingClientRect()
      return { x: r.left + r.width / 2, y: r.top + r.height / 2 }
    })
    // Tolerância de 2px para o arredondamento de sub-pixel, e não mais.
    expect(Math.abs(center.x - 512), 'fora do centro horizontal').toBeLessThanOrEqual(2)
    expect(Math.abs(center.y - 384), 'fora do centro vertical').toBeLessThanOrEqual(2)
  })

  /**
   * O TABULEIRO NUMA JANELA À PARTE sai da aba original, e volta quando ela
   * fecha. Só o navegador mede: são DOIS documentos conversando pelo evento
   * `storage`, que nenhum teste de handler vê. No celular não há segunda tela,
   * e o botão de abrir a janela não aparece.
   */
  test('o tabuleiro numa janela sai da aba original e volta quando ela fecha', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto(MESA)
    const board = page.locator('[data-gm-col="board"]')
    const rail = page.locator('[data-gm-rail="board"]')
    // O CONTROLE: antes da janela, o tabuleiro está na aba e a faixa não.
    await expect(board).toBeVisible()
    await expect(rail).toBeHidden()

    const windowed = await page.context().newPage()
    await windowed.goto(`${MESA}?janela=tabuleiro`)
    await expect(board, 'a janela abriu e o tabuleiro continuou na aba original').toBeHidden()
    await expect(rail).toBeVisible()

    await windowed.close({ runBeforeUnload: true })
    await expect(board, 'a janela fechou e o tabuleiro não voltou').toBeVisible()

    await page.setViewportSize({ width: 390, height: 844 })
    await expect(page.getByRole('button', { name: 'Abrir o tabuleiro numa janela' })).toBeHidden()
  })

  /**
   * A BUSCA DO GRUPO filtra no navegador, pelo sinal, sem ida ao servidor: só
   * um navegador com o Datastar avalia o `data-show` de cada linha.
   */
  test('a busca do Grupo esconde quem não casa e devolve ao apagar', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto(MESA)
    await page.getByRole('navigation', { name: 'Consultas do mestre' }).getByRole('button', { name: 'Grupo' }).click()
    const rows = page.locator('.ds-gm-member')
    // O CONTROLE: o Grupo tem gente, senão "ninguém visível" não prova nada.
    await expect(rows.first()).toBeVisible()
    const search = page.getByRole('searchbox', { name: 'Buscar no grupo' })
    await search.fill('zzz-ninguém')
    await expect(rows.filter({ visible: true })).toHaveCount(0)
    await search.fill('')
    await expect(rows.first()).toBeVisible()
  })

  /**
   * O CADERNO LÊ POR PADRÃO e escreve ao toque; o Esc e sair da caixa voltam à
   * leitura. Só o navegador mede: é foco, `blur` e a tecla — e o Esc tem de
   * ficar com o caderno, sem chegar ao "voltar" do documento.
   */
  test('as notas leem por padrão, escrevem ao toque e voltam à leitura com o Esc', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto(MESA)
    const reading = page.getByRole('button', { name: 'Editar as notas da sessão' })
    const box = page.getByRole('textbox', { name: 'Notas da sessão' })
    // O CONTROLE: abre lendo, sem a caixa.
    await expect(reading).toBeVisible()
    await expect(box).toBeHidden()

    await reading.click()
    await expect(box, 'o toque não abriu a caixa').toBeVisible()
    await expect(box, 'a caixa abriu sem o cursor').toBeFocused()

    await page.keyboard.press('Escape')
    await expect(box, 'o Esc não voltou à leitura').toBeHidden()
    await expect(reading).toBeVisible()
    await expect(page, 'o Esc escapou do caderno e saiu da mesa').toHaveURL(new RegExp(`${MESA}$`))

    await reading.click()
    await expect(box).toBeFocused()
    await page.getByRole('navigation', { name: 'Consultas do mestre' }).getByRole('button', { name: 'Notas' }).click()
    await expect(box, 'sair da caixa não voltou à leitura').toBeHidden()
  })
})
