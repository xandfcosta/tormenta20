import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

/**
 * Limiar de container arbitrário não entra no código (ALE-173, P8).
 *
 * Havia sete limiares em dezessete usos, quatro deles escritos à mão
 * (`@[28rem]`, `@[30rem]`, `@[34rem]`, `@[44rem]`) convivendo com os nomeados.
 * A medição mostrou que não eram sete decisões: todos respondem à mesma
 * pergunta — "cabe duas colunas nesta caixa?" — com números ligeiramente
 * diferentes, porque não havia escala de onde escolher.
 *
 * Este guarda LÊ o código-fonte em vez de montar uma tela, e é de propósito:
 * o defeito não é visual, é de vocabulário. Uma tela renderizada não sabe
 * dizer se o 30rem foi escolhido ou inventado; o arquivo sabe.
 */
function arquivosTsx(raiz: string): string[] {
  return readdirSync(raiz).flatMap((nome) => {
    const caminho = join(raiz, nome)
    if (statSync(caminho).isDirectory()) return arquivosTsx(caminho)
    return caminho.endsWith('.tsx') && !caminho.endsWith('.test.tsx') ? [caminho] : []
  })
}

describe('a escala de container', () => {
  it('não tem limiar arbitrário', () => {
    const infratores = arquivosTsx(join(import.meta.dirname, '../..'))
      .flatMap((caminho) => {
        const achados = readFileSync(caminho, 'utf8').match(/@\[[\d.]+rem\]:/g) ?? []
        return achados.map((a) => `${caminho.split('/src/')[1]}: ${a}`)
      })

    expect(
      infratores,
      'limiar de container escrito à mão — use a escala nomeada (@sm @md @lg @xl @2xl)',
    ).toEqual([])
  })
})
