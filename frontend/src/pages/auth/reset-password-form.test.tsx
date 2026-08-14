import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/shared/api/api'
import { ResetPasswordForm } from './reset-password-form'

async function fill(password: string, confirm: string) {
  const user = userEvent.setup()
  await user.type(screen.getByLabelText('Nova senha'), password)
  await user.type(screen.getByLabelText('Confirmar senha'), confirm)
  await user.click(screen.getByRole('button', { name: 'Salvar nova senha' }))
}

describe('ResetPasswordForm', () => {
  it('envia a senha escolhida', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    render(() => <ResetPasswordForm onSubmit={onSubmit} />)

    await fill('senha-nova-longa', 'senha-nova-longa')

    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith('senha-nova-longa'))
  })

  // O typo aqui é pior que no registro: o link já foi gasto, e a conta ficaria
  // com uma senha que o jogador não consegue reproduzir.
  it('não envia quando a confirmação não confere', async () => {
    const onSubmit = vi.fn()
    render(() => <ResetPasswordForm onSubmit={onSubmit} />)

    await fill('senha-nova-longa', 'senha-nova-longo')

    expect(await screen.findByText('As senhas não conferem')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('traduz a recusa do link em vez de mostrar a mensagem da API', async () => {
    const onSubmit = vi.fn().mockRejectedValue(new ApiError(403, 'Reset link is invalid or expired'))
    render(() => <ResetPasswordForm onSubmit={onSubmit} />)

    await fill('senha-nova-longa', 'senha-nova-longa')

    expect(await screen.findByText(/Este link não vale mais/)).toBeInTheDocument()
    expect(screen.queryByText('Reset link is invalid or expired')).not.toBeInTheDocument()
  })
})
