import { type Browser, type Page, expect, test } from '@playwright/test'

/**
 * O REALTIME COM DOIS CLIENTES na Mesa.
 *
 * Por que e2e — e é o fluxo AO VIVO ENTRE DOIS CLIENTES, a justificativa que o
 * guia da casa nomeia: um handler que publicasse para a sala errada, ou para o
 * próprio remetente, passaria por todo teste de handler em Go e por todo caso de
 * uma aba só. O mestre veria a própria ação e o jogador não veria nada.
 *
 * SERIAL, e numa sessão só destes casos: eles escrevem uns sobre os outros, e a
 * 5 não é a das outras suítes.
 */
test.describe.configure({ mode: 'serial' })

const MESA = '/campanhas/1/sessoes/5'

/** As duas telas da mesma mesa, uma por papel. */
async function asDuasTelas(browser: Browser) {
  const gm = await browser.newContext({ storageState: '.auth/user.json' })
  const player = await browser.newContext({ storageState: '.auth/player.json' })
  const gmScreen = await gm.newPage()
  const playerScreen = await player.newPage()
  await gmScreen.goto(MESA)
  await playerScreen.goto(MESA)
  // OS DOIS CONECTADOS ANTES DE AGIR. Sem isto o caso mede uma corrida: o
  // mestre agiria antes de o stream do jogador existir, e a ausência do eco
  // seria lida como "não propagou".
  await expect(gmScreen.getByRole('button', { name: /^Abrir a iniciativa/ }).first()).toBeVisible()
  await expect(playerScreen.getByRole('group', { name: 'O que ver na sessão' })).toBeVisible()
  return {
    telaDoMestre: gmScreen,
    telaDoJogador: playerScreen,
    // `catch` na limpeza, sempre: fechar contexto pode lançar e SUBSTITUIR o
    // erro de verdade do caso.
    fecha: async () => {
      await gm.close().catch(() => {})
      await player.close().catch(() => {})
    },
  }
}

/**
 * A gaveta da fila, que é onde o mestre comanda.
 *
 * IDEMPOTENTE, e não é conveniência: a gaveta é MODAL, então com ela já aberta o
 * botão que a abre fica atrás dela. O sintoma não é "não achei o botão" — é um
 * timeout de clique sobre um seletor que casou, com "subtree intercepts pointer
 * events" no log, que aponta para o lugar errado.
 */
async function openTheTracker(page: Page) {
  const drawer = page.locator('#tracker-drawer')
  if (await drawer.getAttribute('open') === null) {
    await page
      .getByRole('button', { name: /^Abrir a iniciativa/ })
      .filter({ visible: true })
      .click()
  }
  await expect(drawer).toHaveAttribute('open', '')
}

async function closeTheTracker(page: Page) {
  await page.getByRole('button', { name: 'Fechar a iniciativa' }).click()
  await expect(page.locator('#tracker-drawer')).not.toHaveAttribute('open', '')
}

/** Põe um combatente na fila pelo gesto do mestre, e devolve o nome dele. */
async function poeNaFila(page: Page, displayName: string) {
  await openTheTracker(page)
  await page.getByRole('button', { name: '+ Combatente' }).click()
  await page.getByLabel('Nome', { exact: true }).fill(displayName)
  await page.getByRole('button', { name: 'Acrescentar' }).click()
  await expect(page.locator('#tracker-drawer').getByText(displayName).first()).toBeVisible()
}

async function tiraDaFila(page: Page, displayName: string) {
  await openTheTracker(page)
  await page.getByRole('button', { name: `Remover ${displayName} da fila` }).click()
  await closeTheTracker(page)
}

/**
 * A cena precisa estar EM CURSO: sem ela o servidor não manda fila à mesa.
 *
 * São DOIS cliques desde a ALE-365: "Iniciar cena" abre os três tipos do livro
 * (p252) e a fila só existe na cena de AÇÃO, que é a que este caso quer.
 */
async function garanteACena(page: Page) {
  const start = page.locator('summary[aria-label="Iniciar uma cena"]').filter({ visible: true })
  if (await start.count()) {
    await start.first().click()
    await page.getByRole('button', { name: 'Iniciar uma cena de Ação' }).filter({ visible: true }).first().click()
  }
  await expect(
    page.getByRole('button', { name: 'Encerrar cena' }).filter({ visible: true }).first(),
  ).toBeVisible()
}

test('cada papel recebe a SUA cena, e não a do outro', async ({ browser }) => {
  const { telaDoMestre: gmScreen, telaDoJogador: playerScreen, fecha: closes } = await asDuasTelas(browser)
  try {
    // O SELETOR DE SUPERFÍCIE é do jogador: ele escolhe entre a própria ficha, a
    // mesa e o tabuleiro. O mestre vê tudo junto, no palco dele.
    await expect(
      playerScreen.getByRole('group', { name: 'O que ver na sessão' }),
      'o jogador não recebeu a cena do jogador',
    ).toBeVisible()
    await expect(
      gmScreen.getByRole('group', { name: 'O que ver na sessão' }),
      'o mestre recebeu a cena do JOGADOR',
    ).toHaveCount(0)

    // E os COMANDOS são do mestre. A trava é na view (`v.Mestre` nil), então o
    // que se afirma aqui é que ela chegou íntegra até o navegador do jogador.
    await expect(
      gmScreen.getByRole('region', { name: 'Controles do mestre' }).first(),
      'o mestre não recebeu os controles do mestre',
    ).toBeVisible()
    await expect(
      playerScreen.getByRole('region', { name: 'Controles do mestre' }),
      'o jogador recebeu os controles do MESTRE',
    ).toHaveCount(0)
  } finally {
    await closes()
  }
})

test('o que o mestre põe na fila aparece na tela do jogador', async ({ browser }) => {
  const echo = `Eco de teste ${Date.now()}`
  const { telaDoMestre: gmScreen, telaDoJogador: playerScreen, fecha: closes } = await asDuasTelas(browser)
  try {
    await garanteACena(gmScreen)
    // O jogador olha a MESA — é onde a fila mora para ele.
    await playerScreen.getByRole('button', { name: 'Mesa', exact: true }).click()

    await poeNaFila(gmScreen, echo)
    await closeTheTracker(gmScreen)

    // A tela do jogador não recarrega: o combatente chega pelo stream.
    await expect(playerScreen.getByText(echo).first()).toBeVisible()

    await tiraDaFila(gmScreen, echo)
    await expect(playerScreen.getByText(echo)).toHaveCount(0)
  } finally {
    await closes()
  }
})

/**
 * A CONDIÇÃO QUE O MESTRE APLICA CHEGA À TELA DO JOGADOR.
 *
 * Ela é afirmada na FILA e não na ficha embutida, e a diferença é de desenho:
 * a superfície "Minha ficha" da Mesa não é região do stream (ver
 * `mesaView.MinhaFicha`), então ela se atualiza pelos comandos DELA e não pelo
 * que o mestre faz. Quem mostra a condição ao vivo é a linha do combatente, que
 * é remendada — e é ela que prova o que este caso existe para provar: a
 * mensagem saiu de um navegador e chegou no outro, na sala certa.
 */
test('a condição que o mestre aplica aparece na fila do jogador', async ({ browser }) => {
  const target = `Condenado ${Date.now()}`
  const { telaDoMestre: gmScreen, telaDoJogador: playerScreen, fecha: closes } = await asDuasTelas(browser)
  try {
    await garanteACena(gmScreen)
    await playerScreen.getByRole('button', { name: 'Mesa', exact: true }).click()
    await poeNaFila(gmScreen, target)

    // O CONTROLE: a linha chegou limpa antes. Sem ele, uma condição herdada de
    // outra corrida faria o caso passar sem nada ter propagado.
    await expect(playerScreen.getByText(target).first()).toBeVisible()
    await expect(playerScreen.getByTitle(/Abalado|-2 em testes/).first()).toHaveCount(0)

    await gmScreen.getByRole('button', { name: `Condições de ${target}` }).click()
    const dialog = gmScreen.locator('#combatant-conditions')
    await dialog.getByRole('button', { name: 'Abalado', exact: true }).click()
    await gmScreen.keyboard.press('Escape')

    // E a tela do jogador aprende sozinha, sem recarregar. É a issue inteira.
    await expect(playerScreen.getByText('ABALADO').first()).toBeVisible()

    await tiraDaFila(gmScreen, target)
    await expect(playerScreen.getByText(target)).toHaveCount(0)
  } finally {
    await closes()
  }
})

/**
 * A CENA como cortina: o mestre encerra e a fila SOME da mesa, enquanto
 * continua inteira na tela dele.
 *
 * As duas metades importam. Sumir da mesa é a regra; CONTINUAR na tela do
 * mestre é o que separa "redigi o que vai para a mesa" de "apaguei a fila" — e a
 * segunda leitura passaria verde afirmando só a primeira.
 */
test('encerrar a cena tira a fila da mesa sem tirá-la do mestre', async ({ browser }) => {
  const echo = `Cortina de teste ${Date.now()}`
  const { telaDoMestre: gmScreen, telaDoJogador: playerScreen, fecha: closes } = await asDuasTelas(browser)
  try {
    await garanteACena(gmScreen)
    await playerScreen.getByRole('button', { name: 'Mesa', exact: true }).click()
    await poeNaFila(gmScreen, echo)
    await closeTheTracker(gmScreen)
    await expect(playerScreen.getByText(echo).first()).toBeVisible()

    await gmScreen
      .getByRole('button', { name: 'Encerrar cena' })
      .filter({ visible: true })
      .first()
      .click()

    await expect(playerScreen.getByText(echo)).toHaveCount(0)
    await openTheTracker(gmScreen)
    await expect(
      gmScreen.locator('#tracker-drawer').getByText(echo).first(),
      'a fila sumiu da tela do MESTRE: isso é apagar, não redigir',
    ).toBeVisible()
    await closeTheTracker(gmScreen)

    // E volta pelo mesmo caminho: a fila estava guardada o tempo todo.
    await garanteACena(gmScreen)
    await expect(playerScreen.getByText(echo).first()).toBeVisible()

    await tiraDaFila(gmScreen, echo)
  } finally {
    await closes()
  }
})

/**
 * O TABULEIRO atravessa a mesa, e a CORTINA o esconde sem apagá-lo. Os dois no
 * mesmo caso porque são o mesmo mecanismo — estado que sai REDIGIDO por papel —
 * e montar um tabuleiro custa caro demais para pagar duas vezes.
 */
test('o tabuleiro que o mestre abre aparece na tela do jogador, e a cortina o esconde', async ({
  browser,
}) => {
  test.setTimeout(90_000)
  const place = `Cripta de teste ${Date.now()}`
  const theBoard = new RegExp(`Tabuleiro · ${place}`)
  const { telaDoMestre: gmScreen, telaDoJogador: playerScreen, fecha: closes } = await asDuasTelas(browser)
  try {
    await playerScreen.getByRole('button', { name: 'Tabuleiro', exact: true }).click()

    // O terreno pode ter sobrado de outra corrida: encerrar antes é o que faz o
    // caso medir o tabuleiro DELE e não o ambiente.
    const end = gmScreen.getByRole('button', { name: 'Encerrar o tabuleiro' })
    if (await end.count()) {
      await end.first().click()
      await gmScreen.getByRole('dialog').getByRole('button', { name: 'Encerrar' }).click()
    }
    await gmScreen.getByRole('button', { name: 'Abrir tabuleiro' }).first().click()
    await gmScreen.locator('#new-place-field').fill(place)
    await gmScreen.getByRole('dialog').getByRole('button', { name: 'Abrir' }).click()
    await expect(gmScreen.getByRole('region', { name: theBoard })).toBeVisible()

    await expect(
      playerScreen.getByRole('region', { name: theBoard }),
      'o tabuleiro do mestre não chegou à mesa',
    ).toBeVisible()

    // A CORTINA: o tabuleiro continua existindo para o mestre e a mesa vê o
    // pano. É a metade que separa "escondi" de "apaguei".
    await gmScreen.getByRole('button', { name: 'Fechar a cortina' }).first().click()
    await expect(playerScreen.getByRole('region', { name: 'Cortina' })).toBeVisible()
    await expect(playerScreen.getByRole('region', { name: theBoard })).toHaveCount(0)
    await expect(
      gmScreen.getByRole('region', { name: theBoard }),
      'a cortina apagou o tabuleiro do MESTRE',
    ).toBeVisible()

    await gmScreen.getByRole('button', { name: 'Abrir a cortina para a mesa' }).first().click()
    await expect(playerScreen.getByRole('region', { name: theBoard })).toBeVisible()

    await gmScreen.getByRole('button', { name: 'Encerrar o tabuleiro' }).first().click()
    await gmScreen.getByRole('dialog').getByRole('button', { name: 'Encerrar' }).click()
    await expect(playerScreen.getByRole('region', { name: theBoard })).toHaveCount(0)
  } finally {
    await closes()
  }
})

/**
 * O DANO DO MESTRE CHEGA NA FICHA QUE O JOGADOR ESTÁ OLHANDO.
 *
 * A superfície "Ficha" não é região do stream — a ficha é sete painéis
 * computados, e recomputá-los a cada tique custaria o preço mais caro da página
 * para descobrir que nada mudou. O que o servidor manda é um SINAL de uma linha
 * (`sheet_version`), e quem repede a ficha é o cliente.
 *
 * As duas metades importam, e a segunda é a que o desenho podia ter perdido:
 * a ficha atualiza, E o jogador continua na seção em que estava. O servidor não
 * sabe qual é — ela viaja na query dos comandos da ficha, e este stream abriu
 * antes de qualquer clique —, então quem a guarda é o sinal `sheet_tab`. Sem ele
 * o repedido devolveria a aba padrão, e quem estivesse lendo Combate no meio de
 * um turno seria jogado de volta para a primeira seção a cada golpe recebido.
 */
test('o dano do mestre chega na ficha do jogador, na seção em que ele está', async ({
  browser,
}) => {
  test.setTimeout(90_000)
  const { telaDoMestre: gmScreen, telaDoJogador: playerScreen, fecha: closes } = await asDuasTelas(browser)
  try {
    await garanteACena(gmScreen)
    await openTheTracker(gmScreen)
    await gmScreen.getByRole('button', { name: 'Adicionar grupo' }).click()

    // O NOME do personagem deste jogador, lido na superfície MESA — que é onde
    // ele está antes de o jogador abrir a ficha, e é o nome que o botão "Ferir
    // …" da gaveta do mestre carrega.
    const initiativeHeader = playerScreen.locator('h2', { hasText: '·' }).first()
    await expect(initiativeHeader).toBeVisible()
    const pcName = (await initiativeHeader.innerText()).split('·').pop()?.trim() ?? ''
    expect(pcName, 'não achei o nome do personagem do jogador').not.toBe('')

    await playerScreen.getByRole('button', { name: 'Ficha', exact: true }).click()
    // Uma seção que NÃO é a que abre: é ela que prova que o remendo respeita
    // onde a pessoa está.
    await playerScreen.getByRole('button', { name: 'Combate' }).click()
    await expect(playerScreen.getByRole('heading', { name: 'Combate' })).toBeVisible()

    // A barra da ficha é `aria-hidden` de propósito — o que se LÊ é a fração.
    const sheetHP = async () => {
      const text = await playerScreen.locator('#sheet-scene').innerText()
      return text.match(/\d+\/\d+/)?.[0] ?? ''
    }
    await expect.poll(sheetHP).toMatch(/\d+\/\d+/)
    const earlier = await sheetHP()

    // O mestre fere O PERSONAGEM DESTE JOGADOR, e não o primeiro da fila: a
    // ordem sai de um d20, então mirar "o primeiro" editaria a ficha de outra
    // pessoa e o caso passaria a afirmar nada.
    await gmScreen
      .locator('#tracker-drawer')
      .getByRole('button', { name: `Ferir ${pcName}` })
      .first()
      .click()

    await expect
      .poll(sheetHP, { timeout: 8000, message: 'a ficha do jogador não soube do dano' })
      .not.toBe(earlier)
    // E ele continua em Combate: o remendo trouxe a seção dele, não a padrão.
    await expect(
      playerScreen.getByRole('heading', { name: 'Combate' }),
      'o remendo devolveu a ficha na aba padrão e tirou o jogador de onde ele estava',
    ).toBeVisible()

    await tiraDaFila(gmScreen, pcName)
  } finally {
    await closes()
  }
})
