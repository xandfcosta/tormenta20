import { type Locator, type Page, expect } from '@playwright/test'

/**
 * Os caminhos da cena do mestre depois da ALE-198.
 *
 * A cena inverteu a hierarquia: o TABULEIRO é a superfície permanente, e a fila
 * do combate, o bestiário, os catálogos e a ficha viraram overlay. Isso mudou
 * como se ALCANÇA cada coisa — não o que ela faz —, e sem um lugar só para
 * esses caminhos a mudança seguinte teria de reescrever vinte specs de novo.
 *
 * Regra de uso: espere por `cenaViva` e não pela iniciativa. Ela agora mora numa
 * gaveta, e "a iniciativa está na tela" deixou de ser sonda de vida da cena.
 */

/**
 * A sonda de vida da cena do mestre: o socket conectado, no cromo que nunca sai
 * da tela. Era o cabeçalho da iniciativa até a ALE-198 o mandar para a faixa do
 * turno — a fila virou gaveta, e uma queda silenciosa no meio do combate não
 * pode depender de alguém abrir uma gaveta para ser vista.
 */
export function cenaViva(page: Page): Locator {
  return page.getByRole('status', { name: 'Conectado' })
}

/** O trilho das consultas: bestiário, encontros, catálogos e notas. */
export function trilhoDeConsultas(page: Page): Locator {
  return page.getByRole('navigation', { name: 'Consultas do mestre' })
}

/** O trilho da fila do combate. Só existe a partir de 1024. */
export function trilhoDaFila(page: Page): Locator {
  return page.getByRole('navigation', { name: 'Fila do combate' })
}

/**
 * Abre uma consulta pelo trilho. Um overlay por vez: abrir uma fecha a gaveta
 * da fila, que é a regra que separa isto dos side sheets empilhados da ALE-122.
 */
export async function abreConsulta(
  page: Page,
  consulta: 'Bestiário' | 'Encontros' | 'Catálogos' | 'Notas',
): Promise<void> {
  await trilhoDeConsultas(page).getByRole('button', { name: consulta, exact: true }).click()
}

/**
 * Abre a gaveta da fila e devolve o diálogo dela.
 *
 * Dois botões, um por faixa de largura, e nunca os dois na tela: a partir de
 * 1024 é o topo do trilho da esquerda; abaixo disso o trilho não existe e quem
 * a alcança é o botão da fileira, que também conta quantos estão na fila.
 */
export async function abreAFila(page: Page): Promise<Locator> {
  const gaveta = page.getByRole('dialog', { name: 'Iniciativa' })
  // IDEMPOTENTE: com a gaveta já aberta abaixo de 1280 ela é MODAL, e o botão
  // que a abre fica atrás dela — clicar de novo esperaria para sempre.
  if (await gaveta.isVisible()) return gaveta
  // UM localizador que casa os dois nomes, e não um `isVisible()` escolhendo
  // entre eles: `isVisible` é um instantâneo SEM espera, e o trilho remonta ao
  // cruzar 1024 — chamado logo depois de um `setViewportSize` ele respondia
  // "não" sobre um trilho que apareceu no quadro seguinte, e o teste ia esperar
  // pelo botão da outra faixa até estourar. Com um localizador só, o `click`
  // espera pelo que existir.
  await page
    .getByRole('button', { name: /^(Abrir a iniciativa|Iniciativa · \d+)$/ })
    .click()
  await expect(gaveta).toBeVisible()
  return gaveta
}

/**
 * Os rótulos DENTRO da gaveta, lidos dos botões de remover.
 *
 * O trilho só existe a partir de 1024, então abaixo disso `labelsNaFila`
 * responde vazio sobre uma fila cheia — e um instantâneo vazio faz a limpeza do
 * teste ir embora sem tirar nada da seed compartilhada.
 */
export async function labelsNaGaveta(gaveta: Locator): Promise<string[]> {
  const botoes = await gaveta.getByRole('button', { name: /^Remover / }).all()
  const labels: string[] = []
  for (const botao of botoes) {
    const nome = (await botao.getAttribute('aria-label')) ?? ''
    labels.push(nome.replace('Remover ', ''))
  }
  return labels
}

/** Fecha a gaveta da fila, se estiver aberta. */
export async function fechaAFila(page: Page): Promise<void> {
  const gaveta = page.getByRole('dialog', { name: 'Iniciativa' })
  if (await gaveta.isVisible()) {
    await page.keyboard.press('Escape')
    await expect(gaveta).toBeHidden()
  }
}

/**
 * Abre a ficha de um combatente pelo TRILHO — o caminho de um clique, e o mesmo
 * que a peça no tabuleiro usa. A ficha é um diálogo desde a ALE-198.
 *
 * Devolve o diálogo. O nome dele é o rótulo do combatente.
 */
export async function abreAFicha(page: Page, label: string): Promise<Locator> {
  await trilhoDaFila(page)
    .getByRole('button', { name: new RegExp(`^Abrir ${escapaRegex(label)}( |$)`) })
    .click()
  const ficha = page.getByRole('dialog', { name: `Ficha de ${label}` })
  await expect(ficha).toBeVisible()
  return ficha
}

/**
 * Abre a ficha pelo caminho que existe em TODA largura: a gaveta da fila.
 *
 * O trilho — e portanto `abreAFicha` — só existe a partir de 1024. Abaixo disso
 * a fila não tem trilho nenhum, e quem quer a ficha passa pela gaveta. Devolve
 * o diálogo da ficha; a gaveta fecha sozinha ao escolher, que é a regra.
 */
export async function abreAFichaPelaGaveta(page: Page): Promise<Locator> {
  const gaveta = await abreAFila(page)
  const rotulo = await gaveta
    .getByRole('button', { name: /^Mudar a iniciativa de / })
    .first()
    .getAttribute('aria-label')
  const nome = (rotulo ?? '').replace('Mudar a iniciativa de ', '')
  if (!nome) throw new Error('a fila está vazia: não há ficha para abrir')
  await gaveta.getByRole('button', { name: nome, exact: true }).click()
  const ficha = page.getByRole('dialog', { name: `Ficha de ${nome}` })
  await expect(ficha).toBeVisible()
  return ficha
}

/** Fecha a ficha do combatente, se estiver aberta. */
export async function fechaAFicha(page: Page): Promise<void> {
  const fechar = page.getByRole('button', { name: 'Fechar o combatente' })
  if (await fechar.isVisible()) await fechar.click()
}

/**
 * Os rótulos na fila, lidos do TRILHO e não da lista: o trilho está sempre na
 * tela a partir de 1024, e ler daqui não exige abrir gaveta nenhuma.
 *
 * Escopado nos ITENS DA LISTA, e não em "todo botão do trilho": o botão de
 * expandir também se chama "Abrir a iniciativa", e ele entraria na conta como
 * um combatente chamado "a iniciativa".
 *
 * O nome acessível de um item é "Abrir <nome> — PV x de y — na vez"; o que
 * interessa é o nome, e ele para no primeiro travessão.
 */
export async function labelsNaFila(page: Page): Promise<string[]> {
  const itens = await trilhoDaFila(page).getByRole('listitem').getByRole('button').all()
  const labels: string[] = []
  for (const item of itens) {
    const nome = (await item.getAttribute('aria-label')) ?? ''
    if (nome.startsWith('Abrir ')) labels.push(nome.slice(6).split(' — ')[0])
  }
  return labels
}

/**
 * O primeiro da fila, ESPERANDO por ele.
 *
 * `labelsNaFila` é um instantâneo sem espera automática, e o trilho remonta ao
 * cruzar 1024 — ler logo depois de um `setViewportSize` pega a árvore antes de
 * o observador da media query ter respondido, e a fila parece vazia com quatro
 * combatentes nela. Custou uma execução da matriz de estados para aparecer.
 */
export async function primeiroDaFila(page: Page): Promise<string> {
  await expect
    .poll(async () => (await labelsNaFila(page)).length, { timeout: 7000 })
    .toBeGreaterThan(0)
  const [primeiro] = await labelsNaFila(page)
  return primeiro ?? ''
}

function escapaRegex(texto: string): string {
  return texto.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * Garante que há CENA em curso, e é IDEMPOTENTE (ALE-210).
 *
 * A cena virou estado que o mestre liga: sem ela não existe avanço de turno, e
 * a fila nem sai do servidor para os jogadores. A seed nasce fora de cena — que
 * é o estado de uma sessão recém-aberta —, então todo teste que fala de turno
 * passa por aqui primeiro.
 *
 * Espera pela VAGA antes de decidir, e não por `isVisible` no "Iniciar cena":
 * `isVisible` é instantâneo sem espera automática, e a faixa remonta ao cruzar
 * 1024 — a armadilha que a `abreAFila` documenta logo acima.
 */
export async function garanteACena(page: Page): Promise<void> {
  const vaga = page
    .getByRole('button', { name: /^(Iniciar cena|Começar|Próximo|Ninguém na fila)/ })
    .first()
  await expect(vaga).toBeVisible()
  const iniciar = page.getByRole('button', { name: 'Iniciar cena' })
  if ((await iniciar.count()) === 0) return
  await iniciar.click()
  await expect(iniciar).toBeHidden()
}

/**
 * Aciona o FIM do ciclo — encerrar ou reiniciar — e confirma.
 *
 * Os dois moram no pé do trilho a partir de 1024 e no menu da sessão abaixo
 * disso: os mesmos nós em dois lugares, porque o trilho não existe no celular.
 * O helper esconde essa escolha, que é de largura e não de comportamento.
 */
export async function acionaOCiclo(
  page: Page,
  botao: 'Encerrar cena' | 'Reiniciar o combate',
  confirmacao: 'Encerrar a cena?' | 'Reiniciar o combate?',
): Promise<void> {
  const alvo = page.getByRole('button', { name: botao })
  if ((await alvo.count()) === 0) {
    await page.getByRole('button', { name: 'Configurações da sessão' }).click()
    await expect(alvo).toBeVisible()
  }
  await alvo.click()
  // Escopado pelo TÍTULO: o gatilho e o botão de confirmar têm nomes vizinhos,
  // e sem o escopo o localizador casa os dois.
  await page
    .getByRole('dialog', { name: confirmacao })
    .getByRole('button', { name: botao.startsWith('Encerrar') ? 'Encerrar' : 'Reiniciar' })
    .click()
  await expect(page.getByRole('dialog', { name: confirmacao })).toBeHidden()
}
