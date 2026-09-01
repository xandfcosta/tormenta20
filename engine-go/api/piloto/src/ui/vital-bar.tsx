import { cn } from '@/lib/utils'
import { FieldLabel } from './section-label'

/**
 * HP fill token by ratio — the COLOR, not just the width, says "how bad".
 * Lives in shared because both the sheet's HUD and the session tracker paint
 * the same bar, and a feature may not import another feature.
 */
export function hpFillVar(percent: number): string {
  if (percent <= 25) return '--hp-critical'
  if (percent <= 50) return '--hp-hurt'
  return '--hp-full'
}

/**
 * O token com que se ESCREVE o rótulo, que não é sempre o mesmo com que se
 * PINTA a barra (ALE-240).
 *
 * Os três tons vitais foram escolhidos como cor de barra, e dois deles servem
 * de texto por acaso — medido sobre o painel: o verde dá 5,34:1 e o âmbar
 * 6,25:1. O `--hp-critical` dá **4,11:1**, abaixo do mínimo de texto pequeno,
 * e o rótulo tem 10px em negrito. Ou seja: exatamente na hora em que a barra
 * grita "este aqui está morrendo", o "PV" ao lado dela é o menos legível da
 * tela.
 *
 * O crítico escreve com a tinta de perigo da casa, que a ALE-237 fechou em
 * 5,21 sobre o painel e 4,58 sobre o elevado. Tinta própria seria um segundo
 * vermelho quase idêntico ao lado do primeiro — mesmo matiz, claridade a um
 * passo —, e a casa tem uma palavra por conceito.
 *
 * O rótulo deixa de ser EXATAMENTE a cor da barra, e isso é deliberado: os dois
 * continuam sendo o mesmo tom, e legibilidade ganha de casamento de pixel.
 */
export function hpInkVar(percent: number): string {
  const preenchimento = hpFillVar(percent)
  return preenchimento === '--hp-critical' ? '--grimorio-crimson-bright' : preenchimento
}

/**
 * Read-only PV/PM bar. A real `progressbar`, so the number is readable by
 * assistive tech and by the E2E suite — the tracker's rows are the one place
 * a player watches someone else's health, and "some green" is not an answer.
 *
 * @example <VitalBar kind="hp" label="PV" current={12} max={30} />
 */
export function VitalBar(props: {
  kind: 'hp' | 'mp'
  label: string
  current: number
  max: number
  class?: string
}) {
  const percent = () =>
    props.max > 0 ? Math.max(0, Math.min(100, (props.current / props.max) * 100)) : 0
  const fillVar = () => (props.kind === 'hp' ? hpFillVar(percent()) : '--mp-arcane')
  const inkVar = () => (props.kind === 'hp' ? hpInkVar(percent()) : '--mp-arcane')

  return (
    <div class={cn('flex items-center gap-1.5', props.class)}>
      {/* O espaçamento sobe de `wider` para `widest` ao adotar o componente:
          era a mesma intenção escrita um degrau diferente, e a caixa é fixa em
          `w-7`, então nada se move. */}
      <FieldLabel
        tom="inherit"
        class="w-7 shrink-0 font-bold"
        style={{ color: `var(${inkVar()})` }}>
        {props.label}
      </FieldLabel>
      <div
        role="progressbar"
        aria-valuenow={props.current}
        aria-valuemin={0}
        aria-valuemax={props.max}
        aria-label={`${props.label} ${props.current} de ${props.max}`}
        class="h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-muted"
      >
        <div
          class="h-full rounded-full transition-[width]"
          style={{ width: `${percent()}%`, background: `var(${fillVar()})` }}
        />
      </div>
      {/* 13px e cor de texto normal, não 10px em muted: PV e PM eram o MENOR
          texto da cena sendo os números mais lidos do combate (ALE-163). */}
      <span class="shrink-0 font-mono text-[13px] tabular-nums text-foreground">
        {props.current}/{props.max}
      </span>
    </div>
  )
}
