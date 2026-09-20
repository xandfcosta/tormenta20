import { test, type Page } from '@playwright/test'
import { expectEveryToggleDrawsItsState } from './support/geometry'

/**
 * OS CONTROLES DE ESTADO DESENHAM O ESTADO QUE DECLARAM.
 *
 * O `expectEveryToggleDrawsItsState` explica o defeito e por que ele só se mede
 * no navegador. Aqui está a VARREDURA: quais telas são visitadas.
 *
 * ## Por que estas, e por que uma lista
 *
 * Porque controle de estado não tem um lugar só — ele aparece em filtro de
 * catálogo, em crachá de ficha, em ferramenta de tabuleiro. Enumerar é remendo,
 * e a seção "Como uma convenção passa a valer" diz isso; o que restaura a
 * amostragem seria um guarda sobre a FORMA, e a ALE-341 mediu que a forma não
 * responde: três varreduras de fonte deram três números, e o navegador derrubou
 * os primeiros que foi conferir.
 *
 * Então a lista é assumida, e o que ela paga é ser a única instância que decide.
 * Cena nova com controle de estado entra aqui à mão — e é a mesma dívida do
 * `appearance_scenes.json`, com a mesma razão.
 */
type Screen = { nome: string; rota: string; raiz?: string; abre?: (page: Page) => Promise<void> }

const THE_SCREENS: Screen[] = [
  { nome: 'o bestiário do mestre', rota: '/mestre/bestiario' },
  { nome: 'o acervo de condições', rota: '/mestre/condicoes' },
  // O NECROMANTE tem oito magias no grimório, e o "preparada" de cada uma é um
  // switch. Uma ficha SEM magia desenharia zero controles, e o caso passaria
  // sobre o vazio — é a razão de o personagem estar no endereço.
  { nome: 'as magias da ficha', rota: '/personagens/3?tab=spells' },
  // A LENDA DE NÍVEL 20 tem DEZOITO situacionais; a maioria das fichas não tem
  // nenhum. Quem escolhe a tela escolhe o DADO junto.
  { nome: 'os situacionais da ficha', rota: '/personagens/7?tab=conditionals' },
  { nome: 'a mochila da ficha', rota: '/personagens/1?tab=bag' },
  // O DIÁLOGO DE CONJURAR é o caso que abriu a ALE-341, e ele só existe depois
  // de um clique: os "Trocar" dos aprimoramentos nascem escondidos. Uma
  // varredura que só carregasse endereços não o alcançaria — é a primeira das
  // quatro formas de "não visitar" que o CLAUDE.md lista.
  {
    nome: 'os aprimoramentos ao conjurar',
    rota: '/personagens/3?tab=spells',
    // A RAIZ É O DIÁLOGO, e isso não é detalhe: varrendo o `body`, o primeiro
    // clique cai num "Preparada" da lista atrás, o servidor redesenha a ficha e
    // o diálogo FECHA — os "Trocar" somem antes de serem medidos, e o caso passa
    // sobre o que não viu. Foi assim que ele passou com o defeito recriado.
    raiz: '[role="dialog"][aria-label="Bola de Fogo"]',
    abre: async (page) => {
      await page.getByRole('button', { name: 'Conjurar Bola de Fogo' }).click()
      await page.waitForTimeout(700)
    },
  },
]

// A ABA DE PODERES ficou de fora, e não por esquecimento: nenhum personagem da
// semente tem poder ESCOLHIDO, então ela desenha zero controles e o caso
// dispararia o controle do medidor em vez de medir. Ela entra no dia em que a
// semente tiver um — e a issue registra isso (ALE-341).

for (const screen of THE_SCREENS) {
  test(`todo controle de ${screen.nome} desenha o estado que declara`, async ({ page }) => {
    await page.goto(screen.rota)
    await page.waitForTimeout(900)
    if (screen.abre) await screen.abre(page)
    await expectEveryToggleDrawsItsState(page, screen.raiz ?? 'body')
  })
}
