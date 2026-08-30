import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { ITEM_FLAG_LABEL } from '@/entities/character/effect-source'

/**
 * As DUAS cópias dos rótulos de flag dizem a mesma coisa (ALE-272, fatia 5).
 *
 * A ficha em Datastar desenha "Sempre ativos (itens equipados)" no servidor, e
 * para isso o Go ganhou um `itemFlagLabel` com as mesmas seis frases que o
 * `ITEM_FLAG_LABEL` daqui. Enquanto a SPA viver há duas cópias, e duas cópias
 * divergem — a primeira versão do painel novo mostrava `cannot-apply-dex-to-defense`
 * cru na tela porque a tradução simplesmente não existia do lado do Go.
 *
 * Este teste MORRE COM A SPA, e é para morrer: quando `frontend/` sair, o Go
 * fica sendo a única cópia e não há o que comparar. Ele é a rede da convivência,
 * não uma regra permanente.
 *
 * Lê o Go como TEXTO de propósito. Importar não é possível daqui, e um `expect`
 * por frase escrito à mão seria uma TERCEIRA cópia — a que ninguém lembraria de
 * atualizar.
 */
describe('os rótulos de flag de item', () => {
  const goSource = readFileSync(
    resolve(__dirname, '../../../../engine-go/api/piloto_ficha_effects.go'),
    'utf8',
  )

  const goLabels = (): Record<string, string> => {
    const bloco = goSource.match(/var itemFlagLabel = map\[string\]string\{([\s\S]*?)\n\}/)
    if (!bloco) throw new Error('não achei o `itemFlagLabel` no Go — o mapa foi renomeado?')
    const achados: Record<string, string> = {}
    for (const linha of bloco[1].matchAll(/"([^"]+)":\s*"([^"]+)"/g)) {
      achados[linha[1] as string] = linha[2] as string
    }
    return achados
  }

  it('dizem as mesmas frases nos dois lados', () => {
    const doGo = goLabels()
    // CONTROLE: sem ele, um regex que parasse de casar daria `{}` e o
    // `toEqual` compararia dois vazios — verde sobre nada.
    expect(Object.keys(doGo).length, 'o regex não colheu nenhum rótulo do Go').toBeGreaterThan(4)
    expect(doGo).toEqual(ITEM_FLAG_LABEL)
  })
})
