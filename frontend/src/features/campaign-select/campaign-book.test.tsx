import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Campaign } from '@/shared/api/api'
import { CampaignBook } from './campaign-book'

function campaign(id: number, name = `Crônica ${id}`): Campaign {
  return {
    id,
    ownerId: 1,
    name,
    description: null,
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
  }
}

/** Phone/reduced-motion by default: no leaf animation to wait on. */
function mockViewport({ wide = false, reduced = true } = {}) {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: media.includes('prefers-reduced-motion') ? reduced : wide,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

beforeEach(() => mockViewport())
afterEach(() => vi.restoreAllMocks())

describe('CampaignBook', () => {
  it('sem sessão ao vivo, oferece abrir a crônica', () => {
    render(
      <CampaignBook
        campaign={campaign(1)}
        isLive={false}
        orderIds={[1]}
        onOpen={vi.fn()}
        onResume={vi.fn()}
      />,
    )
    expect(screen.getByRole('button', { name: /Abrir crônica/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Continuar a sessão/ })).not.toBeInTheDocument()
  })

  /**
   * Regressão ALE-78: o mapa de sessões ao vivo é um fan-out separado, então
   * chega DEPOIS da lista — o alvo muda só no `isLive`, com o mesmo id. A
   * máquina de virada deduplicava por id e engolia essa atualização, deixando o
   * livro em "Abrir crônica" enquanto o rail já mostrava a brasa.
   */
  it('reflete a sessão ao vivo quando ela chega depois (mesma crônica)', () => {
    const view = render(
      <CampaignBook
        campaign={campaign(1)}
        isLive={false}
        orderIds={[1]}
        onOpen={vi.fn()}
        onResume={vi.fn()}
      />,
    )

    view.rerender(
      <CampaignBook
        campaign={campaign(1)}
        isLive={true}
        orderIds={[1]}
        onOpen={vi.fn()}
        onResume={vi.fn()}
      />,
    )

    expect(screen.getByRole('button', { name: /Continuar a sessão/ })).toBeInTheDocument()
    expect(screen.getByText('Sessão ao vivo')).toBeInTheDocument()
  })

  it('vale também no spread do desktop, com a folha virando', () => {
    mockViewport({ wide: true, reduced: false })
    const view = render(
      <CampaignBook
        campaign={campaign(1)}
        isLive={false}
        orderIds={[1]}
        onOpen={vi.fn()}
        onResume={vi.fn()}
      />,
    )

    view.rerender(
      <CampaignBook
        campaign={campaign(1)}
        isLive={true}
        orderIds={[1]}
        onOpen={vi.fn()}
        onResume={vi.fn()}
      />,
    )

    expect(screen.getByRole('button', { name: /Continuar a sessão/ })).toBeInTheDocument()
  })

  it('trocar de crônica troca o conteúdo do livro', () => {
    const view = render(
      <CampaignBook
        campaign={campaign(1, 'A Queda de Tauron')}
        isLive={false}
        orderIds={[1, 2]}
        onOpen={vi.fn()}
        onResume={vi.fn()}
      />,
    )
    expect(screen.getByText('A Queda de Tauron')).toBeInTheDocument()

    view.rerender(
      <CampaignBook
        campaign={campaign(2, 'Segredos de Wynlla')}
        isLive={false}
        orderIds={[1, 2]}
        onOpen={vi.fn()}
        onResume={vi.fn()}
      />,
    )

    expect(screen.getByText('Segredos de Wynlla')).toBeInTheDocument()
  })
})
