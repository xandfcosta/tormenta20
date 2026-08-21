import { For, Show } from 'solid-js'
import type { CreatureBlock } from '@/shared/api/creature-types'
import { MONSTER_SIZE_LABEL, MONSTER_TIPO_LABEL, formatNd } from './monster-format'
import { FieldLabel, SectionLabel } from '@/shared/ui/section-label'

/**
 * O bloco de criatura na tela, seja ele o verbete do livro ou o que o MESTRE
 * escreveu (ALE-137). Uma apresentação só, porque é uma forma só: "BANDIDO —
 * ND 1/4 — Humanoide (humano) Médio" (p289) tem a mesma estrutura do Ogro.
 *
 * A ordem é a que o dono pediu e não a do livro: os ATAQUES vêm primeiro porque
 * é o que se usa a cada turno; ND, atributos, resistências e o resto ficam
 * abaixo, para consulta (ALE-122).
 *
 * Só CONSULTA: mostra "+9 / 2d8+7" e o mestre rola no dado dele — o app não
 * rola ataque em lugar nenhum, e inventar isso aqui criaria uma segunda
 * gramática de rolagem só para NPC.
 *
 * @example <CreatureStatBlock block={bloco} bookPage={293} />
 */
export function CreatureStatBlock(props: { block: CreatureBlock; bookPage?: number }) {
  return (
    <div class="space-y-3">
      <Show when={props.block.attacks.length > 0}>
        <section aria-label="Ataques" class="space-y-1">
          <BlockTitle>Ataques</BlockTitle>
          <For each={props.block.attacks}>
            {(attack) => (
              <p class="flex flex-wrap items-baseline gap-x-2 text-xs">
                <span class="font-medium text-foreground">{attack.name}</span>
                <span class="font-mono text-grimorio-gold">{signed(attack.attackBonus)}</span>
                <span class="font-mono text-muted-foreground">{attack.damage}</span>
                <Show when={attack.ranged}>
                  <FieldLabel>
                    à distância
                  </FieldLabel>
                </Show>
                <Show when={attack.special}>
                  {(special) => (
                    <span class="text-2xs text-muted-foreground">({special()})</span>
                  )}
                </Show>
              </p>
            )}
          </For>
        </section>
      </Show>

      <section
        aria-label="Identidade"
        class="flex flex-wrap items-baseline gap-x-3 gap-y-1 text-2xs text-muted-foreground"
      >
        <span>
          ND {formatNd(props.block.nd)} · {MONSTER_TIPO_LABEL[props.block.tipo]} ·{' '}
          {MONSTER_SIZE_LABEL[props.block.size]}
        </span>
        {/* Iniciativa e Percepção abrem o bloco impresso (p289) e são as duas
            primeiras coisas que o mestre rola numa cena. O modelo as tinha
            desde a ALE-137 e esta tela nunca as mostrou — nem para as criaturas
            que o próprio mestre escreve (ALE-151). */}
        <span class="flex items-baseline gap-1">
          INI <span class="font-mono text-sm text-foreground">{signed(props.block.iniciativa)}</span>
        </span>
        <span class="flex items-baseline gap-1">
          PER <span class="font-mono text-sm text-foreground">{signed(props.block.percepcao)}</span>
        </span>
        <span class="flex items-baseline gap-1">
          DEF <span class="font-mono text-sm text-foreground">{props.block.defesa}</span>
        </span>
        {/* PM só aparece em conjurador — é a linha "Pontos de Mana" do livro, e
            a maioria das criaturas não a tem. */}
        <Show when={props.block.pm !== undefined}>
          <span class="flex items-baseline gap-1">
            PM <span class="font-mono text-sm text-foreground">{props.block.pm}</span>
          </span>
        </Show>
        <span class="flex items-baseline gap-1">
          Deslocamento <span class="font-mono text-foreground">{props.block.deslocamento}</span>
        </span>
        <Show when={props.bookPage}>{(page) => <span>p{page()}</span>}</Show>
      </section>

      <section
        aria-label="Atributos e resistências"
        class="grid grid-cols-3 gap-x-3 gap-y-1 text-2xs sm:grid-cols-6"
      >
        <For each={attributeCells(props.block)}>
          {(cell) => (
            <span class="flex items-baseline justify-between gap-1 rounded-none border border-grimorio-iron px-1.5 py-0.5">
              <span class="uppercase tracking-wide text-muted-foreground">{cell.label}</span>
              <span class="font-mono text-foreground">{signed(cell.value)}</span>
            </span>
          )}
        </For>
      </section>

      <Show when={props.block.skills.length > 0}>
        <section aria-label="Perícias" class="space-y-1">
          <BlockTitle>Perícias</BlockTitle>
          <p class="flex flex-wrap gap-x-3 text-2xs text-muted-foreground">
            <For each={props.block.skills}>
              {(skill) => (
                <span>
                  {skill.name} <span class="font-mono text-foreground">{signed(skill.bonus)}</span>
                  {/* O bônus CONDICIONAL do livro, "(+14 em pântanos)": ele muda
                      a rolagem e some se não for mostrado (ALE-151). */}
                  <Show when={skill.nota}>{(nota) => <span> ({nota()})</span>}</Show>
                </span>
              )}
            </For>
          </p>
        </section>
      </Show>

      <Show when={props.block.equipment || props.block.treasure}>
        <section
          aria-label="Equipamento e tesouro"
          class="space-y-0.5 text-2xs text-muted-foreground"
        >
          <Show when={props.block.equipment}>
            {(equipment) => (
              <p>
                <span class="uppercase tracking-wide">Equipamento</span> {equipment()}
              </p>
            )}
          </Show>
          <Show when={props.block.treasure}>
            {(treasure) => (
              <p>
                <span class="uppercase tracking-wide">Tesouro</span> {treasure()}
              </p>
            )}
          </Show>
        </section>
      </Show>

      <Show when={props.block.specialAbilities.length > 0}>
        <section aria-label="Habilidades" class="space-y-1">
          <BlockTitle>Habilidades</BlockTitle>
          <ul class="ml-4 list-disc space-y-0.5 text-2xs text-muted-foreground">
            <For each={props.block.specialAbilities}>{(ability) => <li>{ability}</li>}</For>
          </ul>
        </section>
      </Show>
    </div>
  )
}

function BlockTitle(props: { children: string }) {
  return (
    <SectionLabel as="h4" tom="gold">
      {props.children}
    </SectionLabel>
  )
}

/**
 * Modificador SEMPRE com sinal: "-2" e "2" são coisas diferentes na hora de
 * somar um teste, e o zero precisa dizer que é zero, não que faltou o dado.
 */
function signed(value: number): string {
  return value >= 0 ? `+${value}` : `${value}`
}

function attributeCells(block: CreatureBlock): { label: string; value: number }[] {
  return [
    { label: 'For', value: block.forca },
    { label: 'Des', value: block.destreza },
    { label: 'Con', value: block.constituicao },
    { label: 'Int', value: block.inteligencia },
    { label: 'Sab', value: block.sabedoria },
    { label: 'Car', value: block.carisma },
    { label: 'Fort', value: block.fortitude },
    { label: 'Refl', value: block.reflexos },
    { label: 'Von', value: block.vontade },
  ]
}
