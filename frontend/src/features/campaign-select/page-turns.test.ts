import { createRoot, createSignal } from 'solid-js'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Campaign } from '@/shared/api/api'
import { type Page, createPageTurns } from './page-turns'

function campaign(id: number): Campaign {
  return {
    id,
    ownerId: 1,
    name: `Campanha ${id}`,
    description: null,
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
  }
}

const page = (id: number): Page => ({ campaign: campaign(id), isLive: false })

/** Desktop + motion allowed, unless a test says otherwise. */
function mockViewport({ wide = true, reduced = false } = {}) {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: media.includes('prefers-reduced-motion') ? reduced : wide,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

afterEach(() => vi.restoreAllMocks())

describe('createPageTurns', () => {
  it('começa mostrando o alvo, sem virada', () => {
    mockViewport()
    createRoot((dispose) => {
      const [target] = createSignal(page(1))
      const turns = createPageTurns(target, () => [1, 2, 3])
      expect(turns.shown().campaign.id).toBe(1)
      expect(turns.turn()).toBeNull()
      dispose()
    })
  })

  it('uma escolha vira a página pra frente', () => {
    mockViewport()
    createRoot((dispose) => {
      const [target, setTarget] = createSignal(page(1))
      const turns = createPageTurns(target, () => [1, 2, 3])

      setTarget(page(2))

      expect(turns.turn()?.from.campaign.id).toBe(1)
      expect(turns.turn()?.to.campaign.id).toBe(2)
      expect(turns.turn()?.dir).toBe(1)
      expect(turns.shown().campaign.id).toBe(2)
      dispose()
    })
  })

  // A direção vem da ordem dos marcadores, não da ordem de clique.
  it('escolher um marcador anterior vira pra trás', () => {
    mockViewport()
    createRoot((dispose) => {
      const [target, setTarget] = createSignal(page(3))
      const turns = createPageTurns(target, () => [1, 2, 3])

      setTarget(page(1))

      expect(turns.turn()?.dir).toBe(-1)
      dispose()
    })
  })

  // O ponto da fila: escolhas feitas DURANTE a animação não cortam nem
  // reiniciam — todas tocam, em ordem.
  it('enfileira escolhas feitas durante a virada e toca uma por vez', () => {
    mockViewport()
    createRoot((dispose) => {
      const [target, setTarget] = createSignal(page(1))
      const turns = createPageTurns(target, () => [1, 2, 3, 4])

      setTarget(page(2))
      setTarget(page(3))
      setTarget(page(4))

      // A primeira virada já estava no ar quando as outras chegaram: começou
      // com a fila vazia, então corre em ritmo normal.
      expect(turns.turn()?.to.campaign.id).toBe(2)
      expect(turns.turn()?.fast).toBe(false)

      // As que esperaram correm rápido, pra fila drenar…
      turns.finishTurn()
      expect(turns.turn()?.to.campaign.id).toBe(3)
      expect(turns.turn()?.fast).toBe(true)

      // …menos a última, que assenta em ritmo normal.
      turns.finishTurn()
      expect(turns.turn()?.to.campaign.id).toBe(4)
      expect(turns.turn()?.fast).toBe(false)

      turns.finishTurn()
      expect(turns.turn()).toBeNull()
      expect(turns.shown().campaign.id).toBe(4)
      dispose()
    })
  })

  // Regressão do bug que a main tem (ALE-78): o mapa de sessões ao vivo chega
  // DEPOIS da lista, então o alvo muda só no `isLive`. Sem isso, o livro fica
  // dizendo "Abrir campanha" enquanto o rail já mostra a brasa.
  it('atualiza no lugar quando só o isLive muda, sem virar página', () => {
    mockViewport()
    createRoot((dispose) => {
      const [target, setTarget] = createSignal<Page>({ campaign: campaign(1), isLive: false })
      const turns = createPageTurns(target, () => [1, 2])

      setTarget({ campaign: campaign(1), isLive: true })

      expect(turns.turn()).toBeNull() // nada de animação
      expect(turns.shown().isLive).toBe(true)
      dispose()
    })
  })

  it('não enfileira a mesma campanha duas vezes seguidas', () => {
    mockViewport()
    createRoot((dispose) => {
      const [target, setTarget] = createSignal(page(1))
      const turns = createPageTurns(target, () => [1, 2])

      setTarget(page(2))
      setTarget(page(2))
      turns.finishTurn()

      expect(turns.turn()).toBeNull() // nada sobrando na fila
      dispose()
    })
  })

  // Em telefone e sob reduced-motion a troca é instantânea: sem folha virando.
  it('no telefone troca direto, sem virada', () => {
    mockViewport({ wide: false })
    createRoot((dispose) => {
      const [target, setTarget] = createSignal(page(1))
      const turns = createPageTurns(target, () => [1, 2])

      setTarget(page(2))

      expect(turns.turn()).toBeNull()
      expect(turns.shown().campaign.id).toBe(2)
      dispose()
    })
  })

  it('respeita prefers-reduced-motion (WCAG 2.3.3)', () => {
    mockViewport({ reduced: true })
    createRoot((dispose) => {
      const [target, setTarget] = createSignal(page(1))
      const turns = createPageTurns(target, () => [1, 2])

      setTarget(page(2))

      expect(turns.turn()).toBeNull()
      expect(turns.shown().campaign.id).toBe(2)
      dispose()
    })
  })
})
