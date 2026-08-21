import { render, screen } from '@solidjs/testing-library'
import { describe, expect, it } from 'vitest'
import { FieldLabel, SectionLabel, SectionTitle } from './section-label'

/**
 * O que se afirma aqui é a SEMÂNTICA, não a aparência.
 *
 * A aparência dos três papéis é decisão de desenho e vive no `/grimorio`, que a
 * desenha com os componentes de verdade e lê os valores do navegador — em jsdom
 * nenhuma classe do Tailwind resolve, então um teste de classe aqui só
 * repetiria a string do arquivo ao lado.
 *
 * O que jsdom testemunha, e que o typechecker NÃO garante, é o elemento que
 * cada um produz. Isso importa: o título de seção é um `<h2>` porque a estrutura
 * de cabeçalhos é o que um leitor de tela usa para pular pela cena, e trocá-lo
 * por um `<p>` degrada a navegação sem mudar um pixel.
 */
describe('a família de rótulos', () => {
  it('o título de seção é um cabeçalho de verdade', () => {
    render(() => <SectionTitle>Distribua os atributos</SectionTitle>)

    expect(screen.getByRole('heading', { name: 'Distribua os atributos', level: 2 })).toBeTruthy()
  })

  it('o cabeçalho de bloco não vira cabeçalho sem quem chama pedir', () => {
    render(() => <SectionLabel>Kit da classe</SectionLabel>)

    // Um `<h4>` solto dentro de um painel quebra a ladeira de níveis da cena:
    // quem quiser um cabeçalho pede um, com o nível que o sítio comporta.
    expect(screen.queryByRole('heading')).toBeNull()
    expect(screen.getByText('Kit da classe')).toBeTruthy()
  })

  it('a semântica é de quem chama: o mesmo desenho vira legenda de grupo', () => {
    render(() => (
      <fieldset>
        <FieldLabel as="legend">Atributos</FieldLabel>
        <input aria-label="Força" />
      </fieldset>
    ))

    expect(screen.getByRole('group', { name: 'Atributos' })).toBeTruthy()
  })

  it('o id chega, para o painel poder ser rotulado por ele', () => {
    render(() => <SectionTitle id="mesa-bestiario">Bestiário</SectionTitle>)

    expect(screen.getByRole('heading', { name: 'Bestiário' }).id).toBe('mesa-bestiario')
  })
})
