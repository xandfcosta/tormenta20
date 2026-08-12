import type { CatalogItem, CatalogSpell, Condition } from '@tormenta20/t20-data'
import { For, type JSX, Show } from 'solid-js'
import { Badge } from '@/shared/ui/badge'
import type { CatalogPower } from './catalog-model'

// Local label maps. The spell/school vocabulary also lives in
// features/character-sheet, but FSD forbids importing sideways between
// features, so the GM catalog keeps its own minimal copies.
const SCHOOL_LABEL: Record<string, string> = {
  evocacao: 'Evocação',
  abjuracao: 'Abjuração',
  encantamento: 'Encantamento',
  adivinhacao: 'Adivinhação',
  convocacao: 'Convocação',
  transmutacao: 'Transmutação',
  ilusao: 'Ilusão',
  necromancia: 'Necromancia',
}
const EXECUTION_LABEL: Record<string, string> = {
  padrao: 'Padrão',
  reacao: 'Reação',
  livre: 'Livre',
  completa: 'Completa',
  movimento: 'Movimento',
}
const RANGE_LABEL: Record<string, string> = {
  pessoal: 'Pessoal',
  toque: 'Toque',
  curto: 'Curto',
  medio: 'Médio',
  longo: 'Longo',
  ilimitado: 'Ilimitado',
}
const ITEM_CATEGORY_LABEL: Record<string, string> = {
  'weapon-simple': 'Arma simples',
  'weapon-martial': 'Arma marcial',
  'weapon-exotic': 'Arma exótica',
  'weapon-firearm': 'Arma de fogo',
  'armor-light': 'Armadura leve',
  'armor-heavy': 'Armadura pesada',
  shield: 'Escudo',
  apparel: 'Vestuário',
  consumable: 'Consumível',
  meal: 'Refeição',
  catalyst: 'Catalisador',
  improvement: 'Melhoria',
  material: 'Material',
  animal: 'Animal',
  vehicle: 'Veículo',
}

function Row(props: { children: JSX.Element }) {
  return (
    <div class="rounded-md border border-grimorio-iron p-2.5 text-sm">{props.children}</div>
  )
}

export function ConditionRow(props: { condition: Condition }) {
  return (
    <Row>
      <p class="flex flex-wrap items-center gap-1.5 font-medium">
        {props.condition.name}
        <For each={props.condition.tags}>
          {(tag) => (
            <Badge variant="outline" class="text-[10px] uppercase">
              {tag}
            </Badge>
          )}
        </For>
      </p>
      <p class="mt-1 text-muted-foreground">{props.condition.description}</p>
      <Show when={props.condition.upgradesTo}>
        {(upgrade) => (
          <p class="mt-1 text-xs text-muted-foreground">
            Agrava para <span class="font-medium">{upgrade()}</span>
          </p>
        )}
      </Show>
    </Row>
  )
}

export function SpellCatalogRow(props: { spell: CatalogSpell }) {
  return (
    <Row>
      <p class="flex flex-wrap items-center gap-1.5 font-medium">
        {props.spell.name}
        <Badge class="text-[10px]">{props.spell.circle}º círculo</Badge>
        <Badge variant="secondary" class="text-[10px]">
          {SCHOOL_LABEL[props.spell.school] ?? props.spell.school}
        </Badge>
      </p>
      <p class="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
        <span>Execução: {EXECUTION_LABEL[props.spell.execution] ?? props.spell.execution}</span>
        <span>Alcance: {RANGE_LABEL[props.spell.range] ?? props.spell.range}</span>
        <span>Duração: {props.spell.duration}</span>
        <Show when={props.spell.resistance}>
          {(resistance) => <span>Resistência: {resistance()}</span>}
        </Show>
      </p>
      <p class="mt-1 text-muted-foreground">{props.spell.baseEffect}</p>
      <Show when={props.spell.augments.length > 0}>
        <p class="mt-1 text-xs text-muted-foreground">
          {props.spell.augments.length} aprimoramento
          {props.spell.augments.length === 1 ? '' : 's'} disponíveis.
        </p>
      </Show>
    </Row>
  )
}

export function PowerCatalogRow(props: { power: CatalogPower }) {
  return (
    <Row>
      <p class="flex flex-wrap items-center gap-1.5 font-medium">
        {props.power.name}
        <Badge variant="outline" class="text-[10px]">
          {props.power.source}
        </Badge>
      </p>
      <p class="mt-1 text-muted-foreground">{props.power.description}</p>
    </Row>
  )
}

export function ItemCatalogRow(props: { item: CatalogItem }) {
  return (
    <Row>
      <p class="flex flex-wrap items-center gap-1.5 font-medium">
        {props.item.name}
        <Badge variant="secondary" class="text-[10px]">
          {ITEM_CATEGORY_LABEL[props.item.category] ?? props.item.category}
        </Badge>
      </p>
      <p class="mt-1 flex flex-wrap gap-x-3 text-xs text-muted-foreground">
        <span>Preço: T$ {props.item.price}</span>
        <span>Espaços: {props.item.slots}</span>
        <Show when={props.item.weapon}>
          {(weapon) => (
            <span>
              Dano: {weapon().damage} · Crít: {weapon().critRange}+/×{weapon().critMult} ·{' '}
              {weapon().type}
            </span>
          )}
        </Show>
      </p>
    </Row>
  )
}
