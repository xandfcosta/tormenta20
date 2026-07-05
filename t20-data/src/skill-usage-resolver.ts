/**
 * Fábrica genérica para o padrão `<skill>UsageByKind(kind)` usado em
 * todos os 26 catálogos de perícias.
 *
 * Cada catálogo define uma união discriminada por `kind` e um resolver
 * que faz throw em kind desconhecido. Antes, o padrão exigia ~9 linhas
 * hand-rolled por perícia (Map + closure + throw). Este utilitário
 * substitui tudo por uma linha:
 *
 *   export const acrobaciaUsageByKind =
 *     makeUsageByKind<AcrobaciaUsageKind, AcrobaciaUsage>(
 *       ACROBACIA_USAGES,
 *       'acrobaciaUsageByKind',
 *     )
 *
 * O contrato preserva a mensagem de erro exata dos resolvers anteriores
 * (`${resolverName}: unknown kind ${kind}`), então os testes existentes
 * continuam válidos sem alteração.
 */

export function makeUsageByKind<
  TKind extends string,
  TUsage extends { kind: TKind },
>(
  usages: readonly TUsage[],
  resolverName: string,
): (kind: TKind) => TUsage {
  const map = new Map<TKind, TUsage>(usages.map((u) => [u.kind, u]))
  return (kind) => {
    const usage = map.get(kind)
    if (!usage) {
      throw new Error(`${resolverName}: unknown kind ${kind}`)
    }
    return usage
  }
}
