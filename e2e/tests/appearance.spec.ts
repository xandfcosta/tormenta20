import { expect, test } from '@playwright/test'
import { medeOContraste } from './support/contrast'
import { expectTextIsDrawnWithAShippedFace } from './support/fonts'
import { expectNothingIsClippedSideways, measureSidewaysClipping } from './support/geometry'
import { MEASURED_SCENES, SESSION_FILE } from './support/measured-scenes'
import { expectCinzelAcimaDoPiso } from './support/typography'
import { expectOneFocusRing } from './support/focus'
import { DS_TEXT_FLOOR, DS_TOUCH_FLOOR, measureDsText, measureDsTouchTargets } from './support/ds-floors'

/**
 * A APARÊNCIA DE TODA CENA QUE DESENHA PÁGINA.
 *
 * E2E porque não há outra testemunha: contraste exige converter oklch para sRGB,
 * e tipografia exige saber em que tamanho a Cinzel foi DESENHADA — herança de
 * CSS, não o que está escrito na `class=`. Em jsdom o `getComputedStyle` devolve
 * o oklch cru e todo elemento mede zero.
 *
 * # AMOSTRAGEM, e não uma lista de endereços
 *
 * O laço é sobre o REGISTRO, e quem força o registro a estar completo é o
 * `convention.TestEveryPageSceneIsMeasuredForAppearance`, em Go. Cena nova sem
 * linha no registro é vermelho lá, com o nome dela na mensagem.
 *
 * # O denominador, e por que ele não é cerimônia
 *
 * Os dois medidores devolvem `{falhas, medidos}`, e o caso afirma os DOIS. "Nada
 * reprovou" e "o seletor não casou com nada" se parecem no terminal, e a segunda
 * é o que acontece quando a cena não carregou — uma página de erro tem pouco
 * texto e nenhum deles reprova.
 *
 * # O que ele NÃO mede
 *
 * A cena em UM estado: a que abre no endereço, sem diálogo aberto nem dado que
 * ramifique. As telas que RAMIFICAM pelo dado continuam precisando do caso
 * próprio — o caminhar pelas sete abas da ficha mede sempre um guerreiro, e
 * metade do painel de Combate só existe para quem conjura. Este arquivo é o
 * PISO, não o teto.
 */

// O PISO DE TEXTO MEDIDO, e ele é MEDIDO e não escolhido.
//
// O controle existe para separar "a cena desenhou" de "o medidor olhou uma
// página de erro" — as duas devolvem uma lista de falhas vazia, e no terminal se
// parecem. Então o número tem de cair ENTRE as duas, e os dois lados foram
// contados no navegador:
//
//   /personagens          441 textos
//   /campanhas/nova         6 textos   ← a cena mais enxuta do app
//   qualquer 404            1 texto
//
// Quatro fica abaixo da cena mais magra e quatro vezes acima da página de erro. Um
// número cravado por palpite reprova cena que está perfeita, e ruído é o que faz
// um guarda ser desligado na segunda semana.
//
// Ele é GLOBAL de propósito. Um número por cena seria uma segunda lista para
// envelhecer, e o que este controle precisa pegar é a página que não carregou —
// não "esta tela tem pouco texto".
const MEASURED_TEXT_FLOOR = 4

// A FACE MAIS LARGA, e o passo é MEDIDO e não escolhido.
//
// A folha pede a face com `font-display: swap`: até o `.woff2` chegar, TODA
// primeira pintura é desenhada com a fonte da máquina de quem abriu. Então um
// leiaute que só cabe com a face entregue corta de verdade, na vida real, em
// todo carregamento frio — e corta de novo no dia em que a face mudar.
//
// O passo de letra é a forma de perguntar isso com um número que não depende da
// máquina: ele perturba a face ENTREGUE em uma quantidade conhecida, enquanto
// medir com a face do sistema devolveria a resposta do computador que rodou a
// suíte — que é o defeito que esta fatia veio consertar.
//
// 0,2em é o que a face do runner do GitHub custou, medida no caso que derrubou
// a CI: o número do atributo pedia 27px nesta bancada e 37px lá, e 0,2em leva
// os 27 a exatamente 37 (ALE-362).
const WIDER_FACE_STEP = '0.2em'

// A DÍVIDA, e ela só ENCOLHE. Está vazia desde que o elenco virou cartões
// (ALE-528); a causa que a enchia era item de GRADE com `min-width: auto`, que
// não deixa a coluna descer abaixo do min-content e o `truncate` agir.
//
// Cena que sair da dívida e continuar na lista REPROVA, senão a lista vira
// mentira sozinha — é a segunda direção da catraca dos arquivos longos.
const WIDER_FACE_DEBT = new Set<string>([])

/**
 * Roda os medidores de UMA navegação, um `test.step` cada, e reprova no fim com
 * a lista de quem reprovou. Os medidores usam `expect` duro por dentro, e o
 * primeiro vermelho calaria os seguintes; aqui cada um fala, com a mensagem
 * dele e o nome do medidor na frente.
 */
async function runMeasures(where: string, measures: Array<[string, () => Promise<void>]>) {
  const failures: string[] = []
  for (const [name, measure] of measures) {
    await test.step(name, async () => {
      try {
        await measure()
      } catch (error) {
        failures.push(`${name}: ${error instanceof Error ? error.message : String(error)}`)
      }
    })
  }
  expect(failures, `medidores que reprovaram em ${where}`).toEqual([])
}

for (const [label, scene] of Object.entries(MEASURED_SCENES)) {
  for (const visit of scene.visits) {
    // O papel entra no NOME porque a Mesa aparece duas vezes: o mestre e o
    // jogador recebem metades diferentes do mesmo endereço, e um relatório com
    // dois casos de nome igual não diz qual reprovou.
    const where = `${label} · ${visit.address} · ${visit.session}`

    // DUAS NAVEGAÇÕES POR CENA, e não uma por medidor (ALE-322). Cada caso
    // custava 1,1–1,5s quase todo de abrir contexto e navegar, e a cena rodava
    // seis: o arquivo era metade da suíte e crescia com o registro. Os
    // medidores que olham a MESMA página dividem a navegação; o que muda a
    // página (a face mais larga injeta estilo) vem por último.
    test.describe(`aparência: ${where}`, () => {
      test.use({ storageState: SESSION_FILE[visit.session] })

      test(`no desktop: contraste, Cinzel, piso de texto e fonte em ${where}`, async ({ page }) => {
        const response = await page.goto(visit.address)
        if (visit.mayBeAbsent && response?.status() === 404) {
          test.skip(true, `esta bancada não serve ${visit.address}`)
          return
        }
        expect(response?.status(), `${visit.address} não respondeu`).toBeLessThan(400)
        // A Cinzel é do grimório e os pisos de 12px e 44px são do Tempestade
        // (ALE-497): cada cena mede os do sistema que ela veste, e o relatório
        // diz qual foi.
        const migrated = (await page.locator('[data-slot="scene-shell"].scene-ds').count()) > 0
        test.info().annotations.push({ type: 'sistema', description: migrated ? 'Tempestade' : 'grimório' })

        await runMeasures(where, [
          ['contraste AA', async () => {
            const contrast = await medeOContraste(page)
            expect(
              contrast.medidos,
              `${where}: o medidor achou ${contrast.medidos} textos — a cena não carregou, e a asserção seguinte não seria evidência de nada`,
            ).toBeGreaterThan(MEASURED_TEXT_FLOOR)
            expect(contrast.falhas, `texto abaixo do AA em ${where}`).toEqual([])
          }],
          ...(migrated
            ? [[`texto de ${DS_TEXT_FLOOR}px ou mais`, async () => {
                const report = await measureDsText(page)
                expect(report.medidos, `${where}: nenhum texto medido no escopo .scene-ds`).toBeGreaterThan(MEASURED_TEXT_FLOOR)
                expect(report.falhas, `texto abaixo de ${DS_TEXT_FLOOR}px no sistema Tempestade em ${where}`).toEqual([])
              }] as [string, () => Promise<void>]]
            : [['Cinzel acima do piso', () => expectCinzelAcimaDoPiso(page, `em ${where}`)] as [string, () => Promise<void>]]),
          // A FACE DESENHADA é o que torna os outros medidores repetíveis: com
          // a face vindo da máquina, eles respondem sobre o computador que rodou
          // a suíte (ALE-362).
          ['fonte entregue', () => expectTextIsDrawnWithAShippedFace(page, where)],
        ])
      })

      // A 390px COM TOQUE, que é como um celular é: `hasTouch` liga o `pointer:
      // coarse`, que sobe o controle para 44px. O corte lateral mede aqui pela
      // mesma razão — conteúdo que não cabe a 390px é conteúdo que ninguém
      // alcança (ALE-342), e o celular de verdade desenha os controles de toque.
      test.describe('no celular', () => {
        test.use({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true })

        test(`no celular: alvo de toque, corte lateral e face mais larga em ${where}`, async ({ page }) => {
          const response = await page.goto(visit.address)
          if (visit.mayBeAbsent && response?.status() === 404) {
            test.skip(true, `esta bancada não serve ${visit.address}`)
            return
          }
          expect(response?.status(), `${visit.address} não respondeu`).toBeLessThan(400)
          const migrated = (await page.locator('[data-slot="scene-shell"].scene-ds').count()) > 0

          await runMeasures(where, [
            ...(migrated
              ? [[`alvo de ${DS_TOUCH_FLOOR}px ou mais`, async () => {
                  const report = await measureDsTouchTargets(page)
                  expect(report.medidos, `${where}: nenhum alvo medido no escopo .scene-ds`).toBeGreaterThan(0)
                  expect(report.falhas, `alvo menor que ${DS_TOUCH_FLOOR}×${DS_TOUCH_FLOOR} no toque em ${where}`).toEqual([])
                }] as [string, () => Promise<void>]]
              : []),
            ['nada cortado de lado', () => expectNothingIsClippedSideways(page, 'body')],
            // A FACE MAIS LARGA por último: ela injeta estilo na página, e
            // qualquer medidor depois dela mediria a página perturbada.
            ['face mais larga', async () => {
              await page.addStyleTag({
                content: `body * { letter-spacing: ${WIDER_FACE_STEP} !important }`,
              })
              const { cortados: clipped, medidos: measured } = await measureSidewaysClipping(page, 'body')
              expect(measured, `${where}: a varredura não achou nó nenhum`).toBeGreaterThan(5)
              if (WIDER_FACE_DEBT.has(where)) {
                expect(
                  clipped,
                  `${where} está na dívida da face mais larga e hoje aguenta o passo. Tire a linha de WIDER_FACE_DEBT — uma dívida que afirma o que já foi pago não é relida por ninguém`,
                ).not.toEqual([])
                return
              }
              expect(
                clipped,
                `${where}: com uma face ${WIDER_FACE_STEP} mais larga o conteúdo é cortado. Toda primeira pintura usa a fonte da máquina (\`font-display: swap\`), então isto corta de verdade em carregamento frio`,
              ).toEqual([])
            }],
          ])
        })
      })
    })
  }
}

// O CONTROLE DOS PISOS: os dois medidores acusam o que reprova de verdade. Sem
// isto, "nada abaixo do piso" e "o seletor não casou com nada" seriam a mesma
// cor no relatório.
test.describe('os pisos do sistema Tempestade', () => {
  test.use({ storageState: SESSION_FILE.gm, viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true })

  // O FOCO do sistema novo é UMA cara POR MODO, e é desenhado e não aceso: a
  // folha viva tem toda peça do kit, então é onde uma segunda cara apareceria
  // primeiro. Uma metade por vez, porque as duas lado a lado têm de fato dois
  // roxos — o claro e o escuro —, e o medidor contaria o modo como defeito.
  for (const [shown, hidden] of [['light', 'dark'], ['dark', 'light']]) {
    test(`o realce de foco é um só na metade ${shown} da folha viva`, async ({ page }) => {
      await page.goto('/sistema')
      await page.addStyleTag({
        content: `.ds-preview[data-preview='${hidden}'], [data-slot='scene-shell'] > header, form[action='/modo'] { display: none }`,
      })
      await expectOneFocusRing(page, `na metade ${shown} da folha viva`, 20)
    })
  }

  test('os medidores acusam um texto e um alvo que reprovam', async ({ page }) => {
    await page.goto('/sistema')
    await page.evaluate(() => {
      const scope = document.querySelector('.scene-ds') as HTMLElement
      const tiny = document.createElement('span')
      tiny.textContent = 'miúdo'
      tiny.style.fontSize = '10px'
      const small = document.createElement('button')
      small.textContent = 'x'
      small.style.cssText = 'width:20px;height:20px;min-width:0;padding:0'
      scope.prepend(tiny, small)
    })
    const text = await measureDsText(page)
    expect(text.falhas).toEqual(['10px: "miúdo"'])
    const touch = await measureDsTouchTargets(page)
    expect(touch.falhas).toEqual(['<button> «x» 20×20'])
  })
})
