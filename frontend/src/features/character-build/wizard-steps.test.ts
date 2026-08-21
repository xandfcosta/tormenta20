import { describe, expect, it } from 'vitest'
import {
  allStepsReady,
  type CharacterFormValues,
  furthestReachableIndex,
  isStepSlug,
  stepAt,
  stepReady,
  WIZARD_STEPS,
  wizardDefaults,
  wizardSteps,
} from './wizard-steps'

const complete: CharacterFormValues = {
  ...wizardDefaults,
  name: 'Thorin',
  races: ['Anão'],
  origin: 'Batedor',
  classes: [{ className: 'Guerreiro', level: 1 }],
  size: 'Médio',
}

describe('stepReady', () => {
  it('raca needs at least one race', () => {
    expect(stepReady('raca', wizardDefaults, {})).toBe(false)
    expect(stepReady('raca', { ...wizardDefaults, races: ['Elfo'] }, {})).toBe(
      true,
    )
  })

  it('classe needs a primary class', () => {
    expect(stepReady('classe', wizardDefaults, {})).toBe(false)
    expect(stepReady('classe', complete, {})).toBe(true)
  })

  it('origem needs an origin', () => {
    expect(stepReady('origem', wizardDefaults, {})).toBe(false)
    expect(stepReady('origem', complete, {})).toBe(true)
  })

  it('atributos is always ready (preset-seeded)', () => {
    expect(stepReady('atributos', wizardDefaults, {})).toBe(true)
  })

  it('identidade needs a name and size', () => {
    expect(stepReady('identidade', { ...complete, name: '  ' }, {})).toBe(false)
    expect(stepReady('identidade', complete, {})).toBe(true)
  })

  // Vitalidade was folded into Identidade (ALE-94): four numbers, two of them
  // derived, floated on a full stage. Its gate had to come along, or a draft
  // with PV atual above PV máx would walk to the Resumo unchallenged.
  it('identidade still enforces current ≤ max', () => {
    expect(
      stepReady('identidade', { ...complete, hpCurrent: 99, hpMax: 10 }, {}),
    ).toBe(false)
    expect(
      stepReady('identidade', { ...complete, mpCurrent: 5, mpMax: 0 }, {}),
    ).toBe(false)
  })
})

describe('WIZARD_STEPS', () => {
  it('has no standalone vitalidade step', () => {
    // A bookmark to the old URL is caught by the same guard as any typo.
    expect(isStepSlug('vitalidade')).toBe(false)
  })
})

describe('furthestReachableIndex', () => {
  it('is the first (raça) step when nothing is chosen', () => {
    expect(furthestReachableIndex(wizardDefaults, {})).toBe(0)
  })

  it('advances one step once its predecessor is ready', () => {
    expect(
      furthestReachableIndex({ ...wizardDefaults, races: ['Anão'] }, {}),
    ).toBe(1)
  })

  it('reaches the last step when every step is ready', () => {
    // Da CAMINHADA deste personagem, não do catálogo: um Guerreiro de nível 1
    // não tem passo de Poderes (ALE-169).
    expect(furthestReachableIndex(complete, {})).toBe(wizardSteps(complete).length - 1)
  })
})

describe('allStepsReady', () => {
  it('false until every required choice is made', () => {
    expect(allStepsReady(wizardDefaults, {})).toBe(false)
  })
  it('true for a complete build', () => {
    expect(allStepsReady(complete, {})).toBe(true)
  })
})

describe('stepAt — andar um passo', () => {
  it('avança e recua na ordem declarada', () => {
    expect(stepAt('raca', 1, WIZARD_STEPS)).toBe('classe')
    expect(stepAt('classe', -1, WIZARD_STEPS)).toBe('raca')
  })

  it('devolve null nas pontas (não circula)', () => {
    expect(stepAt('raca', -1, WIZARD_STEPS)).toBeNull()
    expect(stepAt('resumo', 1, WIZARD_STEPS)).toBeNull()
  })
})

describe('isStepSlug — slug vindo da URL', () => {
  it('aceita um passo real', () => {
    expect(isStepSlug('pericias')).toBe(true)
  })

  it('recusa qualquer outra coisa', () => {
    expect(isStepSlug('inventario')).toBe(false)
    expect(isStepSlug('')).toBe(false)
  })
})

/**
 * A caminhada é derivada da regra, e não uma lista escrita à mão (ALE-169).
 *
 * O passo de Poderes era atravessado por TODO personagem novo como uma tela
 * preta com uma frase, porque a forja cria nível 1 e a primeira vaga de poder
 * é do SEGUNDO nível de uma classe.
 */
describe('wizardSteps — os passos que este personagem atravessa', () => {
  it('tira Poderes quando a classe não rende vaga nenhuma', () => {
    const slugs = wizardSteps(complete).map((s) => s.slug)

    expect(slugs).not.toContain('poderes')
    expect(slugs).toHaveLength(WIZARD_STEPS.length - 1)
  })

  it('devolve Poderes assim que existe uma vaga', () => {
    const nivel2 = { ...complete, classes: [{ className: 'Guerreiro', level: 2 }] }

    expect(wizardSteps(nivel2).map((s) => s.slug)).toContain('poderes')
  })

  it('nunca reordena nem inventa passo', () => {
    const catalogo = WIZARD_STEPS.map((s) => s.slug)
    const caminhada = wizardSteps(complete).map((s) => s.slug)

    expect(caminhada).toEqual(catalogo.filter((slug) => caminhada.includes(slug)))
  })

  /** O endereço guardado de um passo que não se aplica não pode virar 404. */
  it('o catálogo continua reconhecendo o slug fora da caminhada', () => {
    expect(wizardSteps(complete).map((s) => s.slug)).not.toContain('poderes')
    expect(isStepSlug('poderes')).toBe(true)
  })
})

describe('stepAt — pula o passo que não se aplica', () => {
  it('anda de Classe direto para Origem num nível 1', () => {
    expect(stepAt('classe', 1, wizardSteps(complete))).toBe('origem')
  })

  it('e volta de Origem para Classe', () => {
    expect(stepAt('origem', -1, wizardSteps(complete))).toBe('classe')
  })
})

/**
 * Um endereço guardado de um passo que saiu não pode virar beco sem saída, e a
 * saída NÃO é redirecionar: desviar dentro de um efeito que observa a URL é um
 * laço. A caminhada simplesmente inclui onde o jogador está.
 */
describe('wizardSteps — quem chega por URL no passo que saiu', () => {
  it('inclui o passo em que o jogador está, mesmo sem vaga', () => {
    expect(wizardSteps(complete, 'poderes').map((s) => s.slug)).toContain('poderes')
  })

  it('e ele some assim que o jogador sai', () => {
    expect(wizardSteps(complete, 'origem').map((s) => s.slug)).not.toContain('poderes')
  })

  it('estando nele, o Próximo e o Voltar andam a partir dele', () => {
    const caminhada = wizardSteps(complete, 'poderes')

    expect(stepAt('poderes', 1, caminhada)).toBe('origem')
    expect(stepAt('poderes', -1, caminhada)).toBe('classe')
  })
})
