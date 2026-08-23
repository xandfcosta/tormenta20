import type { Monster } from '@/shared/api/catalog-types'

/**
 * Cache próprio do BESTIÁRIO, com acessor SÍNCRONO — o mesmo contrato dos
 * outros catálogos (`rules-catalog-cache`, `abilities-cache`, `spell-cache`):
 * o `ensureCatalogs` prepara antes de qualquer tela renderizar, e quem lê não
 * toca em query nenhuma.
 *
 * ELE ERA A EXCEÇÃO, E A EXCEÇÃO CUSTOU (ALE-199). O comentário que ficava no
 * `bestiaryCatalogQueryOptions` dizia, com todas as letras: "só a Mesa do
 * Mestre e o adicionar da sessão o leem, e eles podem esperar uma query como
 * qualquer outra tela". A premissa é falsa DENTRO da sessão — ali esperar uma
 * query desanexa a cena inteira, porque o `Suspense` mais próximo é o que o
 * roteador põe em volta do route match, e reinserir o nó reinicia toda animação
 * abaixo dele (a família ALE-95/96/121/122).
 *
 * O que a bissecção mostrou, e que muda o remédio: **não é a BUSCA, é a criação
 * do recurso**. Medido no e2e, com o guarda `percorrer as consultas não
 * desanexa a cena`:
 *
 *   componente vazio, sem query ........... não desanexa
 *   só o `useQuery`, sem filtro nem lista .. DESANEXA
 *   só o `useQuery`, com o cache PREPARADO . DESANEXA
 *
 * Ou seja: `settledQuery` não salva aqui. Ele impede a leitura pendente de
 * `.data`, mas quem suspende é o recurso nascer dentro de um dono reativo novo
 * — e o portal do Kobalte cria um a cada abertura. A única saída é não haver
 * recurso: por isso acessor síncrono, e não mais um `queryOptions`.
 *
 * O preço são 13 KB comprimidos no prime do `__root`, contra os 40 KB que as
 * magias já custam ali. Barato pelo padrão que os outros dezoito seguem.
 */
let monstros: readonly Monster[] = []
let porId: ReadonlyMap<string, Monster> = new Map()
let primed = false

/** Chamado pelo `ensureCatalogs`, antes de qualquer tela renderizar. */
export function primeBestiary(bestiary: readonly Monster[]): void {
  monstros = bestiary
  porId = new Map(bestiary.map((monster) => [monster.id, monster]))
  primed = true
}

/**
 * O bestiário inteiro, sem suspender.
 *
 * @example allMonsters().filter((m) => m.nd === '1')
 */
export function allMonsters(): readonly Monster[] {
  return monstros
}

/** Um verbete por id, ou `undefined` se ele não existe no catálogo. */
export function getMonster(id: string): Monster | undefined {
  return porId.get(id)
}

/** True depois do prime — para um portão em tempo de render. */
export function isBestiaryPrimed(): boolean {
  return primed
}
