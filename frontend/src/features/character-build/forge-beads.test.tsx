import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ForgeBeads } from './forge-beads'

const renderBeads = (props?: Partial<Parameters<typeof ForgeBeads>[0]>) => {
  const onJump = vi.fn()
  render(() => <ForgeBeads current="poderes" reachable={4} onJump={onJump} {...props} />)
  return { onJump }
}

describe('ForgeBeads', () => {
  it('anuncia a posição em linguagem simples, não em algarismo romano', () => {
    renderBeads()

    // "III" é decoração; o leitor de tela recebe a frase.
    expect(screen.getByText('Passo 3 de 9 · Poderes')).toBeInTheDocument()
  })

  it('mostra o algarismo romano do passo atual', () => {
    renderBeads({ current: 'raca' })

    expect(screen.getByText('I')).toBeInTheDocument()
  })

  it('marca a conta atual para o leitor de tela', () => {
    renderBeads()

    expect(screen.getByRole('button', { name: /Poderes/ })).toHaveAttribute(
      'aria-current',
      'step',
    )
  })

  it('anda para um passo já alcançado', async () => {
    const { onJump } = renderBeads()

    await userEvent.click(screen.getByRole('button', { name: /Classe/ }))

    expect(onJump).toHaveBeenCalledWith('classe')
  })

  it('não anda para um passo ainda trancado', async () => {
    const { onJump } = renderBeads({ reachable: 2 })

    const locked = screen.getByRole('button', { name: /Identidade/ })
    expect(locked).toBeDisabled()
    await userEvent.click(locked)

    expect(onJump).not.toHaveBeenCalled()
  })

  it('o passo atual não é um alvo de clique', async () => {
    const { onJump } = renderBeads()

    await userEvent.click(screen.getByRole('button', { name: /Poderes/ }))

    expect(onJump).not.toHaveBeenCalled()
  })
})
