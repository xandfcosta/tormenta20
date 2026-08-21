import { Dumbbell, Star, Trash2 } from 'lucide-solid'
import { For, Show, splitProps } from 'solid-js'
import {
  ATTRIBUTE_ABBR,
  ATTRIBUTE_KEYS,
  type ExpertiseDef,
  expertiseStateFor,
  trainingBonusForLevel,
} from '@/entities/character/expertise'
import { expertiseFromSheet } from '@/entities/character/computed-sheet'
import { createMediaQuery } from '@/shared/lib/media-query'
import type { AttributeKey, Character } from '@/shared/api/api'
import type { ComputedSheetV2 } from '@/shared/lib/computed-sheet-v2'
import { cn } from '@/shared/lib/utils'
import { DialogTrigger } from '@/shared/ui/dialog'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { ExpertiseBreakdown } from './expertise-breakdown'
import type { ExpertisePatch } from './expertise-mutations'
import { signed } from './signed'

export type ExpertiseRowProps = {
  character: Character
  def: ExpertiseDef
  sheet: ComputedSheetV2
  onPatch: (patch: ExpertisePatch) => void
  /** Only custom "ofícios" can be deleted. */
  onDelete?: () => void
  /**
   * A linha do MESTRE, no painel do combatente: só o total e o nome, com os
   * controles subindo para a linha do nome. Ver `SheetPanelProps.glance`.
   */
  glance?: boolean
}

/**
 * One perícia: its total (which opens the breakdown), the name, the trained
 * toggle, the attribute it keys off, and the chips that preview the math.
 *
 * As chips REPETEM o diálogo de decomposição palavra por palavra (½ nível,
 * atributo, treino, outros), e isso custa uma segunda linha por perícia. Na
 * ficha do jogador vale a pena: é a ficha dele e ele quer auditar o número sem
 * abrir nada. Na tela do mestre não — ele quer o número (ALE-145), então em
 * `glance` a linha fica só com o total, o nome e os dois controles.
 */
export function ExpertiseRow(props: ExpertiseRowProps) {
  /**
   * O TELEFONE usa o mesmo arranjo do `glance` (ALE-183): o seletor de atributo
   * sobe para a linha do nome e a fileira de chips deixa de existir. Medido a
   * 390px, isso tira 27px dos 108 de cada perícia e leva a janela de rolagem de
   * 4,3 para 5,7 perícias visíveis.
   *
   * Os chips não somem do app — eles REPETEM, palavra por palavra, o diálogo de
   * decomposição que o próprio total abre. No telefone a auditoria fica a um
   * toque em vez de ocupar um quarto de cada linha.
   *
   * A chave é largura + ORIENTAÇÃO, e não só largura: o celular DEITADO tem
   * 844px de largura com 390 de altura, e é justamente onde 27px por linha
   * custam mais caro (ALE-162). Altura é proibida — o teclado virtual a muda e
   * reconstrói o componente no meio da digitação.
   */
  const telefone = createMediaQuery(
    '(max-width: 639px), (max-width: 1023px) and (orientation: landscape)',
  )
  /** Sem a fileira de chips: por ser a ficha do mestre ou por ser telefone. */
  const semChips = () => props.glance || telefone()
  const state = () => expertiseStateFor(props.character, props.def)
  // Every standard + custom perícia is on the sheet; `?? 0` is only a type guard.
  const entry = () => expertiseFromSheet(props.sheet, props.def.name)
  const total = () => entry()?.total ?? 0
  const halfLevel = () => Math.floor(props.character.level / 2)
  const trainBonus = () => (state().trained ? trainingBonusForLevel(props.character.level) : 0)
  const itemBonus = () => entry()?.itemBonus ?? 0
  const locked = () => !!props.def.trainedOnly && !state().trained
  // O Indefeso (e tudo que o livro define como indefeso) FALHA automaticamente
  // em Reflexos — não é um número, então a linha não mostra total (p394).
  const autoFail = () => props.sheet.autoFailExpertises.includes(props.def.name)

  return (
    <ExpertiseBreakdown
      name={props.def.name}
      total={total()}
      locked={locked()}
      halfLevel={halfLevel()}
      attrAbbr={ATTRIBUTE_ABBR[state().attribute]}
      attrMod={entry()?.attrValue ?? 0}
      trainBonus={trainBonus()}
      itemBonus={itemBonus()}
      contributions={entry()?.itemContributions ?? []}
    >
      {/* `items-center` em glance: sem a segunda linha, alinhar pelo topo
          deixava o nome fora do eixo da caixa do total. */}
      <div
        class={cn(
          // `min-w-0`: item de grade/flex não encolhe abaixo do conteúdo por
          // padrão, e o `<select>` de atributo tem largura intrínseca — bastava
          // o painel perder ~15px (a barra de rolagem CLÁSSICA, que Linux e
          // Windows desenham DENTRO da caixa, ao contrário da sobreposta) para
          // a linha inteira pintar para fora do pai a 390px. Medido: 6px de
          // sobra a 375 e 21px a 360.
          'flex min-w-0 gap-2.5 rounded-none border border-grimorio-iron transition-colors hover:border-grimorio-gold/50',
          state().trained && 'bg-grimorio-panel',
          semChips() ? 'items-center p-1.5' : 'items-start p-2.5',
        )}
      >
        {/* Both the badge and the name open the breakdown; the toggle, the
            attribute select and delete stay interactive — they are not
            triggers. Kobalte composes via `as=`, where Radix used `asChild`. */}
        <DialogTrigger
          as={TotalBadge}
          total={total()}
          locked={locked()}
          autoFail={autoFail()}
          compact={props.glance}
        />
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-1.5">
            <DialogTrigger
              as="button"
              type="button"
              class="min-w-0 flex-1 truncate text-left text-sm text-foreground hover:underline"
            >
              {props.def.name}
            </DialogTrigger>
            <Show when={props.def.trainedOnly}>
              <TrainedOnlyStar locked={locked()} />
            </Show>
            {/* O seletor de atributo aparece numa das duas linhas, nunca nas
                duas: é o MESMO componente com as mesmas props, só muda de
                lugar. Sem chips embaixo, a segunda linha inteira deixa de
                existir e a linha cai de ~68px para ~48px. */}
            <Show when={semChips()}>
              <AttributeSelect
                name={props.def.name}
                value={state().attribute}
                sheet={props.sheet}
                onChange={(attribute) => props.onPatch({ attribute })}
              />
            </Show>
            <TrainedToggle
              trained={state().trained}
              name={props.def.name}
              onToggle={(next) => props.onPatch({ trained: next })}
            />
            <Show when={props.onDelete}>
              {(onDelete) => <DeleteExpertiseButton name={props.def.name} onDelete={onDelete()} />}
            </Show>
          </div>
          <Show when={!semChips()}>
            <div class="mt-1.5 flex flex-wrap items-center gap-1.5">
              <AttributeSelect
                name={props.def.name}
                value={state().attribute}
                sheet={props.sheet}
                onChange={(attribute) => props.onPatch({ attribute })}
              />
              <Chip label="½lvl" value={String(halfLevel())} />
              <Chip label="treino" value={signed(trainBonus())} />
              <DialogTrigger as="button" type="button" class="inline-flex hover:brightness-110">
                <Chip label="outros" value={signed(itemBonus())} />
              </DialogTrigger>
            </div>
          </Show>
        </div>
      </div>
    </ExpertiseBreakdown>
  )
}

/**
 * The perícia's number, doubling as the breakdown trigger. Locked (trained-only
 * and still untrained) reads as a dashed, dimmed box — the old line-through was
 * illegible on small mono digits.
 */
function TotalBadge(props: {
  total: number
  locked: boolean
  autoFail?: boolean
  /** A caixa é o que fixa a altura da linha: 44px dela viram 32 em glance. */
  compact?: boolean
}) {
  // A falha automática não tem número para mostrar. O traço carrega o sentido
  // visualmente e o rótulo acessível o diz por extenso — a composição continua
  // no diálogo, porque o jogador ainda quer saber o que ele PERDEU.
  const label = () =>
    props.autoFail ? 'Falha automática — detalhar' : `Detalhar ${signed(props.total)}`
  // `splitProps` e não `{...props}` cru: o resto vem do `DialogTrigger` (o
  // `as=` do Kobalte injeta onClick e o estado do diálogo) e PRECISA chegar ao
  // botão, mas os nossos quatro não são atributos de `<button>` — espalhados
  // junto, viravam `total="10" locked="false"` no DOM.
  const [, trigger] = splitProps(props, ['total', 'locked', 'autoFail', 'compact'])
  return (
    <button
      {...trigger}
      type="button"
      aria-label={label()}
      class={cn(
        'flex shrink-0 items-center justify-center rounded-none border font-mono font-bold',
        props.compact ? 'size-8 text-sm' : 'size-11 text-lg',
        props.autoFail
          ? 'border-destructive/60 bg-destructive/10 text-destructive'
          : props.locked
            ? 'border-dashed border-grimorio-iron text-muted-foreground/50'
            : 'border-grimorio-iron bg-grimorio-panel-raised text-grimorio-gold',
      )}
    >
      {props.autoFail ? '—' : signed(props.total)}
    </button>
  )
}

/** Star marking a trained-only perícia; amber while it is locked out. */
function TrainedOnlyStar(props: { locked: boolean }) {
  return (
    <Tooltip>
      <TooltipTrigger
        as="button"
        type="button"
        aria-label="Apenas treinada"
        class="inline-flex shrink-0 cursor-help"
      >
        <Star
          aria-hidden="true"
          class={cn(
            'size-3',
            props.locked
              ? 'fill-amber-500 text-amber-500'
              : 'fill-none text-muted-foreground/60',
          )}
        />
      </TooltipTrigger>
      <TooltipContent>Pode ser usada apenas quando treinada</TooltipContent>
    </Tooltip>
  )
}

function TrainedToggle(props: { trained: boolean; name: string; onToggle: (next: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={props.trained}
      aria-label={`${props.name} treinada`}
      onClick={() => props.onToggle(!props.trained)}
      class={cn(
        'inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-wider transition-colors',
        props.trained
          ? 'border-grimorio-gold/60 text-grimorio-gold'
          : 'border-grimorio-iron text-muted-foreground hover:border-grimorio-gold/40',
      )}
    >
      <Dumbbell aria-hidden="true" class="size-3" />
      Treino
    </button>
  )
}

function DeleteExpertiseButton(props: { name: string; onDelete: () => void }) {
  return (
    <button
      type="button"
      aria-label={`Remover ${props.name}`}
      onClick={() => props.onDelete()}
      class="inline-flex shrink-0 rounded-none p-1 text-muted-foreground transition-colors hover:text-destructive"
    >
      <Trash2 aria-hidden="true" class="size-3.5" />
    </button>
  )
}

/**
 * Which attribute the perícia keys off. Shows the FINAL modifier (race and item
 * bonuses folded in), not the raw sheet value — otherwise the row disagrees
 * with its own breakdown and total.
 */
function AttributeSelect(props: {
  name: string
  value: AttributeKey
  sheet: ComputedSheetV2
  onChange: (attribute: AttributeKey) => void
}) {
  return (
    <select
      value={props.value}
      onChange={(event) => props.onChange(event.currentTarget.value as AttributeKey)}
      aria-label={`${props.name} atributo`}
      // `min-w-0 max-w-full`: o `<select>` nativo se dimensiona pela opção mais
      // larga, e sem isto é ele quem empurra a linha para fora quando a coluna
      // aperta.
      class="h-6 min-w-0 max-w-full cursor-pointer rounded-full border border-grimorio-iron bg-transparent px-2 font-mono text-[11px] outline-none focus:ring-2 focus:ring-ring"
    >
      <For each={ATTRIBUTE_KEYS}>
        {(key) => (
          <option value={key}>
            {ATTRIBUTE_ABBR[key]} {signed(props.sheet.attributes[key].total)}
          </option>
        )}
      </For>
    </select>
  )
}

function Chip(props: { label: string; value: string }) {
  return (
    <span class="inline-flex items-center gap-1 rounded-full border border-grimorio-iron px-2 py-0.5 text-[10px] text-muted-foreground">
      <span class="uppercase tracking-wider">{props.label}</span>
      <span class="font-mono text-foreground">{props.value}</span>
    </span>
  )
}
