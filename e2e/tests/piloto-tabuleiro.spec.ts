import { expect, type Page, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'

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

/** Uma mesa só desta corrida. Devolve o endereço e como se livrar dela. */
async function mesaDescartavel(page: Page): Promise<{ mesa: string; apagar: () => Promise<void> }> {
  const nome = `E2E Descartável tabuleiro ${Date.now()}-${Math.floor(Math.random() * 1e6)}`

  // A FIXTURE vai pela API e não pela tela, de propósito: montar campanha e
  // sessão clicando gastaria meia dúzia de navegações em cada caso para chegar
  // ao que se quer medir, e nenhuma delas é o assunto deste arquivo. O que é
  // medido — o tabuleiro — vai pela tela inteiro.
  //
  // Estas rotas JSON estão na lista de órfãs da ALE-247. Quando elas caírem,
  // esta fixture troca de caminho e nenhum dos casos abaixo muda: eles só
  // precisam de um endereço de mesa.
  const criada = await page.request.post('/api/campaigns', {
    data: { name: nome, description: 'Criada e apagada pelo E2E do tabuleiro.' },
  })
  expect(criada.ok(), `criar a campanha descartável: ${criada.status()}`).toBeTruthy()
  const campanha = (await criada.json()).id as number

  const sessao = await page.request.post(`/api/campaigns/${campanha}/sessions`, {
    data: { sessionNumber: 1, title: 'Sessão do E2E' },
  })
  expect(sessao.ok(), `criar a sessão descartável: ${sessao.status()}`).toBeTruthy()
  const sid = (await sessao.json()).id as number

  return {
    mesa: `/mesa/${campanha}/${sid}`,
    // A LIMPEZA NÃO PODE FALAR MAIS ALTO QUE O DEFEITO (ALE-245), e eu aprendi
    // isto de novo na primeira corrida deste arquivo: sem o `catch`, um caso que
    // falhou no meio deixa a página num estado em que o `delete` estoura, e o
    // relatório mostra o erro da FAXINA no lugar do erro do teste — com a linha
    // apontando para o `finally`. A campanha órfã custa uma linha na lista; o
    // defeito escondido custa uma sessão inteira.
    apagar: async () => {
      try {
        await page.request.delete(`/api/campaigns/${campanha}`)
      } catch {
        // A mesa descartável fica para trás. É o preço certo a pagar.
      }
    },
  }
}

/**
 * Abre a GAVETA da fila, que é onde a lista inteira passou a morar (ALE-269).
 *
 * A forma do mestre virou SHELL: o trilho de 80px responde "de quem é a vez", e
 * dano, ordem, condição, "+ Combatente" e "Adicionar grupo" desceram para uma
 * gaveta pela esquerda — a mesma decisão que a ALE-198 tomou na SPA, onde a
 * fila inteira vive num `SidePanel`.
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
async function fechaAFila(page: Page): Promise<void> {
  await page.getByRole('button', { name: 'Fechar a iniciativa' }).click()
  await expect(page.locator('#gaveta-da-fila'), 'a gaveta da fila não fechou').not.toHaveAttribute(
    'open',
    '',
  )
}

async function abreAFila(page: Page): Promise<void> {
  await page
    .getByRole('button', { name: /^Abrir a iniciativa/ })
    .filter({ visible: true })
    .click()
  await expect(page.locator('#gaveta-da-fila'), 'a gaveta da fila não abriu').toHaveAttribute(
    'open',
    '',
  )
}

/**
 * Põe UMA peça no mapa, e este passo não é enfeite: a camada de MOVER — a que
 * cobria o marcador — só é desenhada quando existe algo movível
 * (`v.AlvoDoMovimento != ""`). Num tabuleiro vazio ela não nasce, e um teste de
 * empilhamento sobre um tabuleiro vazio não enfrenta o que veio guardar.
 *
 * Descobri isto SABOTANDO: tirei o `z-index` do marcador e o caso continuou
 * verde. Ele media um palco onde nada cobria nada.
 */
async function poeUmaPecaNoMapa(page: Page): Promise<void> {
  await abreAFila(page)
  await page.getByRole('button', { name: '+ Combatente' }).click()
  // `exact` porque `getByLabel` casa por SUBSTRING: desde o editor de bloco
  // (ALE-269) a mesma cena tem "Nome do NPC", e `'Nome'` passou a resolver para
  // dois campos. É a segunda vez nesta fatia que um seletor único por acidente
  // deixa de ser — a outra foi a camada de clique.
  await page.getByLabel('Nome', { exact: true }).fill('Ogro do E2E')
  await page.getByRole('button', { name: 'Acrescentar' }).click()
  // A GAVETA FECHA ANTES de o teste voltar ao mapa, e esta ordem é a jornada de
  // verdade: monta-se a fila na gaveta, fecha-se, e põe-se no mapa pela faixa do
  // tabuleiro. Ela é MODAL — deixá-la aberta torna inerte tudo o que está atrás,
  // e o `Pôr no mapa` da faixa (que vem antes no DOM, então é o que o `.first()`
  // acha) ficaria coberto por ela. O sintoma é "dialog intercepts pointer
  // events" num seletor que casou, e não um "não achei".
  await fechaAFila(page)
  await page.getByRole('button', { name: 'Pôr no mapa', exact: true }).first().click()
  // Escopado ao DIÁLOGO: o nome do combatente aparece também na fila atrás dele,
  // e um seletor de página inteira acha os dois.
  const dialogo = page.locator('#por-no-mapa')
  await dialogo.getByRole('button', { name: /Ogro do E2E/ }).click()
  await dialogo.getByRole('button', { name: 'Pôr no mapa', exact: true }).click()
  await expect(page.locator('.tabuleiro-peca'), 'a peça não entrou no mapa').toHaveCount(1)
}

/** Abre o tabuleiro pela TELA, que é o gesto de verdade. */
async function abreOTabuleiro(page: Page, mesa: string): Promise<void> {
  await page.goto(mesa, { waitUntil: 'domcontentloaded' })
  await page.getByRole('button', { name: 'Abrir tabuleiro' }).click()
  // `exact` porque `getByLabel` casa por SUBSTRING, e o diálogo do acervo se
  // chama "Lugares da campanha" — com acervo na mesa, `'Lugar'` resolve para
  // DOIS e o helper estoura em strict mode (ALE-271). O sintoma não aponta para
  // a causa: ele diz "waiting for getByLabel('Lugar')" num campo que está lá.
  await page.getByLabel('Lugar', { exact: true }).fill('Taverna do E2E')
  await page.getByRole('button', { name: 'Abrir', exact: true }).click()
  // A CENA e não o PLANO: desde a ALE-203 o plano é uma ORIGEM de tamanho zero
  // num plano infinito, e o Playwright chama de invisível todo elemento sem
  // caixa. Quem tem o retângulo agora é a janela que recorta.
  await page.locator('.tabuleiro-cena').waitFor({ timeout: 10_000 })
}

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
  const { mesa, apagar } = await mesaDescartavel(page)
  try {
    await abreOTabuleiro(page, mesa)

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
  const { mesa, apagar } = await mesaDescartavel(page)
  try {
    await abreOTabuleiro(page, mesa)
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
 * ZERO, e a conta do clique é `offsetX + $vistax` num elemento IRMÃO desse
 * plano. Nenhuma camada abaixo do navegador tem geometria para testemunhar
 * isso — em jsdom todo elemento mede zero, e a asserção passaria verde com as
 * duas contas erradas.
 *
 * A asserção é geométrica e não aritmética, pela mesma razão de lá: recalcular
 * `floor((x + vista) / quadrado)` aqui seria comparar a implementação consigo
 * mesma.
 */
test('depois de arrastar a vista, a casa pintada é a que estava sob o dedo', async ({ page }) => {
  const { mesa, apagar } = await mesaDescartavel(page)
  try {
    await abreOTabuleiro(page, mesa)

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
  const { mesa, apagar } = await mesaDescartavel(page)
  try {
    await abreOTabuleiro(page, mesa)

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
  const { mesa, apagar } = await mesaDescartavel(page)
  try {
    await abreOTabuleiro(page, mesa)
    await poeUmaPecaNoMapa(page)

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
  const { mesa, apagar } = await mesaDescartavel(page)
  try {
    await abreOTabuleiro(page, mesa)
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
  const { mesa, apagar } = await mesaDescartavel(page)
  try {
    await abreOTabuleiro(page, mesa)
    await poeUmaPecaNoMapa(page)

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
  const { mesa, apagar } = await mesaDescartavel(page)
  try {
    await abreOTabuleiro(page, mesa)
    await poeUmaPecaNoMapa(page)
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
  const { mesa, apagar } = await mesaDescartavel(page)
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

    await abreOTabuleiro(page, mesa)
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
 * O submenu de duplicar acrescenta TRÊS botões por peça, e quem os esconde é o
 * `data-show`, que é CSS. Com o MENU aberto e o submenu fechado eles não podem
 * estar no caminho do teclado: seriam três paradas de Tab que não se veem, e a
 * pessoa navegando pelo menu passaria por elas antes de chegar ao ✕.
 *
 * O MENU É ABERTO ANTES, e essa linha é a diferença entre um guarda e um enfeite.
 * A primeira versão deste caso media com o menu FECHADO, e passou verde com o
 * submenu sabotado para `data-show="true"` — porque um filho de pai
 * `display:none` é invisível de qualquer jeito. Ele afirmava no nome uma coisa e
 * media outra, e só a sabotagem o denunciou.
 *
 * Por que e2e: o que se mede é `checkVisibility` com o `data-show` aplicado de
 * verdade, e a herança de `display` de um pai para o filho. Em jsdom todo
 * elemento mede zero e o caso passaria verde sobre os três botões no ar.
 */
test('o submenu de duplicar só entra no caminho do teclado quando é aberto', async ({ page }) => {
  const { mesa, apagar } = await mesaDescartavel(page)
  try {
    await abreOTabuleiro(page, mesa)
    await poeUmaPecaNoMapa(page)

    const botoesDoSubmenu = () =>
      page.evaluate(
        () =>
          [...document.querySelectorAll('.tabuleiro-peca-copia button')].filter((b) =>
            (b as HTMLElement).checkVisibility(),
          ).length,
      )

    // O MENU ABERTO é a premissa: com ele fechado, o submenu seria invisível pela
    // herança do pai e o caso não mediria o `data-show` dele.
    await page.locator('.tabuleiro-peca').first().click({ button: 'right' })
    const duplicar = page.getByRole('button', { name: /^Duplicar / })
    await expect(
      duplicar,
      'o menu da peça não abriu: sem ele o caso mede a herança do pai, não o submenu',
    ).toBeVisible()

    expect(
      await botoesDoSubmenu(),
      'o submenu fechado deixou botão alcançável pelo Tab dentro de um menu aberto',
    ).toBe(0)

    // O CONTROLE POSITIVO: aberto, os três aparecem. Sem ele, "nenhum botão
    // visível" seria também o resultado de um seletor que parou de casar.
    await duplicar.click()
    await expect
      .poll(botoesDoSubmenu, {
        message: 'o submenu não abriu: a asserção acima estaria medindo uma camada que nunca aparece',
      })
      .toBe(3)
  } finally {
    await apagar()
  }
})
