import { render, screen, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { RaceChoiceState } from './grant-helpers'
import { RacePicker } from './race-picker'

const OPTIONS = ['Anão', 'Elfo', 'Humano', 'Lefou', 'Suraggel']

function renderPicker(value: string[] = [], choices: RaceChoiceState = {}) {
  const onChange = vi.fn()
  const onChoice = vi.fn()
  render(() => (
    <RacePicker
      options={OPTIONS}
      value={value}
      choices={choices}
      onChange={onChange}
      onChoice={onChoice}
    />
  ))
  return { onChange, onChoice }
}

describe('RacePicker — catálogo', () => {
  it('separa as raças em Comuns e Outras', () => {
    renderPicker()

    expect(screen.getByRole('listbox', { name: 'Comuns' })).toBeInTheDocument()
    expect(screen.getByRole('listbox', { name: 'Outras' })).toBeInTheDocument()
  })

  it('mostra a assinatura de atributo no ladrilho', () => {
    renderPicker()

    expect(screen.getByRole('option', { name: /Anão/ })).toHaveTextContent('+2 CON')
  })

  it('busca sem acento encontra a raça acentuada', async () => {
    renderPicker()

    await userEvent.type(screen.getByLabelText('Buscar raça'), 'anao')

    expect(screen.getByRole('option', { name: /Anão/ })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: /Elfo/ })).not.toBeInTheDocument()
  })

  it('busca sem resultado responde em vez de mostrar nada', async () => {
    renderPicker()

    await userEvent.type(screen.getByLabelText('Buscar raça'), 'draconato')

    expect(screen.getByText(/Nenhuma raça para/)).toBeInTheDocument()
  })

  it('escolher acrescenta a raça', async () => {
    const { onChange } = renderPicker()

    await userEvent.click(screen.getByRole('option', { name: /Anão/ }))

    expect(onChange).toHaveBeenCalledWith(['Anão'])
  })

  it('escolher a já escolhida remove (é alternância)', async () => {
    const { onChange } = renderPicker(['Anão'])

    await userEvent.click(screen.getByRole('option', { name: /Anão/ }))

    expect(onChange).toHaveBeenCalledWith([])
  })
})

describe('RacePicker — detalhe da escolhida', () => {
  it('sem escolha, convida em vez de ficar em branco', () => {
    renderPicker()

    expect(screen.getByText(/Escolha uma raça para ver o que ela concede/)).toBeInTheDocument()
  })

  it('a primeira raça é a primária e vale mecanicamente', () => {
    renderPicker(['Anão'])

    const detail = within(screen.getByRole('region', { name: 'Raça escolhida' }))
    expect(detail.getByText('Anão · primária')).toBeInTheDocument()
    expect(detail.getByText('+2 CON')).toBeInTheDocument()
  })

  it('a segunda raça não aplica nada até o mestre autorizar', async () => {
    const { onChoice } = renderPicker(['Anão', 'Elfo'])

    const optIn = screen.getByRole('button', { name: /Aplicar propriedades/ })
    expect(optIn).toHaveAttribute('aria-pressed', 'false')
    // Sem opt-in, nem a escolha de atributo da secundária aparece.
    expect(screen.queryByText(/Distribua/)).not.toBeInTheDocument()

    await userEvent.click(optIn)
    expect(onChoice).toHaveBeenCalledWith('Elfo', { applied: true })
  })

  it('avisa enquanto a escolha de atributo da raça está pendente', () => {
    renderPicker(['Humano'])

    expect(screen.getByText('Escolha de atributo pendente.')).toBeInTheDocument()
  })

  it('some com o aviso quando os +1 foram todos colocados', () => {
    renderPicker(['Humano'], {
      Humano: { floatingPicks: ['strength', 'dexterity', 'constitution'] },
    })

    expect(screen.queryByText('Escolha de atributo pendente.')).not.toBeInTheDocument()
  })

  it('a Deformidade do Lefou aparece junto do resto da raça', () => {
    renderPicker(['Lefou'])

    expect(screen.getByText(/2 bônus de \+2/)).toBeInTheDocument()
  })

  it('encaminha a escolha de atributo com o nome da raça certa', async () => {
    const { onChoice } = renderPicker(['Humano'])

    await userEvent.click(screen.getByRole('button', { name: 'FOR' }))

    expect(onChoice).toHaveBeenCalledWith('Humano', { floatingPicks: ['strength'] })
  })
})
