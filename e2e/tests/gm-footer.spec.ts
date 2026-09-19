import { expect, type Page, test } from '@playwright/test'

/**
 * O RODAPÉ DO MESTRE na Mesa. Cada caso está aqui porque o navegador é a ÚNICA
 * testemunha:
 *
 *  1. **O nome do sinal.** Nome de ATRIBUTO é minusculado pelo analisador de
 *     HTML, e nenhuma outra camada faz isso. O teste de handler manda um corpo
 *     escrito à mão e passa verde sobre o defeito; jsdom não tem Datastar.
 *  2. **A consulta de contêiner.** É leiaute real medido em altura — jsdom mede
 *     zero e diria verde sobre um rodapé enrolado em três fileiras.
 *  3. **A centralização do `<dialog>`.** Depende da margem que o `preflight` do
 *     Tailwind zera e da camada de topo do navegador.
 */

test.use({ storageState: '.auth/user.json' })

const MESA = '/campanhas/1/sessoes/4'

const rodape = 'section[aria-label="Controles do mestre"]'

/**
 * Abre a GAVETA da fila, onde moram dano, ordem, condição, "+ Combatente" e
 * "Adicionar grupo".
 *
 * Sem este passo os botões existem no HTML dentro de um `<dialog>` FECHADO, que
 * o navegador esconde com `display:none`. O sintoma não é "não achei o botão":
 * é um TIMEOUT de clique em cima de um seletor que casou.
 *
 * UM seletor nas duas larguras: acima de 1024 quem abre é o ⤢ do trilho, abaixo
 * é o botão da fileira de consultas, e os dois têm o mesmo prefixo de nome
 * acessível de propósito. O `visible` é o que escolhe entre eles — o outro está
 * no DOM com `display:none`, e sem o filtro o `.first()` acertaria o escondido.
 */
async function openTheTracker(page: Page): Promise<void> {
  await page
    .getByRole('button', { name: /^Abrir a iniciativa/ })
    .filter({ visible: true })
    .click()
  await expect(page.locator('#tracker-drawer'), 'a gaveta da fila não abriu').toHaveAttribute(
    'open',
    '',
  )
}

test.describe('O rodapé do mestre', () => {
  /**
   * O DEFEITO QUE SÓ O NAVEGADOR MOSTRA.
   *
   * Uma chave de atributo escrita em camelCase chega ao Datastar MINÚSCULA,
   * porque nome de atributo é minusculado pelo analisador — e ele então liga um
   * sinal NOVO com esse nome. O fio passa a levar os DOIS, o declarado com o
   * valor intocado e o minúsculo com a escolha real, e o servidor lê o primeiro:
   * um número plausível no lugar do certo. O `_` de `rest_quality` atravessa o
   * parser intacto.
   *
   * O guarda Go ao lado NÃO pega isto: ele monta o corpo à mão e por isso
   * afirma o servidor, não a página. Este afirma o FIO.
   *
   * A requisição é interceptada e ABORTADA de propósito: o que se mede é o que
   * a página manda, e descansar de verdade mexeria nos PV de fichas que outros
   * specs leem.
   */
  test('a qualidade que o mestre escolhe é a que vai no fio', async ({ page }) => {
    await page.goto(MESA)

    let corpo: string | null = null
    await page.route('**/descanso/dia', async (rota) => {
      corpo = rota.request().postData()
      await rota.abort()
    })

    await page.getByRole('button', { name: 'Recuperar · dia' }).first().click()
    const qualidade = page.getByLabel('Qualidade do descanso')
    await expect(qualidade).toBeVisible()
    // O CONTROLE de que o sinal declarado chega ao elemento: sem ele o select
    // nasceria na primeira opção ("ruim") e a tela mentiria antes de qualquer
    // escolha.
    await expect(qualidade).toHaveValue('normal')

    await qualidade.selectOption('luxuosa')
    await page.getByRole('button', { name: 'Descansar' }).click()

    await expect.poll(() => corpo, { message: 'o clique não postou nada' }).not.toBeNull()
    const sinais = JSON.parse(corpo ?? '{}') as Record<string, unknown>
    expect(sinais.rest_quality, `o fio levou ${corpo}`).toBe('luxuosa')
    // E leva UM nome só. Dois — o declarado e o que o `data-bind` inventou — é
    // exatamente a forma do defeito, e ela passa despercebida porque o valor
    // certo ESTÁ lá, só que na chave que o servidor não lê.
    expect(Object.keys(sinais).filter((k) => k.toLowerCase().includes('quality'))).toHaveLength(1)
  })

  /**
   * A recuperação e o encerrar cena descem para a gaveta quando o palco é
   * baixo, e quem os move é a consulta de contêiner: leiaute decidido a partir
   * de altura MEDIDA, que por isso não tinha como nascer no Go. No celular
   * DEITADO cada fileira de cromo a menos é uma linha de combatente a mais; em
   * PÉ há altura de sobra e esconder seria custo sem troca.
   *
   * `toBeVisible` e não `getBoundingClientRect`: medido, um filho de `<details>`
   * fechado ainda DEVOLVE retângulo (o navegador usa `content-visibility`), e
   * ler aquele retângulo como "está na tela" é o instrumento mentindo com cara
   * de resultado.
   */
  test('a recuperação sai da fileira quando o palco é baixo', async ({ page }) => {
    await page.goto(MESA)
    const naFileira = page.locator(`${rodape} .stage-tall-only`).getByRole('button', {
      name: 'Expirar efeitos · cena',
    })
    const gaveta = page.locator(`${rodape} details.stage-drawer`)

    // Localizador de CSS e não `getByRole('button')`: `<summary>` não expõe o
    // papel de botão, então aquele localizador não acha nada — e `toBeHidden`
    // sobre o que não existe passa VERDE, porque "não achei" e "está escondido"
    // são a mesma linha para ele. Este casa o elemento nas DUAS alturas, e aí a
    // visibilidade responde pela consulta de contêiner.
    const abrir = page.locator('summary[aria-label="Mais comandos da mesa"]')

    await page.setViewportSize({ width: 390, height: 844 })
    await expect(naFileira, 'em pé a recuperação fica à vista').toBeVisible()
    await expect(abrir, 'em pé a gaveta não tem por que existir').toBeHidden()

    await page.setViewportSize({ width: 844, height: 390 })
    await expect(naFileira, 'deitado ela sai da fileira').toBeHidden()
    await expect(abrir, 'deitado a gaveta é o único caminho até ela').toBeVisible()

    // A fileira ÚNICA é a troca inteira: se o rodapé enrolasse, esconder os
    // botões não teria comprado nada. 61px foi o medido com uma fileira; o teto
    // dá folga para o alvo de toque de 44px mais a borda, e denuncia a segunda.
    const altura = await page.locator(rodape).evaluate((el) => el.getBoundingClientRect().height)
    expect(altura, 'o rodapé enrolou em mais de uma fileira').toBeLessThan(80)

    // E a gaveta aberta cabe na janela: ela sobe (`bottom-full`) porque nasce no
    // rodapé, e subir é o que pode estourar por cima num palco de 390.
    await abrir.click()
    const painel = gaveta.locator('div').first()
    await expect(painel.getByRole('button', { name: 'Expirar efeitos · cena' })).toBeVisible()
    const caixa = await painel.evaluate((el) => {
      const r = el.getBoundingClientRect()
      return { top: r.top, bottom: r.bottom, left: r.left, right: r.right }
    })
    expect(caixa.top, 'a gaveta estourou por cima da janela').toBeGreaterThanOrEqual(0)
    expect(caixa.left, 'a gaveta estourou pela esquerda').toBeGreaterThanOrEqual(0)
    expect(caixa.right, 'a gaveta estourou pela direita').toBeLessThanOrEqual(844)
  })

  /**
   * A LINHA DA FILA com os quatro verbos do mestre.
   *
   * Quatro alvos de 36px mais o número da iniciativa mais o nome, numa tela de
   * 390: é o formato onde a fileira estoura, e estourar aqui significa a lixeira
   * saindo pela borda ou o nome empurrando os verbos para fora da caixa. jsdom
   * mede zero e passaria verde sobre os dois.
   *
   * Quem cede é o NOME, que tem `truncate` — e é isso que a asserção prende: os
   * verbos INTEIROS dentro da linha, em vez de "a página não rola de lado", que
   * é mais fraco e não vê recorte dentro de um contêiner.
   */
  test('os verbos da linha cabem na fila a 390px', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await page.goto(MESA)

    // O CONTROLE: sem linha na fila não há o que medir, e um `toBeLessThan`
    // sobre uma lista vazia passa verde dizendo nada. O "Adicionar grupo" é
    // idempotente, então chamá-lo aqui não depende do que outro spec deixou.
    await openTheTracker(page)
    await page.getByRole('button', { name: '+ Adicionar grupo' }).click()
    const linha = page.locator('#table ol li').first()
    await expect(linha).toBeVisible()
    await expect(linha.getByRole('button', { name: /^Ferir / })).toBeVisible()
    // E a linha medida TEM o crachá, senão a medida é de outra linha que não a
    // que corre risco: o selo de 5 letras é o que empurra os verbos, e uma
    // fileira sem ele passaria verde sobre o caso que importa.
    await expect(linha.getByText('Ficha', { exact: true })).toBeVisible()

    const medida = await linha.evaluate((el) => {
      const caixa = el.getBoundingClientRect()
      const verbos = el.querySelector('div.shrink-0') as HTMLElement
      const v = verbos.getBoundingClientRect()
      return {
        recorte: el.scrollWidth - el.clientWidth,
        verbosForaPelaDireita: v.right - caixa.right,
        verbosForaPelaEsquerda: caixa.left - v.left,
      }
    })
    expect(medida.recorte, 'a linha recortou o próprio conteúdo').toBeLessThanOrEqual(1)
    expect(medida.verbosForaPelaDireita, 'os verbos saíram pela direita da linha').toBeLessThanOrEqual(1)
    expect(medida.verbosForaPelaEsquerda, 'os verbos saíram pela esquerda da linha').toBeLessThanOrEqual(1)
  })

  /**
   * O COMANDO REMENDA A CENA SOZINHO, sem depender do stream.
   *
   * Medir "o remendo chega antes do batimento" seria VÁCUO: o stream acorda a
   * cada mutação e entregaria a mesma tela igualmente rápido, então o teste
   * passaria verde com o remendo REMOVIDO. O que o remendo compra é
   * INDEPENDÊNCIA — então a forma honesta de medi-lo é cortar o stream: com o
   * canal fechado, o único caminho possível para o DOM mudar é a resposta do
   * POST.
   *
   * O verbo escolhido é o OLHO e não o avanço, e isso é deliberado: dois cliques
   * o devolvem ao estado original, e a sessão é compartilhada por seis specs.
   * Iniciar cena deixaria estado ligado para os outros, e encerrá-la expira os
   * efeitos de duração cena das fichas do grupo. O avanço tem guarda de fio no
   * Go (`TestTheCommandPatchesTheSceneRightAway`), pelo mesmo caminho de código.
   */
  test('o comando redesenha a cena mesmo com o stream cortado', async ({ page }) => {
    // Cortado ANTES da carga: o `data-init` abre o stream ao montar a página, e
    // abortar depois deixaria uma conexão viva que poderia entregar o remendo e
    // fazer o teste passar pelo motivo errado.
    // REGEX e não glob: o `@get` do Datastar anexa os sinais da página como
    // query string, então a URL é `.../fluxo?datastar={...}` e um glob
    // terminado em `/fluxo` não casa — o teste passaria verde com o stream
    // ABERTO, medindo exatamente o caminho que ele existe para excluir. Quem
    // denuncia isso é o controle abaixo.
    let tentouAbrir = 0
    await page.route(/\/campanhas\/\d+\/sessoes\/\d+\/fluxo(\?|$)/, async (rota) => {
      tentouAbrir++
      await rota.abort()
    })

    await page.goto(MESA)
    await openTheTracker(page)
    await page.getByRole('button', { name: '+ Adicionar grupo' }).click()

    const olho = page
      .locator('#table ol li')
      .first()
      .getByRole('button', { name: /^(Ocultar|Revelar) os PV de / })
    await expect(olho).toBeVisible()
    // O CONTROLE de que o canal está mesmo fechado: a página TENTOU abri-lo e
    // foi barrada. Sem isto, um `data-init` que deixasse de existir faria este
    // teste medir uma página sem stream por acidente, e não por corte.
    expect(tentouAbrir, 'a página nem tentou abrir o stream — o corte não prova nada').toBeGreaterThan(0)

    const antes = await olho.getAttribute('aria-pressed')
    await olho.click()

    // O DOM mudou com o stream fechado: só a resposta do POST podia ter feito
    // isso. E mudou UMA vez — `aria-pressed` é o oposto, e não o valor de volta.
    const depois = antes === 'true' ? 'false' : 'true'
    await expect(olho).toHaveAttribute('aria-pressed', depois)

    // Devolve a linha ao que era: a sessão é compartilhada.
    await olho.click()
    await expect(olho).toHaveAttribute('aria-pressed', antes ?? 'false')
  })

  /**
   * NENHUM DIÁLOGO FECHADO ROUBA O CLIQUE DA CENA.
   *
   * O navegador dá `display:none` a `<dialog>` fechado, e um `flex` utilitário
   * do Tailwind SOBREPÕE isso: o diálogo fechado ocupa a tela inteira e engole o
   * clique de todo botão da Mesa. Sem este guarda o sintoma é um punhado de
   * timeouts em outros casos; com ele, o defeito aparece NOMEADO.
   *
   * Ele varre `dialog:not([open])` em vez de citar ids — amostragem e não
   * enumeração, então vale para o diálogo que nascer amanhã.
   *
   * Só o navegador responde: é a folha do agente do usuário disputando
   * especificidade com a folha compilada, e nem jsdom nem teste de handler têm
   * as duas.
   */
  test('nenhum diálogo fechado rouba o clique da cena', async ({ page }) => {
    await page.goto(MESA)

    const fechados = await page.evaluate(() =>
      [...document.querySelectorAll('dialog:not([open])')].map((d) => ({
        id: d.id,
        display: getComputedStyle(d).display,
      })),
    )
    // O CONTROLE: a cena do mestre TEM diálogos. Uma lista vazia passaria verde
    // dizendo nada, e é o que aconteceria se os ids mudassem.
    expect(fechados.length, 'a cena não tem diálogo nenhum para medir').toBeGreaterThan(0)
    for (const d of fechados) {
      expect(d.display, `o diálogo #${d.id} está fechado e ocupando a tela`).toBe('none')
    }
  })

  /**
   * O `<dialog>` modal centralizado.
   *
   * O `preflight` do Tailwind zera a margem de TODO elemento, e a centralização
   * do `<dialog>` modal é justamente a `margin: auto` que o navegador aplica
   * sobre `inset: 0` — sem o conserto, o diálogo nasce grudado no canto.
   *
   * Nenhuma camada abaixo desta enxerga isso: é regra do agente do usuário
   * combinada com a folha compilada, e a camada de topo só existe no navegador.
   */
  test('o diálogo do descanso nasce centralizado', async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 768 })
    await page.goto(MESA)
    await page.getByRole('button', { name: 'Recuperar · dia' }).first().click()

    const caixa = page.locator('dialog#day-rest')
    await expect(caixa).toBeVisible()
    const centro = await caixa.evaluate((el) => {
      const r = el.getBoundingClientRect()
      return { x: r.left + r.width / 2, y: r.top + r.height / 2 }
    })
    // Tolerância de 2px para o arredondamento de sub-pixel, e não mais: o
    // defeito que isto pega punha o centro a centenas de pixels do lugar.
    expect(Math.abs(centro.x - 512), 'fora do centro horizontal').toBeLessThanOrEqual(2)
    expect(Math.abs(centro.y - 384), 'fora do centro vertical').toBeLessThanOrEqual(2)
  })
})
