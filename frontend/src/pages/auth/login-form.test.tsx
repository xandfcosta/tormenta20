import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/shared/api/api'
import { LoginForm } from './login-form'

const CREDENTIALS = { email: 'mestre@t20.local', password: 'mestre123456' }

async function fillAndSubmit(email: string, password: string) {
  const user = userEvent.setup()
  await user.type(screen.getByLabelText('E-mail'), email)
  await user.type(screen.getByLabelText('Senha'), password)
  await user.click(screen.getByRole('button', { name: 'Entrar' }))
}

describe('LoginForm', () => {
  // 'rende os campos e o botão' saiu na ALE-187: o caso abaixo DIGITA nesses
  // mesmos campos e clica nesse mesmo botão, então ele já falharia se algum
  // sumisse. Afirmar presença antes de usar é dizer duas vezes.

  it('envia as credenciais quando o formulário é válido', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    render(() => <LoginForm onSubmit={onSubmit} />)

    await fillAndSubmit(CREDENTIALS.email, CREDENTIALS.password)

    await waitFor(() => expect(onSubmit).toHaveBeenCalledWith(CREDENTIALS))
  })

  it('barra e-mail inválido no cliente, sem chamar a API', async () => {
    const onSubmit = vi.fn()
    render(() => <LoginForm onSubmit={onSubmit} />)

    await fillAndSubmit('nao-e-email', 'mestre123456')

    expect(await screen.findByText('E-mail inválido')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('exige senha', async () => {
    const onSubmit = vi.fn()
    render(() => <LoginForm onSubmit={onSubmit} />)

    const user = userEvent.setup()
    await user.type(screen.getByLabelText('E-mail'), CREDENTIALS.email)
    await user.click(screen.getByRole('button', { name: 'Entrar' }))

    expect(await screen.findByText('Informe sua senha')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('mostra a mensagem do servidor quando o login é recusado', async () => {
    const onSubmit = vi.fn().mockRejectedValue(new ApiError(401, 'Credenciais inválidas'))
    render(() => <LoginForm onSubmit={onSubmit} />)

    await fillAndSubmit(CREDENTIALS.email, 'senha-errada')

    expect(await screen.findByText('Credenciais inválidas')).toBeInTheDocument()
  })

  it('mostra o erro por campo que o servidor devolve', async () => {
    const onSubmit = vi
      .fn()
      .mockRejectedValue(new ApiError(400, 'Dados inválidos', { email: ['E-mail não cadastrado'] }))
    render(() => <LoginForm onSubmit={onSubmit} />)

    await fillAndSubmit(CREDENTIALS.email, CREDENTIALS.password)

    expect(await screen.findByText('E-mail não cadastrado')).toBeInTheDocument()
  })

  it('traduz uma falha inesperada em vez de vazar o erro cru', async () => {
    const onSubmit = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'))
    render(() => <LoginForm onSubmit={onSubmit} />)

    await fillAndSubmit(CREDENTIALS.email, CREDENTIALS.password)

    expect(await screen.findByText('Erro inesperado. Tente novamente.')).toBeInTheDocument()
  })

  it('desabilita o botão enquanto envia, pra não duplicar o login', async () => {
    let release = () => {}
    const onSubmit = vi.fn().mockImplementation(() => new Promise<void>((r) => (release = r)))
    render(() => <LoginForm onSubmit={onSubmit} />)

    await fillAndSubmit(CREDENTIALS.email, CREDENTIALS.password)

    const button = screen.getByRole('button', { name: 'Entrando…' })
    await waitFor(() => expect(button).toBeDisabled())
    release()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Entrar' })).toBeEnabled())
  })
})
