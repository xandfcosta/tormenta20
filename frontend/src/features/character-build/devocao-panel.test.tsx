import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { DevocaoPanel } from './devocao-panel'

describe('DevocaoPanel — poder concedido à escolha (p96)', () => {
  it('lista os 4 poderes do deus com descrição', () => {
    render(
      <DevocaoPanel
        godName="Khalmyr"
        value=""
        onChange={() => {}}
        raceNames={['Humano']}
        classNames={[]}
      />,
    )
    expect(screen.getByText('Devoção a Khalmyr')).toBeInTheDocument()
    expect(screen.getAllByRole('radio')).toHaveLength(4)
    expect(screen.getByText('Coragem Total')).toBeInTheDocument()
    expect(screen.getByText(/imune a efeitos de medo/i)).toBeInTheDocument()
  })

  it('clicar escolhe; clicar de novo desfaz', () => {
    const onChange = vi.fn()
    const { rerender } = render(
      <DevocaoPanel
        godName="Khalmyr"
        value=""
        onChange={onChange}
        raceNames={['Humano']}
        classNames={[]}
      />,
    )
    fireEvent.click(screen.getByText('Coragem Total'))
    expect(onChange).toHaveBeenCalledWith('Coragem Total')
    rerender(
      <DevocaoPanel
        godName="Khalmyr"
        value="Coragem Total"
        onChange={onChange}
        raceNames={['Humano']}
        classNames={[]}
      />,
    )
    fireEvent.click(screen.getByText('Coragem Total'))
    expect(onChange).toHaveBeenLastCalledWith('')
  })

  it('aviso consultivo quando raça/classe fora da lista de devotos', () => {
    render(
      <DevocaoPanel
        godName="Arsenal"
        value=""
        onChange={() => {}}
        raceNames={['Elfo']}
        classNames={['Bardo']}
      />,
    )
    expect(screen.getByText(/fora da lista de devotos/i)).toBeInTheDocument()
  })

  it('Humano nunca gera aviso (exceção p96)', () => {
    render(
      <DevocaoPanel
        godName="Arsenal"
        value=""
        onChange={() => {}}
        raceNames={['Humano']}
        classNames={['Bardo']}
      />,
    )
    expect(screen.queryByText(/fora da lista/i)).not.toBeInTheDocument()
  })

  it('deus desconhecido → nada', () => {
    const { container } = render(
      <DevocaoPanel
        godName="Cthulhu"
        value=""
        onChange={() => {}}
        raceNames={[]}
        classNames={[]}
      />,
    )
    expect(container).toBeEmptyDOMElement()
  })
})
