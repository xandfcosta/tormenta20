import { classExpertisesFor } from '@/shared/lib/rules-tables-cache'

/**
 * Slots de treinamento de perícia que uma classe expõe à Forja, com a regra do
 * "+Inteligência perícias" dobrada dentro: cada classe treina sua lista fixa,
 * resolve o slot ou/ou e escolhe `chooseCount + max(0, intMod)` do seu pool
 * (Cap 1 — classes treinam perícias extras iguais ao modificador de
 * Inteligência). `null` para classe desconhecida.
 *
 * Morava no `t20-data` junto com a TABELA; a tabela desceu para o catálogo
 * servido e só a regra ficou aqui (ALE-102).
 *
 * @example classExpertiseSlots('Guerreiro', 2)
 */
export function classExpertiseSlots(
  className: string,
  intMod = 0,
): {
  fixed: string[]
  eitherOr?: { options: [string, string] }
  chooseCount: number
  choosePool: string[]
} | null {
  const entry = classExpertisesFor(className)
  if (!entry) return null
  return {
    fixed: entry.fixed,
    ...(entry.eitherOr ? { eitherOr: entry.eitherOr } : {}),
    chooseCount: entry.chooseCount + Math.max(0, intMod),
    choosePool: entry.choosePool,
  }
}
