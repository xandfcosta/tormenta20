/**
 * As PEÇAS DA SPA como elementos customizados, para a folha de especificação
 * mostrar os dois stacks lado a lado (ALE-251).
 *
 * Esta é a primeira ilha em `solid-element` do projeto, e o lugar foi escolhido
 * por ser o de menor risco: ninguém JOGA nesta tela. Quando a ficha precisar de
 * ilha de verdade — e ela vai —, o mecanismo já terá rodado em produção.
 *
 * ## `noShadowDOM()` não é opcional aqui, é o que faz a página funcionar
 *
 * O padrão do `solid-element` é montar num shadow root, e ali dentro as classes
 * do Tailwind NÃO alcançam: a folha de estilo mora no documento, não no shadow.
 * As peças renderizariam sem estilo nenhum — numa página cujo trabalho inteiro
 * é mostrar como elas são. As variáveis CSS atravessariam (custom property
 * herda através do shadow), então o resultado seria pior que quebrado: seria
 * parcialmente certo, com as cores no lugar e a forma não.
 *
 * ## Por que envolver em vez de reimplementar
 *
 * A regra nº 2 do grimório é que ele usa as peças DE VERDADE. Uma imitação com
 * as mesmas classes mentiria no primeiro dia em que alguém mexesse no original
 * — e o objetivo desta seção é justamente flagrar divergência entre os dois
 * stacks. Uma cópia não flagraria nada: ela concordaria consigo mesma.
 */
import { customElement, noShadowDOM } from 'solid-element'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { FramedPanel } from '@/shared/ui/framed-panel'
import { VitalBar } from '@/shared/ui/vital-bar'

type VarianteDeBotao = 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive' | 'link'
type TamanhoDeBotao = 'default' | 'sm' | 'lg' | 'icon'
type VarianteDeChip = 'default' | 'secondary' | 'outline' | 'destructive'

customElement('spa-botao', { variante: 'default', tamanho: 'default', texto: 'Abrir' }, (props) => {
  noShadowDOM()
  return (
    <Button variant={props.variante as VarianteDeBotao} size={props.tamanho as TamanhoDeBotao}>
      {props.texto}
    </Button>
  )
})

customElement('spa-chip', { variante: 'default', texto: 'ativo' }, (props) => {
  noShadowDOM()
  return <Badge variant={props.variante as VarianteDeChip}>{props.texto}</Badge>
})

customElement('spa-campo', { placeholder: '', desabilitado: false }, (props) => {
  noShadowDOM()
  return <Input placeholder={props.placeholder} disabled={props.desabilitado} />
})

customElement('spa-painel', { titulo: '' }, (props) => {
  noShadowDOM()
  return <FramedPanel title={props.titulo}>conteúdo</FramedPanel>
})

// Os atributos chegam como STRING: `atual="42"` é o texto "42", e passá-lo cru
// faria a barra calcular `"42" / "57"` — que em JS é 0.736..., por acaso certo,
// e depois `Math.min` com string quebraria em silêncio noutro caminho. O
// `Number()` é a fronteira entre o HTML e o componente.
customElement('spa-barra-vital', { tipo: 'hp', rotulo: 'Vida', atual: '42', max: '57' }, (props) => {
  noShadowDOM()
  return (
    <VitalBar
      kind={props.tipo as 'hp' | 'mp'}
      label={props.rotulo}
      current={Number(props.atual)}
      max={Number(props.max)}
    />
  )
})
