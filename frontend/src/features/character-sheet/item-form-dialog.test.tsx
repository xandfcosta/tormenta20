import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Button } from '@/shared/ui/button'
import { ItemFormDialog } from './item-form-dialog'
import { ItemRefused } from './item-mutations'

afterEach(() => {
  document.body.innerHTML = ''
})

function renderDialog(
  onSubmit: (input: unknown) => Promise<void>,
  initial?: { name: string; quantity: number; slots: number },
) {
  render(() => (
    <ItemFormDialog
      title="Novo item"
      submitLabel="Adicionar"
      initial={initial}
      onSubmit={onSubmit as never}
      trigger={(open) => (
        <Button type="button" onClick={open}>
          Custom
        </Button>
      )}
    />
  ))
  return userEvent.setup()
}

describe('ItemFormDialog', () => {
  it('envia nome aparado com quantidade e espaços', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    const user = renderDialog(onSubmit)

    await user.click(screen.getByRole('button', { name: 'Custom' }))
    await user.type(screen.getByLabelText('Nome'), '  Corda de seda  ')
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    expect(onSubmit).toHaveBeenCalledWith({ name: 'Corda de seda', quantity: 1, slots: 1 })
  })

  it('cobra um nome antes de chamar o backend', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    const user = renderDialog(onSubmit)

    await user.click(screen.getByRole('button', { name: 'Custom' }))
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    expect(screen.getByText('Informe um nome.')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  // Carga anda de meio em meio (T20 mede em 0,5 espaço); 0,7 não existe.
  it('recusa espaços fora do passo de 0,5', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    const user = renderDialog(onSubmit)

    await user.click(screen.getByRole('button', { name: 'Custom' }))
    await user.type(screen.getByLabelText('Nome'), 'Tocha')
    await user.clear(screen.getByLabelText('Espaços'))
    await user.type(screen.getByLabelText('Espaços'), '0.7')
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    expect(screen.getByText(/múltiplo de 0,5/)).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  // A regra recusou antes do fio: o texto dela É a informação, não pode virar
  // "erro inesperado".
  it('mostra a recusa de domínio sem fechar o formulário', async () => {
    const onSubmit = vi.fn().mockRejectedValue(new ItemRefused('Mochila sobrecarregada.'))
    const user = renderDialog(onSubmit)

    await user.click(screen.getByRole('button', { name: 'Custom' }))
    await user.type(screen.getByLabelText('Nome'), 'Bigorna')
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Mochila sobrecarregada.')
    expect(screen.getByLabelText('Nome')).toBeInTheDocument()
  })

  it('semeia os campos ao editar um item existente', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    const user = renderDialog(onSubmit, { name: 'Espada longa', quantity: 2, slots: 1.5 })

    await user.click(screen.getByRole('button', { name: 'Custom' }))
    expect(screen.getByLabelText('Nome')).toHaveValue('Espada longa')
    expect(screen.getByLabelText('Quantidade')).toHaveValue(2)
    expect(screen.getByLabelText('Espaços')).toHaveValue(1.5)

    await user.click(screen.getByRole('button', { name: 'Adicionar' }))
    expect(onSubmit).toHaveBeenCalledWith({ name: 'Espada longa', quantity: 2, slots: 1.5 })
  })
})
