import { expect, type Page, test } from '@playwright/test'
import { expectOneFocusRing } from './support/focus'

/**
 * AS SUPERFÍCIES DA CASA — onde ela ESCREVE com tinta semântica —, e as duas
 * medições que percorrem todas elas.
 *
 * **Cena nova onde a casa escreva tinta semântica entra nesta lista, no MESMO
 * commit que a cria**: o regime aqui é ENUMERAÇÃO, e a que alguém esquecer nasce
 * sem medição, em silêncio. Um guarda só mede o que ele VISITA.
 */
const HOUSE_SURFACES = [
  {
    where: 'no menu do jogador',
    visit: async (page: Page) => {
      // O PAINEL ELEVADO, que é o `--popover`. Ele só existe depois do clique,
      // e é por isso que nenhum guarda o tinha medido.
      await page.goto('/')
      await page.getByRole('button', { name: 'Menu de Mestre' }).click()
      await expect(page.getByRole('button', { name: 'Sair' })).toBeVisible()
    },
  },
  {
    where: 'no veredito do encontro',
    visit: async (page: Page) => {
      // Com o rascunho vazio não há selo: os tons dele são medidos pelas visitas do
      // `appearance_scenes.json`.
      await page.goto('/mestre/encontros')
      await expect(page.getByRole('heading', { name: 'Encontros' })).toBeVisible()
    },
  },
  {
    where: 'no livro de campanhas',
    visit: async (page: Page) => {
      // O PERGAMINHO, que é creme e inverte a conta inteira. Ancorado no
      // cabeçalho e NÃO em "Sessão ao vivo": exigir uma sessão viva seria uma
      // asserção que mede o banco.
      await page.goto('/campanhas')
      await expect(page.getByRole('heading', { name: 'Campanhas' })).toBeVisible()
    },
  },
  {
    where: 'na aba de lugares da crônica',
    visit: async (page: Page) => {
      // O ACERVO DA CAMPANHA, sobre o PERGAMINHO da crônica — creme, que
      // inverte a conta inteira como a lista de campanhas acima.
      await page.goto('/campanhas/4?tab=lugares')
      // Ancorado no GESTO e não num título: as seções da crônica são abertas por
      // sobrancelha (`SectionLabel`, um `<p>`) e não por `<h2>`, então exigir um
      // heading aqui seria exigir uma forma que a cena não tem.
      await expect(page.getByRole('button', { name: 'Novo lugar' })).toBeVisible()
      await expect(page.getByText('Cripta de Thwor')).toBeVisible()
    },
  },
  {
    where: 'na janela das notas',
    visit: async (page: Page) => {
      // A CENA DAS NOTAS FORA DA MESA: mesmo painel, sem mapa em volta.
      //
      // Endereço direto e não navegado: o gesto que a abre é um `window.open`,
      // e uma janela nova não é a `page` deste guarda.
      await page.goto('/campanhas/1/sessoes/4/notas')
      await expect(page.getByRole('heading', { name: /Notas/ })).toBeVisible()
    },
  },
  {
    where: 'no rascunho de lugar',
    visit: async (page: Page) => {
      // O RASCUNHO é a superfície do TABULEIRO fora da sessão, e escreve tinta
      // que nenhuma outra entrada desta lista alcança: a tarja dourada do modo,
      // o nome da peça sobre o chão de cripta, e o terreno difícil por baixo.
      //
      // A cena vem da SEMENTE e não de um clique de criação: criar o lugar aqui
      // deixaria uma linha para trás no banco compartilhado da corrida.
      //
      // MAS O CAMINHO É NAVEGADO, e não um id no endereço: id de AUTOINCREMENT
      // numa fixture compartilhada quebra por uma mudança sem relação nenhuma, e
      // o erro ("heading não encontrado") não aponta para a causa.
      await page.goto('/campanhas/4?tab=lugares')
      await page.getByRole('link', { name: 'Montar' }).first().click()
      // O NÍVEL importa: o nome do lugar aparece DUAS vezes na tela — no `<h1>`
      // da moldura e no `<h2>` da cena, que é o mesmo desenho da Mesa. Sem o
      // nível o seletor casa com os dois e estoura em `strict mode`.
      await expect(page.getByRole('heading', { level: 1, name: 'Cripta de Thwor' })).toBeVisible()
      // E o CONTROLE do que esta entrada existe para medir: a tarja do modo, que
      // é a tinta que nenhuma outra cena desta lista alcança.
      await expect(page.getByText('a mesa não vê')).toBeVisible()
    },
  },
] as const

test.describe('As superfícies da casa', () => {
  /**
   * Todo foco da casa tem a MESMA cara, em TODA superfície.
   *
   * O medidor mora em `support/focus.ts` e não aqui: instrumento dentro de um
   * chamador tem exatamente um chamador. O que fica aqui é a lista de
   * superfícies — cena nova entra numa linha e ganha a medição.
   *
   * O caminhar pelas SETE ABAS da ficha NÃO mora aqui de propósito: ele vive no
   * `sheet.spec.ts`, junto do contraste e da tipografia, porque à parte seria
   * enumeração e a aba que nascer amanhã nasceria sem medição.
   */
  for (const surface of HOUSE_SURFACES) {
    test(`o realce de foco é o mesmo ${surface.where}`, async ({ page }) => {
      await surface.visit(page)
      await expectOneFocusRing(page, surface.where)
    })
  }
})
