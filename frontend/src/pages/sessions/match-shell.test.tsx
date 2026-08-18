import { RouterProvider, createMemoryHistory, createRootRoute, createRouter } from '@tanstack/solid-router'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { FakeStorage } from '@/shared/test/fake-storage'
import { createSfx, createSfxToggle } from '@/shared/lib/sfx'
import type { SfxName, SfxPlayer } from '@/shared/lib/sfx-player'
import { createUiStore } from '@/shared/stores/ui-store'
import { MatchShell } from './match-shell'

/** Named fake for the audio backend — no Web Audio in the tests. */
class FakeSfxPlayer implements SfxPlayer {
  readonly played: SfxName[] = []
  play(name: SfxName): void {
    this.played.push(name)
  }
}

/**
 * A partida carrega o escopo de tokens (`.scene-grimorio`) mas precisa PUBLICAR
 * o elemento: sem isso todo overlay porta para o `body` e a sessão ao vivo abre
 * diálogos claros do shadcn sobre a mesa escura (ALE-122).
 */
function renderShell() {
  const player = new FakeSfxPlayer()
  const ui = createUiStore(new FakeStorage())
  const toggleSfx = createSfxToggle(ui, createSfx(ui, () => player))
  const root = createRootRoute({
    component: () => (
      <MatchShell
        campaignId={1}
        title="Sessão 4"
        sfxEnabled={ui.sfx()}
        onToggleSfx={toggleSfx}
      >
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
  return { ...view, ui, player, user: userEvent.setup() }
}

describe('MatchShell', () => {
  it('abre os diálogos DENTRO da cena, para herdarem os tokens do grimório', async () => {
    const { container, user } = renderShell()

    await user.click(await screen.findByRole('button', { name: 'Reiniciar' }))

    const scene = container.querySelector('.scene-grimorio')
    expect(scene).not.toBeNull()
    expect(scene?.contains(screen.getByRole('dialog'))).toBe(true)
  })

  // O controle de som só existia no rodapé do Hub: para mudar de ideia sobre
  // áudio numa sessão AO VIVO, o jogador tinha de sair da mesa (ALE-165).
  it('liga o som sem sair da mesa, e o próprio clique confirma em áudio', async () => {
    const { ui, player, user } = renderShell()

    await user.click(await screen.findByRole('button', { name: 'Ligar o som' }))

    expect(ui.sfx()).toBe(true)
    expect(player.played).toEqual(['select'])
    expect(await screen.findByRole('button', { name: 'Desligar o som' })).toBeTruthy()
  })
})
