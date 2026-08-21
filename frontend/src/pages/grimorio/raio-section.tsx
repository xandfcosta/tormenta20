import { RadiusSwatch, SpecBlock, SpecSection } from './spec-primitives'

/**
 * A ladeira do raio, desenhada — e é a seção que existe por um defeito.
 *
 * A cena redefine `--radius`, e a escala do shadcn é derivada dele por
 * `sm = R−4, md = R−2, lg = R, xl = R+4`. Com o R antigo, de 4px, a fórmula
 * DEGENERAVA: `sm` caía em zero e passava a significar "quadrado", que é
 * trabalho do `rounded-none`. Ninguém conseguia prever, lendo o TSX, se
 * `rounded-sm` ia desenhar canto (ALE-173, P7).
 *
 * Por isso cada amostra mostra o pixel que o navegador resolveu, e não um
 * número que alguém digitou aqui.
 */

const ESCALA = [
  { classe: 'rounded-none', nome: 'rounded-none' },
  { classe: 'rounded-sm', nome: 'rounded-sm' },
  { classe: 'rounded-md', nome: 'rounded-md' },
  { classe: 'rounded-lg', nome: 'rounded-lg' },
  { classe: 'rounded-xl', nome: 'rounded-xl' },
  { classe: 'rounded-full', nome: 'rounded-full' },
]

/**
 * Raios que NÃO pertencem à escala, e não devem ser alinhados a ela: são a
 * medida de um OBJETO. Uma capa de livro é fisicamente mais redonda que a
 * página dentro dela, e forçar as duas ao mesmo degrau desenharia um objeto que
 * não existe.
 */
const OBJETOS = [
  { classe: 'grimorio-frame', nome: 'moldura' },
  { classe: 'grimorio-leather', nome: 'capa do livro' },
  { classe: 'rounded-[3px]', nome: 'peça do tabuleiro' },
]

export function RaioSection() {
  return (
    <SpecSection id="raio" titulo="Raio">
      <SpecBlock
        titulo="A escala"
        nota="Ferro forjado não tem canto macio: o padrão da cena é quadrado, e quadrado se escreve rounded-none. Os dois degraus de cima existem para o que QUISER ser mais macio."
      >
        {ESCALA.map((r) => (
          <RadiusSwatch classe={r.classe} nome={r.nome} />
        ))}
      </SpecBlock>

      <SpecBlock
        titulo="Raios de objeto"
        nota="Estes vêm cravados no CSS de propósito e não são violação da escala — não os alinhe."
      >
        {OBJETOS.map((r) => (
          <RadiusSwatch classe={r.classe} nome={r.nome} />
        ))}
      </SpecBlock>
    </SpecSection>
  )
}
