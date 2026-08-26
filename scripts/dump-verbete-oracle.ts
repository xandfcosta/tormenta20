/**
 * Gera `engine-go/api/testdata/verbete-para-bloco-do-js.json` a partir do
 * `creature-from-monster.ts` da SPA (ALE-269, superfície 6b).
 *
 * MESMO ARGUMENTO DO ORÁCULO DO MARKDOWN: enquanto as duas telas existirem, o
 * mestre pode copiar o Ogro pela SPA numa noite e pelo piloto na outra, e as
 * duas cópias têm de sair IGUAIS — o bloco vai para a mesma coluna do mesmo
 * banco. Digitar os esperados em Go seria uma segunda transcrição da regra, e é
 * aí que os dois lados passam a discordar em silêncio.
 *
 * As criaturas escolhidas NÃO são as bonitas: são as que carregam as armadilhas
 * que a ALE-151 documentou, e cada uma está aqui por um motivo nomeado.
 *
 * Uso: node scripts/dump-verbete-oracle.ts
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { creatureFromMonster } from '../frontend/src/features/gm-tools/creature-from-monster.ts'

const AQUI = dirname(fileURLToPath(import.meta.url))
const BESTIARIO = resolve(AQUI, '../engine-go/catalog/data/bestiary.json')
const SAIDA = resolve(AQUI, '../engine-go/api/testdata/verbete-para-bloco-do-js.json')

/** Cada id vem com a razão de estar aqui — um oráculo de casos fáceis mede o fácil. */
const ESCOLHIDOS: { id: string; nota: string }[] = [
  { id: 'ogro', nota: 'o caso comum: ataque, perícias, habilidade especial' },
  { id: 'zumbi', nota: 'ATRIBUTO AUSENTE — o livro escreve travessão em Int (p297), e o bloco vira 0' },
]

const bestiario = JSON.parse(readFileSync(BESTIARIO, 'utf8'))
const lista: Record<string, unknown>[] = Array.isArray(bestiario) ? bestiario : bestiario.monsters

/** Um conjurador qualquer, para o PM ponteiro atravessar; e um sem, para a AUSÊNCIA dele. */
const comPM = lista.find((m) => m.pm !== undefined && m.pm !== null)
const semPM = lista.find((m) => m.pm === undefined || m.pm === null)
if (comPM) ESCOLHIDOS.push({ id: comPM.id as string, nota: 'conjurador: o PM existe e atravessa' })
if (semPM) ESCOLHIDOS.push({ id: semPM.id as string, nota: 'sem PM: a AUSÊNCIA atravessa, e não vira zero' })

const casos = ESCOLHIDOS.map(({ id, nota }) => {
  const verbete = lista.find((m) => m.id === id)
  if (!verbete) throw new Error(`o bestiário não tem ${id} — o oráculo mediria nada`)
  return { id, nota, bloco: creatureFromMonster(verbete as never) }
})

writeFileSync(
  SAIDA,
  `${JSON.stringify(
    {
      _: 'GERADO por scripts/dump-verbete-oracle.ts. Não edite à mão.',
      casos,
    },
    null,
    2,
  )}\n`,
)
console.log(`escrito: ${SAIDA} (${casos.length} casos: ${casos.map((c) => c.id).join(', ')})`)
