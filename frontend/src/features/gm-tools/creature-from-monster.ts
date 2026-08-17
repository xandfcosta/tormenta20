import type { Monster } from '@/shared/api/catalog-types'
import type { CreatureBlock, CreatureSkill } from '@/shared/api/creature-types'

/**
 * Um verbete do bestiário virando ponto de partida para o bloco do mestre
 * (ALE-137) — é o "editar este ogro" que a issue pedia.
 *
 * Os campos estruturais passam direto, porque o livro modela criatura e NPC do
 * mesmo jeito. O que NÃO existe no catálogo fica vazio para o mestre preencher:
 * iniciativa, percepção, PM, equipamento e tesouro se perderam na importação e
 * voltam pela ALE-151. Preenchê-los com zero aqui seria inventar número de
 * livro, que é pior que deixar em branco.
 *
 * @example creatureFromMonster(ogro).sourceMonsterId // 'ogro'
 */
export function creatureFromMonster(monster: Monster): CreatureBlock {
  const { skills, rest } = extractSkills(monster.specialAbilities)
  return {
    nd: monster.nd,
    tipo: monster.tipo,
    size: monster.size,
    // Não vêm do catálogo (ALE-151): zero aqui é "o mestre ainda não disse",
    // e a tela do editor mostra o campo vazio esperando.
    iniciativa: 0,
    percepcao: 0,
    defesa: monster.defesa,
    fortitude: monster.fortitude,
    reflexos: monster.reflexos,
    vontade: monster.vontade,
    hp: monster.hp,
    deslocamento: monster.deslocamento,
    forca: monster.forca,
    destreza: monster.destreza,
    constituicao: monster.constituicao,
    inteligencia: monster.inteligencia,
    sabedoria: monster.sabedoria,
    carisma: monster.carisma,
    attacks: monster.attacks.map((attack) => ({
      name: attack.name,
      attackBonus: attack.attackBonus,
      damage: attack.damage,
      ...(attack.special ? { special: attack.special } : {}),
    })),
    skills,
    equipment: '',
    treasure: '',
    specialAbilities: rest,
    sourceMonsterId: monster.id,
  }
}

/**
 * A linha "Perícias: Furtividade +5, Intimidação +6." que a importação jogou
 * dentro de `specialAbilities` como texto (37 dos 80 verbetes). Ela É o dado —
 * estruturá-la aqui evita que o mestre redigite o que o livro já diz.
 *
 * Se a frase não casar com a forma esperada, ela CONTINUA na lista de
 * habilidades: perder a linha seria pior que não estruturá-la.
 */
function extractSkills(abilities: readonly string[]): {
  skills: CreatureSkill[]
  rest: string[]
} {
  const skills: CreatureSkill[] = []
  const rest: string[] = []
  for (const ability of abilities) {
    const match = /^per[íi]cias:\s*(.+?)\.?$/i.exec(ability.trim())
    const parsed = match ? parseSkillList(match[1]) : []
    if (parsed.length === 0) {
      rest.push(ability)
      continue
    }
    skills.push(...parsed)
  }
  return { skills, rest }
}

/** "Furtividade +5, Intimidação +6" → dois pares nome/bônus. */
function parseSkillList(list: string): CreatureSkill[] {
  const out: CreatureSkill[] = []
  for (const piece of list.split(',')) {
    // O nome pode ter espaço e acento ("Adestrar Animais +9"); o bônus é a
    // última coisa da fatia e vem sempre com sinal no livro.
    const match = /^(.+?)\s*([+-]\d+)$/.exec(piece.trim())
    if (!match) return []
    out.push({ name: match[1].trim(), bonus: Number(match[2]) })
  }
  return out
}
