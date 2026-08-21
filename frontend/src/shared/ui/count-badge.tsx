import { type VariantProps, cva } from 'class-variance-authority'
import { Show } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * A pílula de CONTAGEM — o número pequeno colado num rótulo (ALE-173, P6).
 *
 * Estava escrita quatro vezes, palavra por palavra, e o que difere entre as
 * cópias não é o desenho: é o tratamento de acessibilidade. Três punham
 * `aria-hidden` com uma linha `sr-only` ao lado; uma não punha nada. É a
 * armadilha que o guia do front registra — `aria-label` num `<span>` é
 * IGNORADO, porque o elemento não tem papel que o carregue, e a pílula
 * acabava anunciando só "3".
 *
 * A pílula é sempre `aria-hidden`; quem diz o significado depende de ONDE ela
 * mora, e o tipo obriga a escolher:
 *
 * - **solta**, colada num rótulo que não é controle: passa `label`, e sai uma
 *   linha `sr-only` ao lado, numa interpolação só — duas ("{n} {palavra}")
 *   viram dois nós de texto e saem lidas partidas.
 * - **dentro de um botão ou aba**, que já tem nome próprio: passa
 *   `anunciadoPeloPai`, e o componente NÃO escreve nada — quem nomeia é o
 *   controle, com `aria-label`.
 *
 * A segunda existe porque medi: o cálculo do nome acessível CONCATENA o texto
 * dos filhos sem inserir separador e normaliza espaço em branco, então uma
 * linha `sr-only` dentro de um botão que já fala sai grudada — "Origem2
 * escolhas pendentes". Pôr um espaço à esquerda não resolve: ele é comido na
 * normalização. Isso não era defeito meu: o nome já era "Origem2" antes desta
 * extração, e só ficou visível ao dar nome ao padrão.
 *
 * Não mostra nada quando não há o que contar: quem decide se o zero aparece é
 * quem chama, porque "zero efeitos" é informação e "zero pendências" é ruído.
 *
 * @example <CountBadge count={pendentes()} label="escolhas pendentes" />
 */
const countBadgeVariants = cva(
  'ml-0.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-3xs font-bold leading-none',
  {
    variants: {
      tom: {
        danger: 'bg-destructive text-white',
        primary: 'bg-primary text-primary-foreground',
        muted: 'bg-muted text-muted-foreground',
      },
    },
    defaultVariants: { tom: 'danger' },
  },
)

export type CountBadgeTom = NonNullable<VariantProps<typeof countBadgeVariants>['tom']>

type CountBadgeProps = { count: number; tom?: CountBadgeTom; class?: string } & (
  | { label: string; anunciadoPeloPai?: never }
  | { label?: never; anunciadoPeloPai: true }
)

export function CountBadge(props: CountBadgeProps) {
  return (
    <>
      <span aria-hidden="true" class={cn(countBadgeVariants({ tom: props.tom }), props.class)}>
        {props.count}
      </span>
      <Show when={props.label}>
        {(label) => <span class="sr-only">{`${props.count} ${label()}`}</span>}
      </Show>
    </>
  )
}
