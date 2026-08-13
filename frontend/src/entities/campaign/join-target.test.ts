import { describe, expect, it } from 'vitest'
import { joinTargetId } from './join-target'

describe('joinTargetId', () => {
  it('com convite, a mesa é a que o token resolveu', () => {
    expect(joinTargetId({ token: 'abc', invitedCampaignId: 7, typedId: '' })).toBe(7)
  })

  // Enquanto o token não resolve não há alvo — e o formulário não pode
  // deixar entrar num lugar que ainda não sabe qual é.
  it('com convite ainda resolvendo, não há alvo', () => {
    expect(joinTargetId({ token: 'abc', invitedCampaignId: undefined, typedId: '' })).toBeNull()
  })

  // O espelho que a versão React mantinha podia discordar do convite; aqui o
  // token manda, aconteça o que acontecer no campo manual.
  it('o convite vence o que foi digitado', () => {
    expect(joinTargetId({ token: 'abc', invitedCampaignId: 7, typedId: '3' })).toBe(7)
  })

  it('sem convite, vale o número digitado', () => {
    expect(joinTargetId({ token: undefined, invitedCampaignId: undefined, typedId: '42' })).toBe(42)
  })

  it('recusa o que não é um id de campanha', () => {
    for (const typedId of ['', '0', '-3', '2.5', 'abc', ' ']) {
      expect(joinTargetId({ token: undefined, invitedCampaignId: undefined, typedId })).toBeNull()
    }
  })
})
