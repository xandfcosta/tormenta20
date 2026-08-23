import { type JSX, For, Show, createMemo, createSignal } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { Input } from '@/shared/ui/input'
import { ChooseAbilitiesDialog } from './choose-abilities-dialog'
import { PowerActionSlot } from './power-action-slot'
import { PowerPlayList } from './power-play-list'
import { ownedPowerSpec } from './power-spec-resolver'
import { normalize } from './normalize'
import { ownedAbilities } from './sheet-search-index'
import { SectionTitle } from '@/shared/ui/section-label'
import { Panel } from '@/shared/ui/panel'

/**
 * O bloco Poderes: UMA lista, a da mesa (ALE-217).
 *
 * Ele tinha dois MODOS — jogo e edição — e o dono resumiu o resultado em "está
 * difícil de ser usada". A medição achou três causas, e as três eram do modo de
 * edição: ele abria sozinho sempre que havia pendência (o estado normal de quem
 * subiu de nível), o cromo dele comia 44% do painel no telefone — 235px de 530,
 * deixando cinco linhas para os 26 a 34 poderes de um personagem de nível 20 —,
 * e as três abas de fonte diziam Raça/Origem/Classe para cinco procedências
 * diferentes.
 *
 * Escolher poder acontece uma vez por nível, e por isso a administração inteira
 * virou `ChooseAbilitiesDialog`: um botão que abre diálogo, a gramática que a
 * casa padronizou (ALE-169, ALE-216). O que sobra aqui é o que a mesa usa — as
 * ações em cima, as passivas recolhidas — com 69px de cromo em vez de 235.
 *
 * A busca continua PLANA e por nome: na mesa o jogador sabe como o poder se
 * chama, não qual fonte o concedeu.
 */
export function AbilitiesPanel(props: { character: Character }) {
  const [query, setQuery] = createSignal('')

  return (
    <Panel as="section" fillHeight>
      <div class="shrink-0 border-b border-grimorio-iron px-3 py-2 sm:px-4">
        <SectionTitle contexto="painel">Poderes</SectionTitle>
      </div>

      <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden p-3 sm:p-4">
        <div class="flex shrink-0 items-center gap-2">
          <Input
            value={query()}
            onInput={(event) => setQuery(event.currentTarget.value)}
            placeholder="Buscar poder ou habilidade…"
            aria-label="Buscar poder ou habilidade"
            class="h-8 min-w-0 flex-1 text-xs"
          />
          <ChooseAbilitiesDialog character={props.character} />
        </div>

        <Show
          when={query().trim() === ''}
          fallback={<FlatAbilityResults character={props.character} query={query()} />}
        >
          <PowerPlayList character={props.character} />
        </Show>
      </div>
    </Panel>
  )
}

/** Flat, source-badged results for the abilities search. */
function FlatAbilityResults(props: { character: Character; query: string }) {
  // normalize: accent-insensitive ("furia" finds "Fúria"), same helper as the
  // sheet's other searches.
  const results = createMemo(() => {
    const search = normalize(props.query.trim())
    return ownedAbilities(props.character).filter((entry) =>
      normalize(entry.name).includes(search),
    )
  })

  return (
    <Show
      when={results().length > 0}
      fallback={<EmptyHint>Nenhum poder para "{props.query}".</EmptyHint>}
    >
      <ul class="min-h-0 flex-1 space-y-2 overflow-y-auto pr-1">
        <For each={results()}>
          {(entry) => (
            <li class="rounded-none border border-border p-2">
              <div class="flex flex-wrap items-center gap-1.5">
                <p class="text-xs font-semibold">{entry.name}</p>
                <span class="rounded-none bg-muted px-1 py-0 text-4xs text-muted-foreground">
                  {entry.source}
                </span>
                <PowerActionSlot
                  spec={ownedPowerSpec(entry)}
                  character={props.character}
                  class="ml-auto"
                />
              </div>
              <p class="mt-0.5 text-2xs leading-snug text-muted-foreground">{entry.detail}</p>
            </li>
          )}
        </For>
      </ul>
    </Show>
  )
}

function EmptyHint(props: { children: JSX.Element }) {
  return <p class="text-xs italic text-muted-foreground">{props.children}</p>
}
