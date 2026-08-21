import { Plus, Trash2 } from 'lucide-solid'
import { For, Index, Show } from 'solid-js'
import type { SetStoreFunction } from 'solid-js/store'
import type {
  CreatureBlock,
  CreatureSize,
  CreatureTipo,
} from '@/shared/api/creature-types'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { MONSTER_SIZE_LABEL, MONSTER_TIPO_LABEL, MONSTER_TIPOS } from './monster-format'

const SIZES: CreatureSize[] = ['minusculo', 'pequeno', 'medio', 'grande', 'enorme', 'colossal']

/**
 * Os campos do bloco de criatura, na ORDEM em que o livro os imprime (p289):
 * identidade, iniciativa e percepção, defesa e resistências, vida e mana,
 * deslocamento, ataques, atributos, perícias, equipamento e tesouro.
 *
 * Editar um store (e não devolver um objeto novo a cada tecla) é o que mantém
 * o foco: a lista de ataques é reconstruída a cada mudança, e um `For` sobre um
 * array recriado perde o campo que está sendo digitado — a armadilha nº 1 do
 * guia do front. Por isso as listas usam `Index`.
 *
 * @example <CreatureBlockForm block={bloco} setBlock={setBloco} />
 */
export function CreatureBlockForm(props: {
  block: CreatureBlock
  setBlock: SetStoreFunction<CreatureBlock>
}) {
  return (
    <div class="space-y-4">
      <Section title="Identidade">
        <Field label="ND" id="creature-nd">
          <NumberInput
            id="creature-nd"
            min={0}
            max={30}
            step={0.25}
            spinnerLabel="ND"
            value={props.block.nd}
            onChange={(nd) => props.setBlock('nd', nd)}
          />
        </Field>
        <Field label="Tipo" id="creature-tipo">
          <select
            id="creature-tipo"
            class={SELECT}
            value={props.block.tipo}
            onChange={(event) => props.setBlock('tipo', event.currentTarget.value as CreatureTipo)}
          >
            <For each={MONSTER_TIPOS}>
              {(tipo) => <option value={tipo}>{MONSTER_TIPO_LABEL[tipo]}</option>}
            </For>
          </select>
        </Field>
        <Field label="Tamanho" id="creature-size">
          <select
            id="creature-size"
            class={SELECT}
            value={props.block.size}
            onChange={(event) => props.setBlock('size', event.currentTarget.value as CreatureSize)}
          >
            <For each={SIZES}>
              {(size) => <option value={size}>{MONSTER_SIZE_LABEL[size]}</option>}
            </For>
          </select>
        </Field>
        <Field label="Deslocamento" id="creature-deslocamento">
          <Input
            id="creature-deslocamento"
            value={props.block.deslocamento}
            placeholder="9m (6q)"
            onInput={(event) => props.setBlock('deslocamento', event.currentTarget.value)}
          />
        </Field>
      </Section>

      <Section title="Combate">
        <NumField label="Iniciativa" block={props.block} setBlock={props.setBlock} campo="iniciativa" />
        <NumField label="Percepção" block={props.block} setBlock={props.setBlock} campo="percepcao" />
        <NumField label="Defesa" block={props.block} setBlock={props.setBlock} campo="defesa" min={0} />
        <NumField label="Pontos de Vida" block={props.block} setBlock={props.setBlock} campo="hp" min={1} />
        {/* PM é opcional porque a linha só existe em conjurador (Centauro Xamã,
            20 PM, p290) — um zero fixo diria "tem mana e está sem". */}
        <Field label="Pontos de Mana" id="creature-pm">
          <div class="flex items-center gap-2">
            <Show
              when={props.block.pm !== undefined}
              fallback={
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={() => props.setBlock('pm', 0)}
                >
                  <Plus aria-hidden="true" class="size-3" /> Conjura
                </Button>
              }
            >
              <NumberInput
                id="creature-pm"
                min={0}
                max={999}
                spinnerLabel="Pontos de Mana"
                value={props.block.pm ?? 0}
                onChange={(pm) => props.setBlock('pm', pm)}
              />
              <Button
                type="button"
                size="sm"
                variant="ghost"
                aria-label="Esta criatura não conjura"
                onClick={() => props.setBlock('pm', undefined)}
              >
                <Trash2 aria-hidden="true" class="size-3.5" />
              </Button>
            </Show>
          </div>
        </Field>
      </Section>

      <Section title="Resistências">
        <NumField label="Fortitude" block={props.block} setBlock={props.setBlock} campo="fortitude" />
        <NumField label="Reflexos" block={props.block} setBlock={props.setBlock} campo="reflexos" />
        <NumField label="Vontade" block={props.block} setBlock={props.setBlock} campo="vontade" />
      </Section>

      <Section title="Atributos">
        <NumField label="For" block={props.block} setBlock={props.setBlock} campo="forca" />
        <NumField label="Des" block={props.block} setBlock={props.setBlock} campo="destreza" />
        <NumField label="Con" block={props.block} setBlock={props.setBlock} campo="constituicao" />
        <NumField label="Int" block={props.block} setBlock={props.setBlock} campo="inteligencia" />
        <NumField label="Sab" block={props.block} setBlock={props.setBlock} campo="sabedoria" />
        <NumField label="Car" block={props.block} setBlock={props.setBlock} campo="carisma" />
      </Section>

      <AttacksEditor block={props.block} setBlock={props.setBlock} />
      <SkillsEditor block={props.block} setBlock={props.setBlock} />

      <Section title="Posses">
        <Field label="Equipamento" id="creature-equipment">
          <Input
            id="creature-equipment"
            value={props.block.equipment}
            placeholder="Clava, couro batido"
            onInput={(event) => props.setBlock('equipment', event.currentTarget.value)}
          />
        </Field>
        <Field label="Tesouro" id="creature-treasure">
          <Input
            id="creature-treasure"
            value={props.block.treasure}
            placeholder="Metade"
            onInput={(event) => props.setBlock('treasure', event.currentTarget.value)}
          />
        </Field>
      </Section>
    </div>
  )
}

function AttacksEditor(props: {
  block: CreatureBlock
  setBlock: SetStoreFunction<CreatureBlock>
}) {
  return (
    <section class="space-y-2">
      <div class="flex items-center justify-between">
        <SectionTitle>Ataques</SectionTitle>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() =>
            props.setBlock('attacks', (list) => [...list, { name: '', attackBonus: 0, damage: '' }])
          }
        >
          <Plus aria-hidden="true" class="size-3" /> Ataque
        </Button>
      </div>
      {/* `Index` e não `For`: a lista se recria a cada tecla, e o `For` (que
          reconcilia por REFERÊNCIA) perderia o foco do campo digitado. */}
      <Index each={props.block.attacks}>
        {(attack, i) => (
          <div class="flex flex-wrap items-end gap-2">
            <Field label="Nome" id={`attack-name-${i}`} class="min-w-[9rem] flex-1">
              <Input
                id={`attack-name-${i}`}
                value={attack().name}
                placeholder="Clava"
                onInput={(event) => props.setBlock('attacks', i, 'name', event.currentTarget.value)}
              />
            </Field>
            <Field label="Ataque" id={`attack-bonus-${i}`} class="w-24">
              <NumberInput
                id={`attack-bonus-${i}`}
                min={-20}
                max={60}
                spinnerLabel={`Ataque ${i + 1}`}
                value={attack().attackBonus}
                onChange={(bonus) => props.setBlock('attacks', i, 'attackBonus', bonus)}
              />
            </Field>
            <Field label="Dano" id={`attack-damage-${i}`} class="w-28">
              <Input
                id={`attack-damage-${i}`}
                value={attack().damage}
                placeholder="1d6+3"
                onInput={(event) =>
                  props.setBlock('attacks', i, 'damage', event.currentTarget.value)
                }
              />
            </Field>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              aria-label={`Remover ataque ${i + 1}`}
              onClick={() =>
                props.setBlock('attacks', (list) => list.filter((_, at) => at !== i))
              }
            >
              <Trash2 aria-hidden="true" class="size-3.5" />
            </Button>
          </div>
        )}
      </Index>
    </section>
  )
}

function SkillsEditor(props: {
  block: CreatureBlock
  setBlock: SetStoreFunction<CreatureBlock>
}) {
  return (
    <section class="space-y-2">
      <div class="flex items-center justify-between">
        <SectionTitle>Perícias</SectionTitle>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => props.setBlock('skills', (list) => [...list, { name: '', bonus: 0 }])}
        >
          <Plus aria-hidden="true" class="size-3" /> Perícia
        </Button>
      </div>
      <Index each={props.block.skills}>
        {(skill, i) => (
          <div class="flex flex-wrap items-end gap-2">
            <Field label="Nome" id={`skill-name-${i}`} class="min-w-[9rem] flex-1">
              <Input
                id={`skill-name-${i}`}
                value={skill().name}
                placeholder="Furtividade"
                onInput={(event) => props.setBlock('skills', i, 'name', event.currentTarget.value)}
              />
            </Field>
            <Field label="Bônus" id={`skill-bonus-${i}`} class="w-24">
              <NumberInput
                id={`skill-bonus-${i}`}
                min={-20}
                max={60}
                spinnerLabel={`Perícia ${i + 1}`}
                value={skill().bonus}
                onChange={(bonus) => props.setBlock('skills', i, 'bonus', bonus)}
              />
            </Field>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              aria-label={`Remover perícia ${i + 1}`}
              onClick={() => props.setBlock('skills', (list) => list.filter((_, at) => at !== i))}
            >
              <Trash2 aria-hidden="true" class="size-3.5" />
            </Button>
          </div>
        )}
      </Index>
    </section>
  )
}

const SELECT =
  'h-9 w-full rounded-sm border border-input bg-transparent px-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50'

/** Um campo numérico simples do bloco, ligado direto ao store. */
function NumField(props: {
  label: string
  campo: 'iniciativa' | 'percepcao' | 'defesa' | 'hp' | 'fortitude' | 'reflexos' | 'vontade'
    | 'forca' | 'destreza' | 'constituicao' | 'inteligencia' | 'sabedoria' | 'carisma'
  block: CreatureBlock
  setBlock: SetStoreFunction<CreatureBlock>
  min?: number
}) {
  const id = () => `creature-${props.campo}`
  return (
    <Field label={props.label} id={id()}>
      <NumberInput
        id={id()}
        min={props.min ?? -20}
        max={999}
        spinnerLabel={props.label}
        value={props.block[props.campo]}
        onChange={(value) => props.setBlock(props.campo, value)}
      />
    </Field>
  )
}

/** Rótulo associado por `for`+`id`: o biome recusa `<label>` envolvendo
 *  componente próprio, que ele não sabe ser controle de formulário. */
function Field(props: { label: string; id: string; class?: string; children: unknown }) {
  return (
    <div class={props.class ?? 'w-28 space-y-1'}>
      <label
        for={props.id}
        class="block text-3xs uppercase tracking-widest text-muted-foreground"
      >
        {props.label}
      </label>
      {props.children as never}
    </div>
  )
}

function Section(props: { title: string; children: unknown }) {
  return (
    <section class="space-y-2">
      <SectionTitle>{props.title}</SectionTitle>
      <div class="flex flex-wrap gap-2">{props.children as never}</div>
    </section>
  )
}

function SectionTitle(props: { children: unknown }) {
  return (
    <h4 class="font-heading text-2xs uppercase tracking-[0.14em] text-grimorio-gold">
      {props.children as never}
    </h4>
  )
}
