import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/shared/api/api'
import { RegisterForm } from './register-form'

const VALID = { email: 'novo@t20.local', password: 'segredo123', name: 'Novo' }

async function fill(fields: { email?: string; name?: string; password?: string; confirm?: string }) {
  const user = userEvent.setup()
  if (fields.email) await user.type(screen.getByLabelText('E-mail'), fields.email)
  if (fields.name) await user.type(screen.getByLabelText('Nome (opcional)'), fields.name)
  if (fields.password) await user.type(screen.getByLabelText('Senha'), fields.password)
  if (fields.confirm) await user.type(screen.getByLabelText('Confirmar senha'), fields.confirm)
  await user.click(screen.getByRole('button', { name: 'Criar conta' }))
}

describe('RegisterForm', () => {
  it('envia e-mail, senha e nome', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    render(() => <RegisterForm onSubmit={onSubmit} />)

    await fill({ ...VALID, confirm: VALID.password })

    await waitFor(() =>
      expect(onSubmit).toHaveBeenCalledWith({
        email: VALID.email,
        password: VALID.password,
        name: VALID.name,
      }),
    )
  })

  // Nome vazio é "sem nome", não string vazia — senão o backend guarda ''.
  it('omite o nome quando deixado em branco', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    render(() => <RegisterForm onSubmit={onSubmit} />)

    await fill({ email: VALID.email, password: VALID.password, confirm: VALID.password })

    await waitFor(() =>
      expect(onSubmit).toHaveBeenCalledWith({
        email: VALID.email,
        password: VALID.password,
        name: undefined,
      }),
    )
  })

  it('exige ao menos 8 caracteres na senha', async () => {
    const onSubmit = vi.fn()
    render(() => <RegisterForm onSubmit={onSubmit} />)

    await fill({ email: VALID.email, password: 'curta', confirm: 'curta' })

    expect(await screen.findByText('A senha precisa ter ao menos 8 caracteres')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  // A confirmação existe pra pegar o erro de digitação ANTES de virar uma
  // senha que o jogador não consegue reproduzir.
  it('barra senhas que não conferem', async () => {
    const onSubmit = vi.fn()
    render(() => <RegisterForm onSubmit={onSubmit} />)

    await fill({ email: VALID.email, password: VALID.password, confirm: 'outracoisa' })

    expect(await screen.findByText('As senhas não conferem')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('mostra o erro por campo que o servidor devolve', async () => {
    const onSubmit = vi
      .fn()
      .mockRejectedValue(new ApiError(409, 'Conflito', { email: ['E-mail já cadastrado'] }))
    render(() => <RegisterForm onSubmit={onSubmit} />)

    await fill({ ...VALID, confirm: VALID.password })

    expect(await screen.findByText('E-mail já cadastrado')).toBeInTheDocument()
  })
})
