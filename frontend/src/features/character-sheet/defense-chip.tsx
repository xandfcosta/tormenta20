import { Shield } from 'lucide-solid'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import type { Character } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'

/**
 * Compact DEF chip: "does a 17 hit me?" has to be answerable without switching
 * blocks. Read-only — the full breakdown lives in the Defesa box of the Combate
 * tab.
 *
 * Nasceu para os viewports onde o cluster de números é `hidden` (<md) e por
 * isso se chamava `MobileDefChip`; a faixa do combatente do mestre usa o mesmo
 * chip em qualquer largura (ALE-145), então o nome deixou de dizer a verdade.
 */
export function DefenseChip(props: {
  character: Character
  activeConditionals: ReadonlySet<string>
  class?: string
}) {
  const defense = () => computedSheetFor(props.character, props.activeConditionals).defense.total
  return (
    // role="img": a bare span is `generic`, which does not take an accessible
    // name — the shield glyph plus the number only read as "Defesa 17" here.
    <span
      role="img"
      class={cn(
        // `shrink-0`: um chip com número não pode encolher abaixo do próprio
        // conteúdo — encolhendo, o escudo e o "16" são pintados 2px para fora
        // da própria caixa. Achado pela `expectNadaEscapa` da ALE-144, que é
        // exatamente o tipo de defeito que nenhuma asserção antiga via.
        'flex shrink-0 items-center gap-1 rounded-md border border-destructive/50 px-1.5 py-0.5',
        'font-mono text-sm font-bold text-destructive',
        props.class,
      )}
      title="Defesa"
      aria-label={`Defesa ${defense()}`}
    >
      <Shield aria-hidden="true" class="size-3.5" />
      {defense()}
    </span>
  )
}
