import { expect, type Page } from '@playwright/test'

/**
 * A MESA DESCARTÁVEL e os passos que chegam ao tabuleiro.
 *
 * Eles moram AQUI e não dentro de um spec porque **instrumento que mora dentro
 * de um chamador tem exatamente um chamador**: a alternativa era copiar
 * sessenta linhas de jornada, que divergem no primeiro dia em que um rótulo da
 * tela mudar.
 */

/** Uma mesa só desta corrida. Devolve o endereço e como se livrar dela. */
export async function disposableTable(page: Page): Promise<{ mesa: string; apagar: () => Promise<void> }> {
  const label = `E2E Descartável tabuleiro ${Date.now()}-${Math.floor(Math.random() * 1e6)}`

  // A FIXTURE vai pela API e não pela tela, de propósito: montar campanha e
  // sessão clicando gastaria meia dúzia de navegações em cada caso para chegar
  // ao que se quer medir, e nenhuma delas é o assunto deste arquivo. O que é
  // medido — o tabuleiro — vai pela tela inteiro. Os casos abaixo só precisam
  // de um endereço de mesa, então trocar este caminho não mexe em nenhum.
  const created = await page.request.post('/api/campanhas', {
    data: { name: label, description: 'Criada e apagada pelo E2E do tabuleiro.' },
  })
  expect(created.ok(), `criar a campanha descartável: ${created.status()}`).toBeTruthy()
  const campaign = (await created.json()).id as number

  const session = await page.request.post(`/api/campanhas/${campaign}/sessoes`, {
    data: { sessionNumber: 1, title: 'Sessão do E2E' },
  })
  expect(session.ok(), `criar a sessão descartável: ${session.status()}`).toBeTruthy()
  const sid = (await session.json()).id as number

  return {
    mesa: `/campanhas/${campaign}/sessoes/${sid}`,
    // A LIMPEZA NÃO PODE FALAR MAIS ALTO QUE O DEFEITO: sem o `catch`, um caso
    // que falhou no meio deixa a página num estado em que o `delete` estoura, e
    // o relatório mostra o erro da FAXINA no lugar do erro do teste. A campanha
    // órfã custa uma linha na lista; o defeito escondido custa uma sessão.
    apagar: async () => {
      try {
        await page.request.delete(`/api/campanhas/${campaign}`)
      } catch {
        // A mesa descartável fica para trás. É o preço certo a pagar.
      }
    },
  }
}

/**
 * Acrescenta UM combatente à fila pelo Adicionar do mestre (ALE-522).
 *
 * O `pv` é opcional porque o formulário o trata assim, e a diferença importa:
 * sem ele a linha não desenha barra de vital nenhuma, e um guarda que precise
 * ver o número mudar mediria uma linha que não tem número.
 *
 * Os campos vão pelo ID e não pelo rótulo: "Nome" e "PV" existem em mais de um
 * diálogo da mesa, e o `getByLabel` casaria todos.
 */
export async function putACombatantInTheTracker(
  page: Page,
  displayName: string,
  pv?: number,
): Promise<void> {
  await page.getByRole('button', { name: 'Adicionar à fila' }).first().click()
  const dialog = page.locator('#gm-add')
  await expect(dialog, 'o Adicionar não abriu').toHaveAttribute('open', '')
  await dialog.getByRole('button', { name: 'Combatente' }).click()
  await page.locator('#gm-new-name').fill(displayName)
  if (pv !== undefined) {
    await page.locator('#gm-new-hp').fill(String(pv))
  }
  await dialog.getByRole('button', { name: 'Adicionar à fila' }).click()
  await expect(dialog, 'o Adicionar não fechou').not.toHaveAttribute('open', '')
}

/**
 * Põe UMA peça no mapa, e este passo não é enfeite: a camada de MOVER — a que
 * cobre o marcador — só é desenhada quando existe algo movível. Num tabuleiro
 * vazio ela não nasce, e um teste de empilhamento sobre um tabuleiro vazio não
 * enfrenta o que veio guardar: tirar o `z-index` do marcador deixa o caso
 * VERDE, medindo um palco onde nada cobre nada.
 */
export async function putATokenOnTheMap(page: Page): Promise<void> {
  await putACombatantInTheTracker(page, 'Ogro do E2E')
  await page.getByRole('button', { name: 'Pôr no mapa', exact: true }).first().click()
  // Escopado ao DIÁLOGO: o nome do combatente aparece também na fila atrás dele,
  // e um seletor de página inteira acha os dois.
  const dialog = page.locator('#populate')
  await dialog.getByRole('button', { name: /Ogro do E2E/ }).click()
  await dialog.getByRole('button', { name: 'Pôr no mapa', exact: true }).click()
  await expect(page.locator('.board-token'), 'a peça não entrou no mapa').toHaveCount(1)
}

/** Abre o tabuleiro pela TELA, que é o gesto de verdade. */
export async function openTheBoard(page: Page, table: string): Promise<void> {
  await page.goto(table, { waitUntil: 'domcontentloaded' })
  await page.getByRole('button', { name: 'Abrir tabuleiro' }).click()
  // `exact` porque `getByLabel` casa por SUBSTRING, e o diálogo do acervo se
  // chama "Lugares da campanha": com acervo na mesa, `'Lugar'` resolve para
  // DOIS e o helper estoura em strict mode. O sintoma não aponta para a causa —
  // ele diz "waiting for getByLabel('Lugar')" num campo que está lá.
  await page.getByLabel('Lugar', { exact: true }).fill('Taverna do E2E')
  await page.getByRole('button', { name: 'Abrir', exact: true }).click()
  // A CENA e não o PLANO: o plano é uma ORIGEM de tamanho zero num plano
  // infinito, e o Playwright chama de invisível todo elemento sem caixa. Quem
  // tem o retângulo é a janela que recorta.
  await page.locator('.board-scene').waitFor({ timeout: 10_000 })
}

/**
 * Mostra o TABULEIRO do mestre nas larguras em que as superfícies se revezam
 * (abaixo de 1024 a mesa abre na Fila, ALE-522). No desktop as três estão à
 * vista, e a barra das superfícies nem aparece.
 */
export async function showTheBoard(page: Page): Promise<void> {
  const surface = page
    .getByRole('navigation', { name: 'Superfícies do mestre' })
    .getByRole('button', { name: 'Tabuleiro' })
  if (await surface.isVisible()) await surface.click()
}

/**
 * Escolhe um PINCEL de terreno: na mesa do mestre eles moram atrás do Terreno,
 * numa fileira que abre e fecha (ALE-522).
 */
export async function pickBrush(page: Page, label: string): Promise<void> {
  await page.getByRole('navigation', { name: 'Ferramentas do mapa' }).getByRole('button', { name: /^Terreno/ }).click()
  await page.getByRole('group', { name: 'Espécies de terreno' }).getByRole('button', { name: label }).click()
}
