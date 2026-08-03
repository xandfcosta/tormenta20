import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Character } from '@/shared/api/api'
import { characterQueryOptions } from '@/entities/character/queries'
import { tempHpPool } from '@/entities/character/temp-hp-pool'
import { useApplyDamage } from '@/entities/character/use-apply-damage'
import { useManualTempHp } from '@/entities/character/use-manual-temp-hp'
import { ResourceAdjustDialog } from './resource-bar'

/**
 * F3 — pool-aware VIDA ✎ dialog: the manual "PV temporários" setter posts
 * `{ manualTempHp }` to :id/active-effects and damage routes through the
 * atomic POST :id/damage (F2) instead of a vitals write.
 */

// Named fake — full Character with an active Alma de Bronze pool of 10.
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
    activeEffects: [
      {
        id: 50,
        catalogId: 'class.barbaro.alma-de-bronze',
        scope: 'scene',
        modifiers: JSON.stringify([
          { target: { k: 'tempHp' }, amount: 10, bonusType: 'untyped', note: 'PV temporários' },
        ]),
        createdAt: '2026-08-01T00:00:00Z',
      },
    ],
    spells: [],
    ...overrides,
  }
}

// Named fake fetch — records every call and answers the two POST endpoints
// the dialog can hit with minimal happy-path bodies.
class FakeCharacterApiFetch {
  calls: { url: string; method: string; body: unknown }[] = []
  install() {
    const handler = async (url: RequestInfo | URL, init?: RequestInit) => {
      const call = {
        url: String(url),
        method: init?.method ?? 'GET',
        body: init?.body ? JSON.parse(String(init.body)) : null,
      }
      this.calls.push(call)
      return new Response(JSON.stringify(this.responseFor(call.url)), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    }
    vi.stubGlobal('fetch', handler as typeof fetch)
  }

  private responseFor(url: string): unknown {
    if (url.endsWith('/damage')) {
      return { hpCurrent: 40, tempHpRemaining: 3, drained: [] }
    }
    return {
      effect: {
        id: 60,
        catalogId: 'manual-temp-hp',
        scope: 'scene',
        modifiers: JSON.stringify([{ target: { k: 'tempHp' }, amount: 12 }]),
        createdAt: '2026-08-01T00:00:00Z',
      },
      displaced: [],
    }
  }
}

/** Harness wiring the dialog exactly like the HUD's VIDA row does. */
function VidaDialogHarness({ character }: { character: Character }) {
  const pool = tempHpPool(character)
  const { applyDamage } = useApplyDamage(character)
  const { setManualTempHp } = useManualTempHp(character)
  return (
    <ResourceAdjustDialog
      label="Vida"
      current={character.hpCurrent}
      max={character.hpMax}
      onSetCurrent={() => {}}
      onDamage={applyDamage}
      tempPool={{ total: pool.total, onSetManual: setManualTempHp }}
    />
  )
}

function renderDialog(character: Character) {
  const qc = new QueryClient()
  qc.setQueryData(characterQueryOptions(character.id).queryKey, character)
  return render(
    <QueryClientProvider client={qc}>
      <VidaDialogHarness character={character} />
    </QueryClientProvider>,
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('ResourceAdjustDialog — pool-aware VIDA (F3)', () => {
  it('shows the current pool and SETS the manual pool via manualTempHp', async () => {
    const fetchFake = new FakeCharacterApiFetch()
    fetchFake.install()
    renderDialog(fakeCharacter())
    fireEvent.click(screen.getByRole('button', { name: 'Editar Vida' }))
    // Current pool total + vale-o-maior helper copy are visible.
    expect(screen.getByText('+10')).toBeTruthy()
    expect(screen.getByText(/vale o maior — não acumulam \(p256\)/)).toBeTruthy()
    fireEvent.change(screen.getByLabelText('PV temporários'), {
      target: { value: '12' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Definir' }))
    await waitFor(() => {
      expect(fetchFake.calls).toContainEqual({
        url: '/api/characters/1/active-effects',
        method: 'POST',
        body: { manualTempHp: 12 },
      })
    })
  })

  it('routes "Remover" through the atomic damage endpoint (F2)', async () => {
    const fetchFake = new FakeCharacterApiFetch()
    fetchFake.install()
    renderDialog(fakeCharacter())
    fireEvent.click(screen.getByRole('button', { name: 'Editar Vida' }))
    fireEvent.change(screen.getByLabelText('Quantidade'), {
      target: { value: '7' },
    })
    fireEvent.submit(screen.getByLabelText('Quantidade').closest('form')!)
    await waitFor(() => {
      expect(fetchFake.calls).toContainEqual({
        url: '/api/characters/1/damage',
        method: 'POST',
        body: { amount: 7 },
      })
    })
  })
})
