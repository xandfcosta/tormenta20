import { type Page, expect, test } from '@playwright/test'
import { abreAFila, abreConsulta, cenaViva } from './support/gm-scene'

/**
 * As GAVETAS do mestre — o `SidePanel` que hospeda bestiário, encontros,
 * catálogos, elenco, regras e a fila do combate desde a ALE-198.
 *
 * Duas promessas que SÓ um browser testemunha, e por isso estão aqui e não em
 * unitário (ALE-207):
 *
 * 1. **O movimento.** jsdom não tem linha do tempo de animação: `getAnimations`
 *    devolve lista vazia e qualquer asserção sobre o que se move passa verde
 *    sobre uma gaveta que gira, some ou entra na diagonal. E o defeito que esta
 *    suíte prende era exatamente esse — o `slide-in-from-bottom` da folha do
 *    telefone continuava valendo acima do `xl`, somado ao `slide-in-from-right`
 *    da coluna, e a gaveta entrava do canto INFERIOR direito, cruzando 958px de
 *    altura numa janela de 1080.
 * 2. **A largura.** É `clamp()` sobre `vw` resolvido pelo motor de leiaute, e
 *    em jsdom não há leiaute nenhum — todo elemento mede zero.
 *
 * Só leitura: abre gaveta, mede, fecha. Nada escreve na seed.
 */

/** O instante ZERO do deslize, medido com a animação pausada. */
type Deslize = {
  /** As propriedades que os quadros animam. A promessa é: só `transform`. */
  props: string[]
  dx: number
  dy: number
  escalaX: number
  escalaY: number
  opacidade: string
  largura: number
}

/**
 * Instala o gravador ANTES do `goto`.
 *
 * `addInitScript` e não `evaluate`: o gravador precisa sobreviver a um
 * recarregamento da página. Com `evaluate` ele mora numa navegação só, e um
 * reload no meio (o HMR do dev server é o caso real) leva junto o ouvinte — o
 * teste então falha com "nenhuma animação" sobre um app que anima certo.
 *
 * Pausa em `currentTime = 0` de propósito: lido solto dentro do
 * `animationstart` o deslize já andou ~11% (medido), e a asserção sobre o
 * percurso viraria uma corrida com o relógio do compositor. Pausado, o
 * `getComputedStyle` resolve o `translate3d(100%, …)` em pixels exatos.
 */
/**
 * A consulta usada para medir o deslize é CATÁLOGOS e não o bestiário, e isso
 * não é gosto: montar a lista do bestiário DESANEXA a cena (ALE-199, a quinta
 * reincidência da família ALE-95). A gaveta é derrubada e remontada no meio da
 * abertura, e o `animationstart` do elemento que morreu não chega ao gravador —
 * o teste então acusa "nenhuma animação" sobre um app que animou, de forma
 * INTERMITENTE, conforme a remontagem ganhe ou perca a corrida. Medido: com o
 * bestiário, dois dos três testes de deslize alternavam entre verde e vermelho
 * a cada rodada.
 *
 * É a mesma razão pela qual o `percorrer as consultas não desanexa a cena`, no
 * `session.spec.ts`, já lista só Encontros, Catálogos e Notas. Consertar a
 * ALE-199 é o que devolve o bestiário às duas listas.
 */
async function gravaODeslize(page: Page): Promise<void> {
  await page.addInitScript(() => {
    const janela = window as unknown as { __deslizes?: unknown[]; __gravando?: boolean }
    // REUSA o array se ele já existe, e registra o ouvinte UMA vez só.
    // `addInitScript` roda a cada documento novo: recriar o array aqui deixaria
    // `window.__deslizes` apontando para um vazio enquanto um ouvinte de antes
    // continua empurrando para o array ÓRFÃO da execução anterior — e o teste
    // lê "nenhuma animação" sobre um app que animou.
    const gravadas: unknown[] = janela.__deslizes ?? []
    janela.__deslizes = gravadas
    if (janela.__gravando) return
    janela.__gravando = true

    // `getAnimations` devolve CSSTransition junto, e contar o TAMANHO da lista
    // passaria verde com a animação desligada (ALE-174). O que se afirma aqui é
    // o conjunto de propriedades que os quadros tocam.
    const propriedadesAnimadas = (animacoes: Animation[]): string[] =>
      [
        ...new Set(animacoes.flatMap((a) => a.effect?.getKeyframes().flatMap(Object.keys) ?? [])),
      ].filter((chave) => !['offset', 'computedOffset', 'easing', 'composite'].includes(chave))

    const noInstanteZero = (alvo: HTMLElement, animacoes: Animation[]) => {
      for (const a of animacoes) {
        a.pause()
        a.currentTime = 0
      }
      const estilo = getComputedStyle(alvo)
      const matriz = new DOMMatrix(estilo.transform)
      const medida = {
        dx: Math.round(matriz.e),
        dy: Math.round(matriz.f),
        escalaX: matriz.a,
        escalaY: matriz.d,
        opacidade: estilo.opacity,
        largura: Math.round(alvo.getBoundingClientRect().width),
      }
      for (const a of animacoes) a.play()
      return medida
    }

    document.addEventListener(
      'animationstart',
      (evento) => {
        const alvo = evento.target as HTMLElement
        if (alvo.getAttribute?.('role') !== 'dialog') return
        const animacoes = alvo.getAnimations()
        gravadas.push({ props: propriedadesAnimadas(animacoes), ...noInstanteZero(alvo, animacoes) })
      },
      true,
    )
  })
}

async function oDeslize(page: Page): Promise<Deslize> {
  // ESPERA a gravação em vez de ler uma vez. O diálogo fica visível antes de o
  // `animationstart` ser despachado, e a leitura imediata era uma corrida: por
  // rodada exatamente UM dos três testes de deslize caía com "nenhuma
  // animação", e qual deles variava. Se a animação de fato não existir, isto
  // ainda falha — só que no fim do tempo, e não por chegar cedo demais.
  await expect
    .poll(
      () => page.evaluate(() => (window as unknown as { __deslizes: unknown[] }).__deslizes.length),
      { message: 'a gaveta abriu sem animação nenhuma' },
    )
    .toBeGreaterThan(0)
  const gravadas = await page.evaluate(
    () => (window as unknown as { __deslizes: Deslize[] }).__deslizes,
  )
  // A PRIMEIRA é a abertura que o gesto do teste provocou; um remonte posterior
  // (HMR) empilharia outras atrás, e elas dizem a mesma coisa.
  return gravadas[0]
}

/** A gaveta deslizou o próprio corpo, e não fez mais nada. */
function expectSoODeslize(deslize: Deslize): void {
  expect(deslize.props, 'os quadros animam mais que o deslize').toEqual(['transform'])
  expect(deslize.opacidade, 'a gaveta desaparece em vez de deslizar').toBe('1')
  expect(deslize.escalaX).toBe(1)
  expect(deslize.escalaY).toBe(1)
}

async function larguraDaGaveta(page: Page, largura: number, altura: number): Promise<number> {
  await page.setViewportSize({ width: largura, height: altura })
  return page.getByRole('dialog').evaluate((no) => Math.round(no.getBoundingClientRect().width))
}

test.describe('As gavetas do mestre', () => {
  test('a consulta entra deslizando pela direita, e só isso', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 })
    await gravaODeslize(page)
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    await abreConsulta(page, 'Catálogos')
    await expect(page.getByRole('dialog', { name: 'Catálogos' })).toBeVisible()

    const deslize = await oDeslize(page)
    expectSoODeslize(deslize)
    // O percurso é o CORPO da gaveta, na horizontal, para dentro da tela. O
    // `dy` é a metade que estava quebrada: valia a altura inteira da janela.
    expect(deslize.dx, 'a gaveta da direita não parte de fora da borda direita').toBe(
      deslize.largura,
    )
    expect(deslize.dy, 'a gaveta ainda entra pela diagonal').toBe(0)
  })

  test('a fila entra pela ESQUERDA, com o mesmo deslize', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 })
    await gravaODeslize(page)
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    await abreAFila(page)

    const deslize = await oDeslize(page)
    expectSoODeslize(deslize)
    expect(deslize.dx, 'a gaveta da fila não parte de fora da borda esquerda').toBe(
      -deslize.largura,
    )
    expect(deslize.dy).toBe(0)
  })

  test('no telefone a folha sobe da borda de baixo', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await gravaODeslize(page)
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()

    await abreConsulta(page, 'Catálogos')
    await expect(page.getByRole('dialog', { name: 'Catálogos' })).toBeVisible()

    const deslize = await oDeslize(page)
    expectSoODeslize(deslize)
    // Abaixo do degrau a gaveta é folha de baixo: o percurso é VERTICAL, e
    // horizontal nenhum — ela ocupa a largura toda e não tem de onde vir.
    expect(deslize.dx).toBe(0)
    expect(deslize.dy).toBeGreaterThan(0)
  })

  test('a largura cresce com a janela e para no teto', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 })
    await page.goto('/campaigns/1/sessions/4')
    await expect(cenaViva(page)).toBeVisible()
    await abreConsulta(page, 'Catálogos')
    await expect(page.getByRole('dialog', { name: 'Catálogos' })).toBeVisible()

    // No degrau a gaveta tem a largura de sempre (26rem): alargar não podia
    // mexer em quem já cabia.
    const noDegrau = await larguraDaGaveta(page, 1280, 800)
    expect(noDegrau).toBe(416)

    const emCheio = await larguraDaGaveta(page, 1920, 1080)
    expect(emCheio, 'a gaveta não cresceu com a janela').toBeGreaterThan(noDegrau)

    const emMonitorGrande = await larguraDaGaveta(page, 2560, 1440)
    expect(emMonitorGrande).toBeGreaterThan(emCheio)

    // O TETO. Acima de 1280 o painel é não modal de propósito (ALE-75): o
    // mestre lê a condição aqui e clica no rastreador atrás, e uma gaveta que
    // crescesse sem parar acabaria cobrindo o que ele quer clicar.
    const emUltrawide = await larguraDaGaveta(page, 3440, 1440)
    expect(emUltrawide, 'a largura não tem teto').toBe(emMonitorGrande)

    // E em toda largura sobram dois terços da janela para a cena.
    for (const [janela, gaveta] of [
      [1280, noDegrau],
      [1920, emCheio],
      [2560, emMonitorGrande],
      [3440, emUltrawide],
    ] as const) {
      expect(gaveta / janela, `a gaveta come a cena @ ${janela}`).toBeLessThanOrEqual(1 / 3)
    }
  })
})
