import { createComputedValue, SpecBlock, SpecSection } from './spec-primitives'
import { cn } from '@/shared/lib/utils'

/**
 * As três famílias e a ladeira de tamanho — com o buraco dela à vista.
 *
 * O buraco é o assunto do P2 da ALE-173: a cena escreve 286 tamanhos ABAIXO do
 * menor degrau declarado (`text-[11px]` ×158, `text-[10px]` ×120, `text-[9px]`
 * ×21), e nenhum deles tem nome. Enquanto não tiverem, esta folha os mostra
 * como o que são: valores arbitrários, fora da escala, repetidos às centenas.
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

const SEM_NOME = [
  { classe: 'text-[11px]', nome: 'text-[11px]', usos: '158 usos' },
  { classe: 'text-[10px]', nome: 'text-[10px]', usos: '120 usos' },
  { classe: 'text-[9px]', nome: 'text-[9px]', usos: '21 usos' },
]

function LinhaDeTexto(props: { classe: string; nome: string; nota?: string }) {
  const [tamanho, refTamanho] = createComputedValue('font-size')
  return (
    <div class="flex w-full items-baseline gap-3 border-b border-grimorio-iron/40 py-1.5">
      <span ref={refTamanho} class={cn('min-w-0 flex-1 truncate', props.classe)}>
        O mestre rolou 17 e o ogro caiu
      </span>
      <span class="shrink-0 font-mono text-[11px] text-muted-foreground">{props.nome}</span>
      <span class="w-14 shrink-0 text-right font-mono text-[11px] text-grimorio-gold">
        {tamanho()}
      </span>
      {props.nota && (
        <span class="w-20 shrink-0 text-right text-[10px] text-muted-foreground">{props.nota}</span>
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

      <SpecBlock titulo="A escala declarada">
        <div class="w-full">
          {DEGRAUS.map((d) => (
            <LinhaDeTexto classe={d.classe} nome={d.nome} />
          ))}
        </div>
      </SpecBlock>

      <SpecBlock
        titulo="Abaixo da escala, sem nome"
        nota="Três tamanhos que a cena usa às centenas e que nenhum token declara. Enquanto seguirem arbitrários, cada um paga a armadilha do valor que só existe no CSS depois de reiniciar o servidor."
      >
        <div class="w-full">
          {SEM_NOME.map((d) => (
            <LinhaDeTexto classe={d.classe} nome={d.nome} nota={d.usos} />
          ))}
        </div>
      </SpecBlock>
    </SpecSection>
  )
}
