import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { InitiativeEntry, SessionRuntimeState } from '@/shared/realtime/realtime'
import { TurnAdvance, TurnCounter } from './turn-controls'

const entrada = (id: string, label = id) => ({ id, label }) as unknown as InitiativeEntry

const estado = (over: Partial<SessionRuntimeState> = {}): SessionRuntimeState => ({
  initiative: [entrada('a', 'Arwen'), entrada('b', 'Ogro'), entrada('c', 'Zumbi 1')],
  round: 1,
  turnIndex: 0,
  turnsTaken: 1,
  // Em cena por padrão: fora dela a faixa mostra "Iniciar cena" e não há turno
  // para avançar (ALE-210).
  sceneActive: true,
  ...over,
})

/**
 * O contador (ALE-142, redesenhado na ALE-184). A posição na rodada sai do
 * estado que já existia. O que este teste protege é quando cada pedaço
 * APARECE — mostrar "Turno 0/3" antes de o combate começar diria que a rodada
 * já anda quando ela não anda.
 */
describe('TurnCounter', () => {
  it('em combate, diz a rodada e a posição', () => {
    render(() => <TurnCounter state={estado({ round: 2, turnIndex: 1 })} />)

    expect(screen.getByText(/Rodada 2/)).toHaveTextContent('Rodada 2 · Turno 2/3')
  })

  it('antes do primeiro turno, só a rodada', () => {
    render(() => <TurnCounter state={estado({ round: 0, turnIndex: -1 })} />)

    const texto = screen.getByText(/Rodada/)
    expect(texto).toHaveTextContent('Rodada 0')
    expect(texto).not.toHaveTextContent('Turno')
  })

  // O total contado pelo servidor SAIU do desenho (ALE-184): "Turno 9/9 · 6 no
  // total" com nove linhas na lista lia como contradição no meio do combate.
  // O campo continua chegando no estado; o que saiu foi a tinta.
  it('o total do servidor não é mais desenhado', () => {
    render(() => <TurnCounter state={estado({ turnIndex: 1, turnsTaken: 14 })} />)

    expect(screen.getByText(/Rodada/)).not.toHaveTextContent('no total')
    expect(screen.queryByText(/14/)).not.toBeInTheDocument()
  })
})

/**
 * O avanço (ALE-132 + ALE-184): o par continua par — voltar e avançar andam na
 * mesma ordem do combate —, mas o peso mudou. O avanço é largo, preenchido e
 * DIZ PARA ONDE VAI; o `‹` fica do tamanho de um ícone, porque desfazer turno é
 * raro. `onlyNext` é a faixa fixa abaixo de 1024, onde a iniciativa some da
 * tela e o mestre não pode perder o botão mais clicado da sessão.
 */
describe('TurnAdvance', () => {
  it('o botão anuncia quem entra, e o par tem os dois verbos', async () => {
    const proximo = vi.fn()
    const anterior = vi.fn()
    render(() => (
      <TurnAdvance state={estado()} connected onNext={proximo} onPrevious={anterior} />
    ))
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'Turno anterior' }))
    // O nome do próximo É o nome acessível: com "▶" o mestre contava a lista
    // para saber para onde ia, e quem lê por leitor de tela contava também.
    await user.click(screen.getByRole('button', { name: 'Próximo: Ogro' }))

    expect(anterior).toHaveBeenCalledOnce()
    expect(proximo).toHaveBeenCalledOnce()
  })

  it('onlyNext deixa apenas o avanço', () => {
    render(() => (
      <TurnAdvance onlyNext state={estado()} connected onNext={vi.fn()} onPrevious={vi.fn()} />
    ))

    expect(screen.getByRole('button', { name: 'Próximo: Ogro' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Turno anterior' })).not.toBeInTheDocument()
  })

  // Sem conexão o clique não chegaria ao servidor: oferecer o botão vivo
  // faria a tela mentir sobre o que aconteceu.
  it('desconectado, ninguém anda o turno', () => {
    render(() => (
      <TurnAdvance state={estado()} connected={false} onNext={vi.fn()} onPrevious={vi.fn()} />
    ))

    expect(screen.getByRole('button', { name: 'Próximo: Ogro' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Turno anterior' })).toBeDisabled()
  })

  // Lista vazia: avançar não tem destino, e um botão vivo que não faz nada é
  // pior que um botão apagado.
  it('sem combatentes, o avanço fica travado', () => {
    render(() => (
      <TurnAdvance
        state={estado({ initiative: [], turnIndex: -1 })}
        connected
        onNext={vi.fn()}
        onPrevious={vi.fn()}
      />
    ))

    expect(screen.getByRole('button', { name: 'Ninguém na fila' })).toBeDisabled()
  })
})
