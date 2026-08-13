import '@testing-library/jest-dom/vitest'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  ACTIVATION_SPECS,
  allGrantedPowerOptions,
  CATALOG_ITEMS,
  CLASS_POWERS_CATALOG,
  CONDITIONS,
  DEUSES,
  GENERAL_POWERS_CATALOG,
  GRANTED_POWERS,
  ORIGENS,
  ORIGINS_CATALOG,
  RACAS,
  RACES_CATALOG,
  SPELL_CATALOG,
  TORMENTA_POWERS,
} from '@tormenta20/t20-data'
import { primeAbilities } from './shared/lib/abilities-cache'
import { primeActivations } from './shared/lib/activation-cache'
import { primeItemCatalog } from './shared/lib/catalog-cache'
import { primeDivinePowers } from './shared/lib/divine-powers-cache'
import { primeRacas } from './shared/lib/racas-cache'
import { primeRulesCatalogs } from './shared/lib/rules-catalog-cache'
import { primeRulesTables } from './shared/lib/rules-tables-cache'
import { primeSpellCatalog } from './shared/lib/spell-cache'

/**
 * The catalogs ship OUT of the bundle: at runtime the app fetches them from
 * /catalog and primes these caches (project_front_decouple_catalog). Tests have
 * no loader, so they prime once from the real t20-data catalogs — otherwise
 * every `getCatalogItem` / `getRace` lookup silently returns undefined and the
 * domain tests fail for a reason that has nothing to do with the rule under
 * test.
 */
primeItemCatalog(CATALOG_ITEMS)
primeAbilities({
  races: RACES_CATALOG,
  origins: ORIGINS_CATALOG,
  classPowers: CLASS_POWERS_CATALOG,
  generalPowers: GENERAL_POWERS_CATALOG,
  deuses: DEUSES,
  grantedPowers: GRANTED_POWERS,
})
primeSpellCatalog(SPELL_CATALOG)
primeRacas(RACAS, ORIGENS)
primeRulesCatalogs(CONDITIONS, TORMENTA_POWERS)
primeDivinePowers(allGrantedPowerOptions())
primeActivations(ACTIVATION_SPECS)

/**
 * As quatro tabelas que o SERVIDOR autora (ALE-102) não vêm mais do t20-data —
 * elas moram no catálogo servido. Os testes leem os mesmos arquivos que o Go
 * embute, então uma tabela editada vale para os dois lados na mesma hora.
 */
const catalogDir = resolve(
  dirname(fileURLToPath(import.meta.url)),
  '../../engine-go/catalog/data',
)
const servedTable = (name: string) =>
  JSON.parse(readFileSync(resolve(catalogDir, `${name}.json`), 'utf8'))

primeRulesTables({
  classExpertises: servedTable('class-expertises'),
  devotoTerms: servedTable('devoto-terms'),
  gmTables: servedTable('gm-tables'),
  dungeonDesign: servedTable('dungeon-design'),
})
