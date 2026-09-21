/**
 * O LEITOR DO LIVRO (ALE-264): o Tormenta 20 aberto na página certa, com o
 * verbete DESTACADO.
 *
 * Por que um visualizador nosso e não o do navegador — as duas razões foram
 * medidas nesta sessão, e nenhuma é preferência:
 *
 *  1. O visualizador do Chrome IGNORA `#search=`. Ele obedece `#page=N` e mais
 *     nada; não há forma de pedir destaque por URL. O dono conferiu na tela.
 *  2. Ele também baixa o arquivo INTEIRO para mostrar uma página. Contado na
 *     interface de loopback: 85 MiB com o PDF cru, 75 MiB com o linearizado —
 *     os dois iguais ao tamanho do arquivo. O `pdf.js` pede FAIXAS de verdade.
 *
 * O que este leitor NÃO é: um substituto do visualizador do navegador. Ele
 * mostra uma página por vez, anda com as setas e destaca o termo. Quem quiser
 * busca, miniaturas e impressão tem o link "abrir fora", que serve o PDF cru.
 * Vendorizar o viewer completo do pdf.js traria locales, cmaps e imagens — dez
 * vezes o peso para funções que a cena não pede.
 */

import type { PDFDocumentProxy, PDFPageProxy } from 'pdfjs-dist'
import * as pdfjs from 'pdfjs-dist'

/** O que o servidor escreve no `<div id="reader">`. */
type Cartaz = {
  livro: string
  worker: string
  /** A página IMPRESSA no rodapé, que é a que o catálogo guarda. */
  pagina: number
  /** Quantas páginas o ARQUIVO tem antes da impressa 1. */
  abertura: number
  termo: string
}

const ZOOM_MIN = 0.5
const ZOOM_MAX = 3

/**
 * A LARGURA MÁXIMA da página desenhada, e ela é conserto de um defeito medido.
 *
 * Sem limite, a página nasce com a largura do contêiner: numa janela de 1900px o
 * canvas ficava 1869×2527 CSS, que a 2× de `devicePixelRatio` são 18 MILHÕES de
 * pixels — ~74 MB de bitmap. O renderizador da aba TRAVOU duas vezes com isso, e
 * num telefone seria pior.
 *
 * 1100 também é melhor de LER: uma página de livro esticada em 1900px força o
 * olho a atravessar a tela inteira por linha. O zoom continua indo além para
 * quem quiser aproximar.
 */
const LARGURA_MAXIMA = 1100

/** O `devicePixelRatio` acima de 2 não acrescenta nitidez visível e quadruplica
 *  o bitmap. É o mesmo limite que o `<img srcset>` da SPA usa. */
const DENSIDADE_MAXIMA = 2

function leOCartaz(root: HTMLElement): Cartaz {
  return {
    livro: root.dataset.livro ?? '',
    worker: root.dataset.worker ?? '',
    pagina: Number(root.dataset.pagina ?? '1'),
    abertura: Number(root.dataset.abertura ?? '0'),
    termo: root.dataset.termo ?? '',
  }
}

/** dobra: minúsculas e sem acento, como a busca do servidor. */
function dobra(text: string): string {
  return text
    .normalize('NFD')
    .replace(/\p{Mn}/gu, '')
    .toLowerCase()
}

/**
 * marcasNoTexto devolve as caixas a destacar nesta página.
 *
 * O destaque é por ITEM de texto e estreitado por PROPORÇÃO de caracteres — o
 * pdf.js entrega a posição do item inteiro, não a de cada letra. Numa fonte
 * proporcional isso é aproximado, e é o certo aqui: a marca é para o olho achar
 * o verbete na página, não para sublinhar a palavra com precisão tipográfica.
 */
function marcasNoTexto(
  items: ReadonlyArray<{ str: string; transform: number[]; width: number; height: number }>,
  term: string,
  viewport: { transform: number[]; scale: number },
): Array<{ esquerda: number; topo: number; largura: number; altura: number }> {
  const target = dobra(term)
  if (!target) return []
  const marks = []
  for (const item of items) {
    const where = dobra(item.str).indexOf(target)
    if (where < 0) continue
    const m = pdfjs.Util.transform(viewport.transform, item.transform)
    const height = Math.hypot(m[2], m[3])
    const itemWidth = item.width * viewport.scale
    const byLetter = item.str.length > 0 ? itemWidth / item.str.length : 0
    marks.push({
      esquerda: m[4] + where * byLetter,
      topo: m[5] - height,
      largura: Math.max(byLetter * target.length, 4),
      altura: height,
    })
  }
  return marks
}

class Leitor {
  private doc: PDFDocumentProxy | null = null
  private zoom = 1
  private renderizando: Promise<void> = Promise.resolve()

  constructor(
    private readonly root: HTMLElement,
    private readonly poster: Cartaz,
    private readonly page: HTMLCanvasElement,
    private readonly layer: HTMLElement,
    private readonly label: HTMLElement,
  ) {}

  /** A página do ARQUIVO, que é o que o pdf.js conta. */
  private get noArquivo(): number {
    return this.poster.pagina + this.poster.abertura
  }

  /**
   * abre carrega o documento e só então LIBERA os controles.
   *
   * Os botões nascem `disabled` e é conserto de um defeito que o e2e denunciou:
   * clicar em "próxima" antes de o documento chegar caía num `return` mudo, e
   * quem estava lendo via um botão que não faz nada. Um PDF de 89 MB tem essa
   * janela de verdade.
   */
  async abre(): Promise<void> {
    pdfjs.GlobalWorkerOptions.workerSrc = this.poster.worker
    // `rangeChunkSize` grande porque a rede é a LAN da mesa e o disco é o do
    // mestre: pedaço pequeno vira muitas idas e volta a pesar mais que baixar.
    this.doc = await pdfjs.getDocument({
      url: this.poster.livro,
      rangeChunkSize: 262144,
    }).promise
    await this.desenha()
    this.root.dataset.pronto = ''
    for (const button of document.querySelectorAll<HTMLButtonElement>('[data-acao]')) {
      button.disabled = false
    }
  }

  vai(step: number): void {
    if (!this.doc) return
    const target = this.noArquivo + step
    if (target < 1 || target > this.doc.numPages) return
    this.poster.pagina += step
    void this.desenha()
  }

  aproxima(factor: number): void {
    this.zoom = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, this.zoom * factor))
    void this.desenha()
  }

  /**
   * desenha serializa os renders: trocar de página duas vezes rápido com dois
   * `render` no mesmo canvas faz o pdf.js lançar "Cannot use the same canvas",
   * e a tela fica em branco sem erro visível para quem está lendo.
   */
  private desenha(): Promise<void> {
    this.renderizando = this.renderizando.then(() => this.pinta())
    return this.renderizando
  }

  private async pinta(): Promise<void> {
    if (!this.doc) return
    const page: PDFPageProxy = await this.doc.getPage(this.noArquivo)
    const availableWidth = Math.min(this.root.clientWidth - 32, LARGURA_MAXIMA)
    const natural = page.getViewport({ scale: 1 })
    const scale = (availableWidth / natural.width) * this.zoom
    const viewport = page.getViewport({ scale: scale })

    // O `devicePixelRatio` é o que separa "legível" de "borrado" numa tela de
    // retina: o canvas nasce com o dobro de pixels e é encolhido por CSS.
    const dpr = Math.min(window.devicePixelRatio || 1, DENSIDADE_MAXIMA)
    this.page.width = Math.floor(viewport.width * dpr)
    this.page.height = Math.floor(viewport.height * dpr)
    this.page.style.width = `${Math.floor(viewport.width)}px`
    this.page.style.height = `${Math.floor(viewport.height)}px`
    const context = this.page.getContext('2d')
    if (!context) return
    context.setTransform(dpr, 0, 0, dpr, 0, 0)
    await page.render({ canvas: this.page, canvasContext: context, viewport }).promise

    const text = await page.getTextContent()
    this.layer.style.width = `${Math.floor(viewport.width)}px`
    this.layer.style.height = `${Math.floor(viewport.height)}px`
    this.layer.replaceChildren(
      ...marcasNoTexto(text.items as never[], this.poster.termo, viewport).map((marker) => {
        const box = document.createElement('span')
        box.className = 'reader-mark'
        box.style.left = `${marker.esquerda}px`
        box.style.top = `${marker.topo}px`
        box.style.width = `${marker.largura}px`
        box.style.height = `${marker.altura}px`
        return box
      }),
    )
    this.label.textContent = `p${this.poster.pagina} de ${this.doc.numPages - this.poster.abertura}`
  }
}

export function montaOLeitor(): void {
  const root = document.getElementById('reader')
  if (!root) return
  const page = root.querySelector<HTMLCanvasElement>('canvas')
  const layer = root.querySelector<HTMLElement>('[data-marcas]')
  const label = document.querySelector<HTMLElement>('[data-pagina-atual]')
  if (!page || !layer || !label) return

  const reader = new Leitor(root, leOCartaz(root), page, layer, label)
  const onClick = (action: string, make: () => void) =>
    document.querySelector(`[data-acao="${action}"]`)?.addEventListener('click', make)
  onClick('anterior', () => reader.vai(-1))
  onClick('proxima', () => reader.vai(1))
  onClick('mais', () => reader.aproxima(1.25))
  onClick('menos', () => reader.aproxima(0.8))
  // As setas leem o livro; o guarda de digitação não é preciso porque esta cena
  // não tem campo nenhum — mas o `dialog[open]` do buscador tem, e ele fica por
  // cima. Sem isto, ⌃K e as setas disputariam a mesma tecla.
  window.addEventListener('keydown', (evt) => {
    if (document.querySelector('dialog[open]')) return
    if (evt.key === 'ArrowRight') reader.vai(1)
    if (evt.key === 'ArrowLeft') reader.vai(-1)
    // O Esc ATRAVESSA o iframe. Dentro do diálogo da cena, o foco fica no
    // documento de dentro, e o Esc do `<dialog>` é do documento de FORA — sem
    // esta linha, quem lê pelo teclado não tem como fechar o livro. Mesma
    // origem, então falar com o pai é permitido; o `try` cobre o dia em que
    // alguém embutir esta cena de outro lugar.
    if (evt.key === 'Escape' && window.parent !== window) {
      try {
        window.parent.document.querySelector<HTMLDialogElement>('#book-in-dialog')?.close()
      } catch {
        /* origem diferente: não há o que fechar daqui */
      }
    }
  })

  void reader.abre().catch((failure: unknown) => {
    // Falhar CALADO aqui seria a tela preta sem explicação: o livro pode não
    // estar configurado, o arquivo pode ter sumido do disco do mestre.
    label.textContent = 'não consegui abrir o livro'
    console.error('[leitor]', failure)
  })
}

montaOLeitor()
