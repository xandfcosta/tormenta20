import { useQueryClient } from '@tanstack/solid-query'
import { validateLearnSpell, validateSpellLearned } from '@/shared/rules/rules-spells'
import { firstErrorMessage } from '@/shared/rules/rules-types'
import { SPELL_BASE_PM_COST } from '@/shared/rules/spells'
import type { CatalogSpell, SpellcasterClass } from '@/shared/api/catalog-types'
import { type AttributeKey } from '@/shared/api/attribute-keys'
import { BookPlus, BookX, Check, Sparkles } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import { bestSpellCd, castableClassesFor, highestCastableCircle } from '@/entities/character/spell-rules'
import type { Character, CharacterSpell } from '@/shared/api/api'
import { hasSpell } from '@/shared/lib/spell-cache'
import { Button } from '@/shared/ui/button'
import { toast } from '@/shared/ui/sonner'
import { cn } from '@/shared/lib/utils'
import { CastSpellDialog } from './cast-spell-dialog'
import { CIRCLE_LABEL, SCHOOL_LABEL } from './spell-labels'
import { spellActions } from './spell-mutations'

/** A spell taught by a power (Totem Espiritual, p42). */
export type GrantedSpellMeta = {
  /** The power that teaches it — shown as a badge; a granted spell cannot be
   *  unlearned, so the spellbook actions hide. */
  sourcePower: string
  /** Key attribute the granting power casts with (Totem: Sab, p42). */
  keyAttribute: AttributeKey
}

/**
 * One catalog row. Collapsed it shows name + círculo + escola + PM + CD +
 * status, plus an always-visible Conjurar for learned spells (icon-only below
 * `sm`) — casting must never require expanding the row. Expanded it adds the
 * full stats, the augment list and the spellbook actions.
 */
export function SpellRow(props: {
  spell: CatalogSpell
  character: Character
  learned: CharacterSpell | null
  /** Spell save CD per casting attribute, from the computed sheet (p173). */
  spellCdByAttribute: Record<AttributeKey, number>
  granted?: GrantedSpellMeta
}) {
  const queryClient = useQueryClient()
  const [open, setOpen] = createSignal(false)
  const [pending, setPending] = createSignal(false)

  const applicableClasses = createMemo<SpellcasterClass[]>(() =>
    castableClassesFor(props.character, props.spell.classes),
  )
  const bestCd = () =>
    props.granted
      ? props.spellCdByAttribute[props.granted.keyAttribute]
      : bestSpellCd(props.character, applicableClasses(), props.spellCdByAttribute)
  // A granted spell is castable at its own circle even for a non-caster.
  const canCast = () =>
    Boolean(props.granted) ||
    props.spell.circle <= highestCastableCircle(props.character, applicableClasses())
  const castable = () => Boolean(props.granted) || Boolean(props.learned)

  const knownIds = () => props.character.spells.map((spell) => spell.catalogSpellId)

  /** Guard with the SHARED rule before mutating, so an optimistic patch is only
   *  applied when the server is going to agree. */
  const run = async (refusal: string | null | undefined, write: () => Promise<void>) => {
    if (refusal) {
      toast.error(refusal)
      return
    }
    setPending(true)
    try {
      await write()
    } catch {
      // spellActions already rolled back and told the player.
    } finally {
      setPending(false)
    }
  }

  const actions = () => spellActions(queryClient, props.character.id)

  const learn = () =>
    run(
      firstErrorMessage(
        validateLearnSpell(knownIds(), props.spell.id, hasSpell(props.spell.id)),
      ),
      () => actions().learn(props.spell.id),
    )
  const setPrepared = (prepared: boolean) =>
    run(firstErrorMessage(validateSpellLearned(knownIds(), props.spell.id)), () =>
      actions().setPrepared(props.spell.id, prepared),
    )
  const unlearn = () => run(undefined, () => actions().unlearn(props.spell.id))

  return (
    <div
      class={cn(
        'rounded-md border border-transparent px-2 py-1 hover:bg-accent/40',
        open() && 'border-border',
      )}
    >
      <div class="flex items-center gap-2">
        <button
          type="button"
          onClick={() => setOpen(!open())}
          aria-expanded={open()}
          class="flex min-w-0 flex-1 flex-wrap items-center gap-x-2 gap-y-1 text-left"
        >
          <span
            class={cn(
              'rounded-sm border border-border px-1 font-mono text-[10px]',
              !canCast() && 'text-muted-foreground',
            )}
          >
            {CIRCLE_LABEL[props.spell.circle]}
          </span>
          {/* basis-32 + wrap: the name claims ~8rem before the chips may share
              the line, so at 390px they wrap below instead of crushing it. */}
          <span class="min-w-0 flex-1 basis-32 truncate text-sm font-medium">
            {props.spell.name}
          </span>
          <span class="ml-auto flex shrink-0 items-center gap-2">
            <span class="hidden text-[10px] uppercase tracking-widest text-muted-foreground sm:inline">
              {SCHOOL_LABEL[props.spell.school]}
            </span>
            <span class="font-mono text-xs text-grimorio-gold">
              {SPELL_BASE_PM_COST[props.spell.circle]} PM
            </span>
            <Show when={bestCd() !== null}>
              <span class="font-mono text-xs text-violet-300">CD {bestCd()}</span>
            </Show>
            <Show when={props.granted} fallback={<LearnedBadge learned={props.learned} />}>
              {(granted) => (
                <span class="rounded-sm border border-border px-1 text-[10px] uppercase tracking-widest">
                  {granted().sourcePower}
                </span>
              )}
            </Show>
          </span>
        </button>
        <Show when={castable()}>
          <CastSpellDialog
            spell={props.spell}
            character={props.character}
            disabled={!canCast()}
            compact
          />
        </Show>
      </div>

      <Show when={open()}>
        <div class="mt-2 space-y-2 border-t border-border pt-2 text-xs">
          <div class="grid gap-1 sm:grid-cols-2">
            <Stat label="Execução" value={props.spell.execution} />
            <Stat label="Alcance" value={props.spell.range} />
            <Stat
              label="Duração"
              value={
                props.spell.duration === 'definida' && props.spell.durationNote
                  ? props.spell.durationNote
                  : props.spell.duration
              }
            />
            <Stat label="Resistência" value={props.spell.resistance ?? '—'} />
            <Stat label="Teste" value={props.spell.saveType} />
            <Stat label="Livro" value={`p${props.spell.bookPage}`} />
          </div>
          <p class="whitespace-pre-line text-foreground">{props.spell.baseEffect}</p>

          <Show when={props.spell.augments.length > 0}>
            <div class="space-y-1">
              <p class="text-[10px] uppercase tracking-widest text-muted-foreground">
                Aprimoramentos
              </p>
              <ul class="space-y-1">
                <For each={props.spell.augments}>
                  {(augment) => (
                    <li class="flex items-start gap-2 rounded-sm border border-border px-2 py-1">
                      <span class="rounded-sm border border-border px-1 font-mono text-[10px]">
                        +{augment.pmCost} PM
                      </span>
                      <span
                        class={cn(
                          'text-[10px] uppercase tracking-widest',
                          augment.kind === 'muda' ? 'text-violet-300' : 'text-emerald-300',
                        )}
                      >
                        {augment.kind}
                      </span>
                      <span class="flex-1 text-foreground">
                        {augment.description}
                        <Show when={augment.classOnly}>
                          {(only) => (
                            <span class="ml-1 italic text-muted-foreground">
                              (apenas {only()})
                            </span>
                          )}
                        </Show>
                        <Show when={augment.requiresCircle !== undefined}>
                          <span class="ml-1 italic text-muted-foreground">
                            (requer {CIRCLE_LABEL[augment.requiresCircle as 0]})
                          </span>
                        </Show>
                      </span>
                    </li>
                  )}
                </For>
              </ul>
            </div>
          </Show>

          <Show when={!canCast() && applicableClasses().length > 0}>
            <p class="flex items-center gap-1 text-[11px] text-foreground">
              <Sparkles aria-hidden="true" class="size-3" />
              Círculo acima do máximo conjurável no nível atual.
            </p>
          </Show>

          {/* A granted spell has no spellbook row to manage. */}
          <Show when={!props.granted}>
            <div class="flex flex-wrap items-center gap-2 border-t border-border pt-2">
              <Show
                when={props.learned}
                fallback={
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    class="h-7 gap-1 text-xs"
                    disabled={pending()}
                    onClick={() => void learn()}
                  >
                    <BookPlus aria-hidden="true" class="size-3.5" />
                    Aprender
                  </Button>
                }
              >
                {(learned) => (
                  <>
                    <Button
                      type="button"
                      size="sm"
                      variant={learned().prepared ? 'default' : 'outline'}
                      class="h-7 gap-1 text-xs"
                      disabled={pending()}
                      onClick={() => void setPrepared(!learned().prepared)}
                    >
                      <Check aria-hidden="true" class="size-3.5" />
                      {learned().prepared ? 'Despreparar' : 'Preparar'}
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      class="h-7 gap-1 text-xs text-red-400 hover:bg-red-950/40"
                      disabled={pending()}
                      onClick={() => void unlearn()}
                    >
                      <BookX aria-hidden="true" class="size-3.5" />
                      Esquecer
                    </Button>
                  </>
                )}
              </Show>
            </div>
          </Show>
        </div>
      </Show>
    </div>
  )
}

function LearnedBadge(props: { learned: CharacterSpell | null }) {
  return (
    <Show when={props.learned}>
      {(learned) => (
        <span
          class={cn(
            'rounded-sm px-1 text-[10px] uppercase tracking-widest',
            learned().prepared
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground',
          )}
        >
          {learned().prepared ? 'Preparada' : 'Aprendida'}
        </span>
      )}
    </Show>
  )
}

function Stat(props: { label: string; value: string | number }) {
  return (
    <div class="flex items-baseline gap-1">
      <span class="text-[10px] uppercase tracking-widest text-muted-foreground">
        {props.label}
      </span>
      <span class="text-foreground">{props.value}</span>
    </div>
  )
}
