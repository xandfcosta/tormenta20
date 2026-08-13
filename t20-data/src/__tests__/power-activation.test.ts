import { describe, expect, it } from 'vitest'
import {
  ACTIVATION_SPECS,
  findActivationByName,
  getActivation,
  maxStepsForLevel,
} from '../power-activation'

/**
 * Fase 1 do redesign Poderes/Efeitos: registry unificado sobre os 16 módulos
 * *-power-mechanics. As specs alimentam os controles da tab Poderes —
 * taxonomia, custo, limite de usos e postura exigida.
 */
describe('power-activation registry', () => {
  it('cobre as quatro fontes com chaves únicas (~280 poderes)', () => {
    expect(ACTIVATION_SPECS.length).toBeGreaterThan(250)
    const prefixes = new Set(ACTIVATION_SPECS.map((s) => s.id.split('.')[0]))
    expect(prefixes).toEqual(new Set(['class', 'race', 'origin', 'deus']))
  })

  it('instantânea simples: Golpe Poderoso — livre, 1 PM, sem limite', () => {
    const spec = getActivation('class.barbaro.golpe-poderoso')!
    expect(spec.kind).toBe('instant')
    expect(spec.action).toBe('livre')
    expect(spec.pmCost).toBe(1)
    expect(spec.uses).toBeNull()
  })

  it('instantânea com limite: Brado Assustador — 1/cena', () => {
    const spec = getActivation('class.barbaro.brado-assustador')!
    expect(spec.kind).toBe('instant')
    expect(spec.uses).toBe('cena')
  })

  it('passiva gatilhada: Alma de Bronze exige a flag furia', () => {
    const spec = getActivation('class.barbaro.alma-de-bronze')!
    expect(spec.kind).toBe('triggered-passive')
    expect(spec.requiresFlag).toBe('furia')
  })

  it('requer postura sem ser passiva: Frenesi é instant + requiresFlag', () => {
    const spec = getActivation('class.barbaro.frenesi')!
    expect(spec.kind).toBe('instant')
    expect(spec.requiresFlag).toBe('furia')
    expect(spec.uses).toBe('rodada')
  })

  it('Bardo mapeia requiresInspiracao para a flag inspiracao', () => {
    const spec = getActivation('class.bardo.golpe-elemental')!
    expect(spec.requiresFlag).toBe('inspiracao')
  })

  it('postura: Fúria com escalada por nível de BÁRBARO (p40)', () => {
    const spec = getActivation('class.barbaro.furia')!
    expect(spec.kind).toBe('stance')
    expect(spec.pmCost).toBe(2)
    const steps = (level: number) => maxStepsForLevel(spec.scaling!, level)
    expect(steps(4)).toBe(0)
    expect(steps(5)).toBe(1)
    expect(steps(6)).toBe(1)
    expect(steps(10)).toBe(2)
    expect(steps(20)).toBe(4)
  })

  /**
   * O catálogo é SERVIDO por HTTP (o front não empacota dado de catálogo), e
   * uma função não sobrevive ao JSON: `maxStepsForLevel` era um método do
   * spec, chegava `undefined` no cliente e derrubava a cena inteira ao abrir
   * qualquer postura que escala — nos dois fronts.
   */
  it('a escalada sobrevive à ida e volta por JSON', () => {
    const spec = getActivation('class.barbaro.furia')!

    const overTheWire = JSON.parse(JSON.stringify(spec)) as typeof spec

    expect(overTheWire.scaling).toEqual(spec.scaling)
    expect(maxStepsForLevel(overTheWire.scaling!, 10)).toBe(2)
  })

  it('fontes não-classe: racial, origem e deus resolvem por slug do nome', () => {
    expect(getActivation('race.dahllan.armadura-de-allihanna')?.uses).toBe('cena')
    expect(getActivation('origin.guarda.detetive')?.pmCost).toBe(1)
    expect(getActivation('deus.arsenal.conjurar-arma')?.action).toBe('padrao')
  })

  it('findActivationByName resolve por nome e prefere classe em ambiguidade', () => {
    expect(findActivationByName('Golpe Poderoso')?.id).toBe(
      'class.barbaro.golpe-poderoso',
    )
    expect(findActivationByName('golpe poderoso')?.id).toBe(
      'class.barbaro.golpe-poderoso',
    )
    expect(findActivationByName('não existe')).toBeUndefined()
  })
})

/**
 * Fase 4: grants hand-authored — efeitos duradouros que o backend materializa
 * como ActiveEffect quando o poder é usado/disparado.
 */
describe('power-activation grants', () => {
  it('Alma de Bronze concede pool de PV temporários por Força (p41)', () => {
    expect(getActivation('class.barbaro.alma-de-bronze')?.grant).toEqual({
      kind: 'temp-hp',
      scope: 'scene',
      attribute: 'strength',
    })
  })

  it('Armadura da Honra e Coração Heroico usam Carisma (p53 / origem)', () => {
    expect(getActivation('class.cavaleiro.armadura-da-honra')?.grant).toEqual({
      kind: 'temp-hp',
      scope: 'scene',
      attribute: 'charisma',
    })
    expect(getActivation('origin.heroi-camponês.coracao-heroico')?.grant).toEqual({
      kind: 'temp-hp',
      scope: 'scene',
      attribute: 'charisma',
    })
  })

  it('En Garde: +2 Defesa flat + margem de ameaça condicional (p47)', () => {
    const grant = getActivation('class.bucaneiro.en-garde')?.grant
    expect(grant?.kind).toBe('active-effect')
    if (grant?.kind !== 'active-effect') return
    expect(grant.scope).toBe('scene')
    expect(grant.modifiers).toContainEqual(
      expect.objectContaining({ target: { k: 'defense' }, amount: 2 }),
    )
    expect(grant.modifiers).toContainEqual(
      expect.objectContaining({
        target: { k: 'critRange' },
        condition: expect.objectContaining({ c: 'context' }),
      }),
    )
  })

  it('Armadura de Allihanna (+2 Defesa) e Dom da Verdade (+5 perícias)', () => {
    const dahllan = getActivation('race.dahllan.armadura-de-allihanna')?.grant
    expect(dahllan?.kind).toBe('active-effect')
    if (dahllan?.kind === 'active-effect') {
      expect(dahllan.modifiers).toEqual([
        expect.objectContaining({ target: { k: 'defense' }, amount: 2 }),
      ])
    }
    const khalmyr = getActivation('deus.khalmyr.dom-da-verdade')?.grant
    expect(khalmyr?.kind).toBe('active-effect')
    if (khalmyr?.kind === 'active-effect') {
      expect(khalmyr.modifiers).toHaveLength(2)
      expect(khalmyr.modifiers[0]?.target).toEqual({
        k: 'expertise',
        name: 'Intuição',
      })
    }
  })

  it('grants só existem em ids registrados e specs sem grant ficam intactas', () => {
    const withGrant = ACTIVATION_SPECS.filter((s) => s.grant)
    expect(withGrant).toHaveLength(6)
    expect(getActivation('class.barbaro.golpe-poderoso')?.grant).toBeUndefined()
  })
})
