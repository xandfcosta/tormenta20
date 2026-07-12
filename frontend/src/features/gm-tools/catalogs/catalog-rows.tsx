import type {
  CatalogItem,
  CatalogSpell,
  Condition,
} from '@tormenta20/t20-data'
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

function Row({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-md border border-border/60 p-2.5 text-sm">
      {children}
    </div>
  )
}

export function ConditionRow({ condition }: { condition: Condition }) {
  return (
    <Row>
      <p className="flex flex-wrap items-center gap-1.5 font-medium">
        {condition.name}
        {condition.tags.map((t) => (
          <Badge key={t} variant="outline" className="text-[10px] uppercase">
            {t}
          </Badge>
        ))}
      </p>
      <p className="mt-1 text-muted-foreground">{condition.description}</p>
      {condition.upgradesTo && (
        <p className="mt-1 text-xs text-muted-foreground">
          Agrava para <span className="font-medium">{condition.upgradesTo}</span>
        </p>
      )}
    </Row>
  )
}

export function SpellCatalogRow({ spell }: { spell: CatalogSpell }) {
  return (
    <Row>
      <p className="flex flex-wrap items-center gap-1.5 font-medium">
        {spell.name}
        <Badge className="text-[10px]">{spell.circle}º círculo</Badge>
        <Badge variant="secondary" className="text-[10px]">
          {SCHOOL_LABEL[spell.school] ?? spell.school}
        </Badge>
      </p>
      <p className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
        <span>Execução: {EXECUTION_LABEL[spell.execution] ?? spell.execution}</span>
        <span>Alcance: {RANGE_LABEL[spell.range] ?? spell.range}</span>
        <span>Duração: {spell.duration}</span>
        {spell.resistance && <span>Resistência: {spell.resistance}</span>}
      </p>
      <p className="mt-1 text-muted-foreground">{spell.baseEffect}</p>
      {spell.augments.length > 0 && (
        <p className="mt-1 text-xs text-muted-foreground">
          {spell.augments.length} aprimoramento
          {spell.augments.length === 1 ? '' : 's'} disponíveis.
        </p>
      )}
    </Row>
  )
}

export function PowerRow({ power }: { power: CatalogPower }) {
  return (
    <Row>
      <p className="flex flex-wrap items-center gap-1.5 font-medium">
        {power.name}
        <Badge variant="outline" className="text-[10px]">
          {power.source}
        </Badge>
      </p>
      <p className="mt-1 text-muted-foreground">{power.description}</p>
    </Row>
  )
}

export function ItemCatalogRow({ item }: { item: CatalogItem }) {
  return (
    <Row>
      <p className="flex flex-wrap items-center gap-1.5 font-medium">
        {item.name}
        <Badge variant="secondary" className="text-[10px]">
          {ITEM_CATEGORY_LABEL[item.category] ?? item.category}
        </Badge>
      </p>
      <p className="mt-1 flex flex-wrap gap-x-3 text-xs text-muted-foreground">
        <span>Preço: T$ {item.price}</span>
        <span>Espaços: {item.slots}</span>
        {item.weapon && (
          <span>
            Dano: {item.weapon.damage} · Crít: {item.weapon.critRange}+/×
            {item.weapon.critMult} · {item.weapon.type}
          </span>
        )}
      </p>
    </Row>
  )
}
