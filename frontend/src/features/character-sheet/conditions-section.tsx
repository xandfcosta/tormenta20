import { useQueryClient } from '@tanstack/solid-query'
import { conditionEffectSummary } from '@/shared/rules/condition-modifiers'
import type { ConditionId } from '@/shared/api/catalog-types'
import { X } from 'lucide-solid'
import { For, Show, createMemo } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { conditionsList, conditionsRecord } from '@/shared/lib/rules-catalog-cache'
import { cn } from '@/shared/lib/utils'
import { parseActiveConditions } from './active-conditions'
import { conditionActions } from './effect-mutations'
import { ApplyConditionDialog } from './apply-condition-dialog'
import { toast } from '@/shared/ui/sonner'
import { SectionTitle } from '@/shared/ui/section-label'

export type ConditionEditing = {
  active: () => ConditionId[]
  /** As condições que ainda cabem, já ordenadas em pt-BR, para o picker. */
  options: () => { value: string; label: string }[]
  /**
   * Aplica e REJEITA se o servidor recusar, sem dizer nada — para quem toma a
   * ação dentro de um diálogo, onde a falha vai inline (ALE-216).
   */
  apply: (id: string) => Promise<void>
  /** `apply` com aviso em toast: para os pickers que vivem FORA de diálogo. */
  add: (id: string) => void
  remove: (id: ConditionId) => void
}

/** Uma frase só para as duas saídas que avisam por toast. */
const FALHA_AO_SALVAR = 'Falha ao salvar condições — a ficha voltou ao valor anterior'

/**
 * Aplicar e remover condições do livro (p394-395). Vive fora do componente para
 * que a aba Efeitos e a faixa do combatente (ALE-145) editem a MESMA coisa sem
 * uma copiar a lógica da outra — o que elas não compartilham é a forma.
 *
 * @example const conditions = createConditionEditing(() => props.character)
 */
export function createConditionEditing(character: () => Character): ConditionEditing {
  const queryClient = useQueryClient()
  const active = createMemo(() => parseActiveConditions(character().activeConditions))

  const save = (next: ConditionId[]) => conditionActions(queryClient, character().id).set(next)

  /** Para as saídas FORA de diálogo, que não têm linha inline: avisa e engole. */
  const comToast = (write: Promise<void>) => void write.catch(() => toast.error(FALHA_AO_SALVAR))

  const options = createMemo(() =>
    conditionsList()
      .filter((condition) => !active().includes(condition.id))
      .sort((a, b) => a.name.localeCompare(b.name, 'pt-BR'))
      .map((condition) => ({ value: condition.id, label: condition.name })),
  )

  const apply = async (id: string) => {
    if (active().includes(id as ConditionId)) return
    await save([...active(), id as ConditionId])
  }

  return {
    active,
    options,
    apply,
    add: (id) => comToast(apply(id)),
    remove: (id) => comToast(save(active().filter((c) => c !== id))),
  }
}

/**
 * Book conditions (caído, agarrado, atordoado… p394-395) — the #1 mid-fight
 * tracking need. Active conditions render as removable chips carrying the
 * mechanical effect they apply; `ApplyConditionDialog` adds from the full
 * catalog.
 *
 * The chips say what each condition DOES because a condition that is only a
 * badge is the bug ALE-28 was: these change the sheet's numbers.
 */
export function ConditionsSection(props: { character: Character }) {
  const conditions = createConditionEditing(() => props.character)

  return (
    <section class="space-y-2 rounded-none border border-grimorio-iron p-3">
      {/* A MESMA fileira do "Efeitos ativos" logo abaixo: título à esquerda,
          botão de adicionar à direita abrindo um diálogo. Antes esta seção
          usava um campo de busca embutido e a de baixo um botão — a mesma
          escolha desenhada de dois jeitos, que é o defeito da ALE-169. */}
      <div class="flex flex-wrap items-center justify-between gap-2">
        <SectionTitle as="h3" contexto="painel" class="text-sm">
          Condições (p394)
        </SectionTitle>
        <ApplyConditionDialog conditions={conditions} />
      </div>
      <Show
        when={conditions.active().length > 0}
        fallback={<p class="text-xs text-muted-foreground">Nenhuma condição ativa.</p>}
      >
        <ul class="flex flex-wrap gap-1.5">
          <For each={conditions.active()}>
            {(id) => <ConditionChip id={id} onRemove={() => conditions.remove(id)} />}
          </For>
        </ul>
      </Show>
    </section>
  )
}

/**
 * Uma condição ativa, com o botão de tirar. `compact` esconde o resumo do
 * efeito: na faixa do combatente o que se lê é "quem está caído", e o resumo
 * ("−2 em testes de FOR…") dobrava a largura do chip.
 */
export function ConditionChip(props: {
  id: ConditionId
  onRemove: () => void
  compact?: boolean
}) {
  const condition = () => conditionsRecord()[props.id]
  return (
    <li
      title={condition().description}
      class={cn(
        'flex items-center gap-1 rounded-sm border border-[color:var(--hp-hurt)]/60 bg-[color:var(--hp-hurt)]/10 font-medium',
        props.compact ? 'px-1.5 py-px text-2xs' : 'px-2 py-1 text-xs',
      )}
    >
      {condition().name}
      {/* The applied mechanical effect, or "lembrete" for the conditions with
          no sheet-number impact (ALE-28). */}
      <Show when={!props.compact}>
        <span class="text-3xs font-normal text-muted-foreground">
          {conditionEffectSummary(props.id)}
        </span>
      </Show>
      <button
        type="button"
        aria-label={`Remover condição ${condition().name}`}
        onClick={() => props.onRemove()}
        class="rounded-md p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground"
      >
        <X aria-hidden="true" class="size-3" />
      </button>
    </li>
  )
}
