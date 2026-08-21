import { type VariantProps, cva } from 'class-variance-authority'
import { Dynamic } from 'solid-js/web'
import { type ComponentProps, type JSX, type ValidComponent, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * A família de rótulos em CAIXA ALTA da casa (ALE-173, P2).
 *
 * Contei 208 ocorrências escritas de 59 jeitos diferentes, e a descoberta que
 * decidiu este arquivo é que elas **não são 59 variações da mesma coisa**: são
 * TRÊS papéis distintos, cada um escrito de vinte jeitos porque não havia de
 * onde copiar. O resto das grafias é deriva — alguém precisou de um cabeçalho,
 * escolheu `tracking-wide` em vez de `tracking-[0.16em]`, e a diferença não
 * significava nada.
 *
 * Os três, medidos no código como ele estava:
 *
 * - `SectionTitle` — o título de uma seção da cena, Cinzel grande e dourado.
 *   20 usos, todos dourados, nenhum apagado.
 * - `SectionLabel` — o cabeçalho de um BLOCO, Cinzel 11px. 17 usos, 15 em `<p>`
 *   e 2 em `<h4>`, apagado por padrão e dourado quando o bloco é o assunto.
 * - `FieldLabel` — o rótulo colado num VALOR, sem Cinzel, 10px, espaçamento
 *   mais largo. 60 usos, 37 em `<span>` — é o "FOR" ao lado do 16.
 *
 * **Um componente só para os três seria pior que as 59 grafias**, porque
 * passaria a esconder que são coisas diferentes: o `FieldLabel` não é
 * cabeçalho de nada, e vesti-lo de Cinzel mudaria a cara da ficha em 60
 * lugares. Eles compartilham o `cva` e nada mais.
 *
 * O elemento é escolha de quem chama (`as`), porque a semântica é do SÍTIO e
 * não da aparência: o mesmo desenho é `<legend>` dentro de um `<fieldset>`,
 * `<label>` ao lado de um campo e `<span>` ao lado de um número. E nenhum deles
 * é `aria-hidden` por padrão — medido, só 3 dos 60 são, e sempre porque o valor
 * ao lado já anuncia o conjunto numa linha `sr-only`.
 */
const rotuloVariants = cva('uppercase', {
  variants: {
    papel: {
      titulo: 'font-heading text-lg',
      secao: 'font-heading text-2xs tracking-[0.16em]',
      campo: 'text-3xs tracking-widest',
    },
    /**
     * O espaçamento do TÍTULO depende de onde ele está (ALE-173, P2). O código
     * já dizia isso antes de alguém decidir: `tracking-wide` estava em painel
     * de tela densa — Mochila, Perícias, Grimório, iniciativa — e `[0.16em]`
     * em passo de forja, ferramenta do /gm e na porta. A divisão é limpa
     * demais para ser deriva.
     *
     * A razão é de leitura: um passo de cena é o único assunto da tela e o
     * título pode respirar; um cabeçalho de painel disputa espaço com nove
     * outros, e apertar é o que o mantém legível ao lado dos vizinhos. Medido,
     * 0,45px contra 2,88px no mesmo tamanho.
     */
    contexto: {
      cena: '',
      painel: '',
    },
    tom: {
      gold: 'text-grimorio-gold',
      muted: 'text-muted-foreground',
      inherit: '',
    },
  },
  // COMPOSTA, e não solta: `contexto` só vale para o TÍTULO. Como variante
  // solta ela emitia um `tracking` em cima do que o papel já define, e o
  // `FieldLabel` — que é `tracking-widest` — saía com o espaçamento do título,
  // porque o último a entrar na string vence o merge. Peguei medindo o que
  // cada variante emite, não lendo.
  compoundVariants: [
    { papel: 'titulo', contexto: 'cena', class: 'tracking-[0.16em]' },
    { papel: 'titulo', contexto: 'painel', class: 'tracking-wide' },
  ],
  defaultVariants: { papel: 'secao', tom: 'muted', contexto: 'cena' },
})

export type RotuloTom = NonNullable<VariantProps<typeof rotuloVariants>['tom']>
export type RotuloContexto = NonNullable<VariantProps<typeof rotuloVariants>['contexto']>

type RotuloProps = Omit<ComponentProps<'span'>, 'style'> & {
  /**
   * O elemento que carrega a semântica do SÍTIO — ou um componente, porque no
   * kit o título de um painel é a peça do Kobalte que registra o rótulo do
   * diálogo, e ela não pode ser trocada por uma tag solta.
   */
  as?: ValidComponent
  tom?: RotuloTom
  /** Só o `SectionTitle` olha: apertado em painel denso, folgado em passo de cena. */
  contexto?: RotuloContexto
  /** Só para cor que o CSS não sabe de antemão — a barra vital pinta o rótulo
   *  com a mesma variável do preenchimento, que muda com o valor. */
  style?: JSX.CSSProperties | string
}

function Rotulo(props: RotuloProps & { papel: 'titulo' | 'secao' | 'campo'; padrao: ValidComponent }) {
  // O resto dos atributos passa direto: um rótulo vira `title` de dica, `for`
  // de campo e `aria-*` conforme o sítio, e enumerá-los aqui só adiaria o
  // próximo que faltasse.
  const [local, rest] = splitProps(props, [
    'as',
    'tom',
    'contexto',
    'class',
    'papel',
    'padrao',
    'children',
  ])
  return (
    <Dynamic
      component={local.as ?? local.padrao}
      class={cn(
        rotuloVariants({ papel: local.papel, tom: local.tom, contexto: local.contexto }),
        local.class,
      )}
      {...rest}
    >
      {local.children}
    </Dynamic>
  )
}

/**
 * O título de uma seção da cena — Cinzel grande, dourado.
 *
 * @example <SectionTitle as="h2" id="forge-step-atributos">Distribua os atributos</SectionTitle>
 */
export function SectionTitle(props: RotuloProps) {
  return <Rotulo {...props} papel="titulo" padrao="h2" tom={props.tom ?? 'gold'} />
}

/**
 * O cabeçalho de um bloco dentro da cena — Cinzel 11px.
 *
 * @example <SectionLabel>Kit da classe</SectionLabel>
 */
export function SectionLabel(props: RotuloProps) {
  return <Rotulo {...props} papel="secao" padrao="p" />
}

/**
 * O rótulo colado num valor — sem Cinzel, o "FOR" ao lado do 16.
 *
 * @example <FieldLabel as="span">{attribute.abbr}</FieldLabel>
 */
export function FieldLabel(props: RotuloProps) {
  return <Rotulo {...props} papel="campo" padrao="span" />
}
