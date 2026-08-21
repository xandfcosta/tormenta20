import { render, screen } from '@solidjs/testing-library'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Tooltip, TooltipContent, TooltipTrigger } from './tooltip'

/**
 * O atraso de abertura é da CASA, não de quem chama.
 *
 * O arquivo do kit pedia `openDelay={150}` a cada chamador — o valor que o
 * `TooltipProvider` do React aplicava na raiz — e nenhum dos três passou. Desde
 * o cutover para Solid todo tooltip do app abria com os 700ms padrão do
 * Kobalte, quase cinco vezes o pretendido, e ninguém notou porque a tela não
 * quebra: ela só fica lenta.
 *
 * O teste existe porque uma nota pedindo não segura nada. Com relógio falso,
 * porque o que se afirma é QUANDO.
 */
function montarTooltip() {
  render(() => (
    <Tooltip>
      <TooltipTrigger>gatilho</TooltipTrigger>
      <TooltipContent>a explicação</TooltipContent>
    </Tooltip>
  ))
  return screen.getByText('gatilho')
}

describe('Tooltip — o atraso da casa', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('abre em 150ms, e não nos 700 do padrão do Kobalte', async () => {
    vi.useFakeTimers()
    const gatilho = montarTooltip()

    gatilho.dispatchEvent(new PointerEvent('pointerenter', { bubbles: true }))
    gatilho.dispatchEvent(new PointerEvent('pointermove', { bubbles: true }))

    await vi.advanceTimersByTimeAsync(149)
    expect(screen.queryByText('a explicação')).not.toBeInTheDocument()

    await vi.advanceTimersByTimeAsync(2)
    expect(screen.getByText('a explicação')).toBeInTheDocument()
  })
})
