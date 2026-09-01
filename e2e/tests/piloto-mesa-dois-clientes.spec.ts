import { type Browser, type Page, expect, test } from '@playwright/test'

/**
 * O REALTIME COM DOIS CLIENTES na Mesa em Datastar (ALE-272, fatia 10b).
 *
 * Este arquivo é o porte do `session-realtime.spec.ts`, que dirigia a sessão da
 * SPA e morre com ela. A GARANTIA não morre junto: ela é do servidor — o que o
 * mestre faz aparece na tela do jogador sem ninguém recarregar nada —, e
 * continuaria valendo com a SPA apagada e sem testemunha nenhuma.
 *
 * É a única justificativa de e2e que o guia da casa nomeia e que nada mais
 * cobre: o que existe prova que UM cliente aperta a mão com o stream. Um handler
 * que publicasse para a sala errada — ou para o próprio remetente — passaria por
 * todo teste de handler em Go e por todo caso de uma aba só: o mestre veria a
 * própria ação e o jogador não veria nada.
 *
 * SERIAL, e na sessão 5: estes casos escrevem uns sobre os outros, e a sessão 5
 * é a que o spec da SPA usava justamente para não atropelar a 4, que é a das
 * outras suítes.
 */
test.describe.configure({ mode: 'serial' })

const MESA = '/mesa/1/5'

/** As duas telas da mesma mesa, uma por papel. */
async function asDuasTelas(browser: Browser) {
  const mestre = await browser.newContext({ storageState: '.auth/user.json' })
  const jogador = await browser.newContext({ storageState: '.auth/player.json' })
  const telaDoMestre = await mestre.newPage()
  const telaDoJogador = await jogador.newPage()
  await telaDoMestre.goto(MESA)
  await telaDoJogador.goto(MESA)
  // OS DOIS CONECTADOS ANTES DE AGIR. Sem isto o caso mede uma corrida: o
  // mestre agiria antes de o stream do jogador existir, e a ausência do eco
  // seria lida como "não propagou".
  await expect(telaDoMestre.getByRole('button', { name: /^Abrir a iniciativa/ }).first()).toBeVisible()
  await expect(telaDoJogador.getByRole('group', { name: 'O que ver na sessão' })).toBeVisible()
  return {
    telaDoMestre,
    telaDoJogador,
    // `catch` na limpeza, sempre: fechar contexto pode lançar e SUBSTITUIR o
    // erro de verdade do caso (ALE-245).
    fecha: async () => {
      await mestre.close().catch(() => {})
      await jogador.close().catch(() => {})
    },
  }
}

/**
 * A gaveta da fila, que é onde o mestre comanda (ALE-269).
 *
 * IDEMPOTENTE, e não é conveniência: a gaveta é MODAL, então com ela já aberta o
 * botão que a abre fica atrás dela. O sintoma não é "não achei o botão" — é um
 * timeout de clique sobre um seletor que casou, com "subtree intercepts pointer
 * events" no log, que aponta para o lugar errado.
 */
async function abreAFila(page: Page) {
  const gaveta = page.locator('#gaveta-da-fila')
  if (await gaveta.getAttribute('open') === null) {
    await page
      .getByRole('button', { name: /^Abrir a iniciativa/ })
      .filter({ visible: true })
      .click()
  }
  await expect(gaveta).toHaveAttribute('open', '')
}

async function fechaAFila(page: Page) {
  await page.getByRole('button', { name: 'Fechar a iniciativa' }).click()
  await expect(page.locator('#gaveta-da-fila')).not.toHaveAttribute('open', '')
}

/** Põe um combatente na fila pelo gesto do mestre, e devolve o nome dele. */
async function poeNaFila(page: Page, nome: string) {
  await abreAFila(page)
  await page.getByRole('button', { name: '+ Combatente' }).click()
  await page.getByLabel('Nome', { exact: true }).fill(nome)
  await page.getByRole('button', { name: 'Acrescentar' }).click()
  await expect(page.locator('#gaveta-da-fila').getByText(nome).first()).toBeVisible()
}

async function tiraDaFila(page: Page, nome: string) {
  await abreAFila(page)
  await page.getByRole('button', { name: `Remover ${nome} da fila` }).click()
  await fechaAFila(page)
}

/** A cena precisa estar EM CURSO: sem ela o servidor não manda fila à mesa. */
async function garanteACena(page: Page) {
  const iniciar = page.getByRole('button', { name: 'Iniciar cena' }).filter({ visible: true })
  if (await iniciar.count()) {
    await iniciar.first().click()
  }
  await expect(
    page.getByRole('button', { name: 'Encerrar cena' }).filter({ visible: true }).first(),
  ).toBeVisible()
}

test('cada papel recebe a SUA cena, e não a do outro', async ({ browser }) => {
  const { telaDoMestre, telaDoJogador, fecha } = await asDuasTelas(browser)
  try {
    // O SELETOR DE SUPERFÍCIE é do jogador: ele escolhe entre a própria ficha, a
    // mesa e o tabuleiro. O mestre vê tudo junto, no palco dele.
    await expect(
      telaDoJogador.getByRole('group', { name: 'O que ver na sessão' }),
      'o jogador não recebeu a cena do jogador',
    ).toBeVisible()
    await expect(
      telaDoMestre.getByRole('group', { name: 'O que ver na sessão' }),
      'o mestre recebeu a cena do JOGADOR',
    ).toHaveCount(0)

    // E os COMANDOS são do mestre. A trava é na view (`v.Mestre` nil), então o
    // que se afirma aqui é que ela chegou íntegra até o navegador do jogador.
    await expect(
      telaDoMestre.getByRole('region', { name: 'Controles do mestre' }).first(),
      'o mestre não recebeu os controles do mestre',
    ).toBeVisible()
    await expect(
      telaDoJogador.getByRole('region', { name: 'Controles do mestre' }),
      'o jogador recebeu os controles do MESTRE',
    ).toHaveCount(0)
  } finally {
    await fecha()
  }
})

test('o que o mestre põe na fila aparece na tela do jogador', async ({ browser }) => {
  const eco = `Eco de teste ${Date.now()}`
  const { telaDoMestre, telaDoJogador, fecha } = await asDuasTelas(browser)
  try {
    await garanteACena(telaDoMestre)
    // O jogador olha a MESA — é onde a fila mora para ele.
    await telaDoJogador.getByRole('button', { name: 'Mesa', exact: true }).click()

    await poeNaFila(telaDoMestre, eco)
    await fechaAFila(telaDoMestre)

    // A tela do jogador não recarrega: o combatente chega pelo stream.
    await expect(telaDoJogador.getByText(eco).first()).toBeVisible()

    await tiraDaFila(telaDoMestre, eco)
    await expect(telaDoJogador.getByText(eco)).toHaveCount(0)
  } finally {
    await fecha()
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
  const alvo = `Condenado ${Date.now()}`
  const { telaDoMestre, telaDoJogador, fecha } = await asDuasTelas(browser)
  try {
    await garanteACena(telaDoMestre)
    await telaDoJogador.getByRole('button', { name: 'Mesa', exact: true }).click()
    await poeNaFila(telaDoMestre, alvo)

    // O CONTROLE: a linha chegou limpa antes. Sem ele, uma condição herdada de
    // outra corrida faria o caso passar sem nada ter propagado.
    await expect(telaDoJogador.getByText(alvo).first()).toBeVisible()
    await expect(telaDoJogador.getByTitle(/Abalado|-2 em testes/).first()).toHaveCount(0)

    await telaDoMestre.getByRole('button', { name: `Condições de ${alvo}` }).click()
    const dialogo = telaDoMestre.locator('#condicoes-do-combatente')
    await dialogo.getByRole('button', { name: 'Abalado', exact: true }).click()
    await telaDoMestre.keyboard.press('Escape')

    // E a tela do jogador aprende sozinha, sem recarregar. É a issue inteira.
    await expect(telaDoJogador.getByText('ABALADO').first()).toBeVisible()

    await tiraDaFila(telaDoMestre, alvo)
    await expect(telaDoJogador.getByText(alvo)).toHaveCount(0)
  } finally {
    await fecha()
  }
})

/**
 * A CENA como cortina (ALE-210): o mestre encerra e a fila SOME da mesa,
 * enquanto continua inteira na tela dele.
 *
 * As duas metades importam. Sumir da mesa é a regra; CONTINUAR na tela do
 * mestre é o que separa "redigi o que vai para a mesa" de "apaguei a fila" — e a
 * segunda leitura passaria verde afirmando só a primeira.
 */
test('encerrar a cena tira a fila da mesa sem tirá-la do mestre', async ({ browser }) => {
  const eco = `Cortina de teste ${Date.now()}`
  const { telaDoMestre, telaDoJogador, fecha } = await asDuasTelas(browser)
  try {
    await garanteACena(telaDoMestre)
    await telaDoJogador.getByRole('button', { name: 'Mesa', exact: true }).click()
    await poeNaFila(telaDoMestre, eco)
    await fechaAFila(telaDoMestre)
    await expect(telaDoJogador.getByText(eco).first()).toBeVisible()

    await telaDoMestre
      .getByRole('button', { name: 'Encerrar cena' })
      .filter({ visible: true })
      .first()
      .click()

    await expect(telaDoJogador.getByText(eco)).toHaveCount(0)
    await abreAFila(telaDoMestre)
    await expect(
      telaDoMestre.locator('#gaveta-da-fila').getByText(eco).first(),
      'a fila sumiu da tela do MESTRE: isso é apagar, não redigir',
    ).toBeVisible()
    await fechaAFila(telaDoMestre)

    // E volta pelo mesmo caminho: a fila estava guardada o tempo todo.
    await garanteACena(telaDoMestre)
    await expect(telaDoJogador.getByText(eco).first()).toBeVisible()

    await tiraDaFila(telaDoMestre, eco)
  } finally {
    await fecha()
  }
})

/**
 * O TABULEIRO atravessa a mesa (ALE-124), e a CORTINA o esconde sem apagá-lo
 * (ALE-202). Os dois no mesmo caso porque são o mesmo mecanismo — estado que sai
 * REDIGIDO por papel — e montar um tabuleiro custa caro demais para pagar duas
 * vezes.
 */
test('o tabuleiro que o mestre abre aparece na tela do jogador, e a cortina o esconde', async ({
  browser,
}) => {
  test.setTimeout(90_000)
  const lugar = `Cripta de teste ${Date.now()}`
  const oMapa = new RegExp(`Tabuleiro · ${lugar}`)
  const { telaDoMestre, telaDoJogador, fecha } = await asDuasTelas(browser)
  try {
    await telaDoJogador.getByRole('button', { name: 'Tabuleiro', exact: true }).click()

    // O terreno pode ter sobrado de outra corrida: encerrar antes é o que faz o
    // caso medir o tabuleiro DELE e não o ambiente.
    const encerrar = telaDoMestre.getByRole('button', { name: 'Encerrar o tabuleiro' })
    if (await encerrar.count()) {
      await encerrar.first().click()
      await telaDoMestre.getByRole('dialog').getByRole('button', { name: 'Encerrar' }).click()
    }
    await telaDoMestre.getByRole('button', { name: 'Abrir tabuleiro' }).first().click()
    await telaDoMestre.locator('#novo-lugar').fill(lugar)
    await telaDoMestre.getByRole('dialog').getByRole('button', { name: 'Abrir' }).click()
    await expect(telaDoMestre.getByRole('region', { name: oMapa })).toBeVisible()

    await expect(
      telaDoJogador.getByRole('region', { name: oMapa }),
      'o tabuleiro do mestre não chegou à mesa',
    ).toBeVisible()

    // A CORTINA: o tabuleiro continua existindo para o mestre e a mesa vê o
    // pano. É a metade que separa "escondi" de "apaguei".
    await telaDoMestre.getByRole('button', { name: 'Fechar a cortina' }).first().click()
    await expect(telaDoJogador.getByRole('region', { name: 'Cortina' })).toBeVisible()
    await expect(telaDoJogador.getByRole('region', { name: oMapa })).toHaveCount(0)
    await expect(
      telaDoMestre.getByRole('region', { name: oMapa }),
      'a cortina apagou o tabuleiro do MESTRE',
    ).toBeVisible()

    await telaDoMestre.getByRole('button', { name: 'Abrir a cortina para a mesa' }).first().click()
    await expect(telaDoJogador.getByRole('region', { name: oMapa })).toBeVisible()

    await telaDoMestre.getByRole('button', { name: 'Encerrar o tabuleiro' }).first().click()
    await telaDoMestre.getByRole('dialog').getByRole('button', { name: 'Encerrar' }).click()
    await expect(telaDoJogador.getByRole('region', { name: oMapa })).toHaveCount(0)
  } finally {
    await fecha()
  }
})

/**
 * O DANO DO MESTRE CHEGA NA FICHA QUE O JOGADOR ESTÁ OLHANDO (ALE-275).
 *
 * A superfície "Ficha" não é região do stream — a ficha é sete painéis
 * computados, e recomputá-los a cada tique custaria o preço mais caro da página
 * para descobrir que nada mudou. O que o servidor manda é um SINAL de uma linha
 * (`fichaversao`), e quem repede a ficha é o cliente.
 *
 * As duas metades importam, e a segunda é a que o desenho podia ter perdido:
 * a ficha atualiza, E o jogador continua na seção em que estava. O servidor não
 * sabe qual é — ela viaja na query dos comandos da ficha, e este stream abriu
 * antes de qualquer clique —, então quem a guarda é o sinal `fichatab`. Sem ele
 * o repedido devolveria a aba padrão, e quem estivesse lendo Combate no meio de
 * um turno seria jogado de volta para a primeira seção a cada golpe recebido.
 */
test('o dano do mestre chega na ficha do jogador, na seção em que ele está', async ({
  browser,
}) => {
  test.setTimeout(90_000)
  const { telaDoMestre, telaDoJogador, fecha } = await asDuasTelas(browser)
  try {
    await garanteACena(telaDoMestre)
    await abreAFila(telaDoMestre)
    await telaDoMestre.getByRole('button', { name: 'Adicionar grupo' }).click()

    // O NOME do personagem deste jogador, lido na superfície MESA — que é onde
    // ele está antes de o jogador abrir a ficha, e é o nome que o botão "Ferir
    // …" da gaveta do mestre carrega.
    const cabecalhoDaIniciativa = telaDoJogador.locator('h2', { hasText: '·' }).first()
    await expect(cabecalhoDaIniciativa).toBeVisible()
    const nomeDoPc = (await cabecalhoDaIniciativa.innerText()).split('·').pop()?.trim() ?? ''
    expect(nomeDoPc, 'não achei o nome do personagem do jogador').not.toBe('')

    await telaDoJogador.getByRole('button', { name: 'Ficha', exact: true }).click()
    // Uma seção que NÃO é a que abre: é ela que prova que o remendo respeita
    // onde a pessoa está.
    await telaDoJogador.getByRole('button', { name: 'Combate' }).click()
    await expect(telaDoJogador.getByRole('heading', { name: 'Combate' })).toBeVisible()

    // A barra da ficha é `aria-hidden` de propósito — o que se LÊ é a fração.
    const oPVdaFicha = async () => {
      const texto = await telaDoJogador.locator('#cena-ficha').innerText()
      return texto.match(/\d+\/\d+/)?.[0] ?? ''
    }
    await expect.poll(oPVdaFicha).toMatch(/\d+\/\d+/)
    const antes = await oPVdaFicha()

    // O mestre fere O PERSONAGEM DESTE JOGADOR, e não o primeiro da fila: a
    // ordem sai de um d20, então mirar "o primeiro" editaria a ficha de outra
    // pessoa e o caso passaria a afirmar nada.
    await telaDoMestre
      .locator('#gaveta-da-fila')
      .getByRole('button', { name: `Ferir ${nomeDoPc}` })
      .first()
      .click()

    await expect
      .poll(oPVdaFicha, { timeout: 8000, message: 'a ficha do jogador não soube do dano' })
      .not.toBe(antes)
    // E ele continua em Combate: o remendo trouxe a seção dele, não a padrão.
    await expect(
      telaDoJogador.getByRole('heading', { name: 'Combate' }),
      'o remendo devolveu a ficha na aba padrão e tirou o jogador de onde ele estava',
    ).toBeVisible()

    await tiraDaFila(telaDoMestre, nomeDoPc)
  } finally {
    await fecha()
  }
})
