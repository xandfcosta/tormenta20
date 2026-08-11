import { render, screen, waitFor, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/shared/api/api'
import { InviteDialog } from './invite-button'

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

/** Opens the dialog and hands back the driver, since every case starts there. */
async function openInvite(
  onRotate: () => Promise<string>,
  onCopy: (text: string) => Promise<void> = vi.fn(),
) {
  const user = userEvent.setup()
  render(() => <InviteDialog onRotate={onRotate} onCopy={onCopy} />)
  await user.click(screen.getByRole('button', { name: /Convite/ }))
  return user
}

describe('InviteDialog', () => {
  it('abre sem link, convidando a gerar o primeiro', async () => {
    await openInvite(vi.fn())

    expect(await screen.findByText(/Nenhum link gerado/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Gerar link' })).toBeInTheDocument()
    expect(screen.queryByLabelText('Link de convite')).not.toBeInTheDocument()
  })

  it('gera o link e mostra a URL que o jogador vai abrir', async () => {
    const user = await openInvite(async () => 'abc123')

    await user.click(await screen.findByRole('button', { name: 'Gerar link' }))

    const field = (await screen.findByLabelText('Link de convite')) as HTMLInputElement
    expect(field.value).toBe(`${window.location.origin}/join/abc123`)
  })

  it('copia exatamente a URL mostrada', async () => {
    const onCopy = vi.fn().mockResolvedValue(undefined)
    const user = await openInvite(async () => 'abc123', onCopy)
    await user.click(await screen.findByRole('button', { name: 'Gerar link' }))

    await user.click(await screen.findByRole('button', { name: 'Copiar link' }))

    expect(onCopy).toHaveBeenCalledWith(`${window.location.origin}/join/abc123`)
  })

  // Com um token na mão a ação vira ROTACIONAR: o texto tem que dizer isso,
  // senão o mestre reemite sem perceber que matou o link já compartilhado.
  it('com token na mão, a ação passa a ser rotacionar', async () => {
    const onRotate = vi.fn().mockResolvedValueOnce('abc123').mockResolvedValueOnce('def456')
    const user = await openInvite(onRotate)
    await user.click(await screen.findByRole('button', { name: 'Gerar link' }))

    await user.click(await screen.findByRole('button', { name: 'Rotacionar convite' }))

    const field = (await screen.findByLabelText('Link de convite')) as HTMLInputElement
    expect(field.value).toBe(`${window.location.origin}/join/def456`)
    expect(onRotate).toHaveBeenCalledTimes(2)
  })

  it('mostra a falha do backend em vez de um link quebrado', async () => {
    const onRotate = vi.fn().mockRejectedValue(new ApiError(403, 'Só o mestre convida'))
    const user = await openInvite(onRotate)

    await user.click(await screen.findByRole('button', { name: 'Gerar link' }))

    expect(await screen.findByText('Só o mestre convida')).toBeInTheDocument()
    expect(screen.queryByLabelText('Link de convite')).not.toBeInTheDocument()
  })

  // O token nunca é cacheado: a fonte da verdade é o banco e rotacionar
  // invalida o valor anterior, então reabrir não pode ressuscitar o antigo.
  it('fechar esquece o token', async () => {
    const user = await openInvite(async () => 'abc123')
    await user.click(await screen.findByRole('button', { name: 'Gerar link' }))
    await screen.findByLabelText('Link de convite')

    // O X do Kobalte também se chama "Fechar" (aria-label pt-BR do kit), então
    // a busca é feita dentro do rodapé para dizer qual controle está sendo usado.
    const footer = document.querySelector('[data-slot="dialog-footer"]') as HTMLElement
    await user.click(within(footer).getByRole('button', { name: 'Fechar' }))
    // Kobalte só devolve o aria-hidden do resto da página depois de fechar de
    // fato — antes disso o gatilho não existe para a árvore de acessibilidade.
    await waitFor(() => expect(screen.getByRole('button', { name: /Convite/ })).toBeVisible())
    await user.click(screen.getByRole('button', { name: /Convite/ }))

    expect(await screen.findByText(/Nenhum link gerado/)).toBeInTheDocument()
  })
})
