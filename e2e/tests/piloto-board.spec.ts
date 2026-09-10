import { expect, type Page, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { openTheBoard, disposableTable, putATokenOnTheMap } from './support/table'

/**
 * O TABULEIRO da Mesa em Datastar (ALE-264, item 7).
 *
 * Por que E2E, e este arquivo tem de justificar cada caso porque e2e é a faixa
 * mais cara do repositório: tudo que o SERVIDOR decide já está preso em Go —
 * quem pode pintar, quem pode marcar, a letra do marcador, o deslocamento da
 * peça que nasce. O que sobra aqui é o que só um navegador tem: LEIAUTE REAL
 * (o plano muda de tamanho de verdade quando o zoom muda), EMPILHAMENTO (um
 * elemento coberto por outro não aparece em HTML nenhum) e o REMENDO DO SSE
 * chegando por cima de um estado que mora no cliente.
 *
 * Os três casos abaixo são exatamente esses três mecanismos. Nenhum deles é
 * "a jornada do mestre" — jornada é mais barata e mais firme como teste de
 * integração, e a regra da casa proíbe gastá-la aqui.
 *
 * TABULEIRO DESCARTÁVEL, e não é preciosismo: abrir tabuleiro na sessão 1 ou 4
 * mexeria em estado que seis specs compartilham, e o resto de um deles derruba
 * a suíte do dia seguinte por um caminho que não aponta para lugar nenhum
 * (está escrito no `auth.setup.ts`). Cada caso cria a própria campanha e a
 * apaga no fim; apagar a campanha leva a sessão e o tabuleiro junto.
 */

test.use({ storageState: '.auth/user.json' })

/**
 * As camadas de clique são TRÊS empilhadas — pintar, marcar e mover — e só uma
 * está visível por vez. Achá-las por posição (`.first()`) pega a errada assim
 * que a ferramenta muda, e o erro sai como "elemento não visível", que não
 * aponta para a causa. O nome acessível existe justamente para dizer qual é
 * qual, e é por ele que se pergunta.
 */
/**
 * A CAMADA de clique, e não qualquer botão com aquele nome.
 *
 * O papel + nome sozinho deixou de bastar na ALE-269, quando o trilho ganhou
 * "Mover a peça" e "Régua" para o jogador: `/Mover/` passou a casar com o botão
 * do trilho E com a camada ("Mover Ogro — escolha a casa"), e o caso morreu em
 * `strict mode violation` — que é o modo certo de descobrir isso, porque a
 * alternativa seria o clique cair no botão errado e o teste medir outra coisa.
 *
 * A classe é o que define a camada e é o que ela sempre teve; o nome sozinho é
 * um prefixo que qualquer ferramenta nova pode voltar a colidir.
 */
const camadaDe = (page: Page, gesto: RegExp) =>
  page.locator('.tabuleiro-casas').and(page.getByRole('button', { name: gesto }))

/**
 * A FERRAMENTA no trilho, e não qualquer botão com aquele nome.
 *
 * Terceira vez que um seletor por nome deixa de ser único nesta cena, e a lição
 * é sempre a mesma: nome de botão é um PREFIXO que a próxima ferramenta pode
 * colidir. Aqui o `exact: true` também parou de servir — o trilho passou a dizer
 * a tecla no nome acessível ("Marcar (tecla 4)"), de propósito, para quem navega
 * por teclado descobrir o atalho.
 *
 * Perguntar DENTRO do trilho resolve os dois: o número pode mudar de lugar e o
 * caso continua apontando para a ferramenta que ele quer.
 */
const ferramenta = (page: Page, nome: string) =>
  page.getByRole('navigation', { name: 'Ferramentas do mapa' }).getByRole('button', { name: nome })

/**
 * O `--quadrado` mudou de dono na ALE-203: o PALCO era a caixa que rolava, e ela
 * saiu junto com a moldura — num plano infinito não há `scrollWidth` para o
 * navegador prender. Quem guarda o enquadramento agora é a CENA, que é a janela
 * que recorta.
 */
const quadrado = (page: Page) =>
  page.locator('.tabuleiro-cena').evaluate((e) => getComputedStyle(e).getPropertyValue('--quadrado').trim())

/**
 * O ZOOM SOBREVIVE AO REMENDO — e esta é A aposta da fatia inteira.
 *
 * O enquadramento não está no HTML de propósito: ele vive em `--quadrado`, no
 * cliente, para o servidor poder redesenhar as peças sem que o mestre perca
 * onde estava olhando. É uma afirmação sobre o que acontece DEPOIS de um patch
 * do SSE chegar, e nenhuma camada abaixo do navegador tem como testemunhá-la —
 * um teste de handler vê o HTML novo e não vê o estado que sobreviveu a ele.
 *
 * O CONTROLE vem antes da asserção: o remendo tem de ter ACONTECIDO. Sem ele,
 * "o zoom não mudou" seria igualmente verdade numa cena que não recebeu nada, e
 * o caso passaria verde sobre um stream morto — que é a família de defeito que
 * o CLAUDE.md desta casa persegue.
 */
test('o zoom e a janela sobrevivem ao remendo do servidor', async ({ page }) => {
  const { mesa, apagar } = await disposableTable(page)
  try {
    await openTheBoard(page, mesa)

    await page.getByRole('button', { name: 'Aproximar o mapa' }).click()
    await page.getByRole('button', { name: 'Aproximar o mapa' }).click()
    const zoomAntes = await quadrado(page)
    expect(zoomAntes, 'o zoom não saiu do padrão — não há o que sobreviver').not.toBe('44px')

    // Uma mudança que vem DO SERVIDOR e redesenha a região do mapa.
    await ferramenta(page, 'Difícil').click()
    await camadaDe(page, /Pintar terreno/).click({ position: { x: 60, y: 60 } })

    // O CONTROLE: o remendo chegou e mudou a cena.
    //
    // O seletor é `.tabuleiro-terreno.tabuleiro-dificil` e não só o segundo: a
    // AMOSTRA do crachá no trilho carrega a mesma classe de espécie, e o
    // seletor curto acharia dois elementos — um deles um quadradinho de
    // legenda que existe desde antes do clique.
    await expect(
      page.locator('.tabuleiro-terreno.tabuleiro-dificil'),
      'o terreno não apareceu — o remendo não aconteceu e o resto não mede nada',
    ).toHaveCount(1)

    expect(await quadrado(page), 'o remendo levou o zoom junto').toBe(zoomAntes)
  } finally {
    await apagar()
  }
})

/**
 * A CONTA DO CLIQUE ACOMPANHA O ZOOM.
 *
 * O quadrado clicado sai do PONTO do clique dividido pelo tamanho da casa, e o
 * tamanho da casa é o mesmo número que o zoom move. Se um dos dois andar sem o
 * outro, o mestre pinta uma casa e outra acende — e a distância entre elas
 * cresce com o zoom, o que faz o defeito parecer "só na hora do combate".
 *
 * A asserção é um INVARIANTE GEOMÉTRICO e não uma conta refeita: a casa que
 * apareceu tem de CONTER o ponto clicado. Recalcular `floor(x / quadrado)` no
 * teste seria comparar a implementação consigo mesma — e passaria verde com as
 * duas erradas do mesmo jeito.
 */
test('depois de aproximar, a casa pintada é a que estava sob o dedo', async ({ page }) => {
  const { mesa, apagar } = await disposableTable(page)
  try {
    await openTheBoard(page, mesa)
    // Aproxima ao máximo: com a casa grande, um erro de conversão de um quadrado
    // já sai da caixa e a asserção o pega. No zoom padrão um erro pequeno pode
    // cair dentro da mesma casa por sorte.
    const mais = page.getByRole('button', { name: 'Aproximar o mapa' })
    while (!(await mais.isDisabled())) await mais.click()

    await page.getByRole('button', { name: 'Camuflagem' }).click()
    const casas = camadaDe(page, /Pintar terreno/)
    const alvo = { x: 150, y: 110 }
    await casas.click({ position: alvo })

    const pintada = page.locator('.tabuleiro-terreno.tabuleiro-camuflagem')
    await expect(pintada, 'nada foi pintado').toHaveCount(1)

    const caixaDaCamada = (await casas.boundingBox())!
    const caixaDaCasa = (await pintada.boundingBox())!
    const pontoX = caixaDaCamada.x + alvo.x
    const pontoY = caixaDaCamada.y + alvo.y

    expect(
      pontoX >= caixaDaCasa.x && pontoX <= caixaDaCasa.x + caixaDaCasa.width,
      `o clique em x=${pontoX} caiu fora da casa pintada (${caixaDaCasa.x}–${caixaDaCasa.x + caixaDaCasa.width})`,
    ).toBe(true)
    expect(
      pontoY >= caixaDaCasa.y && pontoY <= caixaDaCasa.y + caixaDaCasa.height,
      `o clique em y=${pontoY} caiu fora da casa pintada (${caixaDaCasa.y}–${caixaDaCasa.y + caixaDaCasa.height})`,
    ).toBe(true)
  } finally {
    await apagar()
  }
})

/**
 * DEPOIS DE ARRASTAR A VISTA, A CASA CONTINUA SENDO A QUE ESTÁ SOB O DEDO.
 *
 * É o mesmo invariante do caso acima, com o gesto que a ALE-203 trouxe. Ele é
 * outro caso e não uma repetição porque o que pode quebrar é outro: ali é o
 * ZOOM entrando na divisão, aqui é a JANELA entrando na soma — e a janela é
 * exatamente o termo que a moldura escondia. O defeito que o dono relatou
 * ("apaguei e não apagou") era esta soma errada, com a moldura crescendo
 * debaixo do ponteiro.
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
  const { mesa, apagar } = await disposableTable(page)
  try {
    await openTheBoard(page, mesa)

    // ARRASTA A VISTA um bom pedaço, com a ferramenta da mão. O deslocamento é
    // deliberadamente NÃO múltiplo do quadrado: um múltiplo esconderia um erro
    // de fase, porque a casa certa e a errada cairiam no mesmo lugar da grade.
    await ferramenta(page, 'Arrastar a vista').click()
    const mao = page.locator('.tabuleiro-vista')
    const caixaDaMao = (await mao.boundingBox())!
    await page.mouse.move(caixaDaMao.x + 400, caixaDaMao.y + 300)
    await page.mouse.down()
    await page.mouse.move(caixaDaMao.x + 173, caixaDaMao.y + 191, { steps: 8 })
    await page.mouse.up()

    // O CONTROLE: a vista ANDOU. Sem ele, "a casa está certa" é verdade também
    // numa tela onde o arrasto não fez nada — que é o verde mais caro que existe.
    const vista = await page
      .locator('.tabuleiro-cena')
      .evaluate((e) => getComputedStyle(e).getPropertyValue('--vista-x').trim())
    expect(vista, 'a vista não saiu do lugar — o arrasto não aconteceu e o resto não mede nada').not.toBe('0px')

    await ferramenta(page, 'Camuflagem').click()
    const casas = camadaDe(page, /Pintar terreno/)
    const alvo = { x: 260, y: 180 }
    await casas.click({ position: alvo })

    const pintada = page.locator('.tabuleiro-terreno.tabuleiro-camuflagem')
    await expect(pintada, 'nada foi pintado').toHaveCount(1)

    const caixaDaCamada = (await casas.boundingBox())!
    const caixaDaCasa = (await pintada.boundingBox())!
    const pontoX = caixaDaCamada.x + alvo.x
    const pontoY = caixaDaCamada.y + alvo.y

    expect(
      pontoX >= caixaDaCasa.x && pontoX <= caixaDaCasa.x + caixaDaCasa.width,
      `com a vista em ${vista}, o clique em x=${pontoX} caiu fora da casa pintada ` +
        `(${caixaDaCasa.x}–${caixaDaCasa.x + caixaDaCasa.width})`,
    ).toBe(true)
    expect(
      pontoY >= caixaDaCasa.y && pontoY <= caixaDaCasa.y + caixaDaCasa.height,
      `o clique em y=${pontoY} caiu fora da casa pintada ` +
        `(${caixaDaCasa.y}–${caixaDaCasa.y + caixaDaCasa.height})`,
    ).toBe(true)
  } finally {
    await apagar()
  }
})

/**
 * O DESENHO DA MEDIDA CABE NO SVG QUE O CARREGA — o guarda de um recorte que não
 * acusa (ALE-203).
 *
 * O `<svg>` MAIS EXTERNO recorta pelo viewport dele, e `overflow: visible` não
 * levanta esse recorte. Quando a régua e o gabarito passaram a viver dentro de um
 * plano de tamanho ZERO, o viewport virou 0×0 e os dois PARARAM DE APARECER — com
 * o `<path>` no DOM, com a caixa certa no lugar certo, com o `fill` certo e com
 * `display: block`. Nada acusava, e a suíte inteira ficou verde por cima disso.
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
  const { mesa, apagar } = await disposableTable(page)
  try {
    await openTheBoard(page, mesa)

    await ferramenta(page, 'Gabarito').click()
    await camadaDe(page, /Pôr o gabarito/).click({ position: { x: 300, y: 200 } })

    const svg = page.locator('.tabuleiro-medida-fundo')
    const desenho = svg.locator('path')

    // O CONTROLE: o servidor respondeu e há desenho. Sem ele, "cabe no SVG" seria
    // verdade sobre um `<path>` vazio, que é o verde que este caso existe para
    // não dar.
    await expect(desenho, 'o gabarito não foi desenhado — não há o que medir').toHaveAttribute('d', /\S/)

    const caixaDoSvg = (await svg.boundingBox())!
    const caixaDoDesenho = (await desenho.boundingBox())!
    expect(
      caixaDoDesenho.x >= caixaDoSvg.x &&
        caixaDoDesenho.x + caixaDoDesenho.width <= caixaDoSvg.x + caixaDoSvg.width &&
        caixaDoDesenho.y >= caixaDoSvg.y &&
        caixaDoDesenho.y + caixaDoDesenho.height <= caixaDoSvg.y + caixaDoSvg.height,
      `o desenho está em (${caixaDoDesenho.x},${caixaDoDesenho.y},${caixaDoDesenho.width}×${caixaDoDesenho.height}) ` +
        `e o SVG em (${caixaDoSvg.x},${caixaDoSvg.y},${caixaDoSvg.width}×${caixaDoSvg.height}): ` +
        `o viewport do svg recorta o gabarito, e ninguém na mesa o vê`,
    ).toBe(true)
  } finally {
    await apagar()
  }
})

/**
 * A JANELA VAI ATRÁS DO FOCO, e sem isto a ALE-203 teria embutido uma regressão
 * de teclado.
 *
 * A rolagem nativa trazia o elemento focado para a vista de graça. Ela saiu com
 * a moldura: a cena recorta com `overflow: hidden` e a página não rola, então
 * não existe mais ancestral rolável — o navegador TENTA e não tem o que rolar.
 * Medido vermelho antes do conserto: com a peça em (-2039,-1268) e a janela em
 * (92,97,1756×807), focar a peça deixava tudo exatamente onde estava. Quem
 * navega por teclado podia focar uma peça que nunca ia conseguir ver.
 *
 * Por que e2e: são FOCO e GEOMETRIA REAL ao mesmo tempo, num elemento cuja
 * posição vem de um `transform` de CSS. Em jsdom todo retângulo é zero e a
 * asserção passaria verde sobre nada.
 *
 * O CONTROLE é a metade que importa: a peça tem de estar FORA antes. Sem ele,
 * "a peça está dentro" é verdade também numa janela que nunca saiu do lugar.
 */
test('a janela vai atrás do foco quando a peça está fora dela', async ({ page }) => {
  const { mesa, apagar } = await disposableTable(page)
  try {
    await openTheBoard(page, mesa)
    await putATokenOnTheMap(page)

    const cena = page.locator('.tabuleiro-cena')
    const peca = page.locator('.tabuleiro-peca')

    // Arrasta a vista para BEM longe da peça, com a ferramenta da mão.
    await ferramenta(page, 'Arrastar a vista').click()
    const caixaDaMao = (await page.locator('.tabuleiro-vista').boundingBox())!
    await page.mouse.move(caixaDaMao.x + caixaDaMao.width - 40, caixaDaMao.y + caixaDaMao.height - 40)
    await page.mouse.down()
    await page.mouse.move(caixaDaMao.x + 20, caixaDaMao.y + 20, { steps: 10 })
    await page.mouse.up()

    const dentroDaJanela = async () => {
      const j = (await cena.boundingBox())!
      const p = (await peca.boundingBox())!
      return p.x >= j.x && p.x + p.width <= j.x + j.width && p.y >= j.y && p.y + p.height <= j.y + j.height
    }

    // O CONTROLE: a peça ficou FORA da janela.
    expect(await dentroDaJanela(), 'a peça continuou visível — o arrasto não afastou nada e o resto não mede').toBe(false)

    await peca.focus()

    expect(
      await dentroDaJanela(),
      'a peça focada continuou fora da janela: quem navega por teclado pode alcançá-la e nunca vê-la',
    ).toBe(true)
  } finally {
    await apagar()
  }
})

/**
 * SHIFT + ARRASTO ENCHE O RETÂNGULO (ALE-203, item 10 do dono).
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
  const { mesa, apagar } = await disposableTable(page)
  try {
    await openTheBoard(page, mesa)
    await ferramenta(page, 'Difícil').click()

    const casas = camadaDe(page, /Pintar terreno/)
    const caixa = (await casas.boundingBox())!
    const de = { x: caixa.x + 120, y: caixa.y + 120 }
    const ate = { x: de.x + 120, y: de.y + 90 }

    // O CONTROLE: o MESMO arrasto sem Shift pinta uma linha, não uma área.
    await page.mouse.move(de.x, de.y)
    await page.mouse.down()
    await page.mouse.move(ate.x, ate.y, { steps: 10 })
    await page.mouse.up()
    const doTraco = await page.locator('.tabuleiro-terreno.tabuleiro-dificil').count()

    // E agora COM Shift, num pedaço virgem do plano.
    const deB = { x: caixa.x + 420, y: caixa.y + 120 }
    const ateB = { x: deB.x + 120, y: deB.y + 90 }
    await page.keyboard.down('Shift')
    await page.mouse.move(deB.x, deB.y)
    await page.mouse.down()
    await page.mouse.move(ateB.x, ateB.y, { steps: 6 })
    // O LAÇO tem de estar na tela ENQUANTO o dedo segura: é a promessa visual do
    // gesto, e sem ela a pessoa arrasta no escuro.
    await expect(page.locator('.tabuleiro-laco'), 'o laço não apareceu durante o arrasto').toBeVisible()
    await page.mouse.up()
    await page.keyboard.up('Shift')

    await expect
      .poll(() => page.locator('.tabuleiro-terreno.tabuleiro-dificil').count(), {
        message: 'o retângulo não encheu a área',
      })
      .toBeGreaterThan(doTraco * 2)

    // E o laço some quando o dedo solta — ele é intenção, não resultado.
    await expect(page.locator('.tabuleiro-laco'), 'o laço ficou na tela depois de soltar').toBeHidden()
  } finally {
    await apagar()
  }
})

/**
 * O MARCADOR É ALCANÇÁVEL, e este caso existe porque ele já não era.
 *
 * As camadas de clique cobrem o plano inteiro e vêm depois no DOM; a de MOVER é
 * a ativa por padrão. O marcador ficava debaixo dela e o clique nunca chegava —
 * com o HTML inteiro correto, os seis guardas de handler verdes, e nada em
 * lugar nenhum dizendo que havia um elemento coberto. Só o navegador vê isso, e
 * é a definição de quando gastar e2e.
 *
 * O gesto é o do mestre no meio da cena: sem trocar de ferramenta, clicar no
 * ponto que ele marcou e mexer nele.
 */
test('o marcador continua clicável por baixo da camada de mover', async ({ page }) => {
  const { mesa, apagar } = await disposableTable(page)
  try {
    await openTheBoard(page, mesa)
    await putATokenOnTheMap(page)

    // `exact` porque o nome do botão do trilho é PREFIXO do nome da camada de
    // clique ("Marcar um lugar — escolha a casa"), e sem ele o seletor casa com
    // os dois.
    await ferramenta(page, 'Marcar').click()
    await camadaDe(page, /Marcar um lugar/).click({ position: { x: 90, y: 90 } })
    const marcador = page.locator('.tabuleiro-marcador')
    await expect(marcador, 'o marcador não nasceu').toHaveCount(1)

    // De volta ao padrão: é assim que a ferramenta fica enquanto o mestre joga,
    // e era exatamente aí que o marcador ficava inalcançável.
    await ferramenta(page, 'Marcar').click()

    // O CONTROLE do empilhamento: a camada que cobria o marcador tem de estar NO
    // AR. Sem ele, "o clique chegou" é verdade num palco onde nada cobria nada —
    // que foi exatamente o estado em que este caso passou verde com o `z-index`
    // removido, antes de a peça entrar na fixture.
    await expect(
      camadaDe(page, /Mover/),
      'a camada de mover não está no ar — o caso não enfrenta o empilhamento que veio medir',
    ).toBeVisible()

    await marcador.click()
    await expect(
      page.locator('.tabuleiro-marcador-acoes'),
      'o clique não chegou ao marcador — alguma camada o cobriu de novo',
    ).toBeVisible()

    // E o que o gesto existe para fazer: revelar para a mesa.
    await expect(marcador, 'o marcador não nasceu escondido').toHaveClass(/escondido/)
    await page.getByRole('button', { name: /^Revelar o marcador/ }).click()
    await expect(marcador, 'revelar não mudou nada na tela do mestre').not.toHaveClass(/escondido/)
  } finally {
    await apagar()
  }
})

/**
 * A PRÉVIA DO ARRASTO: a seta e a distância aparecem ENQUANTO o dedo arrasta
 * (ALE-203, pedido do dono: "durante o drag do token, mostre a seta apontando
 * para o token movimentando e mostre a distância na seta").
 *
 * POR QUE E2E, que é a faixa mais cara e precisa se justificar: o que se mede
 * aqui só existe DENTRO de um gesto de ponteiro com movimentos intermediários.
 * O Go prova o que a rota `/previa/` responde — quatro guardas, um deles provado
 * vermelho — e nada mais: a ligação entre o `pointermove` e aquela rota mora
 * numa string de expressão do Datastar, que nenhum compilador lê e nenhum teste
 * de handler exercita. Quebrada, ela não dá erro: o atributo continua no HTML,
 * inteiro e com cara de certo, e o arrasto simplesmente não desenha nada.
 *
 * CENTRALIZAR ANTES DE ARRASTAR não é arrumação, e esta linha custou uma
 * investigação inteira: a peça pode cair debaixo do trilho de ferramentas, o
 * `boundingBox` devolve a caixa de um elemento COBERTO sem reclamar, e o
 * `mouse.down` acerta o trilho. O gesto não acontece, e a leitura vira "a prévia
 * não funciona" — apontando para o código que está certo. Antes de ler o
 * silêncio como defeito, o caso confere que a peça TEM o gesto pendurado.
 */
test('a seta e a distância aparecem durante o arrasto da peça', async ({ page }) => {
  const { mesa, apagar } = await disposableTable(page)
  try {
    await openTheBoard(page, mesa)
    await putATokenOnTheMap(page)
    await page.getByLabel('Centralizar nas peças').click()

    const peca = page.locator('.tabuleiro-peca').first()
    // O CONTROLE, antes de qualquer ausência virar conclusão: o gesto está
    // pendurado nesta peça? Sem isto, "não achei prévia" seria verdade também
    // sobre uma peça que ninguém pode arrastar.
    await expect(peca, 'a peça não tem o gesto de arrasto: o canal não está aberto').toHaveAttribute(
      'data-on:pointermove__window',
      /previa/,
    )

    const caixa = await peca.boundingBox()
    if (!caixa) throw new Error('a peça não tem caixa: o arrasto não tem de onde partir')
    const x = caixa.x + caixa.width / 2
    const y = caixa.y + caixa.height / 2

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
      page.locator('.tabuleiro-previa-cabe'),
      'o arrasto não desenhou a seta viva',
    ).toHaveAttribute('d', /^M /)
    await expect(
      page.locator('.tabuleiro-medida-frente text').filter({ hasText: /m$/ }).first(),
      'a seta viva não diz a distância em metros',
    ).toBeVisible()

    await page.mouse.up()
    // E A PRÉVIA SOME ao soltar: sobrevivendo, ela ficaria por cima da seta de
    // verdade, com o mesmo formato e outra medida — dois caminhos na tela e
    // nenhum jeito de saber qual vale.
    await expect(
      page.locator('.tabuleiro-previa-cabe'),
      'a seta viva sobreviveu ao soltar',
    ).toHaveAttribute('d', '')
  } finally {
    await apagar()
  }
})

/**
 * O PAINEL DE VERBOS CABE NO TELEFONE, COM ACERVO (ALE-271).
 *
 * Por que e2e, e este arquivo cobra a justificativa de cada caso: o que estoura
 * é LARGURA REAL de texto renderizado. O servidor não mede caixa — ele escreve
 * "Lugares da campanha · 3" e não sabe que aquilo dá 144px numa janela de 390.
 * Só o navegador sabe, e o defeito é exatamente a soma das larguras.
 *
 * O ESTADO é o assunto, e é por isso que o caso semeia lugares: o painel sem
 * acervo não tem o botão largo, e medi-lo assim é medir outro painel. Foi essa a
 * lacuna que deixou o defeito viver — a cena estava nas listas dos guardas, e o
 * estado que a quebra não estava em lugar nenhum.
 *
 * MEDIDO antes do conserto, com a Mesa a 390px: o painel começava em x = −122 e
 * "Centralizar o mapa" (x = −117) e "Afastar o mapa" (x = −72) ficavam fora da
 * janela — inalcançáveis, e o zoom é de TODO MUNDO.
 */
test('o painel de verbos cabe a 390px com a campanha tendo acervo', async ({ page }) => {
  const { mesa, apagar } = await disposableTable(page)
  try {
    const campanha = mesa.split('/')[2]
    // O ACERVO pela porta de verdade — a mesma que a aba de lugares usa. Três
    // basta: o que muda a largura é o botão EXISTIR e a contagem ter dígito.
    for (const nome of ['Taverna do E2E', 'Cripta do E2E', 'Ruínas do E2E']) {
      const criado = await page.request.post(`/campanhas/${campanha}/lugares/novo`, {
        form: { name: nome, ground: 'cripta' },
      })
      expect(criado.ok(), `semear o lugar ${nome}: ${criado.status()}`).toBeTruthy()
    }

    await openTheBoard(page, mesa)
    await page.setViewportSize({ width: 390, height: 844 })
    await page.waitForTimeout(300)

    // O CONTROLE, e sem ele o caso não mede nada: se o botão do acervo não
    // estiver na tela, o painel medido é o estreito, e o guarda passa verde
    // sobre o painel que nunca quebrou.
    const acervo = page.locator('.tabuleiro-verbos-da-cena button').filter({ hasText: /Lugares|3/ })
    await expect(
      acervo.first(),
      'o botão do acervo não está no painel — o caso mediria um painel sem o item que o estoura',
    ).toBeVisible()

    await expectDentroDaJanela(page)
  } finally {
    await apagar()
  }
})

/**
 * A SEGUNDA CAMADA do menu da peça só aparece quando pedida (ALE-206).
 *
 * Ela é POPOVER NATIVO desde a fatia do colar, e a mudança de mecanismo é o que
 * este caso protege: como camada `absolute` ela media 314×325 e passava 122px da
 * janela a 844×390, com quatro dos seis modos inalcançáveis; `position: fixed`
 * não resolvia porque o plano tem `transform`, que vira bloco de contenção.
 *
 * O que se mede aqui é o FECHADO: um popover escondido não pode deixar botão no
 * caminho do teclado. São seis por peça, e dez zumbis dariam sessenta paradas de
 * Tab sobre um mapa em que nenhuma se vê.
 *
 * A primeira versão deste caso media com o MENU fechado e passou verde com o
 * submenu sabotado — um filho de pai `display:none` é invisível de qualquer
 * jeito. Ele afirmava no nome uma coisa e media outra. Por isso o menu é aberto
 * antes: é a linha que separa um guarda de um enfeite.
 *
 * Por que e2e: `display` computado, a top layer e `checkVisibility` só existem
 * num navegador. Em jsdom todo elemento mede zero e o caso passaria verde sobre
 * os seis botões no ar.
 */
test('o submenu de duplicar só entra no caminho do teclado quando é aberto', async ({ page }) => {
  const { mesa, apagar } = await disposableTable(page)
  try {
    await openTheBoard(page, mesa)
    await putATokenOnTheMap(page)

    const botoesDoSubmenu = () =>
      page.evaluate(
        () =>
          [...document.querySelectorAll('.tabuleiro-peca-copia button')].filter((b) =>
            (b as HTMLElement).checkVisibility(),
          ).length,
      )

    // O MENU ABERTO é a premissa: com ele fechado, o submenu seria invisível pela
    // herança do pai e o caso não mediria o popover.
    await page.locator('.tabuleiro-peca').first().click({ button: 'right' })
    const duplicar = page.getByRole('button', { name: /^Duplicar / })
    await expect(
      duplicar,
      'o menu da peça não abriu: sem ele o caso mede a herança do pai, não o popover',
    ).toBeVisible()

    expect(
      await botoesDoSubmenu(),
      'o submenu fechado deixou botão alcançável pelo Tab dentro de um menu aberto',
    ).toBe(0)

    // O CONTROLE POSITIVO: aberto, os seis aparecem — três de duplicar aqui e
    // três de copiar para colar.
    await duplicar.click()
    await expect
      .poll(botoesDoSubmenu, {
        message: 'o submenu não abriu: a asserção acima estaria medindo uma camada que nunca aparece',
      })
      .toBe(6)

    // E ele CABE na janela em toda forma, que é o que o popover comprou: a
    // camada `absolute` de antes passava 122px da borda a 844×390.
    //
    // REABERTO em cada formato, e não redimensionado com ele no ar: quem
    // posiciona é o `ancora()` no `beforetoggle`, e ele não roda de novo num
    // `resize`. Medir sem reabrir mede a conta do formato ANTERIOR — deu 9px de
    // estouro num painel que, reaberto, sobra 8.
    // OS DOIS FORMATOS desde a ALE-294, e o EM PÉ é o que prende aquela issue.
    //
    // Aqui morava "só o deitado", porque a 390px de LARGURA a peça nascia em
    // (3,0) — debaixo do painel de verbos — e o clique direito ia para o botão
    // de afastar em vez de abrir o menu dela. A peça passou a nascer abaixo da
    // faixa do cromo (`tabuleiro.TopChromeRows`), e é este caso que mantém
    // aquele número honesto: painel mais alto ou zoom padrão menor põem a peça
    // de volta debaixo do painel, o menu não abre e o `.tabuleiro-peca-copia`
    // não existe para medir. É a única testemunha possível — quem cobre um
    // elemento e quem recebe o clique só existem num navegador.
    for (const [nome, w, h] of [
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
      await page.locator('.tabuleiro-peca').first().click({ button: 'right' })
      await page.getByRole('button', { name: /^Duplicar / }).click()
      await page.waitForTimeout(300)
      const escapou = await page.evaluate(() => {
        const p = document.querySelector('.tabuleiro-peca-copia')!.getBoundingClientRect()
        return {
          abaixo: Math.round(p.bottom - window.innerHeight),
          direita: Math.round(p.right - window.innerWidth),
        }
      })
      expect(escapou.abaixo, `a camada passa ${escapou.abaixo}px do pé da janela ${nome}`).toBeLessThanOrEqual(1)
      expect(escapou.direita, `a camada passa ${escapou.direita}px da borda direita ${nome}`).toBeLessThanOrEqual(1)
    }
  } finally {
    await apagar()
  }
})

/**
 * COPIAR guarda a decisão, e cada CTRL+V repete (ALE-206).
 *
 * A issue pedia a pergunta no momento de COLAR; perguntar a cada tecla mataria o
 * valor do teclado, que é repetir. Decisão do dono: a pergunta é feita uma vez,
 * no menu, e o colar só executa.
 *
 * Por que e2e, e este arquivo cobra a justificativa: o `CTRL + V` é um atalho de
 * TECLADO competindo com o colar do navegador, e o quadrado de destino é o centro
 * da JANELA — uma conta de pixels que só existe com zoom e vista reais. Nada
 * disso tem testemunha fora de um navegador.
 *
 * O caso NÃO reprova a regra de PV, que já está presa em Go (`bondForMode`): o
 * que ele prende é a ligação — copiar enche a área, a tecla dispara, e o segundo
 * CTRL+V põe outro.
 */
test('copiar guarda o modo, e cada CTRL+V põe outro igual', async ({ page }) => {
  const { mesa, apagar } = await disposableTable(page)
  try {
    await openTheBoard(page, mesa)
    await putATokenOnTheMap(page)

    // A FAIXA da área começa VAZIA, e este é o controle: sem ele, uma faixa que
    // aparecesse sempre passaria pelas asserções de baixo sem provar nada.
    const faixa = page.locator('.tabuleiro-area')
    await expect(faixa, 'a faixa da área nasceu visível com a área vazia').toBeHidden()

    await page.locator('.tabuleiro-peca').first().click({ button: 'right' })
    await page.getByRole('button', { name: /^Duplicar / }).click()
    await page.getByRole('button', { name: /^Copiar .* para colar: com PV próprio/ }).click()

    await expect(faixa, 'copiar não acendeu a faixa da área').toBeVisible()
    await expect(faixa).toContainText('com PV próprio')

    const antes = await page.locator('.tabuleiro-peca').count()
    await page.keyboard.press('Control+v')
    await expect
      .poll(() => page.locator('.tabuleiro-peca').count(), { message: 'o primeiro CTRL+V não colou' })
      .toBe(antes + 1)
    await page.keyboard.press('Control+v')
    await expect
      .poll(() => page.locator('.tabuleiro-peca').count(), {
        message: 'o segundo CTRL+V não colou: a área não sobreviveu ao remendo da cena',
      })
      .toBe(antes + 2)

    // ESVAZIAR é um BOTÃO e nunca o Esc: o `scene.js` mapeia Escape para "voltar"
    // e o mata no documento — medido, e o `railKeyboard` já o registra.
    await page.getByRole('button', { name: 'Esvaziar a área de transferência' }).click()
    await expect(faixa, 'esvaziar não apagou a faixa').toBeHidden()
    await page.keyboard.press('Control+v')
    await page.waitForTimeout(500)
    expect(
      await page.locator('.tabuleiro-peca').count(),
      'o CTRL+V colou com a área vazia',
    ).toBe(antes + 2)
  } finally {
    await apagar()
  }
})
