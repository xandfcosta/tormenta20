import { expect, test } from '@playwright/test'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('A cena de personagens', () => {
  test.use({ storageState: '.auth/user.json' })

  /**
   * A tecla `D` abre o dossiê, e a dica `D` na tela diz a verdade.
   *
   * O guarda existe por causa da dica: anunciar um atalho morto ensina errado, e
   * é mais fácil escrever a dica do que ligar a tecla. Aqui os dois andam juntos
   * ou nenhum anda.
   *
   * E2E porque teclado é do navegador — e porque o painel é `position: fixed`, o
   * que já me enganou uma vez: `offsetParent` é NULL para elemento fixo, então
   * um verificador escrito com ele reporta "fechado" com o painel aberto na
   * cara. A checagem é por `display`.
   */
  test('a tecla D abre e fecha o dossiê do herói em cena', async ({ page }) => {
    await page.goto('/personagens')
    await page.getByRole('option').first().focus()

    const dossie = page.locator('aside[aria-label^="Dossiê"]').first()
    await expect(dossie).toBeHidden()

    await page.keyboard.press('d')
    await expect(dossie).toBeVisible()
    // Ele traz o que o servidor já tinha: as habilidades da raça vêm do
    // catálogo EMBUTIDO, e o navegador não baixou catálogo nenhum para isso.
    await expect(dossie.getByText(/HABILIDADES DE/i)).toBeVisible()

    await page.keyboard.press('d')
    await expect(dossie).toBeHidden()
  })

  /**
   * Trocar de herói não pede nada ao servidor — mesmo contrato da cena de
   * campanhas, e o guarda conta as requisições pelo mesmo motivo.
   *
   * Aqui ele vale ainda mais: a Defesa de CADA herói já vem computada na
   * página. Se alguém trocar isso por uma busca sob demanda, andar no elenco
   * passa a custar uma chamada da `ComputeSheetV2` por passo.
   */
  test('as setas trocam de herói sem pedir nada ao servidor', async ({ page }) => {
    await page.goto('/personagens')
    const opcoes = page.getByRole('option')
    await expect(opcoes.first()).toBeVisible()

    let pedidos = 0
    page.on('request', () => {
      pedidos++
    })

    await opcoes.first().focus()
    await page.keyboard.press('ArrowRight')

    await expect(opcoes.nth(1)).toHaveAttribute('aria-selected', 'true')
    await expect(opcoes.first()).toHaveAttribute('aria-selected', 'false')
    expect(pedidos, 'andar no elenco foi à rede — o cursor deixou de ser sinal').toBe(0)
  })

  /**
   * O ⏎ leva à FICHA, e nenhuma outra camada vê essa garantia: o guarda em Go
   * conhece só o HTML de um lado.
   */
  test('⏎ no trilho abre a ficha do herói em cena', async ({ page }) => {
    await page.goto('/personagens')
    const primeiro = page.getByRole('option').first()
    await primeiro.focus()
    const nome = (await primeiro.getAttribute('aria-label'))?.split(' · ')[0]

    await page.keyboard.press('Enter')

    await expect(page).toHaveURL(/\/personagens\/\d+$/)
    await expect(page.getByRole('heading', { name: nome, level: 1 }).first()).toBeVisible()
  })

  /**
   * A vaga de criar é POSIÇÃO DE CURSOR, e a seta chega nela. O guarda em Go
   * afirma que ela declara `role=option` e escreve o cursor; que a SETA de fato
   * pare ali e que o ⏎ leve à Forja é do teclado, e teclado é do navegador.
   *
   * Sem o ⏎ aqui a gramática morre na última posição: a tecla que abriu tudo
   * até então não faz nada justamente onde não há ficha para abrir.
   */
  test('a seta alcança a vaga de criar e ⏎ leva à Forja', async ({ page }) => {
    await page.goto('/personagens')
    // "Forjar um herói", e não "…um NOVO herói": o marcador MOSTRA o rótulo em
    // vez de o esconder num `aria-label`, e ele é o mesmo texto do título do
    // palco. O retrato tracejado segue dizendo "novo" — lá a palavra distingue a
    // vaga das capas ao redor, e aqui ela só truncaria em 208px.
    const vaga = page.getByRole('option', { name: 'Forjar um herói' })
    await expect(vaga).toBeVisible()

    await page.getByRole('option').first().focus()
    // Anda até o fim do trilho: a vaga é a última posição, sempre.
    for (let i = 0; i < 30; i++) await page.keyboard.press('ArrowRight')
    await expect(vaga).toHaveAttribute('aria-selected', 'true')
    await expect(page.getByRole('heading', { name: 'Forjar um herói' })).toBeVisible()

    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/personagens\/nova/)
  })

  /**
   * O NOME do vizinho fica sempre à mostra: esconder a legenda atrás de
   * `group-hover` a apaga no toque e sob navegação por teclado, que são os dois
   * modos em que duas iniciais não dizem quem vem a seguir.
   *
   * E2E porque a garantia é de CSS COMPUTADO: em jsdom nenhuma folha se aplica,
   * e o guarda em Go só sabe que o texto está no HTML — texto no HTML com
   * `opacity: 0` passaria nos dois.
   */
  test('os peeks mostram o nome do vizinho sem precisar de hover', async ({ page }) => {
    await page.goto('/personagens')
    await page.getByRole('option').first().focus()
    // Um passo para dentro, para haver vizinho dos DOIS lados do palco.
    await page.keyboard.press('ArrowRight')

    for (const lado of [/^Anterior:/, /^Próximo:/]) {
      const peek = page.getByRole('button', { name: lado })
      const legenda = peek.locator('span').last()
      await expect(legenda).not.toHaveText('')
      await expect(legenda).toHaveCSS('opacity', '1')
    }
  })

  /**
   * O retrato não escorrega nas pontas do elenco.
   *
   * No primeiro herói não há vizinho à esquerda, e a caixa vazia entra no lugar
   * dele — sem ela o retrato desliza para a esquerda ao chegar ali, e o palco
   * dança a cada passo. É LAYOUT medido, e por isso é aqui: contar `div`s por
   * classe num guarda de Go afirmaria a forma do DOM e não a garantia.
   */
  test('o retrato fica no mesmo lugar nas pontas do elenco', async ({ page }) => {
    await page.goto('/personagens')
    const retratoVisivel = () =>
      page.locator('a[aria-label^="Abrir ficha de"]:visible').first().boundingBox()

    // A ENTRADA DO PALCO desloca o retrato por 220ms de propósito, e isso não
    // afrouxa esta garantia: ela é sobre a posição em REPOUSO. Sem a espera, a
    // medição pega o meio de uma animação e mede um instante que ninguém vê
    // parado.
    //
    // A espera é pelas animações DESTA cena, pelo nome: `document.getAnimations()`
    // devolve também os `animate-pulse` da tela, que são INFINITOS — esperar
    // "nenhuma rodando" nunca terminaria.
    const palcoAssentado = () =>
      page.waitForFunction(() =>
        document
          .getAnimations()
          .filter((a) => /^(palcoEntra|placaSobe)/.test((a as CSSAnimation).animationName ?? ''))
          .every((a) => a.playState !== 'running'),
      )

    await page.getByRole('option').first().focus()
    await palcoAssentado()
    const naPonta = await retratoVisivel()

    await page.keyboard.press('ArrowRight')
    await expect(page.getByRole('button', { name: /^Anterior:/ })).toBeVisible()
    await palcoAssentado()
    const noMeio = await retratoVisivel()

    expect(naPonta, 'retrato não medido na ponta').not.toBeNull()
    expect(noMeio, 'retrato não medido no meio').not.toBeNull()
    expect(Math.round(noMeio!.x), 'o retrato escorregou ao sair da ponta').toBe(
      Math.round(naPonta!.x),
    )

    // E a VAGA de criar ocupa a mesma posição de um herói. Ela não tem nome
    // longo, nem vitais, nem resumo — e sem fileiras invisíveis do tamanho
    // deles a coluna centralizada puxa o retrato 74px para cima, que é o maior
    // salto do trilho inteiro.
    for (let i = 0; i < 30; i++) await page.keyboard.press('ArrowRight')
    await palcoAssentado()
    const vaga = await page
      .locator('a[aria-label="Forjar um novo herói"]:visible')
      .first()
      .boundingBox()
    expect(vaga, 'vaga de criar não medida').not.toBeNull()
    expect(Math.round(vaga!.y), 'o palco pulou na vaga de criar').toBe(Math.round(naPonta!.y))
    expect(Math.round(vaga!.x), 'a vaga de criar não está onde os heróis estão').toBe(
      Math.round(naPonta!.x),
    )
  })

  // Tela nova se valida nos seis formatos — regra da casa, e overflow é layout.
  test('personagens: sem scroll horizontal nos seis formatos', async ({ page }) => {
    await page.goto('/personagens')
    await expect(page.getByRole('listbox', { name: 'Personagens' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
  })

})
