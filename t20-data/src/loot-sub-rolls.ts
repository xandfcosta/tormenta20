/**
 * Loot sub-roll tables — PDF Cap 8 Recompensas, p331-332.
 *
 * Tabela 8-3 (Itens Diversos), Tabela 8-4 (Equipamento — sub-tables
 * arma/armadura/esotérico), Tabela 8-5 (Itens Superiores — sub-tables
 * armas/armaduras-escudos/esotéricos).
 *
 * Kind routing (p330 "Resultados da Tabela"): equipamento e superior
 * usam 1d6 pra escolher sub-tabela — 1-3 arma, 4-5 armadura/escudo,
 * 6 esotérico. Deterministic com Rng seedável (`loot-rng.ts`).
 */
import { randInt, rollPercentile, type Rng } from './loot-rng'

// ─── Types ──────────────────────────────────────────────────────────
export type EquipKind = 'arma' | 'armadura' | 'esoterico'

type NameRow = readonly [lo: number, hi: number, name: string]

/**
 * Superior property row. `countsAsTwo` reflects footnote 1 (Atroz,
 * Pungente, Sob medida): conta como duas melhorias; se o item só
 * possuir uma, role novamente.
 */
type SuperiorRow = readonly [
  lo: number,
  hi: number,
  name: string,
  countsAsTwo?: true,
]

export type EquipmentRoll = { kind: EquipKind; name: string }

export type SuperiorProperty = {
  name: string
  countsAsTwo: boolean
  /** Populated when property is "Material especial" (footnote 2). */
  material?: SpecialMaterial
}

export type SuperiorRoll = {
  kind: EquipKind
  properties: readonly SuperiorProperty[]
}

export type SpecialMaterial =
  | 'aço-rubi'
  | 'adamante'
  | 'gelo eterno'
  | 'madeira Tollon'
  | 'matéria vermelha'
  | 'mitral'

/** Footnote 2 material roll (1d6). */
export const MATERIAL_ROLL_TABLE: readonly SpecialMaterial[] = Object.freeze([
  'aço-rubi',
  'adamante',
  'gelo eterno',
  'madeira Tollon',
  'matéria vermelha',
  'mitral',
])

// ─── Tabela 8-3: Itens Diversos (p331) ──────────────────────────────
export const DIVERSOS_ROWS: readonly NameRow[] = Object.freeze([
  [1, 2, 'Ácido'],
  [3, 4, 'Água benta'],
  [5, 5, 'Alaúde élfico'],
  [6, 6, 'Algemas'],
  [7, 8, 'Baga-de-fogo'],
  [9, 23, 'Bálsamo restaurador'],
  [24, 24, 'Bandana'],
  [25, 25, 'Bandoleira de poções'],
  [26, 30, 'Bomba'],
  [31, 31, 'Botas reforçadas'],
  [32, 32, 'Camisa bufante'],
  [33, 33, 'Capa esvoaçante'],
  [34, 34, 'Capa pesada'],
  [35, 35, 'Casaco longo'],
  [36, 36, 'Chapéu arcano'],
  [37, 38, 'Coleção de livros'],
  [39, 40, 'Cosméticos'],
  [41, 42, 'Dente-de-dragão'],
  [43, 43, 'Enfeite de elmo'],
  [44, 44, 'Elixir do amor'],
  [45, 46, 'Equipamento de viagem'],
  [47, 56, 'Essência de mana'],
  [57, 57, 'Estojo de disfarces'],
  [58, 58, 'Farrapos de ermitão'],
  [59, 59, 'Flauta mística'],
  [60, 66, 'Fogo alquímico'],
  [67, 67, 'Gorro de ervas'],
  [68, 69, 'Líquen lilás'],
  [70, 70, 'Luneta'],
  [71, 71, 'Luva de pelica'],
  [72, 73, 'Maleta de medicamentos'],
  [74, 74, 'Manopla'],
  [75, 75, 'Manto eclesiástico'],
  [76, 78, 'Mochila de aventureiro'],
  [79, 80, 'Musgo púrpura'],
  [81, 81, 'Organizador de pergaminhos'],
  [82, 83, 'Ossos de monstro'],
  [84, 85, 'Pó de cristal'],
  [86, 87, 'Pó de giz'],
  [88, 88, 'Pó do desaparecimento'],
  [89, 89, 'Robe místico'],
  [90, 91, 'Saco de sal'],
  [92, 92, 'Sapatos de camurça'],
  [93, 94, 'Seixo de âmbar'],
  [95, 95, 'Sela'],
  [96, 96, 'Tabardo'],
  [97, 97, 'Traje da corte'],
  [98, 99, 'Terra de cemitério'],
  [100, 100, 'Veste de seda'],
])

// ─── Tabela 8-4: Equipamento (p331) ─────────────────────────────────
export const ARMA_ROWS: readonly NameRow[] = Object.freeze([
  [1, 3, 'Adaga'],
  [4, 5, 'Alabarda'],
  [6, 7, 'Alfange'],
  [8, 10, 'Arco curto'],
  [11, 13, 'Arco longo'],
  [14, 15, 'Azagaia'],
  [16, 16, 'Balas (20)'],
  [17, 18, 'Besta leve'],
  [19, 20, 'Besta pesada'],
  [21, 23, 'Bordão'],
  [24, 24, 'Chicote'],
  [25, 27, 'Cimitarra'],
  [28, 30, 'Clava'],
  [31, 31, 'Corrente de espinhos'],
  [32, 33, 'Espada bastarda'],
  [34, 38, 'Espada curta'],
  [39, 43, 'Espada longa'],
  [44, 46, 'Flechas (20)'],
  [47, 49, 'Florete'],
  [50, 51, 'Foice'],
  [52, 53, 'Funda'],
  [54, 55, 'Gadanho'],
  [56, 56, 'Katana'],
  [57, 59, 'Lança'],
  [60, 60, 'Lança montada'],
  [61, 63, 'Maça'],
  [64, 66, 'Machadinha'],
  [67, 67, 'Machado anão'],
  [68, 70, 'Machado de batalha'],
  [71, 73, 'Machado de guerra'],
  [74, 74, 'Machado táurico'],
  [75, 76, 'Mangual'],
  [77, 77, 'Marreta'],
  [78, 80, 'Martelo de guerra'],
  [81, 83, 'Montante'],
  [84, 84, 'Mosquete'],
  [85, 85, 'Pedras (20)'],
  [86, 88, 'Picareta'],
  [89, 90, 'Pique'],
  [91, 92, 'Pistola'],
  [93, 93, 'Rede'],
  [94, 96, 'Tacape'],
  [97, 98, 'Tridente'],
  [99, 100, 'Virotes (20)'],
])

export const ARMADURA_ROWS: readonly NameRow[] = Object.freeze([
  [1, 5, 'Couro'],
  [6, 10, 'Brunea'],
  [11, 25, 'Completa'],
  [26, 30, 'Cota de malha'],
  [31, 45, 'Couraça'],
  [46, 55, 'Couro batido'],
  [56, 65, 'Escudo leve'],
  [66, 80, 'Escudo pesado'],
  [81, 85, 'Gibão de peles'],
  [86, 90, 'Loriga segmentada'],
  [91, 100, 'Meia armadura'],
])

export const ESOTERICO_ROWS: readonly NameRow[] = Object.freeze([
  [1, 10, 'Bolsa de pó'],
  [11, 25, 'Cajado arcano'],
  [26, 35, 'Cetro elemental'],
  [36, 42, 'Costela de lich'],
  [43, 50, 'Dedo de ente'],
  [51, 55, 'Luva de ferro'],
  [56, 65, 'Medalhão de prata'],
  [66, 75, 'Orbe cristalino'],
  [76, 85, 'Tomo hermético'],
  [86, 100, 'Varinha arcana'],
])

// ─── Tabela 8-5: Itens Superiores (p332) ────────────────────────────
export const SUPERIOR_ARMA_ROWS: readonly SuperiorRow[] = Object.freeze([
  [1, 10, 'Atroz', true],
  [11, 13, 'Banhada a ouro'],
  [14, 23, 'Certeira'],
  [24, 26, 'Cravejada de gemas'],
  [27, 36, 'Cruel'],
  [37, 39, 'Discreta'],
  [40, 44, 'Equilibrada'],
  [45, 48, 'Harmonizada'],
  [49, 53, 'Injeção alquímica'],
  [54, 55, 'Macabra'],
  [56, 65, 'Maciça'],
  [66, 75, 'Material especial'],
  [76, 80, 'Mira telescópica'],
  [81, 90, 'Precisa'],
  [91, 100, 'Pungente', true],
])

export const SUPERIOR_ARMADURA_ROWS: readonly SuperiorRow[] = Object.freeze([
  [1, 15, 'Ajustada'],
  [16, 19, 'Banhada a ouro'],
  [20, 23, 'Cravejada de gemas'],
  [24, 28, 'Delicada'],
  [29, 32, 'Discreta'],
  [33, 37, 'Espinhos'],
  [38, 40, 'Equilibrada'],
  [41, 50, 'Material especial'],
  [51, 55, 'Polida'],
  [56, 80, 'Reforçada'],
  [81, 90, 'Selada'],
  [91, 100, 'Sob medida', true],
])

export const SUPERIOR_ESOTERICO_ROWS: readonly SuperiorRow[] = Object.freeze([
  [1, 4, 'Banhado a ouro'],
  [5, 19, 'Canalizador'],
  [20, 23, 'Cravejado de gemas'],
  [24, 27, 'Discreto'],
  [28, 42, 'Energético'],
  [43, 57, 'Harmonizado'],
  [58, 60, 'Macabro'],
  [61, 69, 'Material especial'],
  [70, 85, 'Poderoso'],
  [86, 100, 'Vigilante'],
])

// ─── Row lookup helpers ─────────────────────────────────────────────
function findRow<R extends readonly [number, number, ...unknown[]]>(
  rows: readonly R[],
  roll: number,
): R {
  const hit = rows.find((r) => roll >= r[0] && roll <= r[1])
  if (!hit) {
    throw new Error(
      `loot-sub-rolls: no row matches d% ${roll} — expected 1-100 with full coverage`,
    )
  }
  return hit
}

// ─── Kind routing (1d6, p330) ───────────────────────────────────────
/**
 * PDF p330: equipamento e superior usam 1d6 para escolher sub-tabela.
 *  1-3 → arma, 4-5 → armadura/escudo, 6 → esotérico.
 */
export function rollEquipKind(rng: Rng): EquipKind {
  const d6 = randInt(rng, 1, 6)
  if (d6 <= 3) return 'arma'
  if (d6 <= 5) return 'armadura'
  return 'esoterico'
}

// ─── Resolvers ──────────────────────────────────────────────────────
/** Tabela 8-3 — pick one Item Diverso by d%. */
export function rollDiverso(rng: Rng): string {
  const roll = rollPercentile(rng)
  return findRow(DIVERSOS_ROWS, roll)[2]
}

function equipRowsFor(kind: EquipKind): readonly NameRow[] {
  if (kind === 'arma') return ARMA_ROWS
  if (kind === 'armadura') return ARMADURA_ROWS
  return ESOTERICO_ROWS
}

function superiorRowsFor(kind: EquipKind): readonly SuperiorRow[] {
  if (kind === 'arma') return SUPERIOR_ARMA_ROWS
  if (kind === 'armadura') return SUPERIOR_ARMADURA_ROWS
  return SUPERIOR_ESOTERICO_ROWS
}

/**
 * Tabela 8-4 — 1d6 escolhe kind (arma/armadura/esotérico) então d%
 * escolhe item específico. Passe `kind` explícito para bypass o 1d6.
 */
export function rollEquipamento(rng: Rng, kind?: EquipKind): EquipmentRoll {
  const chosen = kind ?? rollEquipKind(rng)
  const rows = equipRowsFor(chosen)
  const roll = rollPercentile(rng)
  const name = findRow(rows, roll)[2]
  return { kind: chosen, name }
}

/**
 * Roll 1 melhoria on Tabela 8-5. Retorna nome + flag countsAsTwo +
 * material (se rolar "Material especial"). Não trata footnote 1
 * reroll aqui — orchestrator `rollSuperior` faz isso.
 */
function rollSuperiorProperty(
  rng: Rng,
  rows: readonly SuperiorRow[],
): SuperiorProperty {
  const roll = rollPercentile(rng)
  const row = findRow(rows, roll)
  const property: SuperiorProperty = {
    name: row[2],
    countsAsTwo: row[3] === true,
  }
  if (row[2] === 'Material especial') {
    const idx = randInt(rng, 0, MATERIAL_ROLL_TABLE.length - 1)
    return { ...property, material: MATERIAL_ROLL_TABLE[idx]! }
  }
  return property
}

/**
 * Tabela 8-5 — role uma vez por melhoria. Footnote 1: se property
 * conta como duas mas item só tem 1 melhoria total, role novamente
 * (até no máx 8 tentativas pra evitar loop teórico infinito).
 *
 * @param improvements — número de melhorias do item (1-4).
 * @param kind — omitido roll 1d6 pra escolher sub-tabela.
 */
export function rollSuperior(
  rng: Rng,
  improvements: 1 | 2 | 3 | 4,
  kind?: EquipKind,
): SuperiorRoll {
  const chosen = kind ?? rollEquipKind(rng)
  const rows = superiorRowsFor(chosen)
  const properties: SuperiorProperty[] = []
  let remaining = improvements
  let safety = 0

  while (remaining > 0 && safety < 16) {
    safety++
    const prop = rollSuperiorProperty(rng, rows)
    // Footnote 1: countsAsTwo mas resta só 1 slot → reroll
    if (prop.countsAsTwo && remaining < 2) continue
    properties.push(prop)
    remaining -= prop.countsAsTwo ? 2 : 1
  }

  return { kind: chosen, properties }
}
