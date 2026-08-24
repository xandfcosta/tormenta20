import { expect, test } from '@playwright/test'
import { VIEWPORTS } from './support/viewports'

/**
 * O CABEÇALHO DA SESSÃO É UM SÓ, E ELE CABE (ALE-201).
 *
 * Eram DUAS faixas empilhadas — a do app e a do "Modo Jogo" — dizendo coisas do
 * mesmo assunto. A de baixo custava ~38px de uma cena que no celular deitado
 * tem 390 de altura, e a ALE-146 já tinha medido que cada faixa a menos é uma
 * linha de combatente a mais.
 *
 * Este guarda prende as DUAS metades da mudança, porque uma sem a outra não é o
 * que se quis:
 *
 *  1. Há UM cabeçalho, e o que a faixa de baixo dizia está nele.
 *  2. Ele CABE nos seis formatos — o teste de transbordo já pegou uma vez, com
 *     o `· Sessão 4` sendo pintado para fora do pai a 390px porque repetia o
 *     que o título ao lado já dizia.
 *
 * E2E e não teste de componente porque o que se mede é LEIAUTE REAL em seis
 * larguras: jsdom mede zero e passaria verde sobre um cabeçalho estourado.
 */
test.describe.configure({ mode: 'serial' })

test.describe('A sessão tem UMA faixa de cabeçalho', () => {
  test('o cabeçalho carrega o estado ao vivo, e não há segunda faixa', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')

    const cabecalho = page.locator('.scene-grimorio header')
    await expect(cabecalho).toBeVisible()

    // O que a faixa de baixo dizia agora vive aqui.
    await expect(cabecalho.getByText('Ao vivo')).toBeVisible()

    // A saída perdeu o texto e ficou só o ícone — é isso que abre a largura
    // para o estado ao vivo caber na mesma linha.
    const sair = cabecalho.getByRole('link', { name: 'Sair da sessão' })
    await expect(sair).toBeVisible()
    await expect(sair).not.toContainText('Sair da sessão')

    // E não sobrou uma segunda faixa com o mesmo assunto: fora do cabeçalho,
    // ninguém mais anuncia "Ao vivo".
    const foraDoCabecalho = page.locator('.scene-grimorio > div', { hasText: 'Ao vivo' })
    await expect(foraDoCabecalho).toHaveCount(0)
  })

  test('o cabeçalho cabe nos seis formatos, sem pintar para fora', async ({ page }) => {
    await page.goto('/campaigns/1/sessions/4')
    const cabecalho = page.locator('.scene-grimorio header')
    await expect(cabecalho).toBeVisible()

    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })

      const transbordo = await cabecalho.evaluate((el) => {
        const pai = el.getBoundingClientRect()
        return [...el.querySelectorAll('*')]
          .filter((filho) => {
            const r = filho.getBoundingClientRect()
            if (r.width === 0 || r.height === 0) return false
            // 1px de folga: arredondamento de subpixel não é transbordo.
            return r.right > pai.right + 1 || r.left < pai.left - 1
          })
          .map((filho) => (filho.textContent ?? '').trim().slice(0, 24))
      })

      expect(transbordo, `${viewport.name}: pintado para fora do cabeçalho`).toEqual([])
    }
  })
})
