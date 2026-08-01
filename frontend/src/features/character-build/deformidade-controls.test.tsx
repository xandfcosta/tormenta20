import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { RaceChoice } from './grant-helpers'
import { DeformidadeControls } from './deformidade-controls'

describe('DeformidadeControls — captura da escolha (Lefou p23)', () => {
  it('renderiza os 2 slots para Lefou', () => {
    render(
      <DeformidadeControls raceName="Lefou" choice={{}} onChange={() => {}} />,
    )
    expect(screen.getByText(/Deformidade · 2 bônus/)).toBeInTheDocument()
    expect(screen.getAllByText(/Perícia \(\+2\)/)).toHaveLength(2)
    expect(
      screen.getByRole('button', { name: /trocar o 2º bônus/i }),
    ).toBeInTheDocument()
  })

  it('não renderiza nada para raça sem a habilidade', () => {
    const { container } = render(
      <DeformidadeControls raceName="Humano" choice={{}} onChange={() => {}} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('toggle de troca limita perícias a 1 e abre o slot de poder', () => {
    const onChange = vi.fn()
    const choice: RaceChoice = {
      deformidade: { pericias: ['Furtividade', 'Percepção'] },
    }
    render(
      <DeformidadeControls raceName="Lefou" choice={choice} onChange={onChange} />,
    )
    fireEvent.click(screen.getByRole('button', { name: /trocar o 2º bônus/i }))
    expect(onChange).toHaveBeenCalledWith({
      deformidade: { pericias: ['Furtividade'], tormentaPower: '' },
    })
  })

  it('com troca ativa mostra o slot de poder + aviso de −1 CAR quando escolhido', () => {
    render(
      <DeformidadeControls
        raceName="Lefou"
        choice={{
          deformidade: { pericias: ['Furtividade'], tormentaPower: 'dentes-afiados' },
        }}
        onChange={() => {}}
      />,
    )
    expect(screen.getByText('Dentes Afiados')).toBeInTheDocument()
    expect(screen.getByText(/−1 Carisma/)).toBeInTheDocument()
    // slot 1 mostra a perícia escolhida; slot 2 virou o slot de poder
    expect(screen.getByText('Furtividade')).toBeInTheDocument()
    expect(screen.queryByText(/Perícia \(\+2\)/)).not.toBeInTheDocument()
  })

  it('desfazer a troca remove o poder e volta o slot de perícia', () => {
    const onChange = vi.fn()
    render(
      <DeformidadeControls
        raceName="Lefou"
        choice={{
          deformidade: { pericias: ['Furtividade'], tormentaPower: 'antenas' },
        }}
        onChange={onChange}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /trocar o 2º bônus/i }))
    expect(onChange).toHaveBeenCalledWith({
      deformidade: { pericias: ['Furtividade'] },
    })
  })
})
