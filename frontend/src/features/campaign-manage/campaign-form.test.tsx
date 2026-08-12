import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { campaignSchema } from '@/entities/campaign/campaign-schema'
import { ApiError } from '@/shared/api/api'
import { CampaignForm } from './campaign-form'

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('campaignSchema', () => {
  // A regra é a mesma nas duas telas; um nome só de espaços passaria num
  // min(1) ingênuo e deixaria a crônica sem título na estante.
  it('recusa nome vazio ou só de espaços', () => {
    expect(campaignSchema.safeParse({ name: '   ', description: '' }).success).toBe(false)
    expect(campaignSchema.safeParse({ name: '', description: '' }).success).toBe(false)
  })

  it('apara o nome antes de entregar', () => {
    const parsed = campaignSchema.parse({ name: '  Mesa do Beco  ', description: '' })
    expect(parsed.name).toBe('Mesa do Beco')
  })

  it('a descrição pode ser vazia, mas não infinita', () => {
    expect(campaignSchema.safeParse({ name: 'Mesa', description: '' }).success).toBe(true)
    expect(campaignSchema.safeParse({ name: 'Mesa', description: 'x'.repeat(2001) }).success).toBe(
      false,
    )
  })
})

function renderForm(onSubmit = vi.fn().mockResolvedValue(undefined), onCancel = vi.fn()) {
  render(() => (
    <CampaignForm
      submitLabel="Abrir crônica"
      pendingLabel="Abrindo…"
      onSubmit={onSubmit}
      onCancel={onCancel}
    />
  ))
  return { onSubmit, onCancel, user: userEvent.setup() }
}

describe('CampaignForm', () => {
  it('começa em branco quando não recebe valores iniciais', () => {
    renderForm()
    expect(screen.getByLabelText('Nome')).toHaveValue('')
    expect(screen.getByLabelText('Descrição')).toHaveValue('')
  })

  it('entrega os valores aparados ao chamador', async () => {
    const { onSubmit, user } = renderForm()

    await user.type(screen.getByLabelText('Nome'), '  Mesa do Beco  ')
    await user.type(screen.getByLabelText('Descrição'), 'Um beco sem saída.')
    await user.click(screen.getByRole('button', { name: 'Abrir crônica' }))

    expect(onSubmit).toHaveBeenCalledWith({
      name: 'Mesa do Beco',
      description: 'Um beco sem saída.',
    })
  })

  it('barra o envio inválido antes de chamar o backend', async () => {
    const { onSubmit, user } = renderForm()

    await user.click(screen.getByRole('button', { name: 'Abrir crônica' }))

    expect(await screen.findByText('Nome é obrigatório')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('mostra a falha do backend sem perder o que foi digitado', async () => {
    const onSubmit = vi.fn().mockRejectedValue(new ApiError(409, 'Já existe uma campanha assim'))
    const { user } = renderForm(onSubmit)
    await user.type(screen.getByLabelText('Nome'), 'Mesa do Beco')

    await user.click(screen.getByRole('button', { name: 'Abrir crônica' }))

    expect(await screen.findByText('Já existe uma campanha assim')).toBeInTheDocument()
    expect(screen.getByLabelText('Nome')).toHaveValue('Mesa do Beco')
  })

  it('usa o rótulo de espera enquanto o backend responde', async () => {
    let release = () => {}
    const onSubmit = vi.fn(() => new Promise<void>((resolve) => (release = resolve)))
    const { user } = renderForm(onSubmit)
    await user.type(screen.getByLabelText('Nome'), 'Mesa do Beco')

    await user.click(screen.getByRole('button', { name: 'Abrir crônica' }))

    expect(screen.getByRole('button', { name: 'Abrindo…' })).toBeDisabled()
    release()
  })

  it('cancelar devolve o controle sem enviar', async () => {
    const { onSubmit, onCancel, user } = renderForm()

    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(onCancel).toHaveBeenCalledOnce()
    expect(onSubmit).not.toHaveBeenCalled()
  })
})
