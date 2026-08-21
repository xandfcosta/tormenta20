import { describe, expect, it } from 'vitest'
import { type CatalogResultRow, catalogVisualRows } from './catalog-model'

const cabecalho = (label: string): CatalogResultRow => ({
  kind: 'header',
  key: `h.${label}`,
  label,
  count: 0,
})
const item = (id: string): CatalogResultRow =>
  ({ kind: 'condition', key: `c.${id}`, value: { id } }) as CatalogResultRow

/**
 * O agrupamento em colunas da busca de catálogos (ALE-170).
 *
 * A lista é virtualizada, então colunas não são grade de CSS: são os dados
 * agrupados antes de entrar na lista. Isso é regra, e as duas metades dela
 * quebram de jeitos diferentes se erradas.
 */
describe('catalogVisualRows', () => {
  it('põe N resultados por fileira', () => {
    const fileiras = catalogVisualRows([item('a'), item('b'), item('c')], 2)

    expect(fileiras).toHaveLength(2)
    expect(fileiras[0]).toMatchObject({ kind: 'cells' })
    expect((fileiras[0] as { cells: unknown[] }).cells).toHaveLength(2)
    expect((fileiras[1] as { cells: unknown[] }).cells).toHaveLength(1)
  })

  it('o cabeçalho fica sozinho na fileira dele', () => {
    const fileiras = catalogVisualRows([cabecalho('Condições'), item('a'), item('b')], 3)

    expect(fileiras[0]).toMatchObject({ kind: 'header', label: 'Condições' })
    expect(fileiras).toHaveLength(2)
  })

  /**
   * Sem reiniciar, a última condição dividiria a fileira com a primeira magia,
   * e o mestre leria duas coisas de catálogos diferentes como se fossem irmãs.
   */
  it('reinicia a contagem a cada grupo', () => {
    const fileiras = catalogVisualRows(
      [cabecalho('Condições'), item('a'), cabecalho('Magias'), item('b')],
      3,
    )

    expect(fileiras.map((f) => f.kind)).toEqual(['header', 'cells', 'header', 'cells'])
  })

  it('uma coluna devolve uma fileira por resultado', () => {
    expect(catalogVisualRows([item('a'), item('b')], 1)).toHaveLength(2)
  })
})
