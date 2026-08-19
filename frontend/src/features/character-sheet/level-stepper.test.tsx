import { render, screen, waitFor } from '@solidjs/testing-library'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import { api, type Character } from '@/shared/api/api'
import { LevelStepper } from './level-stepper'

/**
 * SUBIR DE NÍVEL (ALE-197, grupo C).
 *
 * O `level-mutations.test.ts` prova a conta e o rollback. O que faltava é a
 * decisão do componente, que a docstring dele enuncia: com UMA classe o degrau
 * sobe direto; com duas, a tela PERGUNTA — adivinhar poria em silêncio um nível
 * de Bardo no Guerreiro, e nada na ficha diria que aconteceu, porque o total
 * subiu do jeito esperado.
 */

function renderStepper(character: Character) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(character.id).queryKey, character)
  render(() => (
    <QueryClientProvider client={client}>
      <LevelStepper character={character} />
    </QueryClientProvider>
  ))
  return { user: userEvent.setup(), client }
}

beforeEach(() => {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: false,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
})

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('LevelStepper', () => {
  it('com uma classe só, sobe direto', async () => {
    const update = vi
      .spyOn(api.characters, 'updateClassLevel')
      .mockResolvedValue({ level: 4, classes: [], hpMax: 40, mpMax: 10 } as never)
    const { user } = renderStepper(
      makeCharacter({ level: 3, classes: [{ className: 'Guerreiro', level: 3 }] }),
    )

    await user.click(screen.getByRole('button', { name: 'Aumentar nível' }))

    // Perguntar "qual classe?" a quem só tem uma seria um clique a mais em
    // toda subida de nível da campanha inteira.
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    await waitFor(() =>
      expect(update).toHaveBeenCalledWith(1, { className: 'Guerreiro', level: 4 }),
    )
  })

  it('multiclasse PERGUNTA antes de escrever', async () => {
    const update = vi.spyOn(api.characters, 'updateClassLevel')
    const { user } = renderStepper(
      makeCharacter({
        level: 5,
        classes: [
          { className: 'Guerreiro', level: 3 },
          { className: 'Bardo', level: 2 },
        ],
      }),
    )

    await user.click(screen.getByRole('button', { name: 'Aumentar nível' }))

    expect(await screen.findByRole('dialog')).toHaveAccessibleName(/Subir nível/)
    // Nada foi escrito ainda: o degrau só existe depois da escolha.
    expect(update).not.toHaveBeenCalled()
  })

  it('a classe escolhida no diálogo é a que sobe', async () => {
    const update = vi
      .spyOn(api.characters, 'updateClassLevel')
      .mockResolvedValue({ level: 6, classes: [], hpMax: 40, mpMax: 10 } as never)
    const { user } = renderStepper(
      makeCharacter({
        level: 5,
        classes: [
          { className: 'Guerreiro', level: 3 },
          { className: 'Bardo', level: 2 },
        ],
      }),
    )
    await user.click(screen.getByRole('button', { name: 'Aumentar nível' }))

    await user.click(await screen.findByRole('button', { name: /Bardo/ }))

    // Nível ABSOLUTO da classe escolhida (2 → 3), não o total do personagem.
    await waitFor(() => expect(update).toHaveBeenCalledWith(1, { className: 'Bardo', level: 3 }))
  })

  it('no nível 1 não dá para descer', () => {
    renderStepper(makeCharacter({ level: 1, classes: [{ className: 'Guerreiro', level: 1 }] }))

    expect(screen.getByRole('button', { name: 'Diminuir nível' })).toBeDisabled()
  })

  it('no teto de 20 não dá para subir', () => {
    renderStepper(makeCharacter({ level: 20, classes: [{ className: 'Guerreiro', level: 20 }] }))

    expect(screen.getByRole('button', { name: 'Aumentar nível' })).toBeDisabled()
  })
})
