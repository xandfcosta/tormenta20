import { expect, test } from '@playwright/test'
import { medeOContraste } from './support/contrast'
import { expectTextIsDrawnWithAShippedFace } from './support/fonts'
import { expectNothingIsClippedSideways, measureSidewaysClipping } from './support/geometry'
import { MEASURED_SCENES, SESSION_FILE } from './support/measured-scenes'
import { expectCinzelAcimaDoPiso } from './support/typography'

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
//   /campanhas/nova        12 textos   ← a cena mais enxuta do app
//   qualquer 404            1 texto
//
// Oito fica abaixo da cena mais magra e oito vezes acima da página de erro. Um
// número cravado por palpite reprova cena que está perfeita, e ruído é o que faz
// um guarda ser desligado na segunda semana.
//
// Ele é GLOBAL de propósito. Um número por cena seria uma segunda lista para
// envelhecer, e o que este controle precisa pegar é a página que não carregou —
// não "esta tela tem pouco texto".
const MEASURED_TEXT_FLOOR = 8

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

// A DÍVIDA, e ela só ENCOLHE.
//
// Três cenas não aguentam o passo hoje, e as duas primeiras são a mesma causa
// que esta fatia consertou no cartão de herói: item de GRADE tem `min-width:
// auto`, então a coluna não desce abaixo do min-content do cartão e o `truncate`
// nunca chega a agir. A terceira é o monograma gigante do palco (`absolute
// inset-0`, `text-[7rem]`), que transborda de propósito e o que falta a ele é a
// marca de isenção que o guarda irmão já tem.
//
// Cena que sair da dívida e continuar na lista REPROVA, senão a lista vira
// mentira sozinha — é a segunda direção da catraca dos arquivos longos.
const WIDER_FACE_DEBT = new Set([
  'campaigns · /campanhas · gm',
  'characters · /personagens · gm',
  'table · /campanhas/1/sessoes/4 · player',
])

for (const [label, scene] of Object.entries(MEASURED_SCENES)) {
  for (const visit of scene.visits) {
    // O papel entra no NOME porque a Mesa aparece duas vezes: o mestre e o
    // jogador recebem metades diferentes do mesmo endereço, e um relatório com
    // dois casos de nome igual não diz qual reprovou.
    const where = `${label} · ${visit.address} · ${visit.session}`

    test.describe(`aparência: ${where}`, () => {
      test.use({ storageState: SESSION_FILE[visit.session] })

      test(`sem texto abaixo do mínimo de contraste do AA em ${where}`, async ({ page }) => {
        const response = await page.goto(visit.address)
        if (visit.mayBeAbsent && response?.status() === 404) {
          test.skip(true, `esta bancada não serve ${visit.address}`)
          return
        }
        expect(response?.status(), `${visit.address} não respondeu`).toBeLessThan(400)

        const contrast = await medeOContraste(page)
        expect(
          contrast.medidos,
          `${where}: o medidor achou ${contrast.medidos} textos — a cena não carregou, e a asserção seguinte não seria evidência de nada`,
        ).toBeGreaterThan(MEASURED_TEXT_FLOOR)
        expect(contrast.falhas, `texto abaixo do AA em ${where}`).toEqual([])
      })

      test(`a Cinzel não desce abaixo do piso de leitura em ${where}`, async ({ page }) => {
        const response = await page.goto(visit.address)
        if (visit.mayBeAbsent && response?.status() === 404) {
          test.skip(true, `esta bancada não serve ${visit.address}`)
          return
        }
        await expectCinzelAcimaDoPiso(page, `em ${where}`)
      })

      // O CORTE LATERAL entra no MESMO laço, e essa é a decisão da ALE-342.
      //
      // O medidor existia desde a ALE-337 e estava ligado numa cena só — o
      // catálogo. Uma cena que voltasse a cortar em qualquer outro lugar não
      // acusava nada, que é a primeira das quatro formas de "não visitar" que o
      // CLAUDE.md lista: a cena não está na lista.
      //
      // Aqui ele não ganha lista PRÓPRIA: pega carona no `MEASURED_SCENES`, que
      // o `convention.TestEveryPageSceneIsMeasuredForAppearance` já obriga a
      // estar completo. Cena nova entra nos três medidores de uma vez, e não há
      // uma segunda enumeração para envelhecer sozinha.
      //
      // A 390px porque é a largura em que sobra menos espaço, e conteúdo que não
      // cabe ali é conteúdo que ninguém alcança — ele não rola.
      test(`nada é cortado de lado a 390px em ${where}`, async ({ page }) => {
        await page.setViewportSize({ width: 390, height: 844 })
        const response = await page.goto(visit.address)
        if (visit.mayBeAbsent && response?.status() === 404) {
          test.skip(true, `esta bancada não serve ${visit.address}`)
          return
        }
        expect(response?.status(), `${visit.address} não respondeu`).toBeLessThan(400)
        await expectNothingIsClippedSideways(page, 'body')
      })

      test(`o leiaute aguenta uma face mais larga em ${where}`, async ({ page }) => {
        await page.setViewportSize({ width: 390, height: 844 })
        const response = await page.goto(visit.address)
        if (visit.mayBeAbsent && response?.status() === 404) {
          test.skip(true, `esta bancada não serve ${visit.address}`)
          return
        }
        expect(response?.status(), `${visit.address} não respondeu`).toBeLessThan(400)
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
      })

      // A FACE DESENHADA entra no mesmo laço pela razão do corte lateral: ela é
      // o que torna os outros três medidores repetíveis. Contraste, piso da
      // Cinzel e corte lateral medem texto DESENHADO, e com a face vindo da
      // máquina os três respondem sobre o computador que rodou a suíte
      // (ALE-362).
      test(`o texto é desenhado com a fonte que o app entrega em ${where}`, async ({ page }) => {
        const response = await page.goto(visit.address)
        if (visit.mayBeAbsent && response?.status() === 404) {
          test.skip(true, `esta bancada não serve ${visit.address}`)
          return
        }
        expect(response?.status(), `${visit.address} não respondeu`).toBeLessThan(400)
        await expectTextIsDrawnWithAShippedFace(page, where)
      })
    })
  }
}
