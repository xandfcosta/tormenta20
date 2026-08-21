import type { AttributeKey } from '@/shared/api/attribute-keys'
import { racasList } from '@/shared/lib/racas-cache'

/**
 * O que uma raça PEDE em matéria de escolha de atributo, achatado para quem vai
 * desenhar o controle.
 *
 * Vive em `shared/lib` porque duas features irmãs precisam da mesma resposta e
 * a FSD não deixa uma importar da outra: a forja, onde a escolha nasce, e a
 * ficha, onde ela pode ser terminada (ALE-169). Ler o `atributoMod` cru dos
 * dois lados seria a mesma regra escrita duas vezes, com garantia de divergir
 * quando uma raça nova aparecer.
 *
 * Lê o cache de raças, então só depois do portão de catálogos.
 */
export type RaceAttributeMeta =
  | { kind: 'none' }
  | {
      kind: 'floating'
      count: number
      value: number
      exclude?: AttributeKey
      penalty?: { attribute: AttributeKey; value: number }
    }
  | { kind: 'subrace'; options: string[] }

/**
 * @example raceAttributeMeta('Humano') // { kind: 'floating', count: 3, value: 1 }
 * @example raceAttributeMeta('Anão') // { kind: 'none' }
 */
export function raceAttributeMeta(name: string): RaceAttributeMeta {
  const mod = racasList().find((r) => r.name === name)?.atributoMod
  if (!mod) return { kind: 'none' }
  if (mod.kind === 'floating') {
    return {
      kind: 'floating',
      count: mod.count,
      value: mod.value,
      exclude: mod.exclude,
      penalty: mod.penalty,
    }
  }
  if (mod.kind === 'subraca-gated') {
    return { kind: 'subrace', options: Object.keys(mod.variants) }
  }
  return { kind: 'none' }
}
