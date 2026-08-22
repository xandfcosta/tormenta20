/**
 * Tibares com separador de milhar pt-BR e no máximo duas casas — o dinheiro do
 * livro tem preços fracionários (T$ 0,1 numa vela) e saldos de cinco dígitos no
 * mesmo campo.
 *
 * O "T$" NÃO entra aqui: metade dos lugares já traz a moeda no rótulo ao lado,
 * e devolvê-la junto produziria "T$ T$ 300".
 *
 * @example formatTibar(1250.5) // "1.250,5"
 */
export function formatTibar(value: number): string {
  return value.toLocaleString('pt-BR', { maximumFractionDigits: 2 })
}
