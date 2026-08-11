import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Button } from './button'

describe('Button', () => {
  it('é um <button type="button"> por padrão, não submit', () => {
    render(() => <Button>Rolar</Button>)
    expect(screen.getByRole('button', { name: 'Rolar' })).toHaveAttribute('type', 'button')
  })

  it('carimba variant e size como data-attrs (seam de estilo e de teste)', () => {
    render(() => (
      <Button variant="destructive" size="sm">
        Excluir
      </Button>
    ))
    const button = screen.getByRole('button')
    expect(button).toHaveAttribute('data-slot', 'button')
    expect(button).toHaveAttribute('data-variant', 'destructive')
    expect(button).toHaveAttribute('data-size', 'sm')
  })

  it('aplica as classes da variante escolhida', () => {
    render(() => <Button variant="outline">Cancelar</Button>)
    expect(screen.getByRole('button').className).toContain('border')
  })

  it('classes do chamador vencem as da variante (tailwind-merge)', () => {
    render(() => <Button class="h-20">Alto</Button>)
    const className = screen.getByRole('button').className
    expect(className).toContain('h-20')
    expect(className).not.toContain('h-9')
  })

  it('dispara onClick', async () => {
    const onClick = vi.fn()
    render(() => <Button onClick={onClick}>Rolar</Button>)
    await userEvent.setup().click(screen.getByRole('button'))
    expect(onClick).toHaveBeenCalledOnce()
  })

  it('não dispara quando desabilitado', async () => {
    const onClick = vi.fn()
    render(() => (
      <Button disabled onClick={onClick}>
        Rolar
      </Button>
    ))
    await userEvent.setup().click(screen.getByRole('button'))
    expect(onClick).not.toHaveBeenCalled()
  })

  it('aceita type="submit" pra formulários', () => {
    render(() => <Button type="submit">Entrar</Button>)
    expect(screen.getByRole('button')).toHaveAttribute('type', 'submit')
  })
})
