import type { APIRequestContext } from '@playwright/test'

/** O nome fixo do herói de teste — é por ele que o setup o reencontra. */
export const POWERS_FIXTURE_NAME = 'E2E Poderes Gerais'

/**
 * Garante que existe UM herói com vagas de poder em aberto, e devolve o id.
 *
 * Idempotente de propósito: o app NÃO apaga personagem (não há rota nem ação na
 * interface), então um teste que criasse um a cada rodada entulharia o elenco
 * exatamente como as crônicas "E2E Descartável" entulharam as mesas. Ele procura
 * pelo nome, cria só na primeira vez e reusa daí em diante.
 *
 * É SETUP, não o assunto: a criação pela Forja tem teste próprio
 * (`forge-listas.spec.ts`), e andar nove passos para chegar na ficha deixaria o
 * teste lento e frágil por motivos que ele não mede.
 */
export async function ensurePowersFixture(request: APIRequestContext): Promise<number> {
  const list = await request.get('/api/characters')
  if (!list.ok()) {
    throw new Error(`listar personagens falhou (${list.status()})`)
  }
  const existing = (await list.json()).find(
    (character: { id: number; name: string }) => character.name === POWERS_FIXTURE_NAME,
  )
  if (existing) return existing.id as number

  // Guerreiro de 6º nível sem poder escolhido: três vagas em aberto, e é a
  // pendência que faz o card da classe abrir sozinho — com ele fechado, a lista
  // de poderes gerais não pinta e não há o que medir.
  const created = await request.post('/api/characters', {
    data: {
      name: POWERS_FIXTURE_NAME,
      races: ['Humano'],
      origin: 'Soldado',
      classes: [{ className: 'Guerreiro', level: 6 }],
      hpMax: 45,
      hpCurrent: 45,
      mpMax: 18,
      mpCurrent: 18,
      strength: 2,
      dexterity: 1,
      constitution: 2,
      intelligence: 0,
      wisdom: 0,
      charisma: 0,
      size: 'Médio',
      displacement: 9,
      trainedExpertises: [],
      items: [],
    },
  })
  if (!created.ok()) {
    throw new Error(`criar personagem falhou (${created.status()}): ${await created.text()}`)
  }
  return (await created.json()).id as number
}
