import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { AdminUser } from '@/shared/api/api'
import { PlayersPanel } from './players-panel'

afterEach(() => {
  document.body.innerHTML = ''
})

function makeUser(overrides: Partial<AdminUser> = {}): AdminUser {
  return {
    id: 2,
    email: 'jogador@t20.local',
    name: 'Jogador',
    isAdmin: false,
    campaigns: 0,
    characters: 0,
    createdAt: '2026-08-01T00:00:00.000Z',
    ...overrides,
  }
}

function setup(users: AdminUser[], currentUserId = 1) {
  const onDelete = vi.fn().mockResolvedValue(undefined)
  const onResetPassword = vi.fn()
  render(() => (
    <PlayersPanel
      users={users}
      currentUserId={currentUserId}
      onDelete={onDelete}
      onResetPassword={onResetPassword}
    />
  ))
  return { onDelete, onResetPassword, user: userEvent.setup() }
}

describe('PlayersPanel', () => {
  it('mostra o que cada conta tem', () => {
    setup([makeUser({ campaigns: 2, characters: 1 })])

    expect(screen.getByText('2 mesas · 1 ficha')).toBeInTheDocument()
  })

  it('marca quem administra', () => {
    setup([makeUser({ isAdmin: true, campaigns: 1, characters: 3 })])

    expect(screen.getByText('admin · 1 mesa · 3 fichas')).toBeInTheDocument()
  })

  // A própria linha aparece na lista, e o menu é o mesmo: apagar a si mesmo
  // levaria as próprias mesas para lugar nenhum (o servidor também recusa).
  it('não oferece apagar a própria conta', () => {
    setup([makeUser({ id: 1, email: 'dono@t20.local', name: 'Dono' })], 1)

    expect(screen.queryByRole('button', { name: /Apagar a conta/ })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Redefinir a senha/ })).toBeInTheDocument()
  })

  // O aviso tem de dizer o preço DESTA conta: um texto genérico não distingue
  // apagar uma conta vazia de apagar a do jogador que mestra duas crônicas.
  it('a confirmação diz o que se perde e para onde vão as mesas', async () => {
    const { user } = setup([makeUser({ campaigns: 2, characters: 3 })])

    await user.click(screen.getByRole('button', { name: 'Apagar a conta de Jogador' }))

    expect(await screen.findByText(/3 fichas vão junto/)).toBeInTheDocument()
    expect(screen.getByText(/2 mesas passam para você/)).toBeInTheDocument()
  })

  it('sem mesas, a confirmação não promete transferência nenhuma', async () => {
    const { user } = setup([makeUser({ campaigns: 0, characters: 1 })])

    await user.click(screen.getByRole('button', { name: 'Apagar a conta de Jogador' }))

    expect(await screen.findByText(/Não há mesas para transferir/)).toBeInTheDocument()
  })

  // Nada irreversível no primeiro clique.
  it('só apaga depois de confirmar', async () => {
    const { user, onDelete } = setup([makeUser()])

    await user.click(screen.getByRole('button', { name: 'Apagar a conta de Jogador' }))
    expect(onDelete).not.toHaveBeenCalled()

    await user.click(await screen.findByRole('button', { name: 'Apagar conta' }))
    expect(onDelete).toHaveBeenCalledOnce()
  })
})
