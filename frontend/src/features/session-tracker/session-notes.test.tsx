import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/shared/api/api'
import type { Session } from '@/shared/api/api'
import { SessionNotes } from './session-notes'

function makeSession(notes: string | null = null): Session {
  return {
    id: 7,
    campaignId: 3,
    title: 'A ponte',
    sessionNumber: 1,
    notes,
    status: 'active',
    startedAt: null,
    endedAt: null,
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
  }
}

function renderNotes(notes: string | null = null) {
  const update = vi.spyOn(api.sessions, 'update').mockResolvedValue(makeSession(notes))
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
  const view = render(() => (
    <QueryClientProvider client={client}>
      <SessionNotes campaignId={3} session={makeSession(notes)} />
    </QueryClientProvider>
  ))
  return { ...view, user, update }
}

describe('SessionNotes', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('mostra o markdown ao lado enquanto o mestre escreve', async () => {
    const { user } = renderNotes()

    await user.type(screen.getByLabelText('Notas da sessão'), '# Cena 1\n- O ogro **fugiu**')

    expect(screen.getByRole('heading', { name: 'Cena 1' })).toBeInTheDocument()
    expect(screen.getByRole('listitem')).toHaveTextContent('O ogro fugiu')
    expect(screen.getByText('fugiu').tagName).toBe('STRONG')
  })

  it('salva sozinha depois da pausa, sem botão de salvar', async () => {
    const { user, update } = renderNotes()

    await user.type(screen.getByLabelText('Notas da sessão'), 'o ogro fugiu')
    expect(screen.queryByRole('button', { name: 'Salvar' })).not.toBeInTheDocument()
    expect(update).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1200)

    expect(update).toHaveBeenCalledWith(3, 7, { notes: 'o ogro fugiu' })
  })

  // O Tabs desmonta a aba inativa: sem descarga na saída, o mestre que escreve e
  // troca para o Bestiário perde o que digitou desde a última pausa.
  it('sair da aba salva o que ainda não tinha sido salvo', async () => {
    const { user, update, unmount } = renderNotes()

    await user.type(screen.getByLabelText('Notas da sessão'), 'tesouro: 300 TO')
    unmount()

    expect(update).toHaveBeenCalledWith(3, 7, { notes: 'tesouro: 300 TO' })
  })
})
