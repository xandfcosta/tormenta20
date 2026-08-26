import { expect, type Page, test } from '@playwright/test'

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
    mesa: `/piloto/mesa/${campanha}/${sid}`,
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
  await page.getByLabel('Lugar').fill('Taverna do E2E')
  await page.getByRole('button', { name: 'Abrir', exact: true }).click()
  await page.locator('.tabuleiro-plano').waitFor({ timeout: 10_000 })
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

const quadrado = (page: Page) =>
  page.locator('.tabuleiro-palco').evaluate((e) => getComputedStyle(e).getPropertyValue('--quadrado').trim())

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
test('o zoom e a rolagem sobrevivem ao remendo do servidor', async ({ page }) => {
  const { mesa, apagar } = await mesaDescartavel(page)
  try {
    await abreOTabuleiro(page, mesa)

    await page.getByRole('button', { name: 'Aproximar o mapa' }).click()
    await page.getByRole('button', { name: 'Aproximar o mapa' }).click()
    const zoomAntes = await quadrado(page)
    expect(zoomAntes, 'o zoom não saiu do padrão — não há o que sobreviver').not.toBe('44px')

    // Uma mudança que vem DO SERVIDOR e redesenha a região do mapa.
    await page.getByRole('button', { name: 'Difícil' }).click()
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
    await page.getByRole('button', { name: 'Marcar', exact: true }).click()
    await camadaDe(page, /Marcar um lugar/).click({ position: { x: 90, y: 90 } })
    const marcador = page.locator('.tabuleiro-marcador')
    await expect(marcador, 'o marcador não nasceu').toHaveCount(1)

    // De volta ao padrão: é assim que a ferramenta fica enquanto o mestre joga,
    // e era exatamente aí que o marcador ficava inalcançável.
    await page.getByRole('button', { name: 'Marcar', exact: true }).click()

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
