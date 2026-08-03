import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getActivation } from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { characterQueryOptions } from '@/entities/character/queries'
import { useConditionalsStore } from '@/shared/stores/conditionals-store'
import { useStanceActivationStore } from '@/shared/stores/stance-activation-store'
import { UsePowerDialog } from './use-power-dialog'

// Live scaling stance — the dialog only renders for specs with `scaling`.
const FURIA = getActivation('class.barbaro.furia')!

// Named fake — full Character shape with neutral defaults; tests override
// only what the dialog reads (classes, mpCurrent, classPowers).
function fakeCharacter(overrides: Partial<Character> = {}): Character {
  return {
    id: 1,
    ownerId: 1,
    name: 'Teste Bárbaro',
    origin: 'Acólito',
    god: null,
    godPower: '',
    tibar: 0,
    level: 6,
    hpMax: 40,
    hpCurrent: 40,
    mpMax: 18,
    mpCurrent: 12,
    strength: 3,
    dexterity: 1,
    constitution: 2,
    intelligence: 0,
    wisdom: 1,
    charisma: 0,
    size: 'Médio',
    displacement: 9,
    proficiencies: '[]',
    raceAbilityChoices: '[]',
    activeConditions: '[]',
    raceAttributeChoices: '{}',
    secondaryRaceChoices: '[]',
    originChoices: '[]',
    classPowers: '[]',
    classChoices: '{}',
    powerChoices: '{}',
    createdAt: '',
    updatedAt: '',
    races: [{ race: 'Humano' }],
    classes: [{ className: 'Bárbaro', level: 6 }],
    expertises: [],
    items: [],
    activeEffects: [],
    spells: [],
    ...overrides,
  }
}

// Named fake for the vitals PATCH the optimistic PM debit fires — records
// calls instead of hitting the network.
class FakeVitalsFetch {
  calls: { url: string; body: unknown }[] = []
  install() {
    const handler = async (url: RequestInfo | URL, init?: RequestInit) => {
      this.calls.push({
        url: String(url),
        body: init?.body ? JSON.parse(String(init.body)) : null,
      })
      return new Response('{}', {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    }
    vi.stubGlobal('fetch', handler as typeof fetch)
  }
}

function renderDialog(character: Character) {
  const qc = new QueryClient()
  qc.setQueryData(characterQueryOptions(character.id).queryKey, character)
  render(
    <QueryClientProvider client={qc}>
      <UsePowerDialog spec={FURIA} character={character} />
    </QueryClientProvider>,
  )
  return qc
}

beforeEach(() => {
  localStorage.clear()
  useConditionalsStore.setState({ active: {} })
  useStanceActivationStore.setState({ records: {} })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('UsePowerDialog — Fúria stepper', () => {
  it('opens with base cost and the flag-group effect preview', () => {
    renderDialog(fakeCharacter())
    fireEvent.click(screen.getByRole('button', { name: 'Ativar Fúria' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Custo total')).toBeInTheDocument()
    expect(screen.getByText('2 PM')).toBeInTheDocument()
    // Fúria's engine modifiers surface as preview lines (attack among them).
    expect(screen.getAllByText('Ataque').length).toBeGreaterThan(0)
    expect(screen.getByText('PM restante após ativar: 10')).toBeInTheDocument()
  })

  it('happy path: +1 step costs 3 PM, activating debits PM and records the payment', async () => {
    const fetchFake = new FakeVitalsFetch()
    fetchFake.install()
    const qc = renderDialog(fakeCharacter())
    fireEvent.click(screen.getByRole('button', { name: 'Ativar Fúria' }))
    fireEvent.click(screen.getByRole('button', { name: 'Aumentar passos' }))
    expect(screen.getByText('3 PM')).toBeInTheDocument()
    expect(screen.getByText('+1 extra (stepper) — anotação, não somado nos totais')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Ativar' }))
    // PM debit callback: optimistic cache write + PATCH body carry 12 − 3.
    await waitFor(() => expect(fetchFake.calls.length).toBe(1))
    expect(fetchFake.calls[0].body).toEqual({ mpCurrent: 9 })
    expect(
      qc.getQueryData<Character>(characterQueryOptions(1).queryKey)?.mpCurrent,
    ).toBe(9)
    // Same store state the old Efeitos toggle produced — flag group all on.
    expect(useConditionalsStore.getState().active[1]?.length).toBeGreaterThan(0)
    expect(useStanceActivationStore.getState().records[1]?.furia).toEqual({
      steps: 1,
      pmPaid: 3,
    })
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    )
  })

  it('clamps steps to maxStepsForLevel (Bárbaro 6 → 1 step)', () => {
    renderDialog(fakeCharacter())
    fireEvent.click(screen.getByRole('button', { name: 'Ativar Fúria' }))
    const plus = screen.getByRole('button', { name: 'Aumentar passos' })
    fireEvent.click(plus)
    expect(plus).toBeDisabled()
  })

  it('shows red insufficient-PM state and disables Ativar', () => {
    renderDialog(fakeCharacter({ mpCurrent: 1 }))
    fireEvent.click(screen.getByRole('button', { name: 'Ativar Fúria' }))
    expect(screen.getByText('PM insuficiente para ativar')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Ativar' })).toBeDisabled()
  })

  it('previews Alma de Bronze temp PV when the power is owned', () => {
    renderDialog(fakeCharacter({ classPowers: '["alma-de-bronze"]' }))
    fireEvent.click(screen.getByRole('button', { name: 'Ativar Fúria' }))
    expect(
      screen.getByText('Alma de Bronze: PV temp = nível + For'),
    ).toBeInTheDocument()
  })
})
