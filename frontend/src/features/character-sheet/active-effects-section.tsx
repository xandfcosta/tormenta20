import { useQueryClient } from '@tanstack/solid-query'
import { Sparkles, X } from 'lucide-solid'
import { For, Show } from 'solid-js'
import { parseEffectModifiers } from '@/entities/character/derived'
import { effectSourceFacts, effectSourceName } from '@/entities/character/effect-source'
import type { ActiveEffect, Character } from '@/shared/api/api'
import { usePowerUses } from '@/shared/stores/power-uses-context'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { ApplyEffectDialog } from './apply-effect-dialog'
import { effectActions } from './effect-mutations'
import { FactChips } from './fact-chips'
import { ModifierList } from './modifier-list'
import { SectionTitle } from '@/shared/ui/section-label'

/**
 * "Efeitos ativos" — consumables and spell buffs that were used and are granting
 * a scene/day-scoped bonus. Server state, unlike the toggles below it: the
 * `/active-effects` endpoints own the list, and ending a scene or a day expires
 * the matching scope.
 *
 * The scene/day boundary also resets the local limited-power-use counters
 * ("usado 1/cena" in the Poderes block) — the same boundary in the book means
 * both, and letting them drift is how a player ends up with a spent power on a
 * fresh scene.
 */
export function ActiveEffectsSection(props: { character: Character }) {
  const queryClient = useQueryClient()
  const powerUses = usePowerUses()
  const actions = () => effectActions(queryClient, props.character.id)
  const effects = () => props.character.activeEffects ?? []

  const endScene = async () => {
    try {
      await actions().endScene()
      powerUses.resetScene(props.character.id)
    } catch {
      // effectActions already told the player; the counters stay put.
    }
  }
  const endDay = async () => {
    try {
      await actions().endDay()
      powerUses.resetDay(props.character.id)
    } catch {
      // idem
    }
  }

  return (
    <section class="rounded-none border border-grimorio-iron p-3">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <SectionTitle as="h3" contexto="painel" class="text-sm">
          Efeitos ativos
        </SectionTitle>
        <div class="flex flex-wrap gap-1">
          <ApplyEffectDialog character={props.character} />
          <ConfirmDialog
            title="Encerrar cena?"
            description="Limpa todos os efeitos de cena (buffs, poções ativas) e zera os usos por cena."
            confirmLabel="Encerrar cena"
            destructive={false}
            onConfirm={() => void endScene()}
            trigger={(open) => (
              <Button
                type="button"
                variant="outline"
                size="sm"
                class="h-6 px-2 text-2xs"
                onClick={open}
              >
                Encerrar cena
              </Button>
            )}
          />
          <ConfirmDialog
            title="Encerrar dia?"
            description="Limpa efeitos de cena e de dia, e zera os usos por cena e por dia."
            confirmLabel="Encerrar dia"
            destructive={false}
            onConfirm={() => void endDay()}
            trigger={(open) => (
              <Button
                type="button"
                variant="outline"
                size="sm"
                class="h-6 px-2 text-2xs"
                onClick={open}
              >
                Encerrar dia
              </Button>
            )}
          />
        </div>
      </div>
      <Show
        when={effects().length > 0}
        fallback={
          <p class="mt-2 text-xs text-muted-foreground">
            Nenhum consumível ativo. Use itens consumíveis na Mochila.
          </p>
        }
      >
        {/* Nomeada porque o diálogo de aplicar lista os MESMOS nomes: sem um
            alvo para escopar, "Escudo da Fé" casa a linha aplicada e a linha do
            catálogo enquanto o diálogo ainda fecha. */}
        <ul aria-label="Efeitos ativos" class="mt-2 space-y-1">
          <For each={effects()}>
            {(effect) => (
              <ActiveEffectRow
                effect={effect}
                onRemove={() => void actions().remove(effect.id).catch(() => {})}
              />
            )}
          </For>
        </ul>
      </Show>
    </section>
  )
}

function ActiveEffectRow(props: { effect: ActiveEffect; onRemove: () => void }) {
  // Name and modifiers resolve for BOTH item and spell sources: the name via the
  // spell-aware resolver, the modifiers from the effect's OWN persisted blob —
  // which works regardless of source, unlike reading the catalog consumable.
  const name = () => effectSourceName(props.effect.catalogId)
  const modifiers = () => parseEffectModifiers(props.effect.modifiers)
  const facts = () => effectSourceFacts(props.effect.catalogId)

  return (
    <li class="rounded-sm border border-bonus/25 bg-bonus/10 px-2 py-1.5">
      <div class="flex items-center gap-2">
        <Sparkles aria-hidden="true" class="size-3.5 shrink-0 text-bonus-ink" />
        <span class="flex-1 truncate text-sm text-foreground">{name()}</span>
        <span
          // A tinta clara não serve AQUI: este crachá é preenchimento FORTE
          // (70%), e `text-bonus-ink` é a cor de escrever no painel, não sobre
          // o próprio bônus. Sobre bloco cheio, a letra é a clara da casa.
          class={
            props.effect.scope === 'day'
              ? 'shrink-0 rounded-full bg-muted px-2 py-0.5 text-3xs font-bold uppercase tracking-widest text-foreground'
              : 'shrink-0 rounded-full bg-bonus/70 px-2 py-0.5 text-3xs font-bold uppercase tracking-widest text-grimorio-fg'
          }
        >
          {props.effect.scope === 'day' ? 'dia' : 'cena'}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          class="size-7 text-foreground hover:bg-penalty/15 hover:text-penalty-ink"
          onClick={() => props.onRemove()}
          aria-label={`Remover ${name()}`}
        >
          <X aria-hidden="true" class="size-3.5" />
        </Button>
      </div>
      <ModifierList
        modifiers={modifiers()}
        class="ml-5 mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-2xs"
      />
      <FactChips facts={facts()} class="ml-5 mt-1" />
      <Show when={modifiers().length === 0 && facts().length === 0}>
        <p class="ml-5 mt-1 text-2xs italic text-muted-foreground">Sem efeito mecânico</p>
      </Show>
    </li>
  )
}
