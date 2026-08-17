import { render, screen } from '@solidjs/testing-library'
import { afterEach, describe, expect, it } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import type { Character } from '@/shared/api/api'
import { CombatStats } from './combat-stats'

/**
 * A condição do livro tem de mover o NÚMERO, não só acrescentar um crachá
 * (ALE-28) — esta é a asserção que desceu de e2e na ALE-144.
 *
 * O que se prova aqui é a LIGAÇÃO: `character.activeConditions` chega ao motor,
 * e o que o motor devolve chega ao ladrilho. O TAMANHO do modificador é do
 * livro e é provado no Go (`conditions_v2_test.go`: Luta −5 com caído, Pontaria
 * intacta), onde a regra é autorada; repeti-lo aqui seria a terceira cópia da
 * mesma regra. Por isso a asserção é "mudou", e o par CaC/Distância é o que
 * separa "a condição chegou" de "algum número mudou".
 *
 * A suíte roda o motor Go de verdade (WASM), então este é o mesmo cálculo da
 * produção.
 *
 * A ficha PRECISA trazer a perícia: sem a linha de Luta o `requireExpertise`
 * devolve um esboço de zeros, e o teste passaria a medir o esboço em vez do
 * motor — foi assim que a primeira versão desta asserção deu 0 dos dois lados.
 */
const EXPERTISES = [
  { name: 'Luta', attribute: 'strength', trained: true, custom: false },
  { name: 'Pontaria', attribute: 'dexterity', trained: true, custom: false },
] as Character['expertises']

afterEach(() => {
  document.body.innerHTML = ''
})

/** Os dois ladrilhos de ataque, lidos da tela como o jogador os lê. */
function ataques(activeConditions: string): { cac: number; dist: number } {
  render(() => (
    <CombatStats
      character={makeCharacter({ activeConditions, expertises: EXPERTISES })}
      activeConditionals={new Set<string>()}
    />
  ))
  const ler = (rotulo: string) => {
    const tile = screen.getByText(rotulo).closest('button')
    const value = tile?.textContent?.match(/[+-]\d+/)?.[0]
    if (!value) throw new Error(`o ladrilho ${rotulo} não mostrou número: ${tile?.textContent}`)
    return Number.parseInt(value, 10)
  }
  const lidos = { cac: ler('Atq CaC'), dist: ler('Atq Dist') }
  document.body.innerHTML = ''
  return lidos
}

describe('CombatStats — condição do livro', () => {
  it('caído derruba o ataque corpo a corpo do ladrilho e deixa o à distância', () => {
    const limpo = ataques('[]')
    const caido = ataques('["caido"]')

    expect(
      caido.cac,
      `sem condição o CaC deu ${limpo.cac}, com caído deu ${caido.cac} — a condição não chegou ao ladrilho`,
    ).toBeLessThan(limpo.cac)
    expect(
      caido.dist,
      'caído não toca em ataque à distância; se este mudou, o modificador foi parar no lugar errado',
    ).toBe(limpo.dist)
  })
})
