import { RouterProvider, createMemoryHistory, createRootRoute, createRouter } from '@tanstack/solid-router'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { MatchShell } from './match-shell'

/**
 * A partida carrega o escopo de tokens (`.scene-grimorio`) mas precisa PUBLICAR
 * o elemento: sem isso todo overlay porta para o `body` e a sessão ao vivo abre
 * diálogos claros do shadcn sobre a mesa escura (ALE-122).
 */
function renderShell() {
  const root = createRootRoute({
    component: () => (
      <MatchShell campaignId={1} title="Sessão 4">
        <ConfirmDialog
          title="Reiniciar o combate?"
          onConfirm={vi.fn()}
          trigger={(open) => (
            <button type="button" onClick={open}>
              Reiniciar
            </button>
          )}
        />
      </MatchShell>
    ),
  })
  const router = createRouter({ routeTree: root, history: createMemoryHistory() })
  const view = render(() => <RouterProvider router={router} />)
  return { ...view, user: userEvent.setup() }
}

describe('MatchShell', () => {
  it('abre os diálogos DENTRO da cena, para herdarem os tokens do grimório', async () => {
    const { container, user } = renderShell()

    await user.click(await screen.findByRole('button', { name: 'Reiniciar' }))

    const scene = container.querySelector('.scene-grimorio')
    expect(scene).not.toBeNull()
    expect(scene?.contains(screen.getByRole('dialog'))).toBe(true)
  })
})
