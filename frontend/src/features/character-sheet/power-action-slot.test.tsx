import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getActivation } from '@tormenta20/t20-data'
import type { ActivationSpec } from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { useConditionalsStore } from '@/shared/stores/conditionals-store'
import { usePowerUsesStore } from '@/shared/stores/power-uses-store'
import { useStanceActivationStore } from '@/shared/stores/stance-activation-store'
import { PowerActionSlot } from './power-action-slot'

// Named fake — full Character shape with neutral defaults; tests override
// only what the slot reads (id, mpCurrent).
function fakeCharacter(overrides: Partial<Character> = {}): Character {
  return {
    id: 1,
    ownerId: 1,
    name: 'Teste Bárbaro',
    origin: 'Acólito',
    god: null,
    godPower: '',
    tibar: 0,
    level: 4,
    hpMax: 20,
    hpCurrent: 20,
    mpMax: 12,
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
    classes: [{ className: 'Bárbaro', level: 4 }],
    expertises: [],
    items: [],
    activeEffects: [],
    spells: [],
    ...overrides,
  }
}

// Named fake spec — hand-built so display strings (cost, uses) are controlled.
function fakeInstantSpec(overrides: Partial<ActivationSpec> = {}): ActivationSpec {
  return {
    id: 'class.teste.poder-teste',
    name: 'Poder Teste',
    kind: 'instant',
    action: 'livre',
    pmCost: 2,
    uses: null,
    bookPage: 1,
    ...overrides,
  }
}

function renderSlot(spec: ActivationSpec | undefined, character: Character) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <PowerActionSlot spec={spec} character={character} />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  localStorage.clear()
  usePowerUsesStore.setState({ uses: {} })
  useConditionalsStore.setState({ active: {} })
  useStanceActivationStore.setState({ records: {} })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('PowerActionSlot — instant', () => {
  it('renders an enabled Usar button with the PM cost and action chip', () => {
    renderSlot(fakeInstantSpec(), fakeCharacter())
    const button = screen.getByRole('button', { name: 'Usar Poder Teste' })
    expect(button).toBeEnabled()
    expect(button).toHaveTextContent('Usar 2 PM')
    expect(screen.getByText('LIVRE · 2 PM')).toBeInTheDocument()
  })

  it('disables with reason when PM is insufficient', () => {
    renderSlot(fakeInstantSpec({ pmCost: 5 }), fakeCharacter({ mpCurrent: 3 }))
    const button = screen.getByRole('button', { name: 'Usar Poder Teste' })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('title', 'PM insuficiente')
  })

  it('shows the scene counter and disables at the limit', () => {
    usePowerUsesStore.getState().bump(1, 'class.teste.poder-teste', 'scene')
    renderSlot(fakeInstantSpec({ uses: 'cena' }), fakeCharacter())
    expect(screen.getByText('usado 1/1 cena')).toBeInTheDocument()
    const button = screen.getByRole('button', { name: 'Usar Poder Teste' })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('title', 'limite por cena atingido')
  })

  it('disables with "requer <flag>" when the stance is off', () => {
    renderSlot(fakeInstantSpec({ requiresFlag: 'furia' }), fakeCharacter())
    const button = screen.getByRole('button', { name: 'Usar Poder Teste' })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('title', 'requer furia')
  })
})

describe('PowerActionSlot — non-instant kinds', () => {
  it('renders nothing without a spec', () => {
    const { container } = renderSlot(undefined, fakeCharacter())
    expect(container).toBeEmptyDOMElement()
  })

  it('renders a passive chip for passives', () => {
    renderSlot(
      fakeInstantSpec({ kind: 'passive', action: 'passivo', pmCost: 0 }),
      fakeCharacter(),
    )
    expect(screen.getByText('Passiva')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

})

// Named fake for the vitals PATCH fired by stance activation — records calls
// instead of hitting the network.
function installFetchFake(): { calls: unknown[] } {
  const recorder = { calls: [] as unknown[] }
  const handler = async (_url: RequestInfo | URL, init?: RequestInit) => {
    recorder.calls.push(init?.body ? JSON.parse(String(init.body)) : null)
    return new Response('{}', {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }
  vi.stubGlobal('fetch', handler as typeof fetch)
  return recorder
}

describe('PowerActionSlot — stance', () => {
  it('renders the Postura chip and a one-tap Ativar for fixed-cost stances', () => {
    renderSlot(fakeInstantSpec({ kind: 'stance', pmCost: 2 }), fakeCharacter())
    expect(screen.getByText('Postura · 2 PM')).toBeInTheDocument()
    const button = screen.getByRole('button', { name: 'Ativar Poder Teste' })
    expect(button).toBeEnabled()
    expect(button).toHaveTextContent('Ativar 2 PM')
  })

  it('disables the fixed-cost Ativar when PM is insufficient', () => {
    renderSlot(
      fakeInstantSpec({ kind: 'stance', pmCost: 5 }),
      fakeCharacter({ mpCurrent: 3 }),
    )
    const button = screen.getByRole('button', { name: 'Ativar Poder Teste' })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('title', 'PM insuficiente')
  })

  it('scaling stances open the stepper dialog instead of one-tap', () => {
    renderSlot(getActivation('class.barbaro.furia'), fakeCharacter())
    expect(screen.getByText('Postura · 2+ PM')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Ativar Fúria' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('active stance shows ATIVA · Encerrar and ends it (state matches the old toggle)', async () => {
    installFetchFake()
    renderSlot(getActivation('class.barbaro.furia'), fakeCharacter())
    fireEvent.click(screen.getByRole('button', { name: 'Ativar Fúria' }))
    fireEvent.click(screen.getByRole('button', { name: 'Ativar' }))
    const encerrar = await screen.findByRole('button', {
      name: 'Encerrar Fúria',
    })
    expect(encerrar).toHaveTextContent('ATIVA · Encerrar')
    expect(useConditionalsStore.getState().active[1]?.length).toBeGreaterThan(0)

    fireEvent.click(encerrar)
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Ativar Fúria' }),
      ).toBeInTheDocument(),
    )
    // Exit is free (no refund) and forgets the payment record.
    expect(useConditionalsStore.getState().active[1] ?? []).toHaveLength(0)
    expect(
      useStanceActivationStore.getState().records[1]?.furia,
    ).toBeUndefined()
  })
})

