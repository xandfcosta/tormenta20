import { For, Show } from 'solid-js'
import type { Monster } from '@/shared/api/catalog-types'
import { MONSTER_SIZE_LABEL, MONSTER_TIPO_LABEL, formatNd } from './monster-format'

/**
 * O bloco do monstro como o mestre usa no meio do combate (ALE-122).
 *
 * A ordem é a que o dono pediu e não a do livro: os ATAQUES vêm primeiro porque
 * é o que se usa a cada turno; ND, atributos, resistências e habilidades ficam
 * abaixo, para consulta. Quem abre um NPC no rastreador via só a barra de PV e
 * saía da tela para procurar a Defesa no bestiário.
 *
 * Só CONSULTA: mostra "+9 / 2d8+7" e o mestre rola no dado dele — o app não
 * rola ataque em lugar nenhum, e inventar isso aqui criaria uma segunda gramática
 * de rolagem só para NPC.
 *
 * @example <MonsterStatBlock monster={ogro} />
 */
export function MonsterStatBlock(props: { monster: Monster }) {
  return (
    <div class="space-y-3">
      <Show when={props.monster.attacks.length > 0}>
        <section aria-label="Ataques" class="space-y-1">
          <BlockTitle>Ataques</BlockTitle>
          <For each={props.monster.attacks}>
            {(attack) => (
              <p class="flex flex-wrap items-baseline gap-x-2 text-xs">
                <span class="font-medium text-foreground">{attack.name}</span>
                <span class="font-mono text-grimorio-gold">{signed(attack.attackBonus)}</span>
                <span class="font-mono text-muted-foreground">{attack.damage}</span>
                <Show when={attack.special}>
                  {(special) => (
                    <span class="text-[11px] text-muted-foreground">({special()})</span>
                  )}
                </Show>
              </p>
            )}
          </For>
        </section>
      </Show>

      <section aria-label="Identidade" class="flex flex-wrap items-baseline gap-x-3 gap-y-1 text-[11px] text-muted-foreground">
        <span>
          ND {formatNd(props.monster.nd)} · {MONSTER_TIPO_LABEL[props.monster.tipo]} ·{' '}
          {MONSTER_SIZE_LABEL[props.monster.size]}
        </span>
        <span class="flex items-baseline gap-1">
          DEF <span class="font-mono text-sm text-foreground">{props.monster.defesa}</span>
        </span>
        <span class="flex items-baseline gap-1">
          Deslocamento <span class="font-mono text-foreground">{props.monster.deslocamento}</span>
        </span>
        <span>p{props.monster.bookPage}</span>
      </section>

      <section aria-label="Atributos e resistências" class="grid grid-cols-3 gap-x-3 gap-y-1 text-[11px] sm:grid-cols-6">
        <For each={attributeCells(props.monster)}>
          {(cell) => (
            <span class="flex items-baseline justify-between gap-1 rounded-sm border border-grimorio-iron px-1.5 py-0.5">
              <span class="uppercase tracking-wide text-muted-foreground">{cell.label}</span>
              <span class="font-mono text-foreground">{signed(cell.value)}</span>
            </span>
          )}
        </For>
      </section>

      <Show when={props.monster.specialAbilities.length > 0}>
        <section aria-label="Habilidades" class="space-y-1">
          <BlockTitle>Habilidades</BlockTitle>
          <ul class="ml-4 list-disc space-y-0.5 text-[11px] text-muted-foreground">
            <For each={props.monster.specialAbilities}>{(ability) => <li>{ability}</li>}</For>
          </ul>
        </section>
      </Show>
    </div>
  )
}

function BlockTitle(props: { children: string }) {
  return (
    <h4 class="font-heading text-[11px] uppercase tracking-[0.14em] text-grimorio-gold">
      {props.children}
    </h4>
  )
}

/**
 * Modificador SEMPRE com sinal: "-2" e "2" são coisas diferentes na hora de
 * somar um teste, e o zero precisa dizer que é zero, não que faltou o dado.
 */
function signed(value: number): string {
  return value >= 0 ? `+${value}` : `${value}`
}

function attributeCells(monster: Monster): { label: string; value: number }[] {
  return [
    { label: 'For', value: monster.forca },
    { label: 'Des', value: monster.destreza },
    { label: 'Con', value: monster.constituicao },
    { label: 'Int', value: monster.inteligencia },
    { label: 'Sab', value: monster.sabedoria },
    { label: 'Car', value: monster.carisma },
    { label: 'Fort', value: monster.fortitude },
    { label: 'Refl', value: monster.reflexos },
    { label: 'Von', value: monster.vontade },
  ]
}
