import { For, type JSX, Show } from 'solid-js'
import type { Monster } from '@/shared/api/catalog-types'
import { xpForNd } from '@/shared/lib/encounter-math'
import { FieldLabel, SectionLabel, SectionTitle } from '@/shared/ui/section-label'
import { formatNd, MONSTER_TIPO_LABEL } from './monster-format'

const signed = (n: number) => (n >= 0 ? `+${n}` : String(n))

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
          {props.monster.size} · p{props.monster.bookPage} · XP {xpForNd(props.monster.nd)}
        </p>
      </div>

      <div class="grid grid-cols-3 gap-2">
        <Stat label="PV" value={props.monster.hp} />
        <Stat label="Defesa" value={props.monster.defesa} />
        <Stat label="Deslocamento" value={props.monster.deslocamento} />
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
    <section>
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
