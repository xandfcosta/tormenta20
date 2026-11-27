/**
 * O CARTÃO QUE FLUTUA no desktop (ALE-536): o do ataque deixa de empurrar as
 * superfícies para baixo e paira sobre a mesa, e a cabeça dele é a alça de
 * arrastar. No celular ele continua no fluxo — quem decide é o CSS, pela
 * largura; aqui só se arrasta quando o cartão está de fato `position: fixed`.
 *
 * # Por que a posição sobrevive ao servidor
 *
 * O stream remenda o cartão a cada mudança da mesa, e o remendo reescreve os
 * atributos para o que o servidor mandou — o `style` posto aqui some no tique.
 * Quem o devolve é o observador do `body`: ele RECOLOCA cada cartão a cada
 * remendo, do armazenamento deste aparelho ou da pilha, e roda antes da
 * pintura. Por isso o cartão não declara `data-preserve-attr`: seria uma
 * segunda forma de dizer a mesma coisa (sabotado: sem ela, o
 * `attack-card-float.spec.ts` continua verde).
 *
 * # Onde nasce, e o recolher
 *
 * O cartão que ninguém arrastou NASCE empilhado embaixo, à direita, acima do
 * chão marcado com `data-float-floor` (a linha do turno do mestre) — e não no
 * topo, onde moram as faixas que todos leem (ver GLOSSARY, **cartão
 * flutuante**). Arrastado, ele sai da pilha e fica onde foi posto. A seta da
 * cabeça (`data-float-fold`) o recolhe até a cabeça; o recolhido também fica
 * no aparelho, e é REAPLICADO a cada remendo pelo mesmo observador.
 */

const STORAGE_KEY = 't20-float-card'
const FOLD_KEY = 't20-float-fold'
const FLEETING_FOLD_KEY = 't20-float-fold-once'
/** O vão entre os cartões da pilha, e entre ela e a borda da janela. */
const DOCK_GAP = 8
const DOCK_EDGE = 12

type Spot = { x: number; y: number }

/** Lê a posição guardada; qualquer coisa que não sejam dois números é nada. */
export function readSpot(raw: string | null): Spot | null {
  if (!raw) return null
  try {
    const spot = JSON.parse(raw) as Partial<Spot>
    if (typeof spot.x !== 'number' || typeof spot.y !== 'number') return null
    if (!Number.isFinite(spot.x) || !Number.isFinite(spot.y)) return null
    return { x: spot.x, y: spot.y }
  } catch {
    return null
  }
}

/** Mantém o cartão inteiro na janela: arrastado para fora, ele não teria alça. */
export function clampSpot(spot: Spot, size: { width: number; height: number }, view: { width: number; height: number }): Spot {
  return {
    x: Math.min(Math.max(0, spot.x), Math.max(0, view.width - size.width)),
    y: Math.min(Math.max(0, spot.y), Math.max(0, view.height - size.height)),
  }
}

function floats(card: HTMLElement): boolean {
  return getComputedStyle(card).position === 'fixed'
}

/** O último lugar de cada cartão, para o remendo que apagou o `style` o reaver sem medir nada (`restoreQuick`). */
const lastSpot = new Map<string, Spot>()

function put(card: HTMLElement, spot: Spot): void {
  card.style.setProperty('--float-x', `${spot.x}px`)
  card.style.setProperty('--float-y', `${spot.y}px`)
  if (card.id) lastSpot.set(card.id, spot)
}

function place(card: HTMLElement, spot: Spot): void {
  const box = card.getBoundingClientRect()
  put(card, clampSpot(spot, box, { width: window.innerWidth, height: window.innerHeight }))
}

/**
 * Devolve JÁ, no mesmo instante do remendo, o lugar e o recolhido que o remendo apagou, sem medir nada: a pilha
 * se refaz no quadro seguinte (`layout`), mas entre os dois o cartão não pode ficar no lugar padrão — nada pinta
 * ali, e mesmo assim quem mede a posição nesse intervalo a leria errada (o `attack-card-float.spec.ts` leu, na CI).
 */
function restoreQuick(storage: Storage): void {
  for (const card of document.querySelectorAll<HTMLElement>('[data-float]')) {
    applyFold(card, storage)
    const spot = card.id ? lastSpot.get(card.id) : undefined
    if (spot && !card.style.getPropertyValue('--float-x')) put(card, spot)
  }
}

/**
 * A CHAVE É POR CARTÃO: o ataque e o Estandarte flutuam ao mesmo tempo, e uma chave só os poria no
 * mesmo lugar, um em cima do outro. O ataque fica com a chave de antes, para quem já o arrastou não o
 * ver voltar à pilha.
 */
function keyOf(card: HTMLElement): string {
  return card.id === 'table-attack' || !card.id ? STORAGE_KEY : `${STORAGE_KEY}:${card.id}`
}

/**
 * O recolhido de um cartão fica guardado por cartão, como a posição. O cartão `data-float-fleeting` (o
 * ataque) guarda numa chave à parte, que vale só enquanto ELE existe: cada ataque é uma decisão nova, e
 * nascer recolhido o faria passar despercebido (decisão do dono).
 */
function foldKeyOf(card: HTMLElement): string {
  return `${card.hasAttribute('data-float-fleeting') ? FLEETING_FOLD_KEY : FOLD_KEY}:${card.id}`
}

/** Esquece o recolhido do cartão passageiro que já sumiu, para o próximo nascer aberto. */
function forgetGoneFleeting(storage: Storage): void {
  for (let i = storage.length - 1; i >= 0; i--) {
    const key = storage.key(i)
    if (!key?.startsWith(`${FLEETING_FOLD_KEY}:`)) continue
    const id = key.slice(FLEETING_FOLD_KEY.length + 1)
    if (!document.querySelector(`#${CSS.escape(id)}[data-float]`)) storage.removeItem(key)
  }
}

/** Aplica o recolhido guardado ao cartão que acabou de aparecer, e diz à seta em que estado ele está. */
function applyFold(card: HTMLElement, storage: Storage): void {
  if (storage.getItem(foldKeyOf(card)) && !card.hasAttribute('data-folded')) card.setAttribute('data-folded', '')
  card.querySelector('[data-float-fold]')?.setAttribute('aria-expanded', card.hasAttribute('data-folded') ? 'false' : 'true')
}

/** O chão da pilha: o topo da linha do turno, ou o pé da janela sem ela. */
function floorOf(): number {
  const floor = document.querySelector('[data-float-floor]')
  return floor ? floor.getBoundingClientRect().top : window.innerHeight
}

/**
 * Põe cada cartão no lugar: o arrastado onde foi posto, e os outros na PILHA de
 * baixo, à direita, na ordem da página — o último encostado no chão.
 */
function layout(storage: Storage): void {
  forgetGoneFleeting(storage)
  const docked: HTMLElement[] = []
  for (const card of document.querySelectorAll<HTMLElement>('[data-float]')) {
    applyFold(card, storage)
    if (!floats(card)) continue
    const spot = readSpot(storage.getItem(keyOf(card)))
    if (spot) place(card, spot)
    else docked.push(card)
  }
  let bottom = floorOf() - DOCK_EDGE
  for (const card of docked.reverse()) {
    const box = card.getBoundingClientRect()
    bottom -= box.height
    put(card, { x: Math.max(DOCK_EDGE, window.innerWidth - box.width - DOCK_EDGE), y: Math.max(DOCK_EDGE, bottom) })
    bottom -= DOCK_GAP
  }
}

/**
 * Liga o arrasto. Ele é DELEGADO no `document`, e não preso ao cartão: o nó do
 * cartão muda quando ele some e volta, e um ouvinte preso ao de agora ficaria
 * pendurado num nó descartado.
 */
export function attachFloatCard(storage: Storage = window.localStorage): void {
  let drag: { card: HTMLElement; dx: number; dy: number; pointer: number } | null = null

  document.addEventListener('pointerdown', (event) => {
    const handle = (event.target as Element | null)?.closest('[data-float-handle]')
    const card = handle?.closest<HTMLElement>('[data-float]')
    // Botão, campo e link dentro da cabeça continuam sendo o que são.
    if (!card || !floats(card) || (event.target as Element).closest('button, a, input')) return
    const box = card.getBoundingClientRect()
    drag = { card, dx: event.clientX - box.left, dy: event.clientY - box.top, pointer: event.pointerId }
    ;(handle as HTMLElement).setPointerCapture(event.pointerId)
    card.dataset.dragging = ''
    event.preventDefault()
  })

  document.addEventListener('pointermove', (event) => {
    if (!drag || event.pointerId !== drag.pointer) return
    place(drag.card, { x: event.clientX - drag.dx, y: event.clientY - drag.dy })
  })

  const drop = (event: PointerEvent) => {
    if (!drag || event.pointerId !== drag.pointer) return
    const box = drag.card.getBoundingClientRect()
    storage.setItem(keyOf(drag.card), JSON.stringify({ x: Math.round(box.left), y: Math.round(box.top) }))
    delete drag.card.dataset.dragging
    drag = null
  }
  document.addEventListener('pointerup', drop)
  document.addEventListener('pointercancel', drop)

  // A SETA recolhe e abre; a pilha se refaz, porque o cartão mudou de altura.
  document.addEventListener('click', (event) => {
    const fold = (event.target as Element | null)?.closest('[data-float-fold]')
    const card = fold?.closest<HTMLElement>('[data-float]')
    if (!card) return
    if (card.toggleAttribute('data-folded')) storage.setItem(foldKeyOf(card), '1')
    else storage.removeItem(foldKeyOf(card))
    layout(storage)
  })

  // UM LAYOUT POR QUADRO: o observador acorda a cada nó remendado em qualquer
  // lugar da página — o arrasto de uma peça remenda muito —, e medir os cartões
  // a cada um forçaria o leiaute repetidas vezes. O quadro roda antes da
  // pintura, então o cartão remendado ainda não aparece fora do lugar.
  let pending = 0
  const soon = () => {
    if (!pending) pending = requestAnimationFrame(() => {
      pending = 0
      layout(storage)
    })
  }
  // O TAMANHO de um cartão muda sem nó novo (o remendo troca o corpo, a seta o
  // recolhe), e a pilha tem de acompanhar: quem está em cima desce ou sobe.
  const sizes = new ResizeObserver(soon)
  const settle = () => {
    restoreQuick(storage)
    for (const card of document.querySelectorAll<HTMLElement>('[data-float]')) sizes.observe(card)
    soon()
  }
  // O CARTÃO NASCE pelo remendo do servidor: o observador fica no `body`, que
  // nunca é remendado, e não num nó que o próximo tique descarta.
  new MutationObserver(settle).observe(document.body, { subtree: true, childList: true })
  window.addEventListener('resize', soon)
  settle()
}
