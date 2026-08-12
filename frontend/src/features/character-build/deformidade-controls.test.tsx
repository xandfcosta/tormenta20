import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { RaceChoice } from './grant-helpers'
import { DeformidadeControls } from './deformidade-controls'

function renderControls(raceName: string, choice: RaceChoice = {}) {
  const onChange = vi.fn()
  render(() => (
    <DeformidadeControls raceName={raceName} choice={choice} onChange={onChange} />
  ))
  return { onChange }
}

describe('DeformidadeControls', () => {
  it('só existe para a raça que tem a habilidade', () => {
    renderControls('Anão')

    expect(screen.queryByText(/Deformidade/)).not.toBeInTheDocument()
  })

  it('oferece os dois bônus de perícia do Lefou', () => {
    renderControls('Lefou')

    expect(screen.getByLabelText('Bônus de perícia 1')).toBeInTheDocument()
    expect(screen.getByLabelText('Bônus de perícia 2')).toBeInTheDocument()
  })

  it('a troca substitui o SEGUNDO bônus — só um pode virar poder (p23)', async () => {
    const { onChange } = renderControls('Lefou', {
      deformidade: { pericias: ['Furtividade', 'Luta'] },
    })

    await userEvent.click(screen.getByRole('button', { name: /Trocar o 2º bônus/ }))

    expect(onChange).toHaveBeenCalledWith({
      deformidade: { pericias: ['Furtividade'], tormentaPower: '' },
    })
  })

  it('desfazer a troca devolve o slot de perícia', async () => {
    const { onChange } = renderControls('Lefou', {
      deformidade: { pericias: ['Furtividade'], tormentaPower: 'dentes-afiados' },
    })

    expect(screen.queryByLabelText('Bônus de perícia 2')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /Trocar o 2º bônus/ }))

    expect(onChange).toHaveBeenCalledWith({ deformidade: { pericias: ['Furtividade'] } })
  })

  it('avisa da perda de Carisma quando o poder é real (p136)', () => {
    renderControls('Lefou', {
      deformidade: { pericias: ['Furtividade'], tormentaPower: 'dentes-afiados' },
    })

    expect(screen.getByText(/−1 Carisma/)).toBeInTheDocument()
  })

  it('não cobra Carisma enquanto o poder não foi escolhido', () => {
    renderControls('Lefou', {
      deformidade: { pericias: ['Furtividade'], tormentaPower: '' },
    })

    expect(screen.queryByText(/−1 Carisma/)).not.toBeInTheDocument()
  })
})
