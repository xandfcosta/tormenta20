import { expect, test } from '@playwright/test'

/**
 * FOCO ATIVA na lista do bestiário — a seta anda e a ficha segue junto, sem
 * exigir um Enter para ver. Decisão do dono: "igual em jogo".
 *
 * E2E porque a pergunta é sobre TECLADO e FOCO de verdade. Medido na marra: por
 * CDP o `element.focus()` move o `document.activeElement` e **não dispara o
 * evento `focus`** quando a janela não tem foco do sistema — nem para um
 * ouvinte próprio. Toda medição de teclado feita por automação de aba mede
 * silêncio e parece medir ausência de comportamento.
 *
 * Mede também o CUSTO, que é o que o dono pediu: quantas idas ao servidor uma
 * travessia de seta custa. O `__debounce` existe para que segurar a seta não
 * peça as 80 fichas do caminho — só a que a pessoa parou para ler.
 */
test.use({ storageState: '.auth/user.json' })

test('a seta anda na lista e a ficha segue junto', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/bestiario')
  await page.waitForLoadState('networkidle')

  const fichaLateral = page.locator('.mesa-painel')
  const primeira = page.locator('a[href*="criatura="]').first()
  await primeira.focus()
  await expect(fichaLateral).toContainText(/\S/)
  const antes = (await fichaLateral.innerText()).slice(0, 40)

  // O CONTROLE: a região está declarada. Sem ela o driver não tem o que dirigir,
  // e "a seta não andou" seria verdade sobre uma tela sem teclado nenhum — que é
  // um defeito diferente e a mensagem apontaria o lugar errado.
  await expect(page.locator('[data-nav-region="lista"]')).toHaveCount(1)

  await page.keyboard.press('ArrowDown')
  await expect
    .poll(async () => (await fichaLateral.innerText()).slice(0, 40), { timeout: 4000 })
    .not.toBe(antes)
})

test('cada passo da seta desenha a ficha sem espera perceptível', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/bestiario')
  await page.waitForLoadState('networkidle')

  const ficha = page.locator('.mesa-painel')
  await page.locator('a[href*="criatura="]').first().focus()

  // PASSO DELIBERADO custa uma ida ao servidor por linha, e isso é o CERTO: quem
  // anda de linha em linha quer ver cada criatura. O `__debounce` não serve para
  // este caso — ele serve para a tecla SEGURADA, que repete a ~30ms e atravessa
  // a lista inteira; aí só a linha onde o dedo parou é pedida.
  //
  // Então o que se mede aqui é LATÊNCIA, não contagem: a pergunta do dono é se
  // a seta "responde igual em jogo", e jogo é quadro a quadro. O teto de 400ms é
  // generoso de propósito — abaixo disso ninguém chama de espera, e acima o
  // desenho deixa de acompanhar o dedo.
  const latencias: number[] = []
  for (let i = 0; i < 8; i++) {
    const antes = (await ficha.innerText()).slice(0, 40)
    const t0 = Date.now()
    await page.keyboard.press('ArrowDown')
    await expect
      .poll(async () => (await ficha.innerText()).slice(0, 40), { timeout: 4000, intervals: [16] })
      .not.toBe(antes)
    latencias.push(Date.now() - t0)
  }

  latencias.sort((a, b) => a - b)
  const mediana = latencias[Math.floor(latencias.length / 2)]
  const pior = latencias[latencias.length - 1]
  console.log(`latência por passo — mediana ${mediana}ms, pior ${pior}ms, todas: ${latencias.join(', ')}`)

  expect(mediana, `mediana de ${mediana}ms por passo de seta`).toBeLessThan(400)
  expect(pior, `pior passo levou ${pior}ms`).toBeLessThan(900)
})

test('depois de uma travessia rápida a ficha é a da linha onde o foco PAROU', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/bestiario')
  await page.waitForLoadState('networkidle')

  await page.locator('a[href*="criatura="]').first().focus()
  // Rápido de propósito: sem `await` entre as teclas, para cair DENTRO da janela
  // do throttle. É o caso que o throttle com borda de subida põe em risco —
  // ele dispara na primeira e limita as seguintes, e se descartar a última a
  // ficha fica mostrando uma criatura pela qual a pessoa só PASSOU.
  await Promise.all(
    Array.from({ length: 12 }, () => page.keyboard.press('ArrowDown')),
  )

  const foco = page.locator(':focus')
  const href = await foco.getAttribute('href')
  const criatura = new URL(href!, 'http://x').searchParams.get('criatura')

  // O CONTROLE: o foco de fato ANDOU. Sem ele, "a ficha bate com o foco" seria
  // verdade trivialmente sobre uma travessia que não saiu do lugar.
  expect(criatura, 'o foco não andou na travessia').toBeTruthy()
  const rotulo = (await foco.innerText()).split('\n')[0].trim()

  await expect
    .poll(async () => (await page.locator('.mesa-painel').innerText()).includes(rotulo), {
      timeout: 4000,
      intervals: [16],
    })
    .toBe(true)
})

/**
 * A FICHA rola pelo teclado quando o foco está nela.
 *
 * O pedido do dono: "preciso conseguir focar na ficha do monstro e usar as setas
 * para scrollar". São DUAS garantias que se parecem e falham por motivos
 * diferentes — o painel tem de ser ALCANÇÁVEL pelo foco, e a seta, uma vez lá,
 * tem de ROLAR em vez de ser comida pelo driver de navegação.
 *
 * A segunda é a que corre risco: a lista ao lado é uma `data-nav-region`, e um
 * driver que preventDefault nas setas indiscriminadamente deixaria a ficha
 * imóvel com o foco dentro dela. O `scene-nav` devolve a tecla ao navegador
 * quando o foco está num controle FORA de qualquer região — este guarda prende
 * esse contrato, que hoje é só um comentário no driver.
 */
test('com o foco na ficha, as setas rolam o painel', async ({ page }) => {
  // Janela baixa de propósito: é o que faz a ficha transbordar. Numa tela alta a
  // ficha inteira cabe, nada rola, e o guarda passaria verde sem medir nada.
  //
  // 680 e não 620, e a diferença é MEDIDA: a consulta de contêiner que decide
  // entre painel lateral e diálogo pede 30rem (480px) de ALTURA do palco, e a
  // 620px de janela o palco mede 476 — quatro abaixo. Lá o painel nem existe, e
  // o controle abaixo acusa "não transborda" quando a verdade é "não está na
  // tela". A 680 o palco mede 536 e a ficha esconde 111px.
  await page.setViewportSize({ width: 1400, height: 680 })
  await page.goto('/piloto/mestre/bestiario')
  await page.waitForLoadState('networkidle')

  const painel = page.locator('.mesa-painel')

  // O CONTROLE, e ele é duplo: o painel precisa ROLAR (senão a asserção final é
  // vazia) e precisa ser FOCÁVEL (senão o teclado nunca chega lá).
  const escondido = await painel.evaluate((e) => e.scrollHeight - e.clientHeight)
  expect(escondido, 'a ficha não transborda nesta janela — o guarda mediria nada').toBeGreaterThan(20)

  // O foco vai no MIOLO e não na `<section>`: a seção é a REGIÃO (é assim que a
  // seta chega até aqui) e o driver só considera item o focável de dentro dela.
  // A rolagem continua sendo da seção, porque rolagem nativa rola o ancestral
  // rolável do elemento focado.
  const miolo = painel.locator('[tabindex="0"]').first()
  await miolo.focus()
  await expect(miolo).toBeFocused()

  const antes = await painel.evaluate((e) => e.scrollTop)
  for (let i = 0; i < 5; i++) await page.keyboard.press('ArrowDown')
  await page.waitForTimeout(200)
  const depois = await painel.evaluate((e) => e.scrollTop)

  expect(
    depois,
    `a ficha não rolou com o foco nela (${antes} → ${depois}): o driver de navegação comeu a seta`,
  ).toBeGreaterThan(antes)
})

/**
 * A seta CRUZA entre a lista e os filtros.
 *
 * O defeito que isto prende: declarar UMA região e não a vizinha deixa a seta
 * presa lá dentro. A lista virou região antes dos filtros, e o efeito foi o dono
 * dizendo "não consigo chegar nos botões ou inputs de filtro com o teclado" —
 * enquanto o TAB chegava normalmente, o que torna o defeito invisível para quem
 * testa com TAB.
 *
 * É a garantia que a filosofia chama de "cross to a neighbouring region at the
 * edge", e ela só existe se as DUAS pontas forem declaradas. Meia gramática é
 * pior que nenhuma: sem região nenhuma a seta rola a página, e com uma só ela
 * prende.
 */
test('a seta sobe da lista para os filtros e volta', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/bestiario')
  await page.waitForLoadState('networkidle')

  // O CONTROLE: as DUAS regiões existem. Sem ele, "a seta não cruzou" seria
  // verdade também sobre uma tela onde nenhuma foi declarada, e a mensagem
  // mandaria procurar no lugar errado.
  await expect(page.locator('[data-nav-region="lista"]')).toHaveCount(1)
  await expect(page.locator('[data-nav-region="filtros"]')).toHaveCount(1)

  await page.locator('a[href*="criatura="]').first().focus()
  await page.keyboard.press('ArrowUp')

  const dentroDosFiltros = await page.evaluate(
    () => !!document.activeElement?.closest('[data-nav-region="filtros"]'),
  )
  expect(dentroDosFiltros, 'a seta para cima não saiu da lista: ficou presa na região').toBe(true)

  // E VOLTA, senão os filtros viram a armadilha que a lista era.
  await page.keyboard.press('ArrowDown')
  const deVoltaNaLista = await page.evaluate(
    () => !!document.activeElement?.closest('[data-nav-region="lista"]'),
  )
  expect(deVoltaNaLista, 'a seta para baixo não voltou para a lista').toBe(true)
})

/**
 * A ficha é alcançável SÓ COM SETAS — o objetivo do dono é não precisar de TAB.
 *
 * O TAB sempre chegou lá; a seta não, porque a ficha não era região e o driver
 * cruza para o vizinho pela geometria entre REGIÕES. Um caminho que só o TAB
 * percorre não é a gramática da casa, é a do navegador.
 */
test('a seta chega na ficha sem nenhum TAB', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/bestiario')
  await page.waitForLoadState('networkidle')

  await page.locator('a[href*="criatura="]').first().focus()
  // O CONTROLE: o ponto de partida é a lista. Sem ele, "chegou na ficha" seria
  // verdade sobre um foco que já estava lá.
  expect(
    await page.evaluate(() => !!document.activeElement?.closest('[data-nav-region="lista"]')),
    'o guarda não partiu da lista',
  ).toBe(true)

  await page.keyboard.press('ArrowRight')

  expect(
    await page.evaluate(() => !!document.activeElement?.closest('[data-nav-region="ficha"]')),
    'a seta para a direita não cruzou da lista para a ficha',
  ).toBe(true)

  // E VOLTA para a lista, senão a ficha vira o beco que ela deixou de ser.
  await page.keyboard.press('ArrowLeft')
  expect(
    await page.evaluate(() => !!document.activeElement?.closest('[data-nav-region="lista"]')),
    'a seta para a esquerda não voltou da ficha para a lista',
  ).toBe(true)
})

/**
 * A legenda de teclado aparece onde a tecla EXISTE, e some onde não existe.
 *
 * O driver só liga em `≥xl` com ponteiro fino. Anunciar seta para quem está no
 * toque é ensinar um atalho que não está lá — a mesma regra do `@tecla`, e a
 * razão de o Esc não estar na legenda (medido: não faz nada nesta cena).
 */
test('a legenda de teclado aparece no laptop e some no telefone', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/bestiario')
  await page.waitForLoadState('networkidle')

  const legenda = page.getByText('trocar de painel')
  await expect(legenda).toBeVisible()
  // Só o que funciona: o Esc está morto nesta cena e não pode ser anunciado.
  await expect(page.locator('body')).not.toContainText('Esc voltar')

  await page.setViewportSize({ width: 640, height: 900 })
  await expect(legenda).toBeHidden()
})

/**
 * UM cursor só, e ele fica na MOLDURA da ficha — não por dentro do que rola.
 *
 * Duas formas erradas antes desta, as duas vistas pelo dono na tela: anel no
 * miolo desenhava por dentro do scroll (acompanha a rolagem, some no corte), e
 * anel na moldura via `:has()` deixou DOIS, porque o miolo continuava pegando a
 * regra global de foco do `index.css`.
 *
 * A causa é uma convenção da casa que eu não seguia: item dentro de
 * `[data-nav-region]` não usa anel de navegador, usa a linguagem de
 * "selecionado" (borda dourada e brilho) — e o seletor dela pede
 * `a`, `button` ou `data-nav-item`. Um `[tabindex]` puro cai na regra geral.
 *
 * E2E porque cascata com `:has()`, camadas e duas folhas só o navegador resolve.
 */
test('a ficha focada acende UM cursor, e na moldura', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 760 })
  await page.goto('/piloto/mestre/bestiario')
  await page.waitForLoadState('networkidle')

  await page.locator('a[href*="criatura="]').first().focus()
  await page.keyboard.press('ArrowRight')

  const medida = await page.evaluate(() => {
    const desenha = (e: Element) => {
      const cs = getComputedStyle(e)
      return cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) > 0
    }
    const painel = document.querySelector('.mesa-painel') as HTMLElement
    return {
      focoNaFicha: !!document.activeElement?.closest('[data-nav-region="ficha"]'),
      comAnel: [...document.querySelectorAll('*')].filter(desenha).map((e) =>
        String((e as HTMLElement).className).slice(0, 30),
      ),
      molduraAcesa: getComputedStyle(painel).boxShadow !== 'none',
      mioloComBrilho:
        getComputedStyle(document.activeElement as HTMLElement).boxShadow !== 'none',
    }
  })

  // O CONTROLE: o foco está na ficha. Sem ele, "nenhum anel" seria verdade sobre
  // uma tela em que nada está focado.
  expect(medida.focoNaFicha, 'o guarda não chegou na ficha').toBe(true)

  expect(medida.comAnel, `anéis de contorno desenhados: ${medida.comAnel.join(' / ')}`).toEqual([])
  expect(medida.molduraAcesa, 'a moldura da ficha não acendeu').toBe(true)
  expect(medida.mioloComBrilho, 'o miolo acendeu por dentro do que rola').toBe(false)
})
