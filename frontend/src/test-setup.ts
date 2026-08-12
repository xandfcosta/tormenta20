import '@testing-library/jest-dom/vitest'
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
