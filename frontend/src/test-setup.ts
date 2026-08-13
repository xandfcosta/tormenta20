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
import { primeEngineCatalogs } from './shared/lib/engine-wasm'
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

/**
 * O MOTOR DE VERDADE nos testes.
 *
 * Os cinco choke points do derive tinham um ramo `import.meta.env.MODE === 'test'`
 * que rodava a cópia TS das regras, para o vitest não precisar de WASM. O efeito
 * colateral é que 108 testes exercitavam uma implementação que a produção NÃO
 * roda — e a fatia 5 vai apagar essa cópia (ALE-109). Carregar o motor Go aqui
 * inverte isso: o que o teste mede passa a ser o que o jogador usa.
 *
 * O carregador de produção é só-navegador (`<script>` + `instantiateStreaming`);
 * em Node o caminho é ler os dois arquivos do disco.
 */
const engineDir = resolve(dirname(fileURLToPath(import.meta.url)), '../public/engine')

const glue = readFileSync(resolve(engineDir, 'wasm_exec.js'), 'utf8')
// eslint-disable-next-line @typescript-eslint/no-implied-eval
new Function(glue).call(globalThis)

type GoRuntime = { importObject: WebAssembly.Imports; run(i: WebAssembly.Instance): void }
const go = new (globalThis as typeof globalThis & { Go: new () => GoRuntime }).Go()
const { instance } = await WebAssembly.instantiate(
  readFileSync(resolve(engineDir, 't20.wasm')),
  go.importObject,
)
// main() trava num select{} e registra os globais do motor — não se espera por ela.
void go.run(instance)

// A MESMA carga que o loader raiz manda em produção, lida dos arquivos que o Go
// embute — então o setup já não depende do t20-data para isto.
primeEngineCatalogs(
  JSON.stringify({
    items: servedTable('items'),
    races: servedTable('race-defs'),
    origins: servedTable('origins'),
    classPowers: servedTable('class-powers'),
    generalPowers: servedTable('general-powers'),
    grantedPowers: servedTable('granted-powers'),
    racas: servedTable('races'),
    tormentaPowerIds: Object.keys(servedTable('tormenta-powers')),
  }),
)
