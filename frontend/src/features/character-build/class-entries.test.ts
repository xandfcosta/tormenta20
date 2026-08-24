import { describe, expect, it } from 'vitest'
import {
  type ClassEntry,
  addClassEntry,
  attributeSpreadOf,
  classPresetSpread,
  removeClassEntry,
  setClassLevel,
  totalClassLevel,
} from './class-entries'
import { wizardDefaults } from './wizard-steps'

const guerreiro: ClassEntry = { className: 'Guerreiro', level: 3 }
const ladino: ClassEntry = { className: 'Ladino', level: 2 }

describe('addClassEntry', () => {
  it('a primeira classe entra no nível 1', () => {
    expect(addClassEntry([], 'Guerreiro')).toEqual([{ className: 'Guerreiro', level: 1 }])
  })

  it('a segunda entra depois, sem mexer na principal', () => {
    expect(addClassEntry([guerreiro], 'Ladino')).toEqual([
      guerreiro,
      { className: 'Ladino', level: 1 },
    ])
  })

  it('classe já escolhida não duplica (o schema recusaria)', () => {
    expect(addClassEntry([guerreiro], 'Guerreiro')).toEqual([guerreiro])
  })
})

describe('removeClassEntry', () => {
  it('tira a classe pedida', () => {
    expect(removeClassEntry([guerreiro, ladino], 'Ladino')).toEqual([guerreiro])
  })

  it('tirar a principal promove a seguinte', () => {
    // A primeira da lista é a PRINCIPAL — quem manda no preset e na semente de
    // PV —, então remover a primeira não pode deixar a lista sem principal.
    expect(removeClassEntry([guerreiro, ladino], 'Guerreiro')).toEqual([ladino])
  })

  it('remover o que não está lá não faz nada', () => {
    expect(removeClassEntry([guerreiro], 'Bardo')).toEqual([guerreiro])
  })
})

describe('setClassLevel', () => {
  it('grava o nível da classe endereçada', () => {
    expect(setClassLevel([guerreiro, ladino], 'Ladino', 5)).toEqual([
      guerreiro,
      { className: 'Ladino', level: 5 },
    ])
  })

  it('prende ao intervalo 1..20 do livro', () => {
    expect(setClassLevel([guerreiro], 'Guerreiro', 0)[0].level).toBe(1)
    expect(setClassLevel([guerreiro], 'Guerreiro', 99)[0].level).toBe(20)
  })

  it('valor não-numérico cai para 1 em vez de envenenar o cálculo', () => {
    expect(setClassLevel([guerreiro], 'Guerreiro', Number.NaN)[0].level).toBe(1)
  })
})

describe('totalClassLevel', () => {
  it('soma os níveis de todas as classes', () => {
    expect(totalClassLevel([guerreiro, ladino])).toBe(5)
  })

  it('sem classe nenhuma o personagem ainda é de nível 1', () => {
    expect(totalClassLevel([])).toBe(1)
  })
})

describe('classPresetSpread — sugestão de atributos da classe', () => {
  // 'devolve os seis atributos da classe' saiu na ALE-187: contava as chaves
  // de um objeto de forma fixa, que o typechecker garante. Os dois casos
  // abaixo ficam — eles afirmam REGRA (classe desconhecida não tem sugestão,
  // e cada classe puxa para o próprio atributo-chave).

  it('classe desconhecida não tem sugestão', () => {
    expect(classPresetSpread('Caçador de Dragões')).toBeNull()
  })

  it('cada classe puxa para o próprio atributo-chave', () => {
    const guerreiroSpread = classPresetSpread('Guerreiro')
    const arcanistaSpread = classPresetSpread('Arcanista')

    expect(guerreiroSpread?.strength).toBeGreaterThan(arcanistaSpread?.strength ?? 0)
    expect(arcanistaSpread?.intelligence).toBeGreaterThan(
      guerreiroSpread?.intelligence ?? 0,
    )
  })
})

describe('attributeSpreadOf — retrato para desfazer', () => {
  it('copia só os seis atributos, nada mais do rascunho', () => {
    // O rascunho inteiro entra; só os seis atributos saem — o desfazer não pode
    // arrastar de volta o nome nem as raças escolhidas depois.
    const spread = attributeSpreadOf({
      ...wizardDefaults,
      name: 'Aknor',
      races: ['Anão'],
      strength: 2,
      dexterity: 1,
      intelligence: -1,
      charisma: 3,
    })

    expect(spread).toEqual({
      strength: 2,
      dexterity: 1,
      constitution: 0,
      intelligence: -1,
      wisdom: 0,
      charisma: 3,
    })
  })
})
