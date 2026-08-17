import type { Monster } from '@/shared/api/catalog-types'
import { creatureFromMonster } from './creature-from-monster'
import { CreatureStatBlock } from './creature-stat-block'

/**
 * O verbete do bestiário na tela.
 *
 * Casca fina sobre o `CreatureStatBlock`: desde a ALE-137 há UMA apresentação
 * de bloco de criatura, porque o livro modela verbete e NPC do mestre da mesma
 * forma. Manter duas cópias faria a do mestre e a do livro divergirem no
 * primeiro ajuste — e o bloco do mestre nasce justamente copiando o verbete.
 *
 * De quebra, a linha "Perícias: Furtividade +5." que a importação deixou como
 * texto passa a sair estruturada aqui também.
 *
 * @example <MonsterStatBlock monster={ogro} />
 */
export function MonsterStatBlock(props: { monster: Monster }) {
  return (
    <CreatureStatBlock
      block={creatureFromMonster(props.monster)}
      bookPage={props.monster.bookPage}
    />
  )
}
