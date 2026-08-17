import { describe, expect, it } from 'vitest'
import { hpFillVar } from '@/shared/ui/vital-bar'
import { adjustPreview, clampResource } from '@/shared/ui/resource-adjust-dialog'

describe('clampResource', () => {
  it('prende entre 0 e o máximo e avisa que prendeu', () => {
    expect(clampResource(-3, 40)).toEqual({ value: 0, clamped: true })
    expect(clampResource(99, 40)).toEqual({ value: 40, clamped: true })
    expect(clampResource(12, 40)).toEqual({ value: 12, clamped: false })
  })
})

describe('adjustPreview', () => {
  const base = { current: 30, max: 40, tempTotal: 0 }

  it('adicionar sobe até o máximo', () => {
    expect(adjustPreview({ ...base, mode: 'add', amount: 5 }).preview).toBe(35)
    expect(adjustPreview({ ...base, mode: 'add', amount: 50 })).toMatchObject({
      preview: 40,
      clamped: true,
    })
  })

  /**
   * O preview espelha o roteamento do endpoint de dano: PV temporário absorve
   * primeiro. Prometer perda de PV que o servidor vai absorver faria o jogador
   * ver um número e receber outro.
   */
  it('remoção é absorvida pelo PV temporário antes de tocar nos PV', () => {
    const result = adjustPreview({ ...base, mode: 'remove', amount: 8, tempTotal: 5 })
    expect(result).toMatchObject({ soak: 5, delta: -3, preview: 27 })
  })

  it('pool maior que o golpe absorve tudo', () => {
    expect(adjustPreview({ ...base, mode: 'remove', amount: 4, tempTotal: 10 })).toMatchObject({
      soak: 4,
      delta: 0,
      preview: 30,
    })
  })

  // Curar não encosta no pool temporário.
  it('adicionar ignora o pool', () => {
    expect(adjustPreview({ ...base, mode: 'add', amount: 4, tempTotal: 10 }).soak).toBe(0)
  })
})

describe('hpFillVar', () => {
  // A cor da barra é informação: um olhar tem de dizer "estou mal".
  it('vira alerta conforme a vida cai', () => {
    expect(hpFillVar(100)).toBe('--hp-full')
    expect(hpFillVar(51)).toBe('--hp-full')
    expect(hpFillVar(50)).toBe('--hp-hurt')
    expect(hpFillVar(26)).toBe('--hp-hurt')
    expect(hpFillVar(25)).toBe('--hp-critical')
    expect(hpFillVar(0)).toBe('--hp-critical')
  })
})
