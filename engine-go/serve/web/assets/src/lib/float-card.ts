/**
 * O CARTÃO QUE FLUTUA no desktop (ALE-536): o do ataque deixa de empurrar as
 * superfícies para baixo e paira sobre a mesa, e a cabeça dele é a alça de
 * arrastar. No celular ele continua no fluxo — quem decide é o CSS, pela
 * largura; aqui só se arrasta quando o cartão está de fato `position: fixed`.
 *
 * # Por que a posição sobrevive ao servidor
 *
 * O stream remenda o cartão a cada mudança da mesa, e o remendo reescreve os
 * atributos para o que o servidor mandou — um `style` posto aqui sumiria no
 * primeiro tique. O cartão declara `data-preserve-attr="style"`, e o Datastar
 * deixa o atributo como está. Quando o cartão SOME e volta (outro ataque), o nó
 * é outro; a posição vem do armazenamento deste aparelho.
 */

const STORAGE_KEY = 't20-float-card'

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

function place(card: HTMLElement, spot: Spot): void {
  const box = card.getBoundingClientRect()
  const inside = clampSpot(spot, box, { width: window.innerWidth, height: window.innerHeight })
  card.style.setProperty('--float-x', `${inside.x}px`)
  card.style.setProperty('--float-y', `${inside.y}px`)
}

/** Põe na posição guardada o cartão que acabou de aparecer sem uma. */
function restore(card: HTMLElement, storage: Storage): void {
  if (card.style.getPropertyValue('--float-x') || !floats(card)) return
  const spot = readSpot(storage.getItem(STORAGE_KEY))
  if (spot) place(card, spot)
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
    storage.setItem(STORAGE_KEY, JSON.stringify({ x: Math.round(box.left), y: Math.round(box.top) }))
    delete drag.card.dataset.dragging
    drag = null
  }
  document.addEventListener('pointerup', drop)
  document.addEventListener('pointercancel', drop)

  const restoreAll = () => {
    for (const card of document.querySelectorAll<HTMLElement>('[data-float]')) restore(card, storage)
  }
  // O CARTÃO NASCE pelo remendo do servidor: o observador fica no `body`, que
  // nunca é remendado, e não num nó que o próximo tique descarta.
  new MutationObserver(restoreAll).observe(document.body, { subtree: true, childList: true })
  window.addEventListener('resize', () => {
    for (const card of document.querySelectorAll<HTMLElement>('[data-float]')) {
      const spot = readSpot(storage.getItem(STORAGE_KEY))
      if (spot && floats(card)) place(card, spot)
    }
  })
  restoreAll()
}
