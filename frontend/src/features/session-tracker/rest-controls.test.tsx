import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { RestControls } from './rest-controls'

/** Named fake for the session socket — o descanso só chama `rest`. */
class FakeRealtime {
  readonly rest = vi.fn()

  asRealtime(): SessionRealtime {
    return { isConnected: () => true, rest: this.rest } as unknown as SessionRealtime
  }
}

function renderRest() {
  const rt = new FakeRealtime()
  const user = userEvent.setup()
  render(() => <RestControls rt={rt.asRealtime()} />)
  return { rt, user }
}

describe('RestControls', () => {
  // Ação rápida é rápida: o descanso de cena não pede nenhum dado, então não
  // pode cobrar um diálogo antes de acontecer.
  it('descanso de cena acontece no clique, sem diálogo', async () => {
    const { rt, user } = renderRest()

    await user.click(screen.getByRole('button', { name: 'Recuperar · cena' }))

    expect(rt.rest).toHaveBeenCalledWith('scene')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  // O de dia precisa da QUALIDADE (p105), e é só por isso que abre diálogo.
  it('descanso de dia pergunta a qualidade e só descansa ao confirmar', async () => {
    const { rt, user } = renderRest()

    await user.click(screen.getByRole('button', { name: 'Recuperar · dia' }))
    expect(rt.rest).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: /Qualidade do descanso/ }))
    await user.click(await screen.findByRole('option', { name: 'Luxuosa (3×)' }))
    await user.click(screen.getByRole('button', { name: 'Descansar' }))

    expect(rt.rest).toHaveBeenCalledWith('day', 'luxuosa')
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('cancelar fecha sem descansar', async () => {
    const { rt, user } = renderRest()

    await user.click(screen.getByRole('button', { name: 'Recuperar · dia' }))
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(rt.rest).not.toHaveBeenCalled()
  })
})
