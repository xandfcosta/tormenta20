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
import * as pdfjs from 'pdfjs-dist'
import type { PDFDocumentProxy, PDFPageProxy } from 'pdfjs-dist'

/** O que o servidor escreve no `<div id="leitor">`. */
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

function leOCartaz(raiz: HTMLElement): Cartaz {
  return {
    livro: raiz.dataset.livro ?? '',
    worker: raiz.dataset.worker ?? '',
    pagina: Number(raiz.dataset.pagina ?? '1'),
    abertura: Number(raiz.dataset.abertura ?? '0'),
    termo: raiz.dataset.termo ?? '',
  }
}

/** dobra: minúsculas e sem acento, como a busca do servidor. */
function dobra(texto: string): string {
  return texto
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
  itens: ReadonlyArray<{ str: string; transform: number[]; width: number; height: number }>,
  termo: string,
  viewport: { transform: number[]; scale: number },
): Array<{ esquerda: number; topo: number; largura: number; altura: number }> {
  const alvo = dobra(termo)
  if (!alvo) return []
  const marcas = []
  for (const item of itens) {
    const onde = dobra(item.str).indexOf(alvo)
    if (onde < 0) continue
    const m = pdfjs.Util.transform(viewport.transform, item.transform)
    const altura = Math.hypot(m[2], m[3])
    const larguraDoItem = item.width * viewport.scale
    const porLetra = item.str.length > 0 ? larguraDoItem / item.str.length : 0
    marcas.push({
      esquerda: m[4] + onde * porLetra,
      topo: m[5] - altura,
      largura: Math.max(porLetra * alvo.length, 4),
      altura,
    })
  }
  return marcas
}

class Leitor {
  private doc: PDFDocumentProxy | null = null
  private zoom = 1
  private renderizando: Promise<void> = Promise.resolve()

  constructor(
    private readonly raiz: HTMLElement,
    private readonly cartaz: Cartaz,
    private readonly tela: HTMLCanvasElement,
    private readonly camada: HTMLElement,
    private readonly rotulo: HTMLElement,
  ) {}

  /** A página do ARQUIVO, que é o que o pdf.js conta. */
  private get noArquivo(): number {
    return this.cartaz.pagina + this.cartaz.abertura
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
    pdfjs.GlobalWorkerOptions.workerSrc = this.cartaz.worker
    // `rangeChunkSize` grande porque a rede é a LAN da mesa e o disco é o do
    // mestre: pedaço pequeno vira muitas idas e volta a pesar mais que baixar.
    this.doc = await pdfjs.getDocument({
      url: this.cartaz.livro,
      rangeChunkSize: 262144,
    }).promise
    await this.desenha()
    this.raiz.dataset.pronto = ''
    for (const botao of document.querySelectorAll<HTMLButtonElement>('[data-acao]')) {
      botao.disabled = false
    }
  }

  vai(passo: number): void {
    if (!this.doc) return
    const alvo = this.noArquivo + passo
    if (alvo < 1 || alvo > this.doc.numPages) return
    this.cartaz.pagina += passo
    void this.desenha()
  }

  aproxima(fator: number): void {
    this.zoom = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, this.zoom * fator))
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
    const pagina: PDFPageProxy = await this.doc.getPage(this.noArquivo)
    const larguraDisponivel = Math.min(this.raiz.clientWidth - 32, LARGURA_MAXIMA)
    const natural = pagina.getViewport({ scale: 1 })
    const escala = (larguraDisponivel / natural.width) * this.zoom
    const viewport = pagina.getViewport({ scale: escala })

    // O `devicePixelRatio` é o que separa "legível" de "borrado" numa tela de
    // retina: o canvas nasce com o dobro de pixels e é encolhido por CSS.
    const dpr = Math.min(window.devicePixelRatio || 1, DENSIDADE_MAXIMA)
    this.tela.width = Math.floor(viewport.width * dpr)
    this.tela.height = Math.floor(viewport.height * dpr)
    this.tela.style.width = `${Math.floor(viewport.width)}px`
    this.tela.style.height = `${Math.floor(viewport.height)}px`
    const contexto = this.tela.getContext('2d')
    if (!contexto) return
    contexto.setTransform(dpr, 0, 0, dpr, 0, 0)
    await pagina.render({ canvas: this.tela, canvasContext: contexto, viewport }).promise

    const texto = await pagina.getTextContent()
    this.camada.style.width = `${Math.floor(viewport.width)}px`
    this.camada.style.height = `${Math.floor(viewport.height)}px`
    this.camada.replaceChildren(
      ...marcasNoTexto(texto.items as never[], this.cartaz.termo, viewport).map((marca) => {
        const caixa = document.createElement('span')
        caixa.className = 'leitor-marca'
        caixa.style.left = `${marca.esquerda}px`
        caixa.style.top = `${marca.topo}px`
        caixa.style.width = `${marca.largura}px`
        caixa.style.height = `${marca.altura}px`
        return caixa
      }),
    )
    this.rotulo.textContent = `p${this.cartaz.pagina} de ${this.doc.numPages - this.cartaz.abertura}`
  }
}

export function montaOLeitor(): void {
  const raiz = document.getElementById('leitor')
  if (!raiz) return
  const tela = raiz.querySelector<HTMLCanvasElement>('canvas')
  const camada = raiz.querySelector<HTMLElement>('[data-marcas]')
  const rotulo = document.querySelector<HTMLElement>('[data-pagina-atual]')
  if (!tela || !camada || !rotulo) return

  const leitor = new Leitor(raiz, leOCartaz(raiz), tela, camada, rotulo)
  const aoClicar = (acao: string, fazer: () => void) =>
    document.querySelector(`[data-acao="${acao}"]`)?.addEventListener('click', fazer)
  aoClicar('anterior', () => leitor.vai(-1))
  aoClicar('proxima', () => leitor.vai(1))
  aoClicar('mais', () => leitor.aproxima(1.25))
  aoClicar('menos', () => leitor.aproxima(0.8))
  // As setas leem o livro; o guarda de digitação não é preciso porque esta cena
  // não tem campo nenhum — mas o `dialog[open]` do buscador tem, e ele fica por
  // cima. Sem isto, ⌃K e as setas disputariam a mesma tecla.
  window.addEventListener('keydown', (evento) => {
    if (document.querySelector('dialog[open]')) return
    if (evento.key === 'ArrowRight') leitor.vai(1)
    if (evento.key === 'ArrowLeft') leitor.vai(-1)
  })

  void leitor.abre().catch((erro: unknown) => {
    // Falhar CALADO aqui seria a tela preta sem explicação: o livro pode não
    // estar configurado, o arquivo pode ter sumido do disco do mestre.
    rotulo.textContent = 'não consegui abrir o livro'
    console.error('[leitor]', erro)
  })
}

montaOLeitor()
