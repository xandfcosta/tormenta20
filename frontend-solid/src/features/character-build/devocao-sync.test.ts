import { describe, expect, it } from 'vitest'
import { devotoSyncPatch } from './devocao-sync'

describe('devotoSyncPatch — deus da Identidade dirige o devoto por classe', () => {
  it('Clérigo + Khalmyr → devoto khalmyr', () => {
    expect(
      devotoSyncPatch({
        god: 'Khalmyr',
        classes: [{ className: 'Clérigo' }],
        classChoices: {},
      }),
    ).toEqual({ 'Clérigo': { devoto: 'khalmyr' } })
  })

  it('já sincronizado → null (sem writes em loop)', () => {
    expect(
      devotoSyncPatch({
        god: 'Khalmyr',
        classes: [{ className: 'Clérigo' }],
        classChoices: { 'Clérigo': { devoto: 'khalmyr' } },
      }),
    ).toBeNull()
  })

  it('sem deus → null (sentinelas Panteão/Paladino do Bem ficam livres)', () => {
    expect(
      devotoSyncPatch({
        god: '',
        classes: [{ className: 'Clérigo' }],
        classChoices: { 'Clérigo': { devoto: 'panteao' } },
      }),
    ).toBeNull()
  })

  it('deus fora da lista da classe → não força (Paladino + Arsenal)', () => {
    expect(
      devotoSyncPatch({
        god: 'Arsenal',
        classes: [{ className: 'Paladino' }],
        classChoices: {},
      }),
    ).toBeNull()
  })

  it('classe sem slot de devoto → null (Bardo)', () => {
    expect(
      devotoSyncPatch({
        god: 'Khalmyr',
        classes: [{ className: 'Bardo' }],
        classChoices: {},
      }),
    ).toBeNull()
  })

  it('sobrescreve sentinela quando o deus é válido (deus é a fonte única)', () => {
    expect(
      devotoSyncPatch({
        god: 'Khalmyr',
        classes: [{ className: 'Clérigo' }],
        classChoices: { 'Clérigo': { devoto: 'panteao' } },
      }),
    ).toEqual({ 'Clérigo': { devoto: 'khalmyr' } })
  })
})
