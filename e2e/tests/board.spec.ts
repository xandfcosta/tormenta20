import { expect, type Page, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { openTheBoard, disposableTable, putATokenOnTheMap } from './support/table'

/**
 * O TABULEIRO da Mesa em Datastar.
 *
 * Por que E2E, e cada caso aqui tem de justificar o gasto: tudo que o SERVIDOR
 * decide já está preso em Go — quem pode pintar, quem pode marcar, a letra do
 * marcador, o deslocamento da peça que nasce. O que sobra é o que só um
 * navegador tem: LEIAUTE REAL (o plano muda de tamanho de verdade quando o zoom
 * muda), EMPILHAMENTO (um elemento coberto por outro não aparece em HTML
 * nenhum) e o REMENDO DO SSE chegando por cima de um estado que mora no
 * cliente. Nenhum caso é "a jornada do mestre" — jornada é mais barata e mais
 * firme como teste de integração.
 *
 * TABULEIRO DESCARTÁVEL: abrir tabuleiro numa sessão compartilhada mexeria em
 * estado de seis specs, e o resto de um deles derruba a suíte do dia seguinte
 * por um caminho que não aponta para lugar nenhum. Cada caso cria a própria
 * campanha e a apaga no fim; apagar a campanha leva a sessão e o tabuleiro
 * junto.
 */

test.use({ storageState: '.auth/user.json' })

/**
 * A CAMADA de clique, e não qualquer botão com aquele nome.
 *
 * São TRÊS camadas empilhadas — pintar, marcar e mover — e só uma está visível
 * por vez: achá-las por posição (`.first()`) pega a errada assim que a
 * ferramenta muda, e o erro sai como "elemento não visível", que não aponta
 * para a causa. Papel + nome também não basta, porque o nome do botão do trilho
 * é PREFIXO do nome da camada ("Mover" casa com o trilho E com "Mover Ogro —
 * escolha a casa"). A classe é o que define a camada.
 */
const camadaDe = (page: Page, gesture: RegExp) =>
  page.locator('.board-squares').and(page.getByRole('button', { name: gesture }))

/**
 * A FERRAMENTA no trilho, e não qualquer botão com aquele nome.
 *
 * `exact: true` não serve: o trilho diz a tecla no nome acessível ("Marcar
 * (tecla 4)"), de propósito, para quem navega por teclado descobrir o atalho.
 * Perguntar DENTRO do trilho resolve o prefixo e o número ao mesmo tempo.
 */
const ferramenta = (page: Page, label: string) =>
  page.getByRole('navigation', { name: 'Ferramentas do mapa' }).getByRole('button', { name: label })

/**
 * Quem guarda o enquadramento é a CENA, a janela que recorta — e não o palco:
 * num plano infinito não há `scrollWidth` para o navegador prender.
 */
const quadrado = (page: Page) =>
  page.locator('.board-scene').evaluate((e) => getComputedStyle(e).getPropertyValue('--quadrado').trim())

/**
 * O ZOOM SOBREVIVE AO REMENDO.
 *
 * O enquadramento não está no HTML de propósito: ele vive em `--quadrado`, no
 * cliente, para o servidor poder redesenhar as peças sem que o mestre perca
 * onde estava olhando. É uma afirmação sobre o que acontece DEPOIS de um patch
 * do SSE chegar, e nenhuma camada abaixo do navegador tem como testemunhá-la —
 * um teste de handler vê o HTML novo e não vê o estado que sobreviveu a ele.
 *
 * O CONTROLE vem antes da asserção: o remendo tem de ter ACONTECIDO. Sem ele,
 * "o zoom não mudou" seria igualmente verdade numa cena que não recebeu nada, e
 * o caso passaria verde sobre um stream morto.
 */
test('o zoom e a janela sobrevivem ao remendo do servidor', async ({ page }) => {
  const { mesa: table, apagar: remove } = await disposableTable(page)
  try {
    await openTheBoard(page, table)

    await page.getByRole('button', { name: 'Aproximar o mapa' }).click()
    await page.getByRole('button', { name: 'Aproximar o mapa' }).click()
    const zoomBefore = await quadrado(page)
    expect(zoomBefore, 'o zoom não saiu do padrão — não há o que sobreviver').not.toBe('44px')

    // Uma mudança que vem DO SERVIDOR e redesenha a região do mapa.
    await ferramenta(page, 'Difícil').click()
    await camadaDe(page, /Pintar terreno/).click({ position: { x: 60, y: 60 } })

    // O CONTROLE: o remendo chegou e mudou a cena.
    //
    // O seletor é `.board-terrain.board-difficult` e não só o segundo: a
    // AMOSTRA do crachá no trilho carrega a mesma classe de espécie, e o
    // seletor curto acharia dois elementos — um deles um quadradinho de
    // legenda que existe desde antes do clique.
    await expect(
      page.locator('.board-terrain.board-difficult'),
      'o terreno não apareceu — o remendo não aconteceu e o resto não mede nada',
    ).toHaveCount(1)

    expect(await quadrado(page), 'o remendo levou o zoom junto').toBe(zoomBefore)
  } finally {
    await remove()
  }
})

/**
 * A CONTA DO CLIQUE ACOMPANHA O ZOOM.
 *
 * O quadrado clicado sai do PONTO do clique dividido pelo tamanho da casa, e o
 * tamanho da casa é o mesmo número que o zoom move. Se um dos dois andar sem o
 * outro, o mestre pinta uma casa e outra acende.
 *
 * A asserção é um INVARIANTE GEOMÉTRICO e não uma conta refeita: a casa que
 * apareceu tem de CONTER o ponto clicado. Recalcular `floor(x / quadrado)` no
 * teste seria comparar a implementação consigo mesma — e passaria verde com as
 * duas erradas do mesmo jeito.
 */
test('depois de aproximar, a casa pintada é a que estava sob o dedo', async ({ page }) => {
  const { mesa: table, apagar: remove } = await disposableTable(page)
  try {
    await openTheBoard(page, table)
    // Aproxima ao máximo: com a casa grande, um erro de conversão de um quadrado
    // já sai da caixa e a asserção o pega. No zoom padrão um erro pequeno pode
    // cair dentro da mesma casa por sorte.
    const more = page.getByRole('button', { name: 'Aproximar o mapa' })
    while (!(await more.isDisabled())) await more.click()

    await page.getByRole('button', { name: 'Camuflagem' }).click()
    const squares = camadaDe(page, /Pintar terreno/)
    const target = { x: 150, y: 110 }
    await squares.click({ position: target })

    const painted = page.locator('.board-terrain.board-concealment')
    await expect(painted, 'nada foi pintado').toHaveCount(1)

    const layerBox = (await squares.boundingBox())!
    const squareBox = (await painted.boundingBox())!
    const pointX = layerBox.x + target.x
    const pointY = layerBox.y + target.y

    expect(
      pointX >= squareBox.x && pointX <= squareBox.x + squareBox.width,
      `o clique em x=${pointX} caiu fora da casa pintada (${squareBox.x}–${squareBox.x + squareBox.width})`,
    ).toBe(true)
    expect(
      pointY >= squareBox.y && pointY <= squareBox.y + squareBox.height,
      `o clique em y=${pointY} caiu fora da casa pintada (${squareBox.y}–${squareBox.y + squareBox.height})`,
    ).toBe(true)
  } finally {
    await remove()
  }
})

/**
 * DEPOIS DE ARRASTAR A VISTA, A CASA CONTINUA SENDO A QUE ESTÁ SOB O DEDO.
 *
 * É o mesmo invariante do caso acima, e não uma repetição: ali é o ZOOM
 * entrando na divisão, aqui é a JANELA entrando na soma.
 *
 * Por que e2e: o deslocamento é um `transform` de CSS sobre um plano de tamanho
 * ZERO, e a conta do clique é `offsetX + $viewport_x` num elemento IRMÃO desse
 * plano. Nenhuma camada abaixo do navegador tem geometria para testemunhar
 * isso — em jsdom todo elemento mede zero, e a asserção passaria verde com as
 * duas contas erradas.
 *
 * A asserção é geométrica e não aritmética, pela mesma razão de lá: recalcular
 * `floor((x + vista) / quadrado)` aqui seria comparar a implementação consigo
 * mesma.
 */
test('depois de arrastar a vista, a casa pintada é a que estava sob o dedo', async ({ page }) => {
  const { mesa: table, apagar: remove } = await disposableTable(page)
  try {
    await openTheBoard(page, table)

    // ARRASTA A VISTA um bom pedaço, com a ferramenta da mão. O deslocamento é
    // deliberadamente NÃO múltiplo do quadrado: um múltiplo esconderia um erro
    // de fase, porque a casa certa e a errada cairiam no mesmo lugar da grade.
    await ferramenta(page, 'Arrastar a vista').click()
    const mao = page.locator('.board-viewport')
    const handBox = (await mao.boundingBox())!
    await page.mouse.move(handBox.x + 400, handBox.y + 300)
    await page.mouse.down()
    await page.mouse.move(handBox.x + 173, handBox.y + 191, { steps: 8 })
    await page.mouse.up()

    // O CONTROLE: a vista ANDOU. Sem ele, "a casa está certa" é verdade também
    // numa tela onde o arrasto não fez nada — que é o verde mais caro que existe.
    const view = await page
      .locator('.board-scene')
      .evaluate((e) => getComputedStyle(e).getPropertyValue('--vista-x').trim())
    expect(view, 'a vista não saiu do lugar — o arrasto não aconteceu e o resto não mede nada').not.toBe('0px')

    await ferramenta(page, 'Camuflagem').click()
    const squares = camadaDe(page, /Pintar terreno/)
    const target = { x: 260, y: 180 }
    await squares.click({ position: target })

    const painted = page.locator('.board-terrain.board-concealment')
    await expect(painted, 'nada foi pintado').toHaveCount(1)

    const layerBox = (await squares.boundingBox())!
    const squareBox = (await painted.boundingBox())!
    const pointX = layerBox.x + target.x
    const pointY = layerBox.y + target.y

    expect(
      pointX >= squareBox.x && pointX <= squareBox.x + squareBox.width,
      `com a vista em ${view}, o clique em x=${pointX} caiu fora da casa pintada ` +
        `(${squareBox.x}–${squareBox.x + squareBox.width})`,
    ).toBe(true)
    expect(
      pointY >= squareBox.y && pointY <= squareBox.y + squareBox.height,
      `o clique em y=${pointY} caiu fora da casa pintada ` +
        `(${squareBox.y}–${squareBox.y + squareBox.height})`,
    ).toBe(true)
  } finally {
    await remove()
  }
})

/**
 * O DESENHO DA MEDIDA CABE NO SVG QUE O CARREGA — o guarda de um recorte que não
 * acusa.
 *
 * O `<svg>` MAIS EXTERNO recorta pelo viewport dele, e `overflow: visible` não
 * levanta esse recorte. Com a régua e o gabarito dentro de um plano de tamanho
 * ZERO, o viewport vira 0×0 e os dois PARAM DE APARECER — com o `<path>` no DOM,
 * com a caixa certa no lugar certo, com o `fill` certo e com `display: block`.
 * Nada acusa.
 *
 * O `toBeVisible` do Playwright também não pega: o `<path>` TEM caixa. Então a
 * asserção é a que descreve o defeito: o desenho tem de estar DENTRO da caixa do
 * SVG. Com o viewport em 0×0 (ou nos 300×150 intrínsecos de um `<svg>` sem
 * `width`/`height`, que foi a segunda forma do mesmo erro), ele não está.
 *
 * Por que e2e: é geometria de SVG dentro de um `transform` de CSS. Nada abaixo do
 * navegador tem viewport para recortar.
 */
test('o gabarito desenhado cabe dentro do SVG que o carrega', async ({ page }) => {
  const { mesa: table, apagar: remove } = await disposableTable(page)
  try {
    await openTheBoard(page, table)

    await ferramenta(page, 'Gabarito').click()
    await camadaDe(page, /Pôr o gabarito/).click({ position: { x: 300, y: 200 } })

    const svg = page.locator('.board-measure-back')
    const drawing = svg.locator('path')

    // O CONTROLE: o servidor respondeu e há desenho. Sem ele, "cabe no SVG" seria
    // verdade sobre um `<path>` vazio, que é o verde que este caso existe para
    // não dar.
    await expect(drawing, 'o gabarito não foi desenhado — não há o que medir').toHaveAttribute('d', /\S/)

    const svgBox = (await svg.boundingBox())!
    const drawingBox = (await drawing.boundingBox())!
    expect(
      drawingBox.x >= svgBox.x &&
        drawingBox.x + drawingBox.width <= svgBox.x + svgBox.width &&
        drawingBox.y >= svgBox.y &&
        drawingBox.y + drawingBox.height <= svgBox.y + svgBox.height,
      `o desenho está em (${drawingBox.x},${drawingBox.y},${drawingBox.width}×${drawingBox.height}) ` +
        `e o SVG em (${svgBox.x},${svgBox.y},${svgBox.width}×${svgBox.height}): ` +
        `o viewport do svg recorta o gabarito, e ninguém na mesa o vê`,
    ).toBe(true)
  } finally {
    await remove()
  }
})

/**
 * A JANELA VAI ATRÁS DO FOCO.
 *
 * A rolagem nativa trazia o elemento focado para a vista de graça, e ela saiu
 * com a moldura: a cena recorta com `overflow: hidden` e a página não rola,
 * então não existe mais ancestral rolável — o navegador TENTA e não tem o que
 * rolar. Sem o conserto, focar uma peça distante deixa tudo onde estava, e quem
 * navega por teclado alcança uma peça que nunca vai conseguir ver.
 *
 * Por que e2e: são FOCO e GEOMETRIA REAL ao mesmo tempo, num elemento cuja
 * posição vem de um `transform` de CSS. Em jsdom todo retângulo é zero e a
 * asserção passaria verde sobre nada.
 *
 * O CONTROLE é a metade que importa: a peça tem de estar FORA antes. Sem ele,
 * "a peça está dentro" é verdade também numa janela que nunca saiu do lugar.
 */
test('a janela vai atrás do foco quando a peça está fora dela', async ({ page }) => {
  const { mesa: table, apagar: remove } = await disposableTable(page)
  try {
    await openTheBoard(page, table)
    await putATokenOnTheMap(page)

    const scene = page.locator('.board-scene')
    const token = page.locator('.board-token')

    // Arrasta a vista para BEM longe da peça, com a ferramenta da mão.
    await ferramenta(page, 'Arrastar a vista').click()
    const handBox = (await page.locator('.board-viewport').boundingBox())!
    await page.mouse.move(handBox.x + handBox.width - 40, handBox.y + handBox.height - 40)
    await page.mouse.down()
    await page.mouse.move(handBox.x + 20, handBox.y + 20, { steps: 10 })
    await page.mouse.up()

    const insideWindow = async () => {
      const j = (await scene.boundingBox())!
      const p = (await token.boundingBox())!
      return p.x >= j.x && p.x + p.width <= j.x + j.width && p.y >= j.y && p.y + p.height <= j.y + j.height
    }

    // O CONTROLE: a peça ficou FORA da janela.
    expect(await insideWindow(), 'a peça continuou visível — o arrasto não afastou nada e o resto não mede').toBe(false)

    await token.focus()

    expect(
      await insideWindow(),
      'a peça focada continuou fora da janela: quem navega por teclado pode alcançá-la e nunca vê-la',
    ).toBe(true)
  } finally {
    await remove()
  }
})

/**
 * SHIFT + ARRASTO ENCHE O RETÂNGULO.
 *
 * O gesto é browser puro e não tem onde ser medido mais barato: `Shift` decidido
 * no `pointerdown`, `setPointerCapture`, o laço posicionado por uma expressão de
 * CSS sobre um plano que um `transform` desloca, e a rota só saindo no
 * `pointerup`. Um teste de handler prova que a ROTA enche a área — e prova zero
 * sobre o gesto que a chama.
 *
 * O CONTROLE é a metade que importa: sem o `Shift`, o mesmo arrasto tem de
 * pintar o TRAÇO e não o retângulo. Sem ele, "12 casas" seria verdade também
 * para um app que ignorasse a tecla e enchesse sempre.
 */
test('Shift + arrasto enche o retângulo, e sem Shift continua traço', async ({ page }) => {
  const { mesa: table, apagar: remove } = await disposableTable(page)
  try {
    await openTheBoard(page, table)
    await ferramenta(page, 'Difícil').click()

    const squares = camadaDe(page, /Pintar terreno/)
    const box = (await squares.boundingBox())!
    const dragStart = { x: box.x + 120, y: box.y + 120 }
    const dragEnd = { x: dragStart.x + 120, y: dragStart.y + 90 }

    // O CONTROLE: o MESMO arrasto sem Shift pinta uma linha, não uma área.
    await page.mouse.move(dragStart.x, dragStart.y)
    await page.mouse.down()
    await page.mouse.move(dragEnd.x, dragEnd.y, { steps: 10 })
    await page.mouse.up()
    const strokeOf = await page.locator('.board-terrain.board-difficult').count()

    // E agora COM Shift, num pedaço virgem do plano.
    const deB = { x: box.x + 420, y: box.y + 120 }
    const ateB = { x: deB.x + 120, y: deB.y + 90 }
    await page.keyboard.down('Shift')
    await page.mouse.move(deB.x, deB.y)
    await page.mouse.down()
    await page.mouse.move(ateB.x, ateB.y, { steps: 6 })
    // O LAÇO tem de estar na tela ENQUANTO o dedo segura: é a promessa visual do
    // gesto, e sem ela a pessoa arrasta no escuro.
    await expect(page.locator('.board-lasso'), 'o laço não apareceu durante o arrasto').toBeVisible()
    await page.mouse.up()
    await page.keyboard.up('Shift')

    await expect
      .poll(() => page.locator('.board-terrain.board-difficult').count(), {
        message: 'o retângulo não encheu a área',
      })
      .toBeGreaterThan(strokeOf * 2)

    // E o laço some quando o dedo solta — ele é intenção, não resultado.
    await expect(page.locator('.board-lasso'), 'o laço ficou na tela depois de soltar').toBeHidden()
  } finally {
    await remove()
  }
})

/**
 * O MARCADOR É ALCANÇÁVEL, e este caso existe porque ele já não era.
 *
 * As camadas de clique cobrem o plano inteiro e vêm depois no DOM; a de MOVER é
 * a ativa por padrão. O marcador ficava debaixo dela e o clique nunca chegava —
 * com o HTML inteiro correto, os guardas de handler verdes, e nada em lugar
 * nenhum dizendo que havia um elemento coberto. Só o navegador vê isso.
 */
test('o marcador continua clicável por baixo da camada de mover', async ({ page }) => {
  const { mesa: table, apagar: remove } = await disposableTable(page)
  try {
    await openTheBoard(page, table)
    await putATokenOnTheMap(page)

    await ferramenta(page, 'Marcar').click()
    await camadaDe(page, /Marcar um lugar/).click({ position: { x: 90, y: 90 } })
    const marker = page.locator('.board-marker')
    await expect(marker, 'o marcador não nasceu').toHaveCount(1)

    // De volta ao padrão: é assim que a ferramenta fica enquanto o mestre joga,
    // e era exatamente aí que o marcador ficava inalcançável.
    await ferramenta(page, 'Marcar').click()

    // O CONTROLE do empilhamento: a camada que cobria o marcador tem de estar NO
    // AR. Sem ele, "o clique chegou" é verdade num palco onde nada cobria nada —
    // e o caso passa verde até com o `z-index` removido.
    await expect(
      camadaDe(page, /Mover/),
      'a camada de mover não está no ar — o caso não enfrenta o empilhamento que veio medir',
    ).toBeVisible()

    await marker.click()
    await expect(
      page.locator('.board-marker-actions'),
      'o clique não chegou ao marcador — alguma camada o cobriu de novo',
    ).toBeVisible()

    // E o que o gesto existe para fazer: revelar para a mesa.
    await expect(marker, 'o marcador não nasceu escondido').toHaveClass(/board-marker-hidden/)
    await page.getByRole('button', { name: /^Revelar o marcador/ }).click()
    await expect(marker, 'revelar não mudou nada na tela do mestre').not.toHaveClass(/board-marker-hidden/)
  } finally {
    await remove()
  }
})

/**
 * A PRÉVIA DO ARRASTO: a seta e a distância aparecem ENQUANTO o dedo arrasta.
 *
 * POR QUE E2E: o que se mede só existe DENTRO de um gesto de ponteiro com
 * movimentos intermediários. O Go prova o que a rota `/previa/` responde e nada
 * mais — a ligação entre o `pointermove` e aquela rota mora numa string de
 * expressão do Datastar, que nenhum compilador lê e nenhum teste de handler
 * exercita. Quebrada, ela não dá erro: o atributo continua no HTML, inteiro e
 * com cara de certo, e o arrasto simplesmente não desenha nada.
 *
 * CENTRALIZAR ANTES DE ARRASTAR não é arrumação: a peça pode cair debaixo do
 * trilho de ferramentas, o `boundingBox` devolve a caixa de um elemento COBERTO
 * sem reclamar, e o `mouse.down` acerta o trilho. O gesto não acontece, e a
 * leitura vira "a prévia não funciona" — apontando para o código que está
 * certo.
 */
test('a seta e a distância aparecem durante o arrasto da peça', async ({ page }) => {
  const { mesa: table, apagar: remove } = await disposableTable(page)
  try {
    await openTheBoard(page, table)
    await putATokenOnTheMap(page)
    await page.getByLabel('Centralizar nas peças').click()

    const token = page.locator('.board-token').first()
    // O CONTROLE, antes de qualquer ausência virar conclusão: o gesto está
    // pendurado nesta peça? Sem isto, "não achei prévia" seria verdade também
    // sobre uma peça que ninguém pode arrastar.
    await expect(token, 'a peça não tem o gesto de arrasto: o canal não está aberto').toHaveAttribute(
      'data-on:pointermove__window',
      /previa/,
    )

    const box = await token.boundingBox()
    if (!box) throw new Error('a peça não tem caixa: o arrasto não tem de onde partir')
    const x = box.x + box.width / 2
    const y = box.y + box.height / 2

    await page.mouse.move(x, y)
    await page.mouse.down()
    // PASSOS INTERMEDIÁRIOS, e é isto que um `dragTo` não tem: a prévia é pedida
    // a cada CASA atravessada, então um salto direto do começo ao fim não a
    // dispara nenhuma vez.
    for (let i = 1; i <= 8; i++) {
      await page.mouse.move(x + i * 44, y)
    }

    // MEDIR NO MEIO DO GESTO: é o único instante em que a prévia existe.
    await expect(
      page.locator('.board-preview-fits'),
      'o arrasto não desenhou a seta viva',
    ).toHaveAttribute('d', /^M /)
    await expect(
      page.locator('.board-measure-front text').filter({ hasText: /m$/ }).first(),
      'a seta viva não diz a distância em metros',
    ).toBeVisible()

    await page.mouse.up()
    // E A PRÉVIA SOME ao soltar: sobrevivendo, ela ficaria por cima da seta de
    // verdade, com o mesmo formato e outra medida — dois caminhos na tela e
    // nenhum jeito de saber qual vale.
    await expect(
      page.locator('.board-preview-fits'),
      'a seta viva sobreviveu ao soltar',
    ).toHaveAttribute('d', '')
  } finally {
    await remove()
  }
})

/**
 * O PAINEL DE VERBOS CABE NO TELEFONE, COM ACERVO.
 *
 * Por que e2e: o que estoura é LARGURA REAL de texto renderizado. O servidor não
 * mede caixa — ele escreve "Lugares da campanha · 3" e não sabe quanto aquilo dá
 * numa janela de 390. O defeito é exatamente a soma das larguras.
 *
 * O ESTADO é o assunto, e é por isso que o caso semeia lugares: o painel sem
 * acervo não tem o botão largo, e medi-lo assim é medir outro painel — foi essa
 * a lacuna que deixou o defeito viver, com a cena nas listas dos guardas e o
 * estado que a quebra em lugar nenhum.
 */
test('o painel de verbos cabe a 390px com a campanha tendo acervo', async ({ page }) => {
  const { mesa: table, apagar: remove } = await disposableTable(page)
  try {
    const campaign = table.split('/')[2]
    // O ACERVO pela porta de verdade — a mesma que a aba de lugares usa. Três
    // basta: o que muda a largura é o botão EXISTIR e a contagem ter dígito.
    for (const label of ['Taverna do E2E', 'Cripta do E2E', 'Ruínas do E2E']) {
      const created = await page.request.post(`/campanhas/${campaign}/lugares/novo`, {
        form: { name: label, ground: 'crypt' },
      })
      expect(created.ok(), `semear o lugar ${label}: ${created.status()}`).toBeTruthy()
    }

    await openTheBoard(page, table)
    await page.setViewportSize({ width: 390, height: 844 })
    await page.waitForTimeout(300)

    // O CONTROLE, e sem ele o caso não mede nada: se o botão do acervo não
    // estiver na tela, o painel medido é o estreito, e o guarda passa verde
    // sobre o painel que nunca quebrou.
    const collection = page.locator('.board-scene-verbs button').filter({ hasText: /Lugares|3/ })
    await expect(
      collection.first(),
      'o botão do acervo não está no painel — o caso mediria um painel sem o item que o estoura',
    ).toBeVisible()

    await expectDentroDaJanela(page)
  } finally {
    await remove()
  }
})

/**
 * A SEGUNDA CAMADA do menu da peça só aparece quando pedida.
 *
 * Ela é POPOVER NATIVO, e o mecanismo é o que este caso protege: como camada
 * `absolute` ela passava da borda da janela a 844×390, com quatro dos seis modos
 * inalcançáveis, e `position: fixed` não resolve porque o plano tem `transform`,
 * que vira bloco de contenção.
 *
 * O que se mede é o FECHADO: um popover escondido não pode deixar botão no
 * caminho do teclado. São seis por peça, e dez peças dariam sessenta paradas de
 * Tab sobre um mapa em que nenhuma se vê.
 *
 * O MENU tem de estar ABERTO na hora de medir o submenu fechado: com o menu
 * fechado, o submenu é invisível pela herança do pai (`display:none`) e o caso
 * passa verde com ele sabotado.
 *
 * Por que e2e: `display` computado, a top layer e `checkVisibility` só existem
 * num navegador. Em jsdom todo elemento mede zero e o caso passaria verde sobre
 * os seis botões no ar.
 */
test('o submenu de duplicar só entra no caminho do teclado quando é aberto', async ({ page }) => {
  const { mesa: table, apagar: remove } = await disposableTable(page)
  try {
    await openTheBoard(page, table)
    await putATokenOnTheMap(page)

    const submenuButtons = () =>
      page.evaluate(
        () =>
          [...document.querySelectorAll('.board-token-copy button')].filter((b) =>
            (b as HTMLElement).checkVisibility(),
          ).length,
      )

    // O MENU ABERTO é a premissa: com ele fechado, o submenu seria invisível pela
    // herança do pai e o caso não mediria o popover.
    await page.locator('.board-token').first().click({ button: 'right' })
    const duplicate = page.getByRole('button', { name: /^Duplicar / })
    await expect(
      duplicate,
      'o menu da peça não abriu: sem ele o caso mede a herança do pai, não o popover',
    ).toBeVisible()

    expect(
      await submenuButtons(),
      'o submenu fechado deixou botão alcançável pelo Tab dentro de um menu aberto',
    ).toBe(0)

    // O CONTROLE POSITIVO: aberto, os seis aparecem — três de duplicar aqui e
    // três de copiar para colar.
    await duplicate.click()
    await expect
      .poll(submenuButtons, {
        message: 'o submenu não abriu: a asserção acima estaria medindo uma camada que nunca aparece',
      })
      .toBe(6)

    // E ele CABE na janela nos DOIS formatos.
    //
    // REABERTO em cada formato, e não redimensionado com ele no ar: quem
    // posiciona é o `ancora()` no `beforetoggle`, e ele não roda de novo num
    // `resize`. Medir sem reabrir mede a conta do formato ANTERIOR — deu 9px de
    // estouro num painel que, reaberto, sobra 8.
    //
    // O formato EM PÉ é também o que mantém o `tabuleiro.TopChromeRows` honesto:
    // painel mais alto ou zoom padrão menor põem a peça de volta debaixo do
    // painel de verbos, o clique direito vai para o botão de afastar, o menu não
    // abre e o `.board-token-copy` não existe para medir.
    for (const [label, w, h] of [
      ['deitado', 844, 390],
      ['em pé', 390, 844],
    ] as const) {
      await page.setViewportSize({ width: w, height: h })
      await page.evaluate(() =>
        document.querySelectorAll('[popover]').forEach((p) => {
          try {
            ;(p as HTMLElement & { hidePopover(): void }).hidePopover()
          } catch {
            // já estava fechado
          }
        }),
      )
      await page.locator('.board-token').first().click({ button: 'right' })
      await page.getByRole('button', { name: /^Duplicar / }).click()
      await page.waitForTimeout(300)
      const escaped = await page.evaluate(() => {
        const p = document.querySelector('.board-token-copy')!.getBoundingClientRect()
        return {
          abaixo: Math.round(p.bottom - window.innerHeight),
          direita: Math.round(p.right - window.innerWidth),
        }
      })
      expect(escaped.abaixo, `a camada passa ${escaped.abaixo}px do pé da janela ${label}`).toBeLessThanOrEqual(1)
      expect(escaped.direita, `a camada passa ${escaped.direita}px da borda direita ${label}`).toBeLessThanOrEqual(1)
    }
  } finally {
    await remove()
  }
})

/**
 * COPIAR guarda a decisão, e cada CTRL+V repete.
 *
 * A pergunta é feita uma vez, no menu, e o colar só executa — perguntar a cada
 * tecla mataria o valor do teclado, que é repetir (decisão do dono).
 *
 * Por que e2e: o `CTRL + V` é um atalho de TECLADO competindo com o colar do
 * navegador, e o quadrado de destino é o centro da JANELA — uma conta de pixels
 * que só existe com zoom e vista reais.
 *
 * O caso NÃO reprova a regra de PV, que já está presa em Go (`bondForMode`): o
 * que ele prende é a ligação — copiar enche a área, a tecla dispara, e o segundo
 * CTRL+V põe outro.
 */
test('copiar guarda o modo, e cada CTRL+V põe outro igual', async ({ page }) => {
  const { mesa: table, apagar: remove } = await disposableTable(page)
  try {
    await openTheBoard(page, table)
    await putATokenOnTheMap(page)

    // A FAIXA da área começa VAZIA, e este é o controle: sem ele, uma faixa que
    // aparecesse sempre passaria pelas asserções de baixo sem provar nada.
    const strip = page.locator('.board-area')
    await expect(strip, 'a faixa da área nasceu visível com a área vazia').toBeHidden()

    await page.locator('.board-token').first().click({ button: 'right' })
    await page.getByRole('button', { name: /^Duplicar / }).click()
    await page.getByRole('button', { name: /^Copiar .* para colar: com PV próprio/ }).click()

    await expect(strip, 'copiar não acendeu a faixa da área').toBeVisible()
    await expect(strip).toContainText('com PV próprio')

    const before = await page.locator('.board-token').count()
    await page.keyboard.press('Control+v')
    await expect
      .poll(() => page.locator('.board-token').count(), { message: 'o primeiro CTRL+V não colou' })
      .toBe(before + 1)
    await page.keyboard.press('Control+v')
    await expect
      .poll(() => page.locator('.board-token').count(), {
        message: 'o segundo CTRL+V não colou: a área não sobreviveu ao remendo da cena',
      })
      .toBe(before + 2)

    // ESVAZIAR é um BOTÃO e nunca o Esc: o `scene.js` mapeia Escape para "voltar"
    // e o mata no documento — medido, e o `railKeyboard` já o registra.
    await page.getByRole('button', { name: 'Esvaziar a área de transferência' }).click()
    await expect(strip, 'esvaziar não apagou a faixa').toBeHidden()
    await page.keyboard.press('Control+v')
    await page.waitForTimeout(500)
    expect(
      await page.locator('.board-token').count(),
      'o CTRL+V colou com a área vazia',
    ).toBe(before + 2)
  } finally {
    await remove()
  }
})
