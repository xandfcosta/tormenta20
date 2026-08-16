import { createRoot } from 'solid-js'
import { describe, expect, it } from 'vitest'
import { createBoardViewport, isVisible } from './board-viewport'

/**
 * A janela sobre o plano infinito (ALE-124). O que se prova aqui é o que a mesa
 * notaria: a vista nasce mostrando o meio e não a quina, o zoom não joga a cena
 * para o canto, e "centralizar" acha o grupo esteja ele onde estiver.
 *
 * Unitário e não integração porque isto é geometria pura — e porque em jsdom
 * NADA mede: a região não tem tamanho, então a única forma de exercitar a
 * medição é chamá-la com números.
 */
const PECA = { id: 't1', label: 'Ogro', x: 0, y: 0, footprint: 1, kind: 'npc' as const }

function withViewport(run: (view: ReturnType<typeof createBoardViewport>) => void) {
  createRoot((dispose) => {
    run(createBoardViewport())
    dispose()
  })
}

describe('a janela do tabuleiro', () => {
  it('nasce centrada na origem, não com a origem na quina', () => {
    withViewport((view) => {
      view.measure(880, 440) // 20 × 10 quadrados de 44px

      expect(view.cols()).toBe(20)
      expect(view.rows()).toBe(10)
      // Origem no MEIO: metade da janela para cada lado do quadrado (0,0).
      expect(view.originX()).toBe(-10)
      expect(view.originY()).toBe(-5)
    })
  })

  // Redimensionar a janela do browser não pode arrastar a vista de volta para a
  // origem — quem estava olhando a briga continua olhando a briga. O que fica
  // parado é o CENTRO: a origem muda por definição quando a janela encolhe.
  it('redimensionar mantém o que estava no meio da tela', () => {
    withViewport((view) => {
      view.measure(880, 440)
      view.pan(30, 0)
      const centroAntes = view.originX() + view.cols() / 2

      view.measure(660, 440)

      // Meio quadrado de arredondamento é o erro máximo aceitável: a origem é
      // inteira, e metade de uma janela ímpar não é.
      expect(Math.abs(view.originX() + view.cols() / 2 - centroAntes)).toBeLessThanOrEqual(0.5)
      // E continua longe da origem: a vista não foi roubada de volta.
      expect(view.originX()).toBeGreaterThan(10)
    })
  })

  // O zoom sozinho não move nada: a janela é medida em QUADRADOS, e quem muda
  // quantos cabem é a remedição que o ResizeObserver dispara logo depois. Este
  // teste imita essa sequência, que é a que acontece na tela.
  it('afastar não joga a cena para o canto', () => {
    withViewport((view) => {
      view.measure(880, 440)
      view.centerOn(100, 100)

      view.zoom(-8)
      view.measure(880, 440)

      // O centro continua sendo (100,100): a diferença entre o centro da janela
      // e o alvo não passa de meio quadrado de arredondamento.
      const centroX = view.originX() + view.cols() / 2
      const centroY = view.originY() + view.rows() / 2
      expect(Math.abs(centroX - 100)).toBeLessThanOrEqual(1)
      expect(Math.abs(centroY - 100)).toBeLessThanOrEqual(1)
    })
  })

  it('centralizar enquadra as peças longe da origem', () => {
    withViewport((view) => {
      view.measure(880, 440)

      view.fit([{ ...PECA, x: 200, y: -80 }])

      expect(isVisible({ ...PECA, x: 200, y: -80 }, {
        originX: view.originX(), originY: view.originY(), cols: view.cols(), rows: view.rows(),
      })).toBe(true)
    })
  })

  // Sem peça nenhuma, "centralizar" volta para a origem: é o único lugar
  // combinado num plano que não tem centro.
  it('sem peças, centralizar volta para a origem', () => {
    withViewport((view) => {
      view.measure(880, 440)
      view.pan(50, 50)

      view.fit([])

      expect(view.originX()).toBe(-10)
      expect(view.originY()).toBe(-5)
    })
  })
})
