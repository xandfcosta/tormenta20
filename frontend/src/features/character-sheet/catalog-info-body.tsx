import type { CatalogItem } from '@/shared/api/item-types'
import type { Modifier } from '@/shared/api/item-types'
import { For, type JSX, Show } from 'solid-js'
import { FactChips } from './fact-chips'
import { describeCondition, describeModifierTarget, formatLoad } from './item-describe'
import { signed } from './signed'

/**
 * The item's full sheet — read inside two dialogs (the catálogo preview and the
 * bag's action sheet), so a new `CatalogItem` field surfaces in both at once.
 *
 * @example <CatalogInfoBody catalog={getCatalogItem('espada-longa')!} />
 */
export function CatalogInfoBody(props: { catalog: CatalogItem }) {
  return (
    <div class="space-y-3 text-xs">
      <div>
        <p class="font-semibold text-grimorio-gold">{props.catalog.name}</p>
        <p class="text-muted-foreground">
          {props.catalog.category} • esp {formatLoad(props.catalog.slots)} • T$ {props.catalog.price}{' '}
          • {props.catalog.equip === 'either' ? 'qualquer equipar' : props.catalog.equip}
          {props.catalog.hands ? ` • ${props.catalog.hands} mão(s)` : ''}
        </p>
      </div>

      <Show when={props.catalog.weapon}>
        {(weapon) => (
          <InfoBlock title="arma">
            <p>
              dano <span class="font-mono">{weapon().damage}</span> • crítico{' '}
              <span class="font-mono">
                {weapon().critRange}/×{weapon().critMult}
              </span>{' '}
              • {weapon().type} • {weapon().purpose}
              {weapon().range ? ` (${weapon().range})` : ''}
            </p>
            <Show when={weapon().traits.length > 0}>
              <p class="text-muted-foreground">propriedades: {weapon().traits.join(', ')}</p>
            </Show>
          </InfoBlock>
        )}
      </Show>

      <Show when={props.catalog.armor}>
        {(armor) => (
          <InfoBlock title="armadura">
            <p>
              Defesa +{armor().defense} • penalidade {armor().penalty} •{' '}
              {armor().heavy ? 'pesada' : 'leve'}
            </p>
          </InfoBlock>
        )}
      </Show>

      <Show when={props.catalog.shield}>
        {(shield) => (
          <InfoBlock title="escudo">
            <p>
              Defesa +{shield().defense} • penalidade {shield().penalty}
            </p>
          </InfoBlock>
        )}
      </Show>

      <InfoBlock title="modificadores">
        <Show
          when={props.catalog.modifiers.length > 0}
          fallback={<p class="text-muted-foreground">Nenhum.</p>}
        >
          <ul class="space-y-0.5">
            <For each={props.catalog.modifiers}>{(modifier) => <ModifierLine modifier={modifier} />}</For>
          </ul>
        </Show>
      </InfoBlock>

      <Show when={props.catalog.displayFacts?.length}>
        <InfoBlock title="outros efeitos">
          <FactChips facts={props.catalog.displayFacts ?? []} />
        </InfoBlock>
      </Show>
    </div>
  )
}

function InfoBlock(props: { title: string; children: JSX.Element }) {
  return (
    <div class="space-y-0.5">
      <p class="text-3xs uppercase tracking-widest text-muted-foreground">{props.title}</p>
      {props.children}
    </div>
  )
}

/** Flags are boolean — their amount (always 1) is bookkeeping, not a bonus,
 *  so it never renders. */
function ModifierLine(props: { modifier: Modifier }) {
  const condition = () => describeCondition(props.modifier)
  return (
    <li class="flex flex-wrap gap-x-1">
      <Show when={props.modifier.target.k !== 'flag'}>
        <span class="font-mono">{signed(props.modifier.amount)}</span>
      </Show>
      <span>{describeModifierTarget(props.modifier.target)}</span>
      <span class="text-3xs text-muted-foreground">[{props.modifier.bonusType}]</span>
      <Show when={condition()}>
        {(text) => <span class="text-3xs text-muted-foreground">— {text()}</span>}
      </Show>
    </li>
  )
}
