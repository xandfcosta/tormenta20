import { For, type JSX, Show, createMemo } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { PickerCombobox } from '@/shared/ui/picker-combobox'
import { ConditionChip, createConditionEditing } from './conditions-section'
import { DefenseChip } from './defense-chip'
import { VitalRows } from './vital-rows'

/**
 * A faixa do combatente no painel do mestre: nome, nível, Defesa, PV/PM e
 * condições — e nada mais (ALE-145).
 *
 * Isto NÃO é o `CharacterHud` encolhido, e a diferença é de propósito, não de
 * tamanho. O HUD é a nameplate do JOGADOR na ficha dele; aqui o mestre responde
 * três perguntas por turno — "quanto de vida ele tem", "acertou a Defesa dele?"
 * e "ele está caído?" — e o resto (ataques, resistências, atributos, fórmulas
 * de arma) mora na aba Combate, a um clique.
 *
 * Medido antes de existir: com o cartão inteiro no lugar desta faixa, o que
 * vinha ANTES da ficha comia 49-51% da região do combatente nos formatos
 * grandes e 65% no celular deitado — e foi isso que forçou o teto de 45% com
 * rolagem por dentro na ALE-125, que era conserto de sintoma.
 *
 * O retrato é um círculo pequeno ao lado do nome, o mesmo vocabulário do token
 * do tabuleiro (iniciais sobre cor determinística): fecha a ALE-126, onde ele
 * era uma tira vertical da altura do cartão.
 *
 * `actions` é o seio de injeção: os controles do painel (aplicar efeito,
 * fechar) entram por aqui em vez de morarem num cabeçalho próprio acima. Não é
 * só economia — o cabeçalho repetia o NOME que a faixa já diz, e ele custava
 * 61px numa região que no celular deitado tem 165 no total.
 * Feature não importa feature: quem monta o `ApplyEffectSelect` é a página.
 */
export function CombatantBand(props: { character: Character; actions?: JSX.Element }) {
  const conditionals = useConditionals()
  const active = createMemo(() => conditionals.active(props.character.id))
  const conditions = createConditionEditing(() => props.character)

  return (
    // @container e não breakpoint de janela: esta faixa vive numa COLUNA de
    // 616-936px numa janela de 1920, e foi exatamente essa confusão que espremeu
    // os atributos a 21px por caixa na ALE-122.
    <div class="@container shrink-0 border-b border-grimorio-iron bg-[var(--grimorio-panel)] px-3 py-1.5 sm:px-4">
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
        <CharacterPortrait
          name={props.character.name}
          size="sm"
          class="size-7 rounded-full border border-grimorio-iron text-[11px]"
        />
        {/* O nível é IRMÃO do nome, não filho: dentro do `truncate` ele era a
            primeira coisa a sumir, e a 390px a faixa mostrava "Arcanista E…"
            sem nível nenhum — justo um dos quatro itens que o dono pediu. */}
        <h2 class="min-w-0 flex-1 truncate font-heading text-sm font-bold tracking-tight">
          {props.character.name}
        </h2>
        <span class="shrink-0 font-mono text-xs text-muted-foreground">
          Nv {props.character.level}
        </span>
        <DefenseChip character={props.character} activeConditionals={active()} />
        <BandConditions character={props.character} conditions={conditions} />
        <Show when={props.actions}>
          <div class="flex shrink-0 items-center gap-2">{props.actions}</div>
        </Show>
      </div>

      {/* Lado a lado quando a coluna dá — duas linhas em vez de quatro é a
          diferença entre a ficha começar no primeiro terço da região ou na
          metade dela. 30rem e não 34: a coluna do combatente mede 518px numa
          janela de 1920 (medido, não suposto — o shell do mestre é de três
          colunas ali), e a 34rem o formato MAIOR era o único a ficar empilhado. */}
      <VitalRows character={props.character} class="mt-1 @[30rem]:grid-cols-2 @[30rem]:gap-x-4" />
    </div>
  )
}

/**
 * As condições ativas e o botão de aplicar, na mesma linha do nome.
 *
 * O aplicador fica AQUI, e não só na aba Efeitos: "você está caído" é a coisa
 * mais frequente que um mestre declara, e a ALE-122 já tinha tirado esse editor
 * de dentro da aba justamente por isso. O que saiu foi o bloco inteiro
 * (título "CONDIÇÕES (p394)" + campo de 256px), que custava ~90px de altura.
 */
function BandConditions(props: {
  character: Character
  conditions: ReturnType<typeof createConditionEditing>
}) {
  return (
    <div class="flex min-w-0 items-center gap-1">
      <Show when={props.conditions.active().length > 0}>
        <ul class="flex flex-wrap items-center gap-1">
          <For each={props.conditions.active()}>
            {(id) => (
              <ConditionChip id={id} compact onRemove={() => props.conditions.remove(id)} />
            )}
          </For>
        </ul>
      </Show>
      <div class="w-36 shrink-0">
        <PickerCombobox
          options={props.conditions.options()}
          onPick={props.conditions.add}
          aria-label="Aplicar condição"
          placeholder="+ condição…"
          emptyMessage="Nenhuma."
        />
      </div>
    </div>
  )
}
