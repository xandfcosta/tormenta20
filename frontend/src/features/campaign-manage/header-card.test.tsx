import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, type Campaign } from '@/shared/api/api'
import { CampaignEditForm } from './header-card'

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

const campaign: Campaign = {
  id: 1,
  ownerId: 1,
  name: 'Snapshot Test',
  description: 'Uma mesa-vitrine.',
  createdAt: '2026-01-01T00:00:00.000Z',
  updatedAt: '2026-01-01T00:00:00.000Z',
  role: 'gm',
}

function renderForm(onSave = vi.fn().mockResolvedValue(undefined), onCancel = vi.fn()) {
  render(() => <CampaignEditForm campaign={campaign} onSave={onSave} onCancel={onCancel} />)
  return { onSave, onCancel, user: userEvent.setup() }
}

describe('CampaignEditForm', () => {
  it('abre com os valores atuais da campanha', () => {
    renderForm()
    expect(screen.getByLabelText('Nome')).toHaveValue('Snapshot Test')
    expect(screen.getByLabelText('Descrição')).toHaveValue('Uma mesa-vitrine.')
  })

  it('salva nome e descrição editados', async () => {
    const { onSave, user } = renderForm()

    await user.clear(screen.getByLabelText('Nome'))
    await user.type(screen.getByLabelText('Nome'), 'Mesa do Beco')
    await user.click(screen.getByRole('button', { name: 'Salvar' }))

    expect(onSave).toHaveBeenCalledWith({
      name: 'Mesa do Beco',
      description: 'Uma mesa-vitrine.',
    })
  })

  // Um nome só de espaços passaria num min(1) ingênuo e deixaria a campanha sem
  // título na lista.
  it('recusa nome vazio ou só de espaços, sem chamar o backend', async () => {
    const { onSave, user } = renderForm()

    await user.clear(screen.getByLabelText('Nome'))
    await user.type(screen.getByLabelText('Nome'), '   ')
    await user.click(screen.getByRole('button', { name: 'Salvar' }))

    expect(await screen.findByText('Nome é obrigatório')).toBeInTheDocument()
    expect(onSave).not.toHaveBeenCalled()
  })

  it('mostra o erro do backend sem perder o que foi digitado', async () => {
    const onSave = vi.fn().mockRejectedValue(new ApiError(409, 'Já existe uma campanha assim'))
    const { user } = renderForm(onSave)

    await user.click(screen.getByRole('button', { name: 'Salvar' }))

    expect(await screen.findByText('Já existe uma campanha assim')).toBeInTheDocument()
    expect(screen.getByLabelText('Nome')).toHaveValue('Snapshot Test')
  })

  it('espalha os erros por campo que o backend atribuiu', async () => {
    const onSave = vi
      .fn()
      .mockRejectedValue(new ApiError(422, 'inválido', { name: ['Nome muito longo'] }))
    const { user } = renderForm(onSave)

    await user.click(screen.getByRole('button', { name: 'Salvar' }))

    expect(await screen.findByText('Nome muito longo')).toBeInTheDocument()
    expect(screen.getByLabelText('Nome')).toHaveAttribute('aria-invalid', 'true')
  })

  it('cancelar devolve o controle ao chamador sem salvar', async () => {
    const { onSave, onCancel, user } = renderForm()

    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(onCancel).toHaveBeenCalledOnce()
    expect(onSave).not.toHaveBeenCalled()
  })
})
