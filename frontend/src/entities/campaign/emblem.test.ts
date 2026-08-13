import { describe, expect, it } from 'vitest'
import { campaignEmblemGradient, campaignInitials, roleLabel } from './emblem'

describe('campaignInitials', () => {
  it('usa as iniciais das duas primeiras palavras', () => {
    expect(campaignInitials('A Queda de Tauron')).toBe('AQ')
  })

  it('nome de uma palavra vira uma letra só', () => {
    expect(campaignInitials('Tormenta')).toBe('T')
  })

  it('cai pra ? quando não há nome', () => {
    expect(campaignInitials('   ')).toBe('?')
  })

  it('ignora espaços extras entre as palavras', () => {
    expect(campaignInitials('  Segredos   de Wynlla ')).toBe('SD')
  })
})

describe('campaignEmblemGradient', () => {
  // O emblema é derivado do nome justamente pra não precisar de imagem salva:
  // mesma crônica, mesmo sigilo, em qualquer reload.
  it('é estável pro mesmo nome', () => {
    expect(campaignEmblemGradient('A Queda de Tauron')).toBe(
      campaignEmblemGradient('A Queda de Tauron'),
    )
  })

  it('distingue crônicas diferentes', () => {
    expect(campaignEmblemGradient('A Queda de Tauron')).not.toBe(
      campaignEmblemGradient('Segredos de Wynlla'),
    )
  })

  it('devolve um gradiente CSS usável', () => {
    expect(campaignEmblemGradient('Tormenta')).toMatch(/^linear-gradient\(155deg, oklch\(/)
  })
})

describe('roleLabel', () => {
  it('gm mestra', () => {
    expect(roleLabel('gm')).toBe('Mestrando')
  })

  it('player joga', () => {
    expect(roleLabel('player')).toBe('Jogando')
  })

  // Payload antigo em cache pode não trazer o papel — não deve virar "Mestrando".
  it('sem papel, assume jogando', () => {
    expect(roleLabel(undefined)).toBe('Jogando')
  })
})
