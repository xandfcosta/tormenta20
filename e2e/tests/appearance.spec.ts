import { expect, test } from '@playwright/test'
import { medeOContraste } from './support/contrast'
import { expectNothingIsClippedSideways } from './support/geometry'
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
    })
  }
}
