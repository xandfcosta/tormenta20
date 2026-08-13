/**
 * As uniões de MAGIA — círculo, escola, execução, alcance, duração, componente,
 * área e resistência. Movidas do `t20-data` (ALE-109) pelo mesmo motivo dos
 * outros vocabulários: são uniões de literais e o Go guarda cada uma como
 * `string`, então gerar daqui perderia a checagem.
 */

/** Círculos de magia. Truque é o círculo 0 — custa 0 PM e não recebe
 *  aprimoramento (p170). */
export type SpellCircle = 0 | 1 | 2 | 3 | 4 | 5

/** As 8 escolas (p170). */
export const SPELL_SCHOOLS = [
  'abjuracao',
  'adivinhacao',
  'convocacao',
  'encantamento',
  'evocacao',
  'ilusao',
  'necromancia',
  'transmutacao',
] as const
export type SpellSchool = (typeof SPELL_SCHOOLS)[number]

/** Execução — como a magia entra na economia de ações (p172). */
export type SpellExecution = 'padrao' | 'movimento' | 'livre' | 'reacao' | 'completa'

/** Alcance (p172). */
export type SpellRange = 'pessoal' | 'toque' | 'curto' | 'medio' | 'longo' | 'ilimitado'

/** Duração (p172-173). */
export type SpellDuration =
  | 'instantanea'
  | 'cena'
  | 'sustentada'
  | 'definida'
  | 'dia'
  | 'permanente'

/** Componentes (p173). */
export type SpellComponent = 'verbal' | 'gestual' | 'material' | 'foco'

/** Formas de área (p173). */
export type AreaShape = 'cilindro' | 'cone' | 'esfera' | 'linha' | 'quadrado' | 'cubo'

/** Como o teste de resistência altera o efeito (p173). */
export type SpellResistance = 'anula' | 'parcial' | 'metade' | 'desacredita'

/** Qual teste a magia pede. */
export type SpellSaveType = 'fortitude' | 'reflexos' | 'vontade' | 'none'

/** "aumenta" acumula; "muda" nunca acumula consigo mesmo (p171). */
export type AugmentKind = 'aumenta' | 'muda'
