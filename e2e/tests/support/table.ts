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
  const nome = `E2E Descartável tabuleiro ${Date.now()}-${Math.floor(Math.random() * 1e6)}`

  // A FIXTURE vai pela API e não pela tela, de propósito: montar campanha e
  // sessão clicando gastaria meia dúzia de navegações em cada caso para chegar
  // ao que se quer medir, e nenhuma delas é o assunto deste arquivo. O que é
  // medido — o tabuleiro — vai pela tela inteiro. Os casos abaixo só precisam
  // de um endereço de mesa, então trocar este caminho não mexe em nenhum.
  const criada = await page.request.post('/api/campanhas', {
    data: { name: nome, description: 'Criada e apagada pelo E2E do tabuleiro.' },
  })
  expect(criada.ok(), `criar a campanha descartável: ${criada.status()}`).toBeTruthy()
  const campanha = (await criada.json()).id as number

  const sessao = await page.request.post(`/api/campanhas/${campanha}/sessoes`, {
    data: { sessionNumber: 1, title: 'Sessão do E2E' },
  })
  expect(sessao.ok(), `criar a sessão descartável: ${sessao.status()}`).toBeTruthy()
  const sid = (await sessao.json()).id as number

  return {
    mesa: `/campanhas/${campanha}/sessoes/${sid}`,
    // A LIMPEZA NÃO PODE FALAR MAIS ALTO QUE O DEFEITO: sem o `catch`, um caso
    // que falhou no meio deixa a página num estado em que o `delete` estoura, e
    // o relatório mostra o erro da FAXINA no lugar do erro do teste. A campanha
    // órfã custa uma linha na lista; o defeito escondido custa uma sessão.
    apagar: async () => {
      try {
        await page.request.delete(`/api/campanhas/${campanha}`)
      } catch {
        // A mesa descartável fica para trás. É o preço certo a pagar.
      }
    },
  }
}

/**
 * Abre a GAVETA da fila, que é onde a lista inteira mora.
 *
 * A forma do mestre é uma SHELL: o trilho de 80px responde "de quem é a vez", e
 * dano, ordem, condição, "+ Combatente" e "Adicionar grupo" ficam numa gaveta
 * pela esquerda.
 *
 * Sem este passo os botões existem no HTML dentro de um `<dialog>` FECHADO, que
 * o navegador esconde com `display:none`. O sintoma não é "não achei o botão":
 * é um TIMEOUT de clique em cima de um seletor que casou — que foi exatamente
 * como estes três casos apareceram no CI.
 *
 * UM seletor nas duas larguras: acima de 1024 quem abre é o ⤢ do trilho, abaixo
 * é o botão da fileira de consultas, e os dois têm o mesmo prefixo de nome
 * acessível de propósito. O `visible` é o que escolhe entre eles — o outro está
 * no DOM com `display:none`, e sem o filtro o `.first()` acertaria o escondido.
 */
export async function closeTheTracker(page: Page): Promise<void> {
  await page.getByRole('button', { name: 'Fechar a iniciativa' }).click()
  await expect(page.locator('#tracker-drawer'), 'a gaveta da fila não fechou').not.toHaveAttribute(
    'open',
    '',
  )
}

export async function openTheTracker(page: Page): Promise<void> {
  // IDEMPOTENTE, porque acrescentar dois combatentes é abrir a gaveta duas
  // vezes: com ela já aberta o botão de abrir está coberto pelo próprio modal,
  // e o sintoma é um timeout de clique num seletor que casou — o mesmo que o
  // comentário de baixo descreve, chegando por outro caminho.
  if (await page.locator('#tracker-drawer[open]').count()) return
  await page
    .getByRole('button', { name: /^Abrir a iniciativa/ })
    .filter({ visible: true })
    .click()
  await expect(page.locator('#tracker-drawer'), 'a gaveta da fila não abriu').toHaveAttribute(
    'open',
    '',
  )
}

/**
 * Acrescenta UM combatente à fila, com a gaveta ABERTA no fim.
 *
 * O `pv` é opcional porque o formulário o trata assim, e a diferença importa:
 * sem ele a linha não desenha barra de vital nenhuma (`trackerBar` só desenha
 * `if b != nil`), e um guarda que precise ver o número mudar mediria uma linha
 * que não tem número.
 */
export async function putACombatantInTheTracker(
  page: Page,
  nome: string,
  pv?: number,
): Promise<void> {
  await openTheTracker(page)
  // O "+ Combatente" é um ALTERNADOR (`$combatant_form = !$combatant_form`),
  // e o formulário fica aberto depois de acrescentar. Clicar sem olhar o estado
  // FECHA o formulário no segundo combatente, e o sintoma é um timeout no campo
  // Nome — que existe, e está escondido.
  const abrir = page.getByRole('button', { name: '+ Combatente' })
  if ((await abrir.getAttribute('aria-expanded')) !== 'true') {
    await abrir.click()
  }
  // `exact` porque `getByLabel` casa por SUBSTRING, e a mesma cena tem "Nome do
  // NPC": sem ele, `'Nome'` resolve para dois campos.
  await page.getByLabel('Nome', { exact: true }).fill(nome)
  if (pv !== undefined) {
    // Pelo ID e não pelo rótulo: `PV` resolve para DOIS campos nesta cena — este
    // e o ajuste do bestiário —, e o `exact` não desempata porque os dois se
    // chamam exatamente "PV".
    await page.locator('#novo-pv').fill(String(pv))
  }
  await page.getByRole('button', { name: 'Acrescentar' }).click()
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
  // A GAVETA FECHA ANTES de o teste voltar ao mapa, e esta ordem é a jornada de
  // verdade: monta-se a fila na gaveta, fecha-se, e põe-se no mapa pela faixa do
  // tabuleiro. Ela é MODAL — deixá-la aberta torna inerte tudo o que está atrás,
  // e o `Pôr no mapa` da faixa (que vem antes no DOM, então é o que o `.first()`
  // acha) ficaria coberto por ela. O sintoma é "dialog intercepts pointer
  // events" num seletor que casou, e não um "não achei".
  await closeTheTracker(page)
  await page.getByRole('button', { name: 'Pôr no mapa', exact: true }).first().click()
  // Escopado ao DIÁLOGO: o nome do combatente aparece também na fila atrás dele,
  // e um seletor de página inteira acha os dois.
  const dialogo = page.locator('#populate')
  await dialogo.getByRole('button', { name: /Ogro do E2E/ }).click()
  await dialogo.getByRole('button', { name: 'Pôr no mapa', exact: true }).click()
  await expect(page.locator('.board-token'), 'a peça não entrou no mapa').toHaveCount(1)
}

/** Abre o tabuleiro pela TELA, que é o gesto de verdade. */
export async function openTheBoard(page: Page, mesa: string): Promise<void> {
  await page.goto(mesa, { waitUntil: 'domcontentloaded' })
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

