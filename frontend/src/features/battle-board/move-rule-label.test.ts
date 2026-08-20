import { describe, expect, it } from 'vitest'
import { regraQueDobrou } from './board-bars'

/**
 * A frase que nomeia a regra do losango (ALE-190).
 *
 * O que se prova aqui é a ESCOLHA DE PALAVRAS, e só ela: as contagens vêm do
 * servidor, do mesmo laço que cobrou o caminho. A aritmética do livro é do
 * motor Go e tem teste lá (`TestPathCostCountsWhichRuleDoubledEachStep`);
 * repeti-la aqui seria a segunda implementação que a ALE-104 apagou.
 */
describe('regraQueDobrou', () => {
  it('caminho reto e limpo não anuncia regra nenhuma', () => {
    // Anunciar "nenhuma dobra" seria ruído na faixa onde a mesa lê o custo.
    expect(regraQueDobrou(0, 0)).toBeNull()
  })

  it('uma diagonal nomeia a regra e cita a página', () => {
    expect(regraQueDobrou(1, 0)).toBe('diagonal custa o dobro (p238)')
  })

  it('terreno difícil sozinho também é nomeado', () => {
    // As duas dobras são regras SEPARADAS no livro, e a tela não pode chamar
    // uma pelo nome da outra: quem lê aprende a contar errado.
    expect(regraQueDobrou(0, 1)).toBe('terreno difícil custa o dobro (p238)')
  })

  it('as duas causas entram na mesma frase', () => {
    expect(regraQueDobrou(1, 1)).toBe('diagonal e terreno difícil custam o dobro (p238)')
  })

  it('conta quantos passos dobraram quando é mais de um', () => {
    expect(regraQueDobrou(3, 0)).toBe('3 diagonais custam o dobro (p238)')
    expect(regraQueDobrou(0, 2)).toBe('2 de terreno difícil custam o dobro (p238)')
  })
})
