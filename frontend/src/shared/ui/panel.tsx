import { type ComponentProps, type ValidComponent, splitProps } from 'solid-js'
import { Dynamic } from 'solid-js/web'
import { cn } from '@/shared/lib/utils'

/**
 * A caixa LISA da cena: borda de ferro de 1px, fundo chapado, canto quadrado
 * (ALE-173, P5).
 *
 * Ela e o `FramedPanel` não são a mesma coisa, e a issue tinha razão nisso — o
 * problema era que só uma tinha nome. O `FramedPanel` é a MOLDURA: borda de
 * ferro mais um filete dourado por dentro, sombra de assento e gradiente. Ele
 * desenha um objeto sobre a mesa, e por isso é caro de repetir. Esta aqui é a
 * superfície de trabalho: onde a informação é densa e a caixa precisa sumir
 * para o conteúdo aparecer.
 *
 * Enquanto a lisa era anônima, cada autor a re-derivava, e elas derivavam:
 * onze escreviam a receita canônica e as outras variavam a borda sem que a
 * diferença significasse nada. As que ficaram de fora significam: borda só no
 * topo ou só na base é DIVISOR e não caixa, e `border-2` e `border-dashed`
 * são ênfase deliberada.
 *
 * Sem respiro embutido, ao contrário do `FramedPanel`: o preenchimento varia
 * com a densidade de cada painel, e um padrão aqui só seria sobrescrito.
 *
 * `fillHeight` é a segunda metade do achado. Seis painéis de aba da ficha
 * repetiam não só a caixa, mas as MESMAS seis classes de arranjo depois dela —
 * `flex h-full min-h-0 flex-1 flex-col overflow-hidden`. Isso não é estilo, é
 * um comportamento com nome que a casa já usa noutro lugar (o `fillHeight` do
 * cartão de iniciativa): ocupo a altura que recebo e rolo por dentro, em vez
 * de empurrar a página.
 *
 * @example <Panel class="p-3 space-y-2">…</Panel>
 * @example <Panel as="section" fillHeight aria-labelledby="mochila">…</Panel>
 */
export function Panel(props: ComponentProps<'div'> & { as?: ValidComponent; fillHeight?: boolean }) {
  const [local, rest] = splitProps(props, ['as', 'class', 'children', 'fillHeight'])
  return (
    <Dynamic
      component={local.as ?? 'div'}
      data-slot="panel"
      class={cn(
        'rounded-none border border-grimorio-iron bg-grimorio-panel',
        local.fillHeight && 'flex h-full min-h-0 flex-1 flex-col overflow-hidden',
        local.class,
      )}
      {...rest}
    >
      {local.children}
    </Dynamic>
  )
}
