import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { DeleteCampaignDialog } from './delete-campaign-button'

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('DeleteCampaignDialog', () => {
  // Destruição irreversível não acontece num clique só: o gatilho abre a
  // confirmação, e é ela que chama o backend.
  it('o gatilho só abre a confirmação, não exclui', async () => {
    const onConfirm = vi.fn()
    render(() => <DeleteCampaignDialog campaignName="Snapshot Test" onConfirm={onConfirm} />)

    await userEvent.setup().click(screen.getByRole('button', { name: /Excluir campanha/ }))

    expect(await screen.findByRole('dialog')).toHaveAccessibleName('Excluir "Snapshot Test"?')
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('avisa o que vai junto antes de confirmar', async () => {
    render(() => <DeleteCampaignDialog campaignName="Snapshot Test" onConfirm={vi.fn()} />)
    await userEvent.setup().click(screen.getByRole('button', { name: /Excluir campanha/ }))

    expect(await screen.findByRole('dialog')).toHaveAccessibleDescription(/sessões e membros/)
  })

  it('confirmar exclui', async () => {
    const onConfirm = vi.fn().mockResolvedValue(undefined)
    const user = userEvent.setup()
    render(() => <DeleteCampaignDialog campaignName="Snapshot Test" onConfirm={onConfirm} />)
    await user.click(screen.getByRole('button', { name: /Excluir campanha/ }))

    await user.click(await screen.findByRole('button', { name: 'Excluir' }))

    expect(onConfirm).toHaveBeenCalledOnce()
  })

  it('cancelar fecha sem excluir', async () => {
    const onConfirm = vi.fn()
    const user = userEvent.setup()
    render(() => <DeleteCampaignDialog campaignName="Snapshot Test" onConfirm={onConfirm} />)
    await user.click(screen.getByRole('button', { name: /Excluir campanha/ }))

    await user.click(await screen.findByRole('button', { name: 'Cancelar' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(onConfirm).not.toHaveBeenCalled()
  })
})
