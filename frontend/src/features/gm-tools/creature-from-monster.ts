import type { Monster } from '@/shared/api/catalog-types'
import type { CreatureBlock } from '@/shared/api/creature-types'

/**
 * Um verbete do bestiário virando ponto de partida para o bloco do mestre
 * (ALE-137) — é o "editar este ogro" que a issue pedia.
 *
 * Os campos passam direto, porque o livro modela criatura e NPC do mesmo jeito.
 *
 * Iniciativa, percepção, PM, perícias, equipamento e tesouro chegaram na
 * ALE-151 — antes dela o catálogo não os tinha, e este arquivo os deixava
 * vazios com uma nota dizendo que voltariam. As perícias vinham RASPADAS do
 * texto de `specialAbilities` ("Perícias: Furtividade +5."), porque era ali que
 * a importação as tinha jogado; agora são campo, e a raspagem morreu junto com
 * o motivo dela.
 *
 * @example creatureFromMonster(ogro).sourceMonsterId // 'ogro'
 */
export function creatureFromMonster(monster: Monster): CreatureBlock {
  return {
    nd: monster.nd,
    tipo: monster.tipo,
    size: monster.size,
    iniciativa: monster.iniciativa,
    percepcao: monster.percepcao,
    ...(monster.pm === undefined ? {} : { pm: monster.pm }),
    defesa: monster.defesa,
    fortitude: monster.fortitude,
    reflexos: monster.reflexos,
    vontade: monster.vontade,
    hp: monster.hp,
    deslocamento: monster.deslocamento,
    // Atributo AUSENTE no livro (o `Int —` do Zumbi) vira zero aqui, e isso é
    // uma perda conhecida: o bloco do mestre é numérico e não sabe dizer
    // "não tem" — dizer exigiria atravessar o struct do Go e o formulário, que
    // é outra frente. A partir da cópia o bloco é DELE e ele edita; o catálogo,
    // que é a fonte, guarda a ausência de verdade (ALE-151).
    forca: monster.forca ?? 0,
    destreza: monster.destreza ?? 0,
    constituicao: monster.constituicao ?? 0,
    inteligencia: monster.inteligencia ?? 0,
    sabedoria: monster.sabedoria ?? 0,
    carisma: monster.carisma ?? 0,
    attacks: monster.attacks.map((attack) => ({
      name: attack.name,
      attackBonus: attack.attackBonus,
      damage: attack.damage,
      ...(attack.special ? { special: attack.special } : {}),
    })),
    skills: monster.skills.map((skill) => ({ ...skill })),
    equipment: monster.equipamento,
    treasure: monster.tesouro,
    specialAbilities: [...monster.specialAbilities],
    sourceMonsterId: monster.id,
  }
}
