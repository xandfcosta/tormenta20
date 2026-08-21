import { For, type JSX, Show, createMemo } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { PickerCombobox } from '@/shared/ui/picker-combobox'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/popover'
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
    <div class="@container shrink-0 border-b border-grimorio-iron bg-grimorio-panel px-3 py-1.5 sm:px-4">
      {/* A ORDEM de quem cede espaço é a regra desta fileira, e a primeira
          versão dela errou justamente isso (ALE-147): o nome era o único item
          com `flex-1 min-w-0`, então era o único a encolher — com duas
          condições ativas ele virava "AI", e as ações caíam numa segunda
          fileira solta. Agora quem cede é o grupo de CONDIÇÕES, que é o que
          cresce sozinho durante o combate, e ele tem para onde ceder: passa de
          dois chips para o popover "⚠+N". */}
      {/* Três grupos IRMÃOS num só wrap, cada um com piso de largura MÍNIMA DO
          PRÓPRIO CONTEÚDO (`min-w-min`). É o piso que impede a sobreposição: um
          grupo sem ele encolhe abaixo do que cabe, e aí os filhos transbordam e
          são desenhados por cima do vizinho. Com piso, o wrap é obrigado a
          mandar o grupo para a linha de baixo — que é a quebra que se quer.

          `min-w-min` e não um valor em rem: eu escrevi `min-w-[15rem]` aqui e
          ele ficou 8px MENOR que o conteúdo real, então o chip de Defesa era
          pintado 2px fora do grupo. Número mágico envelhece a cada campo que
          entra na faixa; `min-content` se mantém sozinho (ALE-144). */}
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
        <div class="flex min-w-min flex-1 items-center gap-2">
          <CharacterPortrait
            name={props.character.name}
            size="sm"
            class="size-7 rounded-full border border-grimorio-iron text-2xs"
          />
          {/* Piso de 7rem: é o que garante ~14 caracteres de nome em qualquer
              largura. O nível é IRMÃO do nome, não filho — dentro do `truncate`
              ele era a primeira coisa a sumir a 390px. */}
          <h2 class="min-w-[7rem] flex-1 truncate font-heading text-sm font-bold tracking-tight">
            {props.character.name}
          </h2>
          <span class="shrink-0 font-mono text-xs text-muted-foreground">
            Nv {props.character.level}
          </span>
          <DefenseChip character={props.character} activeConditionals={active()} />
        </div>

        {/* O piso muda com a largura porque o CONTEÚDO muda: abaixo de 30rem os
            chips somem e sobra o gatilho "⚠ N" mais o seletor. */}
        <div class="flex min-w-min flex-1 items-center gap-1">
          <BandConditions character={props.character} conditions={conditions} />
        </div>
        {/* Fechar o combatente é a saída da tela e aplicar efeito é verbo de
            turno — nenhum dos dois pode ser empurrado para fora (ALE-147). */}
        <Show when={props.actions}>
          <div class="flex min-w-min flex-1 items-center gap-2">{props.actions}</div>
        </Show>
      </div>

      {/* Lado a lado quando a coluna dá — duas linhas em vez de quatro é a
          diferença entre a ficha começar no primeiro terço da região ou na
          metade dela. 30rem e não 34: a coluna do combatente mede 518px numa
          janela de 1920 (medido, não suposto — o shell do mestre é de três
          colunas ali), e a 34rem o formato MAIOR era o único a ficar empilhado. */}
      <VitalRows character={props.character} class="mt-1 @lg:grid-cols-2 @lg:gap-x-4" />
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
  const total = () => props.conditions.active().length

  return (
    <div class="flex min-w-0 flex-1 items-center gap-1">
      {/* Os chips são o atalho de leitura, e SOMEM primeiro: abaixo de 30rem a
          fileira não tem largura para eles e para os dois seletores ao mesmo
          tempo — medido a 390px, onde ela estourava em 28px e levava o botão de
          fechar para fora da tela (ALE-147). O gatilho abaixo continua dizendo
          quantas são, então nada fica escondido sem aviso. */}
      {/* `shrink-0`, não `min-w-0`: com `min-w-0` esta caixa encolhia ABAIXO do
          próprio conteúdo e os chips transbordavam dela — o seletor era
          desenhado por cima do segundo chip. Encolher aqui é seguro de proibir
          porque a lista é limitada a dois; o resto vive no popover. */}
      <ul class="hidden shrink-0 items-center gap-1 @lg:flex">
        <For each={props.conditions.active().slice(0, VISIBLE_CONDITIONS)}>
          {(id) => <ConditionChip id={id} compact onRemove={() => props.conditions.remove(id)} />}
        </For>
      </ul>
      {/* Uma condição a mais nunca alarga a fileira: entra no popover, e lá
          continua removível. Sem isto, uma cena com cinco condições reproduz a
          ALE-147 — os chips cresciam sem limite e comiam o nome. */}
      <Show when={total() > 0}>
        <Popover>
          <PopoverTrigger
            as="button"
            type="button"
            aria-label={
              total() === 1 ? 'Ver a condição ativa' : `Ver as ${total()} condições ativas`
            }
            class="shrink-0 rounded-md border border-[color:var(--hp-hurt)]/60 bg-[color:var(--hp-hurt)]/15 px-1.5 py-px text-2xs font-semibold text-[color:var(--hp-hurt)]"
          >
            ⚠ {total()}
          </PopoverTrigger>
          <PopoverContent class="w-64">
            <ul class="flex flex-wrap gap-1.5">
              <For each={props.conditions.active()}>
                {(id) => <ConditionChip id={id} onRemove={() => props.conditions.remove(id)} />}
              </For>
            </ul>
          </PopoverContent>
        </Popover>
      </Show>
      <PickerCombobox
        class="min-w-[6rem] flex-1"
        options={props.conditions.options()}
        onPick={props.conditions.add}
        aria-label="Aplicar condição"
        placeholder="+ condição…"
        emptyMessage="Nenhuma."
      />
    </div>
  )
}

/**
 * Quantos chips de condição cabem na fileira antes de o resto viver só no
 * popover. Dois, e não os três do `ConditionPips`: aqui eles dividem a linha
 * com o seletor de aplicar e com as ações do painel.
 */
const VISIBLE_CONDITIONS = 2
