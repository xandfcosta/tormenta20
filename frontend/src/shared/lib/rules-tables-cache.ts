import type {
  ClassTrainedExpertises,
  DevotoTerms,
  DungeonDesign,
  GmTables,
} from '@/shared/api/api'

/**
 * Cache das quatro tabelas de livro que o servidor autora — perícias por classe,
 * termos de devoto, tabelas de rolagem do Improviso e desenho de masmorra.
 *
 * Elas eram `import`s de tempo de build do `@tormenta20/t20-data` e por isso os
 * últimos bytes de catálogo no bundle (ALE-102). O ganho não é tamanho — são 8
 * KB — e sim cortar quatro dependências que prendiam o pacote vivo.
 *
 * Mesmo contrato dos outros `*-cache`: primado pelo loader raiz ANTES de
 * qualquer consumidor renderizar, com acessores SÍNCRONOS.
 */
let classExpertises: Readonly<Record<string, ClassTrainedExpertises>> = {}
let devotoTerms: DevotoTerms = { openTerms: [], termToNames: {} }
let gmTables: GmTables | null = null
let dungeonDesign: DungeonDesign | null = null

export function primeRulesTables(tables: {
  classExpertises: Readonly<Record<string, ClassTrainedExpertises>>
  devotoTerms: DevotoTerms
  gmTables: GmTables
  dungeonDesign: DungeonDesign
}): void {
  classExpertises = tables.classExpertises
  devotoTerms = tables.devotoTerms
  gmTables = tables.gmTables
  dungeonDesign = tables.dungeonDesign
}

/** Perícias treinadas pela classe, ou `undefined` para classe desconhecida. */
export function classExpertisesFor(className: string): ClassTrainedExpertises | undefined {
  return classExpertises[className]
}

export function devotoTermsTable(): DevotoTerms {
  return devotoTerms
}

/**
 * As tabelas do Improviso. Lança se lidas antes do prime — a ferramenta do
 * mestre não tem o que mostrar sem elas, e um `null` silencioso viraria uma tela
 * vazia sem explicação.
 */
export function gmTablesData(): GmTables {
  if (!gmTables) {
    throw new Error('rules-tables: gm-tables lido antes do prime (ensureCatalogs)')
  }
  return gmTables
}

export function dungeonDesignData(): DungeonDesign {
  if (!dungeonDesign) {
    throw new Error('rules-tables: dungeon-design lido antes do prime (ensureCatalogs)')
  }
  return dungeonDesign
}
