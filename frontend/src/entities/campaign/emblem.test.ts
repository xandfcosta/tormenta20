import { initials } from '@/shared/lib/initials'
import { describe, expect, it } from 'vitest'
import { roleLabel } from './emblem'

describe('initials', () => {
  it('usa as iniciais das duas primeiras palavras', () => {
    expect(initials('A Queda de Tauron')).toBe('AQ')
  })

  it('nome de uma palavra vira uma letra só', () => {
    expect(initials('Tormenta')).toBe('T')
  })

  it('cai pra ? quando não há nome', () => {
    expect(initials('   ')).toBe('?')
  })

  it('ignora espaços extras entre as palavras', () => {
    expect(initials('  Segredos   de Wynlla ')).toBe('SD')
  })
})

// O bloco `campaignEmblemGradient` saiu na ALE-187, e o pior dos três casos era
// o primeiro: ele comparava `campaignEmblemGradient(x)` com ELE MESMO, ou seja a
// implementação contra si própria — só falharia se a função virasse aleatória.
//
// A determinação de verdade mora um nível abaixo, no `hueFromName`, e desde a
// ALE-238 ela é afirmada com VALORES FIXADOS (186, 278, 22) que valem como
// contrato com a segunda implementação, em Go. Os outros dois casos eram o
// espalhamento — coberto lá também — e um regex de formato CSS.

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

  // A mesa alheia só aparece para quem administra, e o servidor lhe dá o papel
  // `gm` nela — dizer "Mestrando" ali afirmaria que a campanha é dele (ALE-120).
  it('a mesa de outro diz de quem é, não o papel', () => {
    expect(roleLabel('gm', 'Bruna')).toBe('Mesa de Bruna')
  })

  it('sem dono declarado, nada muda', () => {
    expect(roleLabel('gm', null)).toBe('Mestrando')
    expect(roleLabel('player', undefined)).toBe('Jogando')
  })
})
