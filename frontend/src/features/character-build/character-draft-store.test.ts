import { beforeEach, describe, expect, it } from 'vitest'
import {
  hasResumableDraft,
  useCharacterDraftStore,
} from './character-draft-store'
import { wizardDefaults } from './wizard-steps'

describe('character-draft-store', () => {
  beforeEach(() => {
    useCharacterDraftStore.getState().reset()
  })

  it('starts from the wizard defaults', () => {
    expect(useCharacterDraftStore.getState().values).toEqual(wizardDefaults)
    expect(useCharacterDraftStore.getState().raceChoices).toEqual({})
  })

  it('stores form values and race choices', () => {
    const { setValues, setRaceChoices } = useCharacterDraftStore.getState()
    setValues({ ...wizardDefaults, name: 'Aknor', races: ['Anão'] })
    setRaceChoices({ Humano: { floatingPicks: ['strength'] } })
    expect(useCharacterDraftStore.getState().values.name).toBe('Aknor')
    expect(useCharacterDraftStore.getState().raceChoices).toEqual({
      Humano: { floatingPicks: ['strength'] },
    })
  })

  it('reset clears back to defaults', () => {
    useCharacterDraftStore.getState().setValues({
      ...wizardDefaults,
      races: ['Elfo'],
    })
    useCharacterDraftStore.getState().reset()
    expect(useCharacterDraftStore.getState().values.races).toEqual([])
  })

  it('hasResumableDraft reflects meaningful progress', () => {
    expect(hasResumableDraft()).toBe(false)
    useCharacterDraftStore
      .getState()
      .setValues({ ...wizardDefaults, races: ['Anão'] })
    expect(hasResumableDraft()).toBe(true)
  })
})
