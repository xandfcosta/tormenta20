import { render, screen } from '@solidjs/testing-library'
import { describe, expect, it } from 'vitest'
import { Button } from './button'

/**
 * Dois casos, e os dois são DECISÕES desta casa (ALE-187).
 *
 * O que saiu era comportamento nativo do `<button>` (dispara clique, não dispara
 * desabilitado, aceita `type="submit"`) e forma de DOM que ninguém promete: as
 * classes da variante e os `data-variant`/`data-size`/`data-slot`, que o teste
 * chamava de "seam de estilo e de teste" e que — conferido — nenhum CSS,
 * nenhum e2e e nenhum outro teste consome. Costura que não costura nada.
 */
describe('Button', () => {
  /**
   * Dentro de um `<form>` o padrão do HTML é `submit`, e um botão de rolar dado
   * que envia formulário é o tipo de defeito que só aparece na tela de quem
   * está jogando. O componente força `button`; isto prende essa escolha.
   */
  it('é um <button type="button"> por padrão, não submit', () => {
    render(() => <Button>Rolar</Button>)
    expect(screen.getByRole('button', { name: 'Rolar' })).toHaveAttribute('type', 'button')
  })

  /**
   * O `tailwind-merge` é o que deixa o chamador ajustar sem `!important` e sem
   * ordem de folha. Sem ele a classe da variante e a do chamador coexistem, e
   * quem ganha vira sorte da ordem no CSS — por isso a asserção é NEGATIVA
   * também: a altura antiga tem de sumir, não só a nova aparecer.
   */
  it('classes do chamador vencem as da variante (tailwind-merge)', () => {
    render(() => <Button class="h-20">Alto</Button>)
    const className = screen.getByRole('button').className
    expect(className).toContain('h-20')
    expect(className).not.toContain('h-9')
  })
})
