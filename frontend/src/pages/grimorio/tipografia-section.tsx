import { createComputedValue, SpecBlock, SpecSection } from './spec-primitives'
import { cn } from '@/shared/lib/utils'

/**
 * As três famílias e a ladeira de tamanho inteira.
 *
 * A metade de baixo dela é a que esta casa usa de verdade e a que o shadcn não
 * tem: `text-xs`, de 12px, é o PISO dele, pensado para formulário — e a mesa é
 * densa por natureza, com nove combatentes e seus vitais numa coluna. Os três
 * degraus abaixo eram escritos como valor arbitrário 321 vezes, sem nome
 * nenhum, até a ALE-173 declará-los.
 */

const FAMILIAS = [
  { classe: 'font-heading', nome: 'font-heading', uso: 'Cinzel — títulos e rótulos de seção' },
  { classe: 'font-sans', nome: 'font-sans', uso: 'corpo de texto' },
  { classe: 'font-mono', nome: 'font-mono', uso: 'números de jogo, com tabular-nums' },
]

const DEGRAUS = [
  { classe: 'text-2xl', nome: 'text-2xl' },
  { classe: 'text-xl', nome: 'text-xl' },
  { classe: 'text-lg', nome: 'text-lg' },
  { classe: 'text-base', nome: 'text-base' },
  { classe: 'text-sm', nome: 'text-sm' },
  { classe: 'text-xs', nome: 'text-xs' },
]

/** Os três que a casa acrescentou abaixo do piso do shadcn (ALE-173). */
const DEGRAUS_DA_CASA = [
  { classe: 'text-2xs', nome: 'text-2xs', uso: 'rótulo de seção' },
  { classe: 'text-3xs', nome: 'text-3xs', uso: 'rótulo de campo' },
  { classe: 'text-4xs', nome: 'text-4xs', uso: 'crachá' },
]

function LinhaDeTexto(props: { classe: string; nome: string; nota?: string }) {
  const [tamanho, refTamanho] = createComputedValue('font-size')
  return (
    <div class="flex w-full items-baseline gap-3 border-b border-grimorio-iron/40 py-1.5">
      <span ref={refTamanho} class={cn('min-w-0 flex-1 truncate', props.classe)}>
        O mestre rolou 17 e o ogro caiu
      </span>
      <span class="shrink-0 font-mono text-2xs text-muted-foreground">{props.nome}</span>
      <span class="w-14 shrink-0 text-right font-mono text-2xs text-grimorio-gold">
        {tamanho()}
      </span>
      {props.nota && (
        <span class="w-20 shrink-0 text-right text-3xs text-muted-foreground">{props.nota}</span>
      )}
    </div>
  )
}

export function TipografiaSection() {
  return (
    <SpecSection id="tipografia" titulo="Tipografia">
      <SpecBlock titulo="Famílias">
        <div class="w-full">
          {FAMILIAS.map((f) => (
            <LinhaDeTexto classe={`${f.classe} text-base`} nome={f.nome} nota={f.uso} />
          ))}
        </div>
      </SpecBlock>

      <SpecBlock titulo="A escala do shadcn">
        <div class="w-full">
          {DEGRAUS.map((d) => (
            <LinhaDeTexto classe={d.classe} nome={d.nome} />
          ))}
        </div>
      </SpecBlock>

      <SpecBlock
        titulo="Abaixo do piso do shadcn"
        nota="O `text-xs` de 12px é o menor degrau que o shadcn tem, e ele foi pensado para formulário. A mesa é densa, então a casa acrescentou três abaixo — eram 321 valores arbitrários sem nome antes da ALE-173."
      >
        <div class="w-full">
          {DEGRAUS_DA_CASA.map((d) => (
            <LinhaDeTexto classe={d.classe} nome={d.nome} nota={d.uso} />
          ))}
        </div>
      </SpecBlock>
    </SpecSection>
  )
}
