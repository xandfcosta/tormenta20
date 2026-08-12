import { describe, expect, it } from 'vitest'
import { wizardDefaults } from '@/features/character-build/wizard-steps'
import { FakeStorage } from '@/shared/test/fake-storage'
import {
  CHARACTER_DRAFT_STORAGE_KEY,
  createCharacterDraftStore,
  readStoredDraft,
} from './character-draft-store'

const stored = (storage: FakeStorage) =>
  JSON.parse(storage.getItem(CHARACTER_DRAFT_STORAGE_KEY) ?? '{}')

describe('createCharacterDraftStore', () => {
  it('começa nos defaults do wizard', () => {
    const draft = createCharacterDraftStore(new FakeStorage())

    expect(draft.values.races).toEqual([])
    expect(draft.values.size).toBe('Médio')
    expect(draft.raceChoices).toEqual({})
    expect(draft.attributeMode()).toBe('free')
  })

  it('grava um campo e persiste', () => {
    const storage = new FakeStorage()
    const draft = createCharacterDraftStore(storage)

    draft.setValue('name', 'Aknor')
    draft.setValue('races', ['Anão'])

    expect(draft.values.name).toBe('Aknor')
    expect(stored(storage).state.values.name).toBe('Aknor')
    expect(stored(storage).state.values.races).toEqual(['Anão'])
  })

  it('patchValues aplica vários campos de uma vez', () => {
    const draft = createCharacterDraftStore(new FakeStorage())

    draft.patchValues({ strength: 2, dexterity: 1 })

    expect(draft.values.strength).toBe(2)
    expect(draft.values.dexterity).toBe(1)
  })

  it('setRaceChoice mexe só na raça endereçada', () => {
    const draft = createCharacterDraftStore(new FakeStorage())

    draft.setRaceChoice('Humano', { floatingPicks: ['strength'] })
    draft.setRaceChoice('Lefou', { applied: true })

    expect(draft.raceChoices.Humano).toEqual({ floatingPicks: ['strength'] })
    expect(draft.raceChoices.Lefou).toEqual({ applied: true })
  })

  it('reset volta aos defaults e limpa o armazenamento', () => {
    const storage = new FakeStorage()
    const draft = createCharacterDraftStore(storage)
    draft.setValue('races', ['Elfo'])
    draft.setAttributeMode('point-buy')

    draft.reset()

    expect(draft.values.races).toEqual([])
    expect(draft.attributeMode()).toBe('free')
    expect(storage.getItem(CHARACTER_DRAFT_STORAGE_KEY)).toBeNull()
  })

  it('hasResumable só conta escolha de identidade (raça ou classe)', () => {
    const draft = createCharacterDraftStore(new FakeStorage())
    expect(draft.hasResumable()).toBe(false)

    // Um nome digitado sozinho não é rascunho: o prompt de "continuar" só
    // aparece quando há uma escolha estrutural que valha a pena retomar.
    draft.setValue('name', 'Aknor')
    expect(draft.hasResumable()).toBe(false)

    draft.setValue('races', ['Anão'])
    expect(draft.hasResumable()).toBe(true)
  })

  it('rehidrata do armazenamento numa sessão nova', () => {
    const storage = new FakeStorage()
    createCharacterDraftStore(storage).setValue('name', 'Aknor')

    expect(createCharacterDraftStore(storage).values.name).toBe('Aknor')
  })
})

describe('readStoredDraft — leitura defensiva', () => {
  it('sem blob → defaults', () => {
    expect(readStoredDraft(null).values).toEqual(wizardDefaults)
  })

  it('JSON corrompido não derruba a Forja', () => {
    expect(readStoredDraft('{{{').values).toEqual(wizardDefaults)
  })

  it('blob parcial completa com os defaults', () => {
    const raw = JSON.stringify({ state: { values: { races: ['Anão'] } } })

    const { values } = readStoredDraft(raw)

    expect(values.races).toEqual(['Anão'])
    expect(values.size).toBe('Médio')
    expect(values.classPowers).toEqual([])
  })

  it('campo com o tipo errado cai para o default em vez de vazar', () => {
    const raw = JSON.stringify({
      state: { values: { races: 'Anão', hpMax: '12', name: 'Aknor' } },
    })

    const { values } = readStoredDraft(raw)

    expect(values.races).toEqual([])
    expect(values.hpMax).toBe(10)
    expect(values.name).toBe('Aknor')
  })

  it('chave desconhecida no blob é descartada', () => {
    const raw = JSON.stringify({ state: { values: { intruso: 1 } } })

    expect(readStoredDraft(raw).values).not.toHaveProperty('intruso')
  })

  it('modo de atributo inválido volta para livre', () => {
    const raw = JSON.stringify({ state: { attributeMode: 'roubado' } })

    expect(readStoredDraft(raw).attributeMode).toBe('free')
  })

  it('raceChoices não-objeto é ignorado', () => {
    const raw = JSON.stringify({ state: { raceChoices: ['Anão'] } })

    expect(readStoredDraft(raw).raceChoices).toEqual({})
  })
})
