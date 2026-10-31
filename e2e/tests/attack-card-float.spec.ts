import { expect, type Page, test } from '@playwright/test'

/**
 * O CARTÃO DO ATAQUE FLUTUA NO DESKTOP e se arrasta pela cabeça (ALE-536).
 *
 * E2E porque os dois mecanismos só existem num navegador: o gesto de ponteiro,
 * e o REMENDO do Datastar, que reescreve os atributos do cartão a cada mudança
 * — sem o `data-preserve-attr="style"`, o cartão voltaria ao lugar padrão no
 * primeiro toque do mestre, e nenhum teste de Go veria.
 *
 * A mesa é a da campanha 7 da semente (o Cavaleiro Postural tem espada), e o
 * alvo é um goblin do bestiário que o caso põe na fila.
 */

const MESA = '/campanhas/7/sessoes/10'
const ALVO = 'Goblin Salteador'

/** Um comando da mesa pela API, devolvendo a frase da recusa ("" quando não houve). */
async function command(page: Page, path: string, data: Record<string, unknown> = {}): Promise<string> {
  const reply = await page.request.post(MESA + path, { headers: { 'datastar-request': 'true' }, data })
  expect(reply.ok(), `${path} respondeu ${reply.status()}`).toBeTruthy()
  const found = /"command_error":"((?:[^"\\]|\\.)*)"/.exec(await reply.text())
  return found ? JSON.parse(`"${found[1]}"`) : ''
}

/** Põe um golpe do cavaleiro contra o alvo na mesa, passando a vez até ser a dele. */
async function proposesAnAttack(page: Page): Promise<void> {
  await command(page, '/cena/iniciar/acao')
  await command(page, '/elenco/17/na-fila')
  // DO BESTIÁRIO, e não escrito à mão: a linha sem bloco não tem Defesa, e o
  // ataque a recusa.
  await command(page, '/bestiario/enviar', {
    creature: 'goblin-salteador', entry_hp: 7, entry_initiative: 1, entry_copies: 1,
  })
  await page.goto(MESA)
  const target = await page.evaluate((label) => {
    const dialog = [...document.querySelectorAll('dialog[id^="combatant-"]')].find((d) =>
      d.querySelector('h2')?.textContent?.includes(label),
    )
    return dialog?.id.replace(/^combatant-/, '') ?? ''
  }, ALVO)
  expect(target, `o ${ALVO} não entrou na fila`).not.toBe('')
  let refusal = ''
  for (let attempt = 0; attempt < 4; attempt++) {
    refusal = await command(page, `/iniciativa/${target}/atacar`)
    if (refusal === '') return
    await command(page, '/iniciativa/proxima-vez')
  }
  throw new Error(`o cavaleiro não conseguiu atacar: ${refusal}`)
}

test('o cartão do ataque flutua e se arrasta no desktop, e no celular mostra só a cabeça', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 })
  await proposesAnAttack(page)
  try {
    await page.goto(MESA)
    const card = page.locator('#table-attack')
    await expect(card.getByText('Para julgar')).toBeVisible()
    expect(await card.evaluate((el) => getComputedStyle(el).position)).toBe('fixed')

    const head = card.locator('[data-float-handle]')
    const before = await card.boundingBox()
    const grip = await head.boundingBox()
    if (!before || !grip) throw new Error('o cartão não tem caixa')
    await page.mouse.move(grip.x + 30, grip.y + grip.height / 2)
    await page.mouse.down()
    await page.mouse.move(grip.x + 30 + 200, grip.y + grip.height / 2 + 150, { steps: 10 })
    await page.mouse.up()
    const dragged = await card.boundingBox()
    expect(Math.round((dragged?.x ?? 0) - before.x)).toBe(200)
    expect(Math.round((dragged?.y ?? 0) - before.y)).toBe(150)

    // O REMENDO QUE MUDA O PRÓPRIO CARTÃO: o crachá trocar é o controle de que
    // ele chegou — sem isso, "ficou no lugar" pode ser "nada foi remendado".
    await card.getByRole('button', { name: 'Acertou' }).click()
    await expect(card.getByText('Esperando o dano')).toBeVisible()
    // A POSIÇÃO, e não a caixa: o cartão encolhe ao perder os três botões.
    const spot = async () => {
      const box = await card.boundingBox()
      return { x: Math.round(box?.x ?? -1), y: Math.round(box?.y ?? -1) }
    }
    const placed = { x: Math.round(dragged?.x ?? 0), y: Math.round(dragged?.y ?? 0) }
    expect(await spot()).toEqual(placed)

    // E VOLTA para onde foi posto quando a página recarrega.
    await page.reload()
    await expect(card.getByText('Esperando o dano')).toBeVisible()
    expect(await spot()).toEqual(placed)

    // NO CELULAR ele fica no fluxo e mostra SÓ A CABEÇA: o corpo é um diálogo
    // que ela abre, para o cartão não comer a tela da mesa.
    await page.setViewportSize({ width: 390, height: 844 })
    expect(await card.evaluate((el) => getComputedStyle(el).position)).toBe('relative')
    const sheet = page.locator('#attack-sheet')
    await expect(sheet).toBeHidden()
    expect((await card.boundingBox())?.height ?? 0).toBeLessThan(100)
    await card.getByRole('button', { name: 'Abrir o ataque' }).click()
    await expect(sheet).toBeVisible()
    await expect(sheet.getByText('rola o dano')).toBeVisible()
  } finally {
    await command(page, '/ataque/cancelar').catch(() => '')
  }
})
