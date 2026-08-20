import { expect, test } from '@playwright/test'
import { ensurePowersFixture } from './support/character'

/**
 * As listas virtualizadas do MESTRE — bestiário e catálogos.
 *
 * `VirtualList` mede as linhas para saber quais existem, e em jsdom todo
 * elemento mede zero: a lista renderiza NENHUMA linha e um teste de unidade
 * passa verde sobre a tela vazia. Foi assim que a ALE-84 entrou em produção com
 * a suíte inteira verde. Só um browser prova que a linha pintou — e que o
 * filtro TROCA o conjunto pintado, que é o outro modo de falha da
 * virtualização: ficar com o que pintou primeiro.
 *
 * Só leitura: filtra e navega, nunca escreve.
 */
test.describe('Listas virtualizadas do mestre', () => {
  test('o bestiário da sessão pinta linhas e o filtro troca o conjunto', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await page.getByRole('tab', { name: 'Bestiário' }).click()

    const busca = page.getByRole('searchbox', { name: 'Buscar criatura' })
    await expect(busca).toBeVisible()
    // A linha só existe se a lista mediu e pintou.
    const antes = await page.getByRole('button', { name: /ND / }).count()
    expect(antes).toBeGreaterThan(0)

    await busca.fill('ogro')
    // Mais de um ogro no bestiário (o comum e o ancião) — `first()` de propósito.
    await expect(page.getByRole('button', { name: /^Ogro/ }).first()).toBeVisible()
    expect(await page.getByRole('button', { name: /ND / }).count()).toBeLessThan(antes)
  })

  test('o catálogo da sessão pinta linhas e a busca as troca', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    await page.getByRole('tab', { name: 'Catálogos' }).click()

    const busca = page.getByRole('searchbox', { name: 'Buscar nos catálogos' })
    await expect(busca).toBeVisible()

    await busca.fill('espada')
    await expect(page.getByText(/Espada/).first()).toBeVisible()

    // O outro modo de falha: a lista guardar o que pintou primeiro. Depois de
    // uma busca sem resultado, nada de "espada" pode sobreviver na tela.
    await busca.fill('zzzzzz')
    await expect(page.getByText(/Espada longa/)).toHaveCount(0)

    // E o terceiro, que só o browser vê: a lista TERMINAR fora do cartão.
    // O painel de aba é um bloco, então o `flex-1` do filho não limitava altura
    // nenhuma e a lista descia até a borda da janela, 12px além do cartão —
    // sem a página rolar, porque a cena inteira é `overflow-hidden`, e por isso
    // "a cena do mestre não rola" ficava verde por cima (ALE-149).
    await busca.fill('')
    await expect(page.getByText('Abalado')).toBeVisible()
    const vazamento = await page.evaluate(() => {
      const rolante = [...document.querySelectorAll('*')].find((el) => {
        const estilo = getComputedStyle(el)
        return (
          /(auto|scroll)/.test(estilo.overflowY) &&
          el.scrollHeight > el.clientHeight + 4 &&
          (el.textContent ?? '').includes('Abalado')
        )
      })
      if (!rolante) throw new Error('não achei a lista de condições rolando')
      const cartao = document.querySelector('[role="tablist"]')?.parentElement
      if (!cartao) throw new Error('não achei o cartão do workspace')
      return Math.round(rolante.getBoundingClientRect().bottom - cartao.getBoundingClientRect().bottom)
    })
    expect(vazamento, `a lista passou ${vazamento}px do fundo do cartão`).toBeLessThanOrEqual(0)
  })

  test('a ferramenta Bestiário pinta a lista e abre a criatura escolhida', async ({ page }) => {
    await page.goto('/gm/bestiario')

    const busca = page.getByRole('searchbox', { name: 'Buscar criatura' })
    await expect(busca).toBeVisible()
    await busca.fill('ogro')

    const linha = page.getByRole('button', { name: /^Ogro/ }).first()
    await expect(linha).toBeVisible()
    await linha.click()

    await expect(page.getByRole('region', { name: 'Criatura escolhida' })).toContainText('Ogro')
  })


  /**
   * A sexta e última lista virtualizada da auditoria: o pool de poderes gerais,
   * dentro da ficha. Ele só pinta com o card da classe ABERTO, e o card abre
   * sozinho quando há escolha pendente — por isso o teste CRIA um Guerreiro de
   * 6º nível sem poder escolhido (três vagas em aberto) em vez de usar um
   * personagem da seed, onde as vagas já estão gastas e a lista fica fechada.
   *
   * O herói é criado pela API na primeira rodada e REUSADO nas seguintes: o app
   * não apaga personagem, então criar um por rodada entulharia o elenco.
   */
  test('o pool de poderes gerais da ficha pinta e filtra', async ({ page, request }) => {
    const id = await ensurePowersFixture(request)
    await page.goto(`/characters/${id}`)
    await page.getByRole('tab', { name: /^Poderes/ }).click()
    // O pool mora dentro do card da classe, e o card mora dentro do grupo
    // "Classe" — os dois recolhidos, nada pinta e não há o que medir.
    await page.getByRole('button', { name: /^Classe/ }).first().click()

    // `textbox`, não `searchbox`: este campo não declara `type="search"` como os
    // irmãos dele (bestiário e loja) — o papel segue o elemento, não o rótulo.
    const busca = page.getByRole('textbox', { name: 'Buscar poder geral' })
    await expect(busca).toBeVisible()
    // A linha do pool é um `div` com caixa de marcar e o nome em texto — não um
    // botão por linha, como nas outras listas.
    await expect(page.getByText('Ataque Poderoso', { exact: true })).toBeVisible()

    await busca.fill('esquiva')
    await expect(page.getByText(/Esquiva/).first()).toBeVisible()
    // O modo de falha da virtualização: guardar o que pintou primeiro.
    await expect(page.getByText('Ataque Poderoso', { exact: true })).toHaveCount(0)
  })
})

/**
 * A lista do bestiário PREENCHE a coluna no tablet em pé (ALE-175).
 *
 * A lista tinha uma tampa de `45vh` cujo comentário dizia proteger "o resto da
 * ferramenta" de 80 linhas. Mas abaixo de `lg` o resto da ferramenta é o painel
 * de detalhe, que ali é `hidden`: a tampa protegia conteúdo que não existe
 * naquele formato. O preço eram 243px mortos em 768×1024 — um quarto da tela —
 * com a lista mostrando 459px de 5216 de conteúdo.
 *
 * O que se afirma é a FAIXA MORTA e não a altura da lista: altura é
 * consequência do formato, e prendê-la seria prender o número errado. O que a
 * cena promete é não deixar banda vazia embaixo do último elemento da coluna.
 *
 * Por que e2e: é caixa contra caixa em altura real. Em jsdom todo elemento mede
 * zero e a faixa morta dá zero em qualquer arranjo.
 */
test('no tablet em pé, a lista do bestiário não deixa faixa morta', async ({ page }) => {
  await page.setViewportSize({ width: 768, height: 1024 })
  await page.goto('/gm/bestiario')
  await expect(page.getByRole('button', { name: /ND / }).first()).toBeVisible()

  const medida = await page.evaluate(() => {
    const secao = document.querySelector('[aria-labelledby=mesa-bestiario]')
    const lista = [...document.querySelectorAll<HTMLElement>('*')].find((n) =>
      (n.className || '').toString().includes('flex-1 rounded-md border'),
    )
    if (!secao || !lista) return null
    return {
      faixaMorta: Math.round(
        secao.getBoundingClientRect().bottom - lista.getBoundingClientRect().bottom,
      ),
      transbordou: lista.scrollHeight > lista.clientHeight + 8,
    }
  })

  expect(medida, 'não achei a lista do bestiário').not.toBeNull()
  // Sem transbordo a asserção não prova nada: seria uma lista que coube.
  expect(medida?.transbordou, 'a lista não transbordou — o teste não mediu nada').toBe(true)
  expect(medida?.faixaMorta ?? 999, 'faixa morta embaixo da lista').toBeLessThanOrEqual(8)
})
