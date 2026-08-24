import { expect, test } from '@playwright/test'

/**
 * O RODAPÉ DO MESTRE na Mesa em Datastar (ALE-263).
 *
 * Três guardas, e cada um está aqui porque o navegador é a ÚNICA testemunha —
 * que é o padrão de justificativa que o `CLAUDE.md` exige de e2e. "É jornada do
 * usuário" não seria motivo: jornada sai mais barata e mais firme em teste de
 * integração.
 *
 *  1. **O nome do sinal.** Nome de ATRIBUTO é minusculado pelo analisador de
 *     HTML, e nenhuma outra camada faz isso. O teste de handler manda um corpo
 *     escrito à mão e passa verde sobre o defeito; jsdom não tem Datastar.
 *  2. **A consulta de contêiner.** É leiaute real medido em altura — jsdom mede
 *     zero e diria verde sobre um rodapé enrolado em três fileiras.
 *  3. **A centralização do `<dialog>`.** Depende da margem que o `preflight` do
 *     Tailwind zera e da camada de topo do navegador.
 *
 * A página é do Go, não da SPA: é o proxy `/piloto` que a alcança em dev, e o
 * mesmo binário a serve no alvo de build. Se o piloto for apagado, este arquivo
 * vai junto.
 */

test.use({ storageState: '.auth/user.json' })

const MESA = '/piloto/mesa/1/4'

const rodape = 'section[aria-label="Controles do mestre"]'

test.describe('O rodapé do mestre (piloto Datastar)', () => {
  /**
   * O DEFEITO QUE SÓ O NAVEGADOR MOSTRA (ALE-263).
   *
   * `data-bind:qualidadeDoDescanso` chega ao Datastar como
   * `data-bind:qualidadedodescanso`, porque nome de atributo é minusculado pelo
   * analisador — e ele então liga um sinal NOVO com esse nome. Medido antes do
   * conserto: o fio levava os DOIS, `"qualidadeDoDescanso":"normal"` (o
   * declarado, que ninguém tocou) e `"qualidadedodescanso":"luxuosa"` (a
   * escolha real), e o servidor lia o primeiro. O mestre escolhia Luxuosa e o
   * grupo descansava em normal — um número plausível no lugar do certo.
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
    await page.route('**/rest/day', async (rota) => {
      corpo = rota.request().postData()
      await rota.abort()
    })

    await page.getByRole('button', { name: '🌙 Recuperar · dia' }).first().click()
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
    expect(sinais.qualidadedodescanso, `o fio levou ${corpo}`).toBe('luxuosa')
    // E leva UM nome só. Dois — o declarado e o que o `data-bind` inventou — é
    // exatamente a forma do defeito, e ela passa despercebida porque o valor
    // certo ESTÁ lá, só que na chave que o servidor não lê.
    expect(Object.keys(sinais).filter((k) => k.toLowerCase().includes('qualidade'))).toHaveLength(1)
  })

  /**
   * A ECONOMIA DA ALE-146, medida (ALE-263).
   *
   * A recuperação e o encerrar cena descem para a gaveta quando o palco é
   * baixo, e quem os move é a consulta de contêiner — a tradução do `palcoBaixo`
   * da SPA, que decide leiaute a partir de altura MEDIDA e por isso não tinha
   * como nascer no Go.
   *
   * Os dois formatos são os que aquela issue nomeou: no celular DEITADO cada
   * fileira de cromo a menos é uma linha de combatente a mais; em PÉ há altura
   * de sobra e esconder seria custo sem troca.
   *
   * `toBeVisible` e não `getBoundingClientRect`: medido, um filho de `<details>`
   * fechado ainda DEVOLVE retângulo (o navegador usa `content-visibility`), e
   * ler aquele retângulo como "está na tela" é o instrumento mentindo com cara
   * de resultado.
   */
  test('a recuperação sai da fileira quando o palco é baixo', async ({ page }) => {
    await page.goto(MESA)
    const naFileira = page.locator(`${rodape} .palco-alto-so`).getByRole('button', {
      name: '⏳ Recuperar · cena',
    })
    const gaveta = page.locator(`${rodape} details.palco-gaveta`)

    // Localizador de CSS e não `getByRole('button')`: `<summary>` não expõe o
    // papel de botão, então aquele localizador não achava nada — e `toBeHidden`
    // sobre o que não existe passa VERDE. A primeira metade deste teste estava
    // passando por vácuo, e só a segunda denunciou, porque "não achei" e "está
    // escondido" são a mesma linha para o `toBeHidden`. Este casa o elemento nas
    // DUAS alturas, e aí a visibilidade responde pela consulta de contêiner.
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
    await expect(painel.getByRole('button', { name: '⏳ Recuperar · cena' })).toBeVisible()
    const caixa = await painel.evaluate((el) => {
      const r = el.getBoundingClientRect()
      return { top: r.top, bottom: r.bottom, left: r.left, right: r.right }
    })
    expect(caixa.top, 'a gaveta estourou por cima da janela').toBeGreaterThanOrEqual(0)
    expect(caixa.left, 'a gaveta estourou pela esquerda').toBeGreaterThanOrEqual(0)
    expect(caixa.right, 'a gaveta estourou pela direita').toBeLessThanOrEqual(844)
  })

  /**
   * A LINHA DA FILA com os quatro verbos do mestre, nos dois formatos de celular
   * (ALE-263).
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
    await page.getByRole('button', { name: '+ Adicionar grupo' }).click()
    const linha = page.locator('#mesa ol li').first()
    await expect(linha).toBeVisible()
    await expect(linha.getByRole('button', { name: /^Ferir / })).toBeVisible()

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
   * O COMANDO REMENDA A CENA SOZINHO, sem depender do stream (ALE-263).
   *
   * A primeira versão deste teste ia medir "o remendo chega antes do batimento",
   * e ela seria VÁCUO: o `sessionStore` avisa quem escuta a cada mutação
   * (`Assinar`), então o stream acorda na hora e entregaria a mesma tela
   * igualmente rápido — o teste passaria verde com o remendo REMOVIDO, medindo o
   * caminho que ele existe para dispensar.
   *
   * O que o remendo compra de verdade é INDEPENDÊNCIA: a cena de quem clicou é
   * redesenhada pela resposta do PRÓPRIO comando. Então a forma honesta de
   * medi-lo é cortar o stream e ver se o clique ainda funciona — com o canal
   * fechado, o único caminho possível para o DOM mudar é a resposta do POST.
   *
   * O verbo escolhido é o OLHO e não o avanço, e isso é deliberado: dois cliques
   * o devolvem ao estado original, e a sessão 1/4 é compartilhada por seis
   * specs. Iniciar cena para medir o avanço deixaria estado ligado para os
   * outros — e encerrá-la expira os efeitos de duração cena das fichas do grupo,
   * que é um efeito colateral bem maior que o teste. O avanço tem guarda de fio
   * no Go (`TestOComandoRemendaACenaNaHora`); o que falta a ele é só o navegador,
   * e o mecanismo é o mesmo caminho de código.
   */
  test('o comando redesenha a cena mesmo com o stream cortado', async ({ page }) => {
    // Cortado ANTES da carga: o `data-init` abre o stream ao montar a página, e
    // abortar depois deixaria uma conexão viva que poderia entregar o remendo e
    // fazer o teste passar pelo motivo errado.
    // REGEX e não glob: o `@get` do Datastar anexa os sinais da página como
    // query string, então a URL é `.../stream?datastar={...}` e um glob
    // terminado em `/stream` não casa. Foi o controle abaixo que denunciou —
    // sem ele este teste teria passado verde com o stream ABERTO, medindo
    // exatamente o caminho que ele existe para excluir.
    let tentouAbrir = 0
    await page.route(/\/piloto\/mesa\/\d+\/\d+\/stream(\?|$)/, async (rota) => {
      tentouAbrir++
      await rota.abort()
    })

    await page.goto(MESA)
    await page.getByRole('button', { name: '+ Adicionar grupo' }).click()

    const olho = page
      .locator('#mesa ol li')
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
   * O `<dialog>` modal centralizado (ALE-263).
   *
   * O `preflight` do Tailwind zera a margem de TODO elemento, e a centralização
   * do `<dialog>` modal é justamente a `margin: auto` que o navegador aplica
   * sobre `inset: 0`. Medido antes do conserto: `top: 0, left: 0` numa janela de
   * 1916×907 — o diálogo nascia grudado no canto.
   *
   * Nenhuma camada abaixo desta enxerga isso: é regra do agente do usuário
   * combinada com a folha compilada, e a camada de topo só existe no navegador.
   */
  test('o diálogo do descanso nasce centralizado', async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 768 })
    await page.goto(MESA)
    await page.getByRole('button', { name: '🌙 Recuperar · dia' }).first().click()

    const caixa = page.locator('dialog#descanso-de-dia')
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
