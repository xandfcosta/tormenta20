import { expect, test } from '@playwright/test'
import { medeOContraste } from './support/contrast'
import { MEASURED_SCENES, SESSION_FILE } from './support/measured-scenes'
import { expectCinzelAcimaDoPiso } from './support/typography'

/**
 * A APARÊNCIA DE TODA CENA QUE DESENHA PÁGINA (ALE-320).
 *
 * E2E porque não há outra testemunha: contraste exige converter oklch para sRGB,
 * e tipografia exige saber em que tamanho a Cinzel foi DESENHADA — herança de
 * CSS, não o que está escrito na `class=`. Em jsdom o `getComputedStyle` devolve
 * o oklch cru e todo elemento mede zero.
 *
 * # Este arquivo substitui uma ENUMERAÇÃO
 *
 * A medição vivia espalhada: catorze cópias de `nenhum texto fica abaixo do
 * mínimo de contraste do AA`, uma por bloco `describe` do
 * `scenes.spec.ts`, mais avulsas no `piloto-sheet`, no `piloto-forge` e
 * no `grimorio`. Dezoito endereços escritos à mão, e nada cobrando a cena que
 * nascesse amanhã — o regime que o `CLAUDE.md` chama de remendo na ALE-252.
 *
 * Aqui o laço é sobre o REGISTRO, e quem força o registro a estar completo é o
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
 * próprio, e a lição está escrita na ALE-272: o caminhar pelas sete abas da
 * ficha media sempre um guerreiro, e metade do painel de Combate só existe para
 * quem conjura. Este arquivo é o PISO, não o teto.
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
// Oito fica abaixo da cena mais magra e oito vezes acima da página de erro. A
// primeira versão cravou 15 por palpite e reprovou `/campanhas/nova`, que estava
// perfeita: um controle mal calibrado vira ruído, e ruído é o que faz um guarda
// ser desligado na segunda semana.
//
// Ele é GLOBAL de propósito. Um número por cena seria uma segunda lista para
// envelhecer, e o que este controle precisa pegar é a página que não carregou —
// não "esta tela tem pouco texto".
const MEASURED_TEXT_FLOOR = 8

for (const [nome, cena] of Object.entries(MEASURED_SCENES)) {
  for (const visita of cena.visits) {
    // O papel entra no NOME porque a Mesa aparece duas vezes: o mestre e o
    // jogador recebem metades diferentes do mesmo endereço, e um relatório com
    // dois casos de nome igual não diz qual reprovou.
    const onde = `${nome} · ${visita.address} · ${visita.session}`

    test.describe(`aparência: ${onde}`, () => {
      test.use({ storageState: SESSION_FILE[visita.session] })

      test(`sem texto abaixo do mínimo de contraste do AA em ${onde}`, async ({ page }) => {
        const resposta = await page.goto(visita.address)
        if (visita.mayBeAbsent && resposta?.status() === 404) {
          test.skip(true, `esta bancada não serve ${visita.address}`)
          return
        }
        expect(resposta?.status(), `${visita.address} não respondeu`).toBeLessThan(400)

        const contraste = await medeOContraste(page)
        expect(
          contraste.medidos,
          `${onde}: o medidor achou ${contraste.medidos} textos — a cena não carregou, e a asserção seguinte não seria evidência de nada`,
        ).toBeGreaterThan(MEASURED_TEXT_FLOOR)
        expect(contraste.falhas, `texto abaixo do AA em ${onde}`).toEqual([])
      })

      test(`a Cinzel não desce abaixo do piso de leitura em ${onde}`, async ({ page }) => {
        const resposta = await page.goto(visita.address)
        if (visita.mayBeAbsent && resposta?.status() === 404) {
          test.skip(true, `esta bancada não serve ${visita.address}`)
          return
        }
        await expectCinzelAcimaDoPiso(page, `em ${onde}`)
      })
    })
  }
}
