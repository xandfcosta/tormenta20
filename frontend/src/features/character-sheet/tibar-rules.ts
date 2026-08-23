/**
 * As três coisas que se fazem com dinheiro na mesa (ALE-224).
 *
 * Na mesa o que acontece é "achamos 350 no baú" e "paguei 80 pela estalagem";
 * escrever o total é o gesto da FORJA (Tabela 3-1, p140) e o de corrigir um erro
 * de digitação. Os três convivem num diálogo só, e a conta mora aqui — fora do
 * componente — porque ela é regra e tem armadilha: o teto, o piso e o binário.
 */
export type ModoDoTibar = 'receber' | 'gastar' | 'corrigir'

/** O verbo que o botão de confirmar mostra em cada modo. */
export const VERBO_DO_MODO: Record<ModoDoTibar, string> = {
  receber: 'Receber',
  gastar: 'Gastar',
  corrigir: 'Corrigir',
}

/**
 * O saldo em que a operação deixa o personagem.
 *
 * Arredonda em DUAS CASAS porque o dinheiro do livro é fracionário (uma vela
 * custa T$ 0,1) e soma binária de décimos não fecha: 1200,3 − 80,1 dá
 * 1120,1999999999998 em ponto flutuante, e esse número iria para o banco e para
 * a tela. Duas casas é a mesma precisão que o `formatTibar` mostra, então o que
 * se lê e o que se grava passam a ser o mesmo número.
 *
 * @example saldoResultante(1200, 'gastar', 80) // 1120
 */
export function saldoResultante(saldo: number, modo: ModoDoTibar, valor: number): number {
  if (modo === 'corrigir') return arredonda(valor)
  return arredonda(modo === 'receber' ? saldo + valor : saldo - valor)
}

function arredonda(value: number): number {
  return Math.round(value * 100) / 100
}

/**
 * A recusa, ANTES da rede, com o número na mensagem.
 *
 * O servidor já recusa negativo e absurdo, mas deixar a rede responder por isso
 * significa o saldo piscando otimista e voltando — e a barra da mochila junto,
 * porque o tibar é carga (p141). Recusar aqui é o mesmo papel dos validadores
 * de `shared/rules`: a UI ANTECIPA, o Go decide.
 *
 * `null` = pode gravar.
 *
 * @example erroDoTibar(50, 'gastar', 80, 1_000_000) // "Você tem T$ 50."
 */
export function erroDoTibar(
  saldo: number,
  modo: ModoDoTibar,
  valor: number,
  teto: number,
): string | null {
  if (!Number.isFinite(valor) || valor < 0) return 'Informe um valor a partir de 0.'
  const proximo = saldoResultante(saldo, modo, valor)
  // Dívida na ficha viraria carga de moeda NEGATIVA, que COMPRARIA espaço na
  // mochila em vez de ocupar (ALE-215). Por isso o piso é zero e não um aviso.
  if (proximo < 0) return `Você tem T$ ${formata(saldo)}.`
  if (proximo > teto) return `O limite é T$ ${formata(teto)}.`
  return null
}

function formata(value: number): string {
  return value.toLocaleString('pt-BR', { maximumFractionDigits: 2 })
}
