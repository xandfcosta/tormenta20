import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { SidePanel } from './side-panel'

/** The panel switches on a WIDTH query — drive it from the test. */
function mockWidth(sharesScreen: boolean): void {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: sharesScreen,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

function renderPanel(sharesScreen: boolean) {
  mockWidth(sharesScreen)
  const onOpenChange = vi.fn()
  render(() => (
    <SidePanel open onOpenChange={onOpenChange} title="Catálogos" description="Consulta rápida">
      <p>corpo do painel</p>
    </SidePanel>
  ))
  return { onOpenChange }
}

const panel = () => screen.getByRole('dialog')

beforeEach(() => {
  Element.prototype.scrollIntoView = vi.fn()
})
afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('SidePanel', () => {
  it('mostra título, descrição e corpo', () => {
    renderPanel(true)

    expect(panel()).toHaveAccessibleName('Catálogos')
    expect(panel()).toHaveAccessibleDescription('Consulta rápida')
    expect(screen.getByText('corpo do painel')).toBeInTheDocument()
  })

  // A cena atrás continua viva de propósito: o mestre lê a condição aqui e
  // aplica no rastreador lá. Marcar como sobreposição mataria as setas da cena.
  it('na tela larga se declara em linha, para o driver de teclado não recuar', () => {
    renderPanel(true)

    expect(panel()).toHaveAttribute('data-nav-inline')
    expect(panel()).toHaveAttribute('data-nav-region', 'side-panel')
  })

  it('na tela estreita é modal de verdade — não sobra espaço para dois', () => {
    renderPanel(false)

    expect(panel()).not.toHaveAttribute('data-nav-inline')
  })

  // São DOIS fechares — o ✕ do topo (tela larga) e a barra do pé (telefone) —,
  // e os dois se chamam pelo NOME DO PAINEL. Nenhum pode anunciar o "Dismiss"
  // que o Kobalte põe por padrão, e nenhum pode dizer só "Fechar": dentro da
  // gaveta da fila esse nome empatava com o do formulário de adicionar
  // (ALE-198). Em jsdom os dois existem porque `xl:hidden` não pinta nada; no
  // browser só um está na tela.
  it('os fechares são em português e dizem o NOME do painel', () => {
    renderPanel(true)

    expect(screen.getAllByRole('button', { name: 'Fechar Catálogos' })).toHaveLength(2)
    expect(screen.queryByRole('button', { name: 'Dismiss' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Fechar' })).not.toBeInTheDocument()
  })

  // O CloseButton do Kobalte sobrescreve o nome acessível com "Dismiss" em
  // inglês, mesmo com "Fechar" escrito dentro dele (armadilha #2). E o rótulo
  // NOMEIA o painel: "Fechar" nu empatava com o "Fechar" do formulário que vive
  // dentro da gaveta da fila (ALE-198).
  it('fecha pela barra de largura cheia, anunciada em português', async () => {
    const { onOpenChange } = renderPanel(false)

    // A BARRA é a que tem texto escrito dentro; o ✕ do topo é só o ícone.
    const bar = screen
      .getAllByRole('button', { name: 'Fechar Catálogos' })
      .find((botao) => botao.textContent?.trim() === 'Fechar')
    if (!bar) throw new Error('a barra de fechar não está na árvore')
    await userEvent.click(bar)

    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  // O Kobalte dispensa por interação externa MESMO não sendo modal. Sem barrar
  // isso, o mestre clica no rastreador atrás e o painel que ele estava lendo
  // some — que é o oposto de dividir a tela (ALE-75).
  it('na tela larga NÃO fecha ao clicar na cena atrás', async () => {
    const { onOpenChange } = renderPanel(true)

    await userEvent.click(document.body)

    expect(onOpenChange).not.toHaveBeenCalled()
  })

  it('na tela larga a cena atrás continua clicável', () => {
    renderPanel(true)

    expect(document.body.style.pointerEvents).not.toBe('none')
  })

  it('na tela estreita a cena atrás fica inerte, como em todo modal', () => {
    renderPanel(false)

    expect(document.body.style.pointerEvents).toBe('none')
  })

  it('o gesto nunca é o único caminho: Esc também fecha', async () => {
    const { onOpenChange } = renderPanel(true)

    await userEvent.keyboard('{Escape}')

    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('fixa o contexto vivo acima do corpo quando recebe um cabeçalho', () => {
    mockWidth(true)
    render(() => (
      <SidePanel open onOpenChange={vi.fn()} title="Adicionar monstro" header={<span>Rodada 3</span>}>
        <p>lista</p>
      </SidePanel>
    ))

    expect(screen.getByText('Rodada 3')).toBeInTheDocument()
  })
})
