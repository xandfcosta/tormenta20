/// <reference types="node" />
// Dev harness (vitest, Node) — NOT shipped in the app bundle. Exports the static
// catalog resources the Go API serves (GET /catalog/:resource + /characters/options)
// to engine-go/catalog-data/*.json, so the Go server serves byte-identical JSON to
// the Nest CatalogService without importing t20-data. Regenerate after a t20-data
// change:
//   GEN_CATALOGS=1 pnpm --filter frontend test catalog-export
import { mkdirSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  ACTIVATION_SPECS,
  allGrantedPowerOptions,
  BESTIARY,
  CATALOG_ITEMS,
  CLASS_POWERS_CATALOG,
  CONDITIONS,
  DEUSES,
  EXPERTISES,
  GENERAL_POWERS_CATALOG,
  GRANTED_POWERS,
  ORIGENS,
  ORIGINS_CATALOG,
  RACAS,
  RACES_CATALOG,
  SIZES,
  SPELL_CATALOG,
  TORMENTA_POWERS,
} from '@tormenta20/t20-data'
import { describe, expect, it } from 'vitest'

// Creation option lists that live in the BACKEND's t20-constants.ts (not
// t20-data), copied verbatim so options.json matches the Nest options() payload.
const RACES = [
  'Humano', 'Anão', 'Dahllan', 'Elfo', 'Goblin', 'Lefou', 'Minotauro', 'Qareen',
  'Suraggel', 'Sílfide', 'Sereia/Tritão', 'Osteon', 'Medusa', 'Kliren', 'Hynne',
  'Golem', 'Trog',
]
const CLASSES = [
  'Arcanista', 'Bárbaro', 'Bardo', 'Bucaneiro', 'Caçador', 'Cavaleiro', 'Clérigo',
  'Druida', 'Guerreiro', 'Inventor', 'Ladino', 'Lutador', 'Nobre', 'Paladino',
]
const ORIGINS = [
  'Acólito', 'Amigo dos Animais', 'Amnésico', 'Aristocrata', 'Artesão', 'Artista',
  'Assistente de Laboratório', 'Batedor', 'Capanga', 'Charlatão', 'Circense',
  'Criminoso', 'Curandeiro', 'Eremita', 'Escravo', 'Estudioso', 'Fazendeiro',
  'Forasteiro', 'Gladiador', 'Guarda', 'Herdeiro', 'Herói Camponês', 'Marujo',
  'Mateiro', 'Membro de Guilda', 'Mercador', 'Minerador', 'Nômade', 'Pivete',
  'Refugiado', 'Seguidor', 'Selvagem', 'Soldado', 'Taverneiro', 'Trabalhador',
]
const GODS = [
  'Tauron', 'Aharadak', 'Allihanna', 'Arsenal', 'Azgher', 'Hyninn', 'Kallyadranoch',
  'Khalmyr', 'Lena', 'Lin-Wu', 'Marah', 'Megalokk', 'Nimb', 'Oceano', 'Sszzaas',
  'Tanna-Toh', 'Tenebra', 'Thwor', 'Thyatis', 'Valkaria', 'Wynna',
]

// resource name → payload, mirroring CatalogService.registry.
const RESOURCES: Record<string, unknown> = {
  spells: SPELL_CATALOG,
  bestiary: BESTIARY,
  items: CATALOG_ITEMS,
  conditions: CONDITIONS,
  deuses: DEUSES,
  races: RACAS,
  origins: ORIGINS_CATALOG,
  'race-defs': RACES_CATALOG,
  'class-powers': CLASS_POWERS_CATALOG,
  'general-powers': GENERAL_POWERS_CATALOG,
  'granted-powers': GRANTED_POWERS,
  origens: ORIGENS,
  'tormenta-powers': TORMENTA_POWERS,
  'divine-powers': allGrantedPowerOptions(),
  activations: ACTIVATION_SPECS,
}

// The character-creation option lists (charactersController.options()).
const OPTIONS = {
  races: RACES,
  classes: CLASSES,
  origins: ORIGINS,
  gods: GODS,
  sizes: SIZES,
  expertises: EXPERTISES,
}

const outDir = resolve(
  dirname(fileURLToPath(import.meta.url)),
  '../../../../engine-go/catalog/data',
)

describe('catalog export — static resources the Go API serves', () => {
  it('dumps every catalog resource + the options payload', () => {
    // The resource set must match the Nest CatalogService registry.
    expect(Object.keys(RESOURCES)).toHaveLength(15)
    if (!process.env.GEN_CATALOGS) return
    mkdirSync(outDir, { recursive: true })
    for (const [name, payload] of Object.entries(RESOURCES)) {
      writeFileSync(resolve(outDir, `${name}.json`), JSON.stringify(payload))
    }
    writeFileSync(resolve(outDir, 'options.json'), JSON.stringify(OPTIONS))
  })
})
