import { render, screen } from '@solidjs/testing-library'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { isInvalid } from './field-frame'
import { TextAreaField } from './textarea-field'
import { TextField } from './text-field'

afterEach(() => {
  document.body.innerHTML = ''
})

describe('isInvalid', () => {
  it('só é inválido com mensagem de fato', () => {
    expect(isInvalid(undefined)).toBe(false)
    expect(isInvalid([])).toBe(false)
    expect(isInvalid(['Nome é obrigatório'])).toBe(true)
  })
})

describe('FieldFrame (via TextField / TextAreaField)', () => {
  // biome's noLabelWithoutControl e leitores de tela concordam: o label tem que
  // apontar pro id do controle, não embrulhá-lo.
  it('associa o rótulo ao controle pelo nome', () => {
    render(() => <TextField name="nome" label="Nome" value="" onInput={vi.fn()} />)
    expect(screen.getByLabelText('Nome')).toHaveAttribute('id', 'nome')
  })

  it('a dica dá lugar às mensagens de erro', () => {
    render(() => (
      <TextField
        name="nome"
        label="Nome"
        value=""
        onInput={vi.fn()}
        hint="Como a mesa vai chamar"
        errors={['Nome é obrigatório']}
      />
    ))
    expect(screen.getByText('Nome é obrigatório')).toBeInTheDocument()
    expect(screen.queryByText('Como a mesa vai chamar')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Nome')).toHaveAttribute('aria-invalid', 'true')
  })
})

describe('TextAreaField', () => {
  it('devolve o texto digitado ao chamador', () => {
    const onInput = vi.fn()
    render(() => (
      <TextAreaField name="descricao" label="Descrição" value="" onInput={onInput} rows={3} />
    ))

    const field = screen.getByLabelText('Descrição') as HTMLTextAreaElement
    expect(field.tagName).toBe('TEXTAREA')
    expect(field).toHaveAttribute('rows', '3')

    field.value = 'Uma mesa nova'
    field.dispatchEvent(new Event('input', { bubbles: true }))
    expect(onInput).toHaveBeenCalledWith('Uma mesa nova')
  })
})
