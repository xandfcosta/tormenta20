import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/shared/api/api'
import type { Session } from '@/shared/api/api'
import { FakeStorage } from '@/shared/test/fake-storage'
import { NOTES_VIEW_KEY } from './notes-view'
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

function renderNotes(notes: string | null = null, storage: Storage = new FakeStorage()) {
  const update = vi.spyOn(api.sessions, 'update').mockResolvedValue(makeSession(notes))
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
  const view = render(() => (
    <QueryClientProvider client={client}>
      <SessionNotes campaignId={3} session={makeSession(notes)} storage={storage} />
    </QueryClientProvider>
  ))
  return { ...view, user, update, storage }
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

  // Numa nota de mesa a quebra de linha é intencional: o markdown padrão
  // juntaria as três num parágrafo corrido.
  it('cada linha digitada vira uma linha na prévia', async () => {
    const { user, container } = renderNotes()

    await user.type(screen.getByLabelText('Notas da sessão'), 'ogro fugiu\ngoblins ficaram\nchoveu')

    expect(container.querySelectorAll('br')).toHaveLength(2)
  })

  // Marcar a tarefa reescreve a NOTA — o estado do checkbox não pode morar ao
  // lado do texto, senão some quando a sessão é reaberta.
  it('marcar a tarefa reescreve a linha e salva na hora', async () => {
    const { user, update } = renderNotes('# Cena\n- [ ] dar XP')

    await user.click(screen.getByRole('checkbox', { name: 'dar XP' }))

    expect(update).toHaveBeenCalledWith(3, 7, { notes: '# Cena\n- [x] dar XP' })
    expect(screen.getByLabelText('Notas da sessão')).toHaveValue('# Cena\n- [x] dar XP')
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

/**
 * Os modos de visualização (ALE-139).
 *
 * Lado a lado FIXO desperdiçava a tela nos dois extremos: com uma nota de duas
 * linhas, duas colunas quase vazias; com uma nota longa, escrever numa coluna
 * estreita. O que se afirma aqui é o que o mestre VÊ e o que sobrevive ao
 * recarregar — a regra pura mora em `notes-view.ts`.
 */
describe('SessionNotes — modos de visualização', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('em Ler, o texto cru sai da tela e o resultado fica', async () => {
    const { user } = renderNotes('# Cena 1')

    await user.click(screen.getByRole('button', { name: 'Ler' }))

    expect(screen.queryByLabelText('Notas da sessão')).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Cena 1' })).toBeInTheDocument()
  })

  it('em Escrever, o resultado sai e o editor ocupa tudo', async () => {
    const { user } = renderNotes('# Cena 1')

    await user.click(screen.getByRole('button', { name: 'Escrever' }))

    expect(screen.getByLabelText('Notas da sessão')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Cena 1' })).not.toBeInTheDocument()
  })

  // É preferência de TRABALHO, não estado da sessão: o mestre não vai
  // reescolher a cada aba que abre.
  it('a escolha gruda', async () => {
    const { user, storage } = renderNotes()

    await user.click(screen.getByRole('button', { name: 'Ler' }))

    expect(storage.getItem(NOTES_VIEW_KEY)).toBe('ler')
  })

  it('e volta escolhida na próxima montagem', () => {
    const storage = new FakeStorage()
    storage.setItem(NOTES_VIEW_KEY, 'ler')

    renderNotes('# Cena 1', storage)

    expect(screen.queryByLabelText('Notas da sessão')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Ler' })).toHaveAttribute('aria-pressed', 'true')
  })

  /** O padrão continua o de hoje: ninguém perde o arranjo que já usava. */
  it('sem nada guardado, abre no lado a lado', () => {
    renderNotes('# Cena 1')

    expect(screen.getByLabelText('Notas da sessão')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Cena 1' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lado a lado' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })
})
