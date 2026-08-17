import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { InitiativeEntry, SessionRuntimeState } from '@/shared/realtime/realtime'
import { TurnControls, TurnCounter } from './turn-controls'

const entrada = (id: string) => ({ id, label: id }) as unknown as InitiativeEntry

const estado = (over: Partial<SessionRuntimeState> = {}): SessionRuntimeState => ({
  initiative: [entrada('a'), entrada('b'), entrada('c')],
  round: 1,
  turnIndex: 0,
  turnsTaken: 1,
  ...over,
})

/**
 * O contador (ALE-142). O total vem CONTADO do servidor; a posição na rodada
 * sai do estado que já existia. O que este teste protege é quando cada pedaço
 * APARECE — mostrar "Turno 0/3" antes de o combate começar diria que a rodada
 * já anda quando ela não anda.
 */
describe('TurnCounter', () => {
  it('em combate, diz rodada, posição e total', () => {
    render(() => <TurnCounter state={estado({ round: 2, turnIndex: 1, turnsTaken: 14 })} />)

    expect(screen.getByText(/Rodada 2/)).toHaveTextContent('Rodada 2 · Turno 2/3 · 14 no total')
  })

  it('antes do primeiro turno, só a rodada', () => {
    render(() => <TurnCounter state={estado({ round: 0, turnIndex: -1, turnsTaken: 0 })} />)

    const texto = screen.getByText(/Rodada/)
    expect(texto).toHaveTextContent('Rodada 0')
    expect(texto).not.toHaveTextContent('Turno')
    expect(texto).not.toHaveTextContent('no total')
  })

  // Sessão gravada antes do campo existir volta sem ele: a linha continua
  // legível em vez de dizer "undefined no total".
  it('sem o campo do servidor, omite o total', () => {
    render(() => <TurnCounter state={estado({ turnsTaken: undefined })} />)

    expect(screen.getByText(/Rodada/)).toHaveTextContent('Rodada 1 · Turno 1/3')
    expect(screen.getByText(/Rodada/)).not.toHaveTextContent('no total')
  })
})

/**
 * O par de turno (ALE-132): dois botões da MESMA família. `onlyNext` é a faixa
 * fixa abaixo de 1024, onde a iniciativa some da tela e o mestre não pode
 * perder o botão mais clicado da sessão.
 */
describe('TurnControls', () => {
  it('o par tem os dois verbos, cada um com nome próprio', async () => {
    const proximo = vi.fn()
    const anterior = vi.fn()
    render(() => <TurnControls connected onNext={proximo} onPrevious={anterior} />)
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'Turno anterior' }))
    await user.click(screen.getByRole('button', { name: 'Próximo turno' }))

    expect(anterior).toHaveBeenCalledOnce()
    expect(proximo).toHaveBeenCalledOnce()
  })

  it('onlyNext deixa apenas o avanço', () => {
    render(() => <TurnControls onlyNext connected onNext={vi.fn()} onPrevious={vi.fn()} />)

    expect(screen.getByRole('button', { name: 'Próximo turno' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Turno anterior' })).not.toBeInTheDocument()
  })

  // Sem conexão o clique não chegaria ao servidor: oferecer o botão vivo
  // faria a tela mentir sobre o que aconteceu.
  it('desconectado, ninguém anda o turno', () => {
    render(() => <TurnControls connected={false} onNext={vi.fn()} onPrevious={vi.fn()} />)

    expect(screen.getByRole('button', { name: 'Próximo turno' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Turno anterior' })).toBeDisabled()
  })
})
