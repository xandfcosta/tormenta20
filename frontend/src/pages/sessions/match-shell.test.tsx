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

  /**
   * A saída é um LINK e não um botão, e isso é regra do porte para Solid: sem
   * `asChild`, um link com cara de botão é um `<a>` vestindo as classes do
   * botão (armadilha #6). Trocá-lo por `<button>` com `navigate()` parece
   * inofensivo e tira o meio-clique, o "abrir em nova aba" e o endereço que o
   * navegador mostra ao passar o mouse.
   *
   * Veio de um e2e (ALE-187) que ia até a sessão só para clicar na saída e
   * conferir a URL. O que ele media de único era o PAPEL do elemento, e papel
   * se afirma aqui — a navegação em si é o router, que tem os testes dele.
   */
  it('a saída da sessão é um link, e aponta para a campanha', async () => {
    renderShell()

    const saida = await screen.findByRole('link', { name: 'Sair da sessão' })
    expect(saida).toHaveAttribute('href', '/campaigns/1')
  })
})
