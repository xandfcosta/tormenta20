import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/shared/api/api'
import { InvitePlayerDialog } from './invite-player-dialog'

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

/** The dialog opens from the Hub's quick menu, so it starts already open here. */
function openDialog(
  onCreate: () => Promise<string>,
  onCopy: (text: string) => Promise<void> = vi.fn(),
) {
  render(() => (
    <InvitePlayerDialog open onOpenChange={vi.fn()} onCreate={onCreate} onCopy={onCopy} />
  ))
  return userEvent.setup()
}

describe('InvitePlayerDialog', () => {
  it('abre sem link, convidando a gerar o primeiro', async () => {
    openDialog(vi.fn())

    expect(await screen.findByText(/Nenhum link gerado/)).toBeInTheDocument()
    expect(screen.queryByLabelText('Link de convite')).not.toBeInTheDocument()
  })

  // A URL É o fluxo: o jogador abre e cai no registro com o convite preenchido.
  it('gera o link de registro que o jogador vai abrir', async () => {
    const user = openDialog(async () => 'abc123')

    await user.click(await screen.findByRole('button', { name: 'Gerar convite' }))

    const field = (await screen.findByLabelText('Link de convite')) as HTMLInputElement
    expect(field.value).toBe(`${window.location.origin}/register?convite=abc123`)
  })

  it('copia exatamente a URL mostrada', async () => {
    const onCopy = vi.fn().mockResolvedValue(undefined)
    const user = openDialog(async () => 'abc123', onCopy)
    await user.click(await screen.findByRole('button', { name: 'Gerar convite' }))

    await user.click(await screen.findByRole('button', { name: 'Copiar link' }))

    expect(onCopy).toHaveBeenCalledWith(`${window.location.origin}/register?convite=abc123`)
  })

  // Um token com caractere de URL (o gerador usa base64url, mas o contrato é do
  // servidor) não pode quebrar o link ao ser colado na query string.
  it('escapa o token na URL', async () => {
    const user = openDialog(async () => 'a b+c')

    await user.click(await screen.findByRole('button', { name: 'Gerar convite' }))

    const field = (await screen.findByLabelText('Link de convite')) as HTMLInputElement
    expect(field.value).toBe(`${window.location.origin}/register?convite=a%20b%2Bc`)
  })

  it('mostra a recusa do servidor em vez de um link inventado', async () => {
    const user = openDialog(() => Promise.reject(new ApiError(403, 'Admin only')))

    await user.click(await screen.findByRole('button', { name: 'Gerar convite' }))

    expect(await screen.findByText('Admin only')).toBeInTheDocument()
    expect(screen.queryByLabelText('Link de convite')).not.toBeInTheDocument()
  })
})
