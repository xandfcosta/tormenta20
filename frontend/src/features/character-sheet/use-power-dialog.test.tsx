import { render, screen, waitFor } from '@solidjs/testing-library'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import { api, type Character } from '@/shared/api/api'
import { activationSpecs } from '@/shared/lib/activation-cache'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { PowerUsesProvider } from '@/shared/stores/power-uses-context'
import { StanceActivationProvider } from '@/shared/stores/stance-activation-context'
import { UsePowerDialog } from './use-power-dialog'
import { fakeConditionals, fakePowerUses, fakeStances } from '@/shared/test/play-stores'

/**
 * ATIVAR A FÚRIA (ALE-197, grupo B).
 *
 * As regras da postura são puras e provadas (`power-rules`): o total, o teto de
 * passos por nível e a decisão. O que nunca foi montado é o diálogo que as usa
 * — e o custo da postura é PM REAL, debitado com rollback. Um teto lido do
 * nível errado ofereceria à mesa um passo que o personagem não tem.
 *
 * A Fúria do livro (p40): base 2 PM, +1 PM por passo, primeiro passo no nível 5
 * de Bárbaro e um passo a cada 5 níveis. A CONTA é do `power-rules`; o que se
 * prova aqui é a LIGAÇÃO — que a tela pergunta ao nível certo e cobra o total.
 */

function furia() {
  const spec = activationSpecs().find((s) => s.id === 'class.barbaro.furia')
  if (!spec) throw new Error('catálogo sem a Fúria do Bárbaro')
  return spec
}

function barbaro(level: number, mpCurrent: number): Character {
  return makeCharacter({
    level,
    classes: [{ className: 'Bárbaro', level }],
    mpMax: 30,
    mpCurrent,
  })
}

function renderDialog(character: Character) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(character.id).queryKey, character)
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={fakeConditionals()}>
        <PowerUsesProvider store={fakePowerUses()}>
          <StanceActivationProvider store={fakeStances()}>
            <UsePowerDialog spec={furia()} character={character} />
          </StanceActivationProvider>
        </PowerUsesProvider>
      </ConditionalsProvider>
    </QueryClientProvider>
  ))
  return { user: userEvent.setup(), client }
}

async function abrir(user: ReturnType<typeof userEvent.setup>, character: Character) {
  await user.click(screen.getByRole('button', { name: `Ativar ${furia().name}` }))
  await screen.findByRole('dialog')
  return character
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

describe('UsePowerDialog', () => {
  it('cada passo soma PM ao total', async () => {
    const heroi = barbaro(10, 20)
    const { user } = renderDialog(heroi)
    await abrir(user, heroi)

    await user.click(screen.getByRole('button', { name: 'Aumentar passos' }))

    expect(screen.getByRole('status', { name: 'Passos extras: 1' })).toBeInTheDocument()
    // Base 2 + 1 do passo. É o número que a mesa lê antes de pagar.
    expect(screen.getByText('3 PM')).toBeInTheDocument()
  })

  it('o teto de passos vem do NÍVEL de classe', async () => {
    // Bárbaro 5: alcança o primeiro passo e para nele.
    const heroi = barbaro(5, 20)
    const { user } = renderDialog(heroi)
    await abrir(user, heroi)

    await user.click(screen.getByRole('button', { name: 'Aumentar passos' }))

    // Oferecer o segundo passo seria oferecer um bônus que o livro não dá
    // neste nível — e o servidor não tem como recusar: o gasto é PM comum.
    expect(screen.getByRole('button', { name: 'Aumentar passos' })).toBeDisabled()
    expect(screen.getByRole('status', { name: 'Passos extras: 1' })).toBeInTheDocument()
  })

  it('sem PM para o total, não dá para ativar', async () => {
    const heroi = barbaro(10, 1)
    const { user } = renderDialog(heroi)
    await abrir(user, heroi)

    // 1 PM na reserva não paga nem a base de 2.
    expect(screen.getByRole('button', { name: 'Ativar' })).toBeDisabled()
  })

  it('ativar cobra o total escolhido e fecha', async () => {
    const updateVitals = vi
      .spyOn(api.characters, 'updateVitals')
      .mockResolvedValue({ hpCurrent: 30, mpCurrent: 17 } as never)
    const heroi = barbaro(10, 20)
    const { user } = renderDialog(heroi)
    await abrir(user, heroi)

    await user.click(screen.getByRole('button', { name: 'Aumentar passos' }))
    await user.click(screen.getByRole('button', { name: 'Ativar' }))

    // 20 − 3: o débito é do total COM os passos, não só da base — cobrar a
    // base daria de graça o bônus que o passo comprou.
    await waitFor(() => expect(updateVitals).toHaveBeenCalledWith(1, { mpCurrent: 17 }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })
})
