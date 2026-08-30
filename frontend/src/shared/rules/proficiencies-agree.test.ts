import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { CLASS_PROFICIENCIES } from './proficiencies'

/**
 * A tabela de proficiência por classe existe em DUAS cópias enquanto a migração
 * para o Datastar não termina: esta, em TypeScript, que a ficha da SPA usa, e o
 * campo `proficiencies` de `engine-go/catalog/data/classes.json`, que a ficha
 * nova usa (ALE-272, fatia 2).
 *
 * As duas estão vivas ao mesmo tempo e ninguém comparava. Uma divergência não
 * levanta erro em lugar nenhum — ela faz o MESMO personagem aparecer proficiente
 * numa ficha e não na outra, e o motor aplicar a penalidade da p142 em metade
 * das telas.
 *
 * Este teste morre junto com a SPA, na última fatia da ALE-272. Até lá ele é a
 * costura, e é o mesmo desenho do `TestDumpAgreesWithEmbeddedCatalog` do Go: não
 * valida CONTEÚDO (quem faz isso é o `TestClassProficienciesTable`, contra o
 * livro) — garante que as duas cópias são a mesma coisa.
 */
describe('as duas cópias da tabela de proficiências', () => {
  const doCatalogo = () => {
    const caminho = resolve(
      __dirname,
      '../../../../engine-go/catalog/data/classes.json',
    )
    const classes: { name: string; proficiencies: string[] }[] = JSON.parse(
      readFileSync(caminho, 'utf8'),
    )
    return Object.fromEntries(classes.map((c) => [c.name, [...c.proficiencies].sort()]))
  }

  it('concordam classe a classe', () => {
    const doTS = Object.fromEntries(
      Object.entries(CLASS_PROFICIENCIES).map(([nome, lista]) => [nome, [...lista].sort()]),
    )
    expect(doTS).toEqual(doCatalogo())
  })
})
