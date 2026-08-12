import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { createSignal } from 'solid-js'
import { describe, expect, it } from 'vitest'
import { NumberInput, clampToRange } from './number-input'

describe('clampToRange', () => {
  it('deixa passar o que já está dentro dos limites', () => {
    expect(clampToRange(3, 1, 9)).toBe(3)
  })

  it('prende nos extremos', () => {
    expect(clampToRange(0, 1, 9)).toBe(1)
    expect(clampToRange(99, 1, 9)).toBe(9)
  })

  it('sem limite, não prende nada', () => {
    expect(clampToRange(-5)).toBe(-5)
  })
})

function renderInput(
  initial: number,
  extra: { min?: number; max?: number; step?: number; spinnerLabel?: string } = {},
) {
  const [value, setValue] = createSignal(initial)
  render(() => (
    <NumberInput aria-label="Espaços" value={value()} onChange={setValue} {...extra} />
  ))
  return value
}

describe('NumberInput', () => {
  // Espaços andam de meio em meio (T20 mede carga em 0,5) — o passo tem de
  // respeitar o `step`, não somar 1 calado.
  it('o spinner soma e subtrai o passo do campo', async () => {
    const value = renderInput(1, { step: 0.5, min: 0.5 })
    const user = userEvent.setup()

    await user.click(screen.getByLabelText('Aumentar'))
    expect(value()).toBe(1.5)

    await user.click(screen.getByLabelText('Diminuir'))
    expect(value()).toBe(1)
  })

  it('não deixa o spinner passar do limite', async () => {
    const value = renderInput(1, { min: 1, max: 2 })
    const user = userEvent.setup()

    await user.click(screen.getByLabelText('Diminuir'))
    expect(value()).toBe(1)
    expect(screen.getByLabelText('Diminuir')).toBeDisabled()

    await user.click(screen.getByLabelText('Aumentar'))
    expect(value()).toBe(2)
    expect(screen.getByLabelText('Aumentar')).toBeDisabled()
  })

  // Uma tela com três campos numéricos anunciava "Aumentar" três vezes e não
  // dizia o quê — o nome do campo entra no rótulo do spinner.
  it('nomeia os spinners quando o campo pede', async () => {
    renderInput(9, { spinnerLabel: 'deslocamento' })

    expect(screen.getByLabelText('Aumentar deslocamento')).toBeInTheDocument()
    expect(screen.getByLabelText('Diminuir deslocamento')).toBeInTheDocument()
  })

  it('digitar atualiza o valor', async () => {
    const value = renderInput(1)
    const user = userEvent.setup()

    await user.clear(screen.getByLabelText('Espaços'))
    await user.type(screen.getByLabelText('Espaços'), '7')
    expect(value()).toBe(7)
  })
})
