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

/**
 * APAGAR PARA DIGITAR OUTRO NÚMERO (ALE-236).
 *
 * O campo emitia `Number('')`, que é ZERO, a cada tecla — então apagar mandava
 * zero ao chamador, o chamador clampava para o mínimo dele, e o valor
 * controlado reescrevia o campo no meio da digitação.
 *
 * Reproduzido no Chrome antes de virar teste, em "Personagens do grupo"
 * (mín 1, máx 8): partindo de 4, um Backspace deixava `1` na tela em vez de
 * vazio, e digitar `3` produzia `13`, clampado a `8`. Quem quis 3 gravou 8.
 *
 * Os testes usam `{backspace}` e NÃO `clear()`: `input[type=number]` não tem
 * API de seleção, então nem `clear()` nem `{selectall}` esvaziam o campo em
 * jsdom, e um teste escrito com eles mede outra coisa. O teste "digitar
 * atualiza o valor" acima passava por acidente justamente por isso — sem
 * `min`, o zero intermediário não era clampado e o dígito colava num "0".
 */
describe('apagar para digitar outro número', () => {
  /** Como o chamador se comporta de verdade: ele é quem clampa (`setPartySize`
   *  e os outros dezenove), e é o clamp que tornava o defeito destrutivo. */
  function renderClampado(inicial: number, min: number, max: number) {
    const [value, setValue] = createSignal(inicial)
    render(() => (
      <NumberInput
        aria-label="Personagens"
        value={value()}
        onChange={(n) => setValue(Math.min(max, Math.max(min, Math.round(n))))}
        min={min}
        max={max}
      />
    ))
    return value
  }

  it('o campo FICA vazio, e o modelo não se mexe', async () => {
    const value = renderClampado(4, 1, 8)
    const user = userEvent.setup()

    await user.type(screen.getByLabelText('Personagens'), '{backspace}')

    expect(screen.getByLabelText('Personagens')).toHaveValue(null)
    expect(value(), 'apagar na tela não pode gravar nada').toBe(4)
  })

  // O gesto inteiro, que é o da issue: sair de 4 e chegar a 3.
  it('apagar e digitar dá o número digitado, não a colagem clampada', async () => {
    const value = renderClampado(4, 1, 8)
    const user = userEvent.setup()

    await user.type(screen.getByLabelText('Personagens'), '{backspace}3')

    expect(screen.getByLabelText('Personagens')).toHaveValue(3)
    expect(value()).toBe(3)
  })

  // Um campo vazio na tela sobre um valor gravado é mentira: sair do campo
  // desfaz o rascunho e mostra o que está valendo.
  it('sair do campo vazio volta a mostrar o valor gravado', async () => {
    const value = renderClampado(4, 1, 8)
    const user = userEvent.setup()

    await user.type(screen.getByLabelText('Personagens'), '{backspace}')
    await user.tab()

    expect(screen.getByLabelText('Personagens')).toHaveValue(4)
    expect(value()).toBe(4)
  })

  // O spinner COMPROMETE: com o campo vazio ele parte do valor de verdade, e
  // não de um zero implícito.
  it('o spinner sobre um campo vazio parte do valor gravado', async () => {
    const value = renderClampado(4, 1, 8)
    const user = userEvent.setup()

    await user.type(screen.getByLabelText('Personagens'), '{backspace}')
    await user.click(screen.getByLabelText('Aumentar'))

    expect(value()).toBe(5)
    expect(screen.getByLabelText('Personagens')).toHaveValue(5)
  })
})
