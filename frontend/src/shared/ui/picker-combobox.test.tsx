import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { type PickerOption, PickerCombobox } from './picker-combobox'

const CONDITIONS: PickerOption[] = [
  { value: 'caido', label: 'Caído' },
  { value: 'cego', label: 'Cego' },
  { value: 'atordoado', label: 'Atordoado' },
]

function Fixture(props: { onPick?: (value: string) => void }) {
  return (
    <PickerCombobox
      options={CONDITIONS}
      aria-label="Aplicar condição"
      placeholder="Aplicar condição…"
      emptyMessage="Nenhuma."
      onPick={(value) => props.onPick?.(value)}
    />
  )
}

describe('PickerCombobox', () => {
  it('escolher uma opção entrega o valor ao chamador', async () => {
    const onPick = vi.fn()
    render(() => <Fixture onPick={onPick} />)
    const user = userEvent.setup()

    await user.click(screen.getByRole('combobox', { name: 'Aplicar condição' }))
    await user.click(await screen.findByRole('option', { name: 'Cego' }))

    expect(onPick).toHaveBeenCalledWith('cego')
  })

  // É um gatilho de ação, não um campo: depois de aplicar "Cego" o campo tem de
  // estar limpo para o próximo, senão só dá pra aplicar uma condição por sessão.
  it('limpa o campo depois de escolher', async () => {
    render(() => <Fixture />)
    const user = userEvent.setup()

    await user.click(screen.getByRole('combobox', { name: 'Aplicar condição' }))
    await user.click(await screen.findByRole('option', { name: 'Cego' }))

    await waitFor(() =>
      expect(screen.getByRole('combobox', { name: 'Aplicar condição' })).toHaveValue(''),
    )
  })

  it('filtra pelo que foi digitado', async () => {
    render(() => <Fixture />)
    const user = userEvent.setup()

    await user.type(screen.getByRole('combobox', { name: 'Aplicar condição' }), 'ator')

    expect(await screen.findByRole('option', { name: 'Atordoado' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Cego' })).not.toBeInTheDocument()
  })

  it('avisa quando a busca não acha nada', async () => {
    render(() => <Fixture />)
    const user = userEvent.setup()

    await user.type(screen.getByRole('combobox', { name: 'Aplicar condição' }), 'zzz')

    expect(await screen.findByText('Nenhuma.')).toBeInTheDocument()
  })

  // Kobalte embute rótulos em inglês ("Show suggestions", "Suggestions") que
  // sobrescrevem qualquer sr-only interno — o app é pt-BR.
  it('não anuncia rótulo em inglês', async () => {
    render(() => <Fixture />)

    expect(screen.queryByRole('button', { name: 'Show suggestions' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Ver opções' })).toBeInTheDocument()

    await userEvent.setup().click(screen.getByRole('button', { name: 'Ver opções' }))
    expect(await screen.findByRole('listbox', { name: 'Opções' })).toBeInTheDocument()
  })
})
