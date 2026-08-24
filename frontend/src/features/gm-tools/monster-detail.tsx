import { For, type JSX, Show } from 'solid-js'
import type { Monster } from '@/shared/api/catalog-types'
import { xpForNd } from '@/shared/lib/encounter-math'
import { FieldLabel, SectionLabel, SectionTitle } from '@/shared/ui/section-label'
import { formatNd, MONSTER_SIZE_LABEL, MONSTER_TIPO_LABEL } from './monster-format'

/**
 * O modificador como o livro escreve, e o TRAVESSÃO quando ele não existe.
 *
 * "Int —" no Zumbi não é zero: é a ausência de Inteligência, e mostrar "+0"
 * diria que ele tem a média de um humano (ALE-151).
 */
const signed = (n: number | null) => (n === null ? '—' : n >= 0 ? `+${n}` : String(n))

/**
 * A creature's full stat block. Shared by the Bestiário's detail pane and (on
 * narrow viewports) its dialog, so the GM reads the same block either way.
 */
export function MonsterDetail(props: { monster: Monster }) {
  return (
    // Capped measure: a stat block is data plus prose, and at 1920 the
    // attribute boxes stretched to ~220px each to hold two characters.
    <div class="max-w-4xl space-y-4 text-sm">
      <div class="space-y-0.5">
        <SectionTitle as="h3">
          {props.monster.name}
        </SectionTitle>
        <p class="text-xs text-muted-foreground">
          ND {formatNd(props.monster.nd)} · {MONSTER_TIPO_LABEL[props.monster.tipo]} ·{' '}
          {MONSTER_SIZE_LABEL[props.monster.size]} · p{props.monster.bookPage} · XP {xpForNd(props.monster.nd)}
        </p>
      </div>

      {/* Iniciativa e Percepção são a PRIMEIRA linha do bloco impresso (p289),
          e o mestre rola as duas antes de qualquer outra coisa no combate.
          Elas não existiam no modelo até a ALE-151. */}
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <Stat label="Iniciativa" value={signed(props.monster.iniciativa)} />
        <Stat label="Percepção" value={signed(props.monster.percepcao)} />
        <Stat label="PV" value={props.monster.hp} />
        <Stat label="Defesa" value={props.monster.defesa} />
        <Stat label="Deslocamento" value={props.monster.deslocamento} />
        <Show when={props.monster.pm !== undefined}>
          <Stat label="PM" value={props.monster.pm ?? 0} />
        </Show>
      </div>


      <Show when={props.monster.attacks.length > 0}>
        <Section title="Ataques">
          <div class="space-y-1">
            <For each={props.monster.attacks}>
              {(attack) => (
                <div class="rounded-sm border border-grimorio-iron p-2">
                  <p class="text-xs font-semibold">
                    {attack.name}{' '}
                    <span class="font-mono text-muted-foreground">
                      {signed(attack.attackBonus)} · {attack.damage}
                    </span>
                  </p>
                  <Show when={attack.special}>
                    {(special) => (
                      <p class="text-2xs text-muted-foreground">{special()}</p>
                    )}
                  </Show>
                </div>
              )}
            </For>
          </div>
        </Section>
      </Show>

      <Show when={props.monster.specialAbilities.length > 0}>
        <Section title="Habilidades especiais">
          <ul class="list-disc space-y-1 pl-5 text-xs">
            <For each={props.monster.specialAbilities}>{(ability) => <li>{ability}</li>}</For>
          </ul>
        </Section>
      </Show>

      <Show when={props.monster.skills.length > 0}>
        <Section title="Perícias">
          <p class="text-xs">
            <For each={props.monster.skills}>
              {(skill, i) => (
                <>
                  {i() > 0 ? ' · ' : ''}
                  {skill.name} <span class="font-mono">{signed(skill.bonus)}</span>
                  <Show when={skill.nota}>
                    {(nota) => <span class="text-muted-foreground"> ({nota()})</span>}
                  </Show>
                </>
              )}
            </For>
          </p>
        </Section>
      </Show>

      {/* Equipamento e Tesouro fecham o bloco impresso, e é onde o livro os põe.
          O equipamento se perdeu INTEIRO na importação — zero dos 80 verbetes o
          tinham — e o tesouro virava um número de XP (ALE-151). */}
      <Show when={props.monster.equipamento}>
        {(equipamento) => (
          <Section title="Equipamento">
            <p class="text-xs">{equipamento()}</p>
          </Section>
        )}
      </Show>

      <Show when={props.monster.tesouro}>
        {(tesouro) => (
          <Section title="Tesouro">
            <p class="text-xs">{tesouro()}</p>
          </Section>
        )}
      </Show>

      {/* Atributos e resistências vêm DEPOIS do que a criatura faz (ALE-170).
          O mestre abre o bestiário no meio do combate para saber o que ela
          rola AGORA, e antes disso ele atravessava doze estatísticas — três
          vitais, seis atributos e três resistências — para chegar nos ataques.
          O bloco que o próprio mestre escreve, o `CreatureStatBlock`, já põe
          Ataques em primeiro: a casa sabia a ordem certa num lugar e não no
          outro. */}
      <Section title="Atributos">
        <div class="grid grid-cols-3 gap-1 sm:grid-cols-6">
          <Stat label="For" value={signed(props.monster.forca)} />
          <Stat label="Des" value={signed(props.monster.destreza)} />
          <Stat label="Con" value={signed(props.monster.constituicao)} />
          <Stat label="Int" value={signed(props.monster.inteligencia)} />
          <Stat label="Sab" value={signed(props.monster.sabedoria)} />
          <Stat label="Car" value={signed(props.monster.carisma)} />
        </div>
      </Section>

      <Section title="Resistências">
        <div class="grid grid-cols-3 gap-1">
          <Stat label="Fortitude" value={signed(props.monster.fortitude)} />
          <Stat label="Reflexos" value={signed(props.monster.reflexos)} />
          <Stat label="Vontade" value={signed(props.monster.vontade)} />
        </div>
      </Section>
    </div>
  )
}

function Section(props: { title: string; children: JSX.Element }) {
  return (
    // `aria-label` e não só o `<h4>`: uma `<section>` SEM nome acessível não é
    // uma região para o leitor de tela — ela vira uma caixa anônima, e o mestre
    // que navega por regiões não consegue pular para Ataques (ALE-151).
    <section aria-label={props.title}>
      <SectionLabel as="h4" class="mb-1">
        {props.title}
      </SectionLabel>
      {props.children}
    </section>
  )
}

function Stat(props: { label: string; value: string | number }) {
  return (
    <div class="rounded-sm border border-grimorio-iron p-2 text-center">
      <FieldLabel as="p">{props.label}</FieldLabel>
      <p class="text-sm font-semibold tabular-nums">{props.value}</p>
    </div>
  )
}
