import { describe, expect, it } from 'vitest'
import { deriveDraftVitals } from './draft-vitals'

type V = Parameters<typeof deriveDraftVitals>[0]

const base = (o: Partial<V> = {}): V => ({
  classes: [{ className: 'Bárbaro', level: 6 }],
  races: [],
  origin: '',
  strength: 3,
  dexterity: 2,
  constitution: 3,
  intelligence: 0,
  wisdom: 1,
  charisma: 0,
  classPowers: [],
  originChoices: [],
  ...o,
})

describe('deriveDraftVitals', () => {
  it('Bárbaro L6: PV = 24 + 5*6 + Con 3*6, PM = 3*6', () => {
    const { pvMax, pmMax } = deriveDraftVitals(base(), {})
    expect(pvMax).toBe(72)
    expect(pmMax).toBe(18)
  })

  it('folds Vontade de Ferro origin benefit into PM (+floor(level/2))', () => {
    const v = base({ origin: 'Refugiado', originChoices: ['poder-vontade-de-ferro'] })
    expect(deriveDraftVitals(v, {}).pmMax).toBe(21) // 18 + floor(6/2)=3
  })

  it('Clérigo Magia Divina (auto) adds +Sabedoria to PM', () => {
    const v = base({ classes: [{ className: 'Clérigo', level: 3 }], wisdom: 4 })
    expect(deriveDraftVitals(v, {}).pmMax).toBe(19) // 5*3 + Sab 4
  })

  it('applies race attribute delta to Con and folds Duro como Pedra (Anão)', () => {
    const v = base({
      classes: [{ className: 'Guerreiro', level: 1 }],
      races: ['Anão'],
      constitution: 2,
    })
    // Con 2 + raça 2 = 4. PV = 20 + 0 + 4*1 = 24; Duro como Pedra = nível+2 = 3 → 27.
    expect(deriveDraftVitals(v, {}).pvMax).toBe(27)
  })

  it('returns zero when no class is chosen', () => {
    expect(deriveDraftVitals(base({ classes: [] }), {})).toEqual({ pvMax: 0, pmMax: 0 })
  })
})
