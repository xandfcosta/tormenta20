import { FakeStorage } from '@/shared/test/fake-storage'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { allConditionals } from '@/entities/character/derived'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character } from '@/shared/api/api'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import {
  type ConditionalsStore,
  createConditionalsStore,
} from '@/shared/stores/conditionals-store'
import { PowerUsesProvider } from '@/shared/stores/power-uses-context'
import { createPowerUsesStore } from '@/shared/stores/power-uses-store'
import { StanceActivationProvider } from '@/shared/stores/stance-activation-context'
import { createStanceActivationStore } from '@/shared/stores/stance-activation-store'
import { StancesSection, activeStanceGroups, stanceSummary } from './stances-section'


const barbaro = (overrides: Partial<Character> = {}) =>
  makeCharacter({
    classes: [{ className: 'Bárbaro', level: 6 }],
    mpMax: 20,
    mpCurrent: 20,
    ...overrides,
  })

/** Ids of every conditional the Fúria flag raises for this character. */
function furiaEntryIds(character: Character): string[] {
  return allConditionals(character, new Set())
    .filter((entry) => entry.effect.flag === 'furia')
    .map((entry) => entry.id)
}

function renderStances(character: Character, tune?: (store: ConditionalsStore) => void) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(character.id).queryKey, character)
  const conditionals = createConditionalsStore(new FakeStorage())
  tune?.(conditionals)
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={conditionals}>
        <PowerUsesProvider store={createPowerUsesStore(new FakeStorage())}>
          <StanceActivationProvider store={createStanceActivationStore(new FakeStorage())}>
            <StancesSection character={character} />
          </StanceActivationProvider>
        </PowerUsesProvider>
      </ConditionalsProvider>
    </QueryClientProvider>
  ))
  return { user: userEvent.setup(), conditionals }
}

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('activeStanceGroups', () => {
  it('só agrupa postura cujas entradas estão TODAS ligadas', () => {
    const character = barbaro()
    const ids = furiaEntryIds(character)
    const partial = new Set([ids[0]])

    expect(activeStanceGroups(allConditionals(character, partial))).toEqual([])
    expect(activeStanceGroups(allConditionals(character, new Set(ids)))).toHaveLength(1)
  })
})

describe('stanceSummary', () => {
  // Bárbaro 6 tem o degrau +3; listar +2 junto seria mostrar um degrau que a
  // resolução do motor já descartou.
  it('mostra só o degrau vencedor', () => {
    const character = barbaro()
    const [group] = activeStanceGroups(allConditionals(character, new Set(furiaEntryIds(character))))

    const summary = stanceSummary(group)

    expect(summary).toContain('+3')
    expect(summary).not.toContain('+2')
  })
})

describe('StancesSection', () => {
  it('sem postura ligada, não pinta a seção', () => {
    renderStances(barbaro())

    expect(screen.queryByText('Posturas ativas')).not.toBeInTheDocument()
  })

  it('postura ligada aparece com o resumo e o que foi pago', () => {
    const character = barbaro()
    renderStances(character, (store) =>
      store.setMany(character.id, furiaEntryIds(character), true),
    )

    expect(screen.getByText('Posturas ativas')).toBeInTheDocument()
    expect(screen.getByText('Fúria')).toBeInTheDocument()
    // Sem registro de pagamento (postura ligada antes desta fase), cai no custo base.
    expect(screen.getByText(/2 PM/)).toBeInTheDocument()
  })

  it('encerrar desliga a flag do grupo inteiro', async () => {
    const character = barbaro()
    const { user, conditionals } = renderStances(character, (store) =>
      store.setMany(character.id, furiaEntryIds(character), true),
    )

    await user.click(screen.getByRole('button', { name: 'Encerrar Fúria' }))

    await waitFor(() => expect(conditionals.active(character.id).size).toBe(0))
  })
})
