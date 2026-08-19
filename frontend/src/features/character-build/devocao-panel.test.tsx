import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { DevocaoPanel } from './devocao-panel'

/**
 * DEVOÇÃO (ALE-197, grupo B).
 *
 * A ALE-197 listou este painel como "Clérigo + Khalmyr TRAVA o devoto". A
 * premissa está errada, e o que segue fixa o que o código realmente faz —
 * porque a diferença é uma decisão de produto escrita na docstring dele: a
 * elegibilidade do p96 é CONSULTIVA. Raça/classe fora da lista de devotos do
 * deus AVISA "negociado com o mestre" e deixa escolher, porque essa chamada é
 * da mesa e não do formulário. E `Clérigo` (como `Humano`) é passe universal na
 * regra: um clérigo de Khalmyr nunca é o caso de aviso.
 *
 * Uma trava aqui seria o defeito, não a regra — e agora quem a introduzir
 * quebra um teste em vez de descobrir na mesa.
 */

function renderPanel(options: {
  races?: string[]
  classes?: string[]
  value?: string
  onChange?: (name: string) => void
}) {
  const onChange = options.onChange ?? vi.fn()
  render(() => (
    <DevocaoPanel
      godName="Khalmyr"
      value={options.value ?? ''}
      onChange={onChange}
      raceNames={options.races ?? ['Elfo']}
      classNames={options.classes ?? ['Arcanista']}
    />
  ))
  return { user: userEvent.setup(), onChange }
}

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('DevocaoPanel', () => {
  it('clérigo não recebe aviso nenhum', async () => {
    renderPanel({ races: ['Elfo'], classes: ['Clérigo'] })

    expect(await screen.findByLabelText('Poder concedido de Khalmyr')).toBeInTheDocument()
    expect(screen.queryByText(/negociado com o mestre/)).not.toBeInTheDocument()
  })

  it('paladino está na lista de Khalmyr e também passa limpo', async () => {
    renderPanel({ races: ['Elfo'], classes: ['Paladino'] })

    // A outra metade da regra: sem a exceção de Clérigo, o que vale é a linha
    // "Devotos" do deus (p96) — Paladinos estão nela.
    expect(await screen.findByLabelText('Poder concedido de Khalmyr')).toBeInTheDocument()
    expect(screen.queryByText(/negociado com o mestre/)).not.toBeInTheDocument()
  })

  it('fora da lista, AVISA — e continua deixando escolher', async () => {
    const { user, onChange } = renderPanel({ races: ['Elfo'], classes: ['Arcanista'] })

    expect(await screen.findByText(/negociado com o mestre/)).toBeInTheDocument()

    // O aviso não vira bloqueio: quem decide é a mesa, e o formulário registra
    // o que ela decidiu.
    await user.click(screen.getByRole('button', { name: /Espada Justiceira/ }))
    expect(onChange).toHaveBeenCalledWith('Espada Justiceira')
  })

  it('clicar no poder já escolhido o desmarca', async () => {
    const { user, onChange } = renderPanel({ value: 'Espada Justiceira' })

    const escolhido = await screen.findByRole('button', { name: /Espada Justiceira/ })
    expect(escolhido).toHaveAttribute('aria-pressed', 'true')

    await user.click(escolhido)

    // Desmarcar tem de existir: é UM poder e a escolha é irreversível na ficha
    // depois; sem o toggle, um clique errado na Forja não teria volta.
    expect(onChange).toHaveBeenCalledWith('')
  })
})
