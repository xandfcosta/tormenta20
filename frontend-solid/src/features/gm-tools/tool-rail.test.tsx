import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { GM_TOOLS } from './gm-tools'
import { ToolRail } from './tool-rail'

const renderRail = (current: 'bestiario' | 'encontros' = 'bestiario') => {
  const onPick = vi.fn()
  render(() => <ToolRail current={current} onPick={onPick} />)
  return { onPick }
}

describe('ToolRail', () => {
  it('oferece todas as ferramentas', () => {
    renderRail()

    const rail = screen.getByRole('navigation', { name: /ferramentas do mestre/i })
    expect(rail).toBeInTheDocument()
    for (const tool of GM_TOOLS) {
      expect(screen.getByRole('button', { name: tool.label })).toBeInTheDocument()
    }
  })

  it('marca a ferramenta em cena para o leitor de tela', () => {
    renderRail('encontros')

    expect(screen.getByRole('button', { name: 'Encontros' })).toHaveAttribute(
      'aria-current',
      'page',
    )
    expect(screen.getByRole('button', { name: 'Bestiário' })).not.toHaveAttribute('aria-current')
  })

  it('troca de ferramenta ao clicar', async () => {
    const { onPick } = renderRail()

    await userEvent.click(screen.getByRole('button', { name: 'Catálogos' }))

    expect(onPick).toHaveBeenCalledWith('catalogos')
  })

  // Uma lista só, não duas árvores: no telefone o mesmo trilho vira barra
  // horizontal por classe. Duas árvores é como um trilho e seu gêmeo divergem.
  it('renderiza UMA lista, não uma por formato', () => {
    renderRail()

    expect(screen.getAllByRole('navigation')).toHaveLength(1)
    expect(screen.getAllByRole('button', { name: 'Bestiário' })).toHaveLength(1)
  })
})
