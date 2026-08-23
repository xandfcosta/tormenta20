import { useQueryClient } from '@tanstack/solid-query'
import { Plus } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import { isItemProficient } from '@/entities/character/derived'
import type { Character, CharacterItem } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { toast } from '@/shared/ui/sonner'
import { cn } from '@/shared/lib/utils'
import { BagEquippedStrip } from './bag-equipped-strip'
import { type BagFilterKey, BAG_FILTERS, filterStowed } from './bag-filters'
import { BagItemSheet } from './bag-item-sheet'
import { partitionBag } from './bag-slots'
import { BagTile } from './bag-tile'
import { CatalogAddDialog } from './catalog-add-dialog'
import { formatLoad, loadLimitLabel } from './item-describe'
import { ItemFormDialog } from './item-form-dialog'
import { itemActions, itemWriteMessage } from './item-mutations'
import { TibarDialog } from './tibar-dialog'
import { tibarAction } from './tibar-write'
import { CargaMeter } from './carga-meter'
import { formatTibar } from '@/shared/lib/format-tibar'
import { FieldLabel, SectionLabel, SectionTitle } from '@/shared/ui/section-label'
import { Panel } from '@/shared/ui/panel'

/**
 * "Mochila" — the game-bag block that replaced the old Inventário table and
 * the Equipado tab. Top: the paper-doll strip (Mãos 2 / Vestidos 4) showing
 * what is empunhado/vestido at a glance. Below: the carga meter, search and
 * category chips over a tile grid of what is stowed. Every tile and every
 * equipped card opens the same action sheet.
 */
export function BagPanel(props: { character: Character }) {
  const queryClient = useQueryClient()
  const actions = () => itemActions(queryClient, props.character.id)

  const [query, setQuery] = createSignal('')
  const [filter, setFilter] = createSignal<BagFilterKey>('all')
  const [openItemId, setOpenItemId] = createSignal<number | null>(null)

  // The conditionals store lands with the Efeitos block (ALE-86); until then
  // the sheet computes against an empty opt-in set.
  const sheet = createMemo(() => computedSheetFor(props.character, new Set<string>()))
  // A carga inteira vem do motor (p141) — espaços ocupados, limite e
  // sobrecarga. A tela somava isto sozinha até a ALE-215, que é uma segunda
  // implementação de regra do livro rodando no navegador.
  const carga = () => sheet().carga
  // Lê `props.character.id` no MOMENTO da escrita, como o `actions()` acima: a
  // fábrica não guarda estado entre chamadas, e capturá-la no corpo congelaria o
  // id do personagem que estava aberto na montagem.
  const saveTibar = (tibar: number) => tibarAction(queryClient, props.character.id)(tibar)

  const partition = createMemo(() => partitionBag(props.character.items))
  const stowed = createMemo(() => filterStowed(partition().stowed, query(), filter()))
  const openItem = createMemo(() => props.character.items.find((it) => it.id === openItemId()))

  /** Quick unequip from the strip — it skips the sheet, so a refusal has
   *  nowhere to render except a toast. */
  const unequip = async (item: CharacterItem) => {
    try {
      await actions().change(item.id, { equipped: null })
    } catch (failure) {
      toast.error(itemWriteMessage(failure, 'Não foi possível desequipar o item.'))
    }
  }

  return (
    <Panel as="section" fillHeight>
      <div class="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-grimorio-iron px-3 py-2 sm:px-4">
        <div class="min-w-0">
          <SectionTitle contexto="painel">Mochila</SectionTitle>
          <p class="text-3xs text-muted-foreground sm:text-xs">
            carga{' '}
            <span
              class={cn(
                'font-mono',
                // Tinta e não cor de barra (ALE-240).
                carga().overloaded ? 'text-grimorio-crimson-bright' : 'text-foreground',
              )}
            >
              {formatLoad(carga().used)}
            </span>{' '}
            / {carga().limit}
            <Show when={carga().overloaded}>
              <FieldLabel tom="inherit" class="ml-2 text-grimorio-crimson-bright">
                sobrecarga
              </FieldLabel>
            </Show>
            {/* Com a regra desligada os espaços continuam contados — o livro
                condiciona o desligamento a "desde que os jogadores não abusem"
                (p141), e não dá para vigiar abuso sem ver o número. Sem esta
                linha, porém, uma carga acima do limite e SEM alerta pareceria
                defeito da tela (ALE-221). A frase é a do LIVRO, e ela diz de
                quem é a decisão sem repetir a palavra "limite", que a linha já
                usa duas vezes. */}
            <Show when={!carga().enforced}>
              <FieldLabel tom="inherit" class="ml-2">
                a critério do mestre
              </FieldLabel>
            </Show>
            <span class="ml-2">
              • {loadLimitLabel(carga().limit, sheet().attributes.strength.total)}
            </span>
          </p>
          <p class="flex items-center gap-1.5 text-3xs text-muted-foreground sm:text-xs">
            <span aria-hidden="true">⛃</span>
            <span class="font-mono text-foreground">T$ {formatTibar(props.character.tibar)}</span>
            <Show when={carga().coins > 0}>
              <span>· {formatLoad(carga().coins)} espaço{carga().coins === 1 ? '' : 's'}</span>
            </Show>
            <TibarDialog tibar={props.character.tibar} onSave={saveTibar} />
          </p>
        </div>
        <div class="flex items-center gap-2">
          <CatalogAddDialog onAdd={(input) => actions().add(input)} />
          <ItemFormDialog
            title="Novo item"
            submitLabel="Adicionar"
            onSubmit={(input) => actions().add(input)}
            trigger={(open) => (
              <Button
                type="button"
                size="sm"
                variant="outline"
                class="h-7 gap-1 text-xs"
                aria-label="Adicionar item custom"
                onClick={open}
              >
                <Plus aria-hidden="true" class="size-3.5" />
                Custom
              </Button>
            )}
          />
        </div>
      </div>

      <div class="min-h-0 flex-1 space-y-3 overflow-y-auto p-3 sm:p-4">
        <BagEquippedStrip
          partition={partition()}
          onOpen={(item) => setOpenItemId(item.id)}
          onUnequip={unequip}
        />

        <div class="space-y-2">
          <div class="flex items-baseline justify-between">
            <SectionLabel as="h3" tom="gold" class="text-3xs font-bold">
              Mochila (guardado)
            </SectionLabel>
            <span class="font-mono text-xs text-muted-foreground">
              {partition().stowed.length} ite{partition().stowed.length === 1 ? 'm' : 'ns'}
            </span>
          </div>

          <CargaMeter carga={carga()} />

          <div class="flex flex-wrap items-center gap-2">
            <Input
              type="search"
              value={query()}
              onInput={(event) => setQuery(event.currentTarget.value)}
              placeholder="Buscar item…"
              aria-label="Buscar item na mochila"
              class="h-8 min-w-32 flex-1 text-xs"
            />
            <div class="flex gap-1">
              <For each={BAG_FILTERS}>
                {(chip) => (
                  <button
                    type="button"
                    onClick={() => setFilter(chip.key)}
                    aria-pressed={filter() === chip.key}
                    class={cn(
                      'rounded-full border px-2 py-0.5 text-3xs uppercase tracking-wider transition-colors',
                      filter() === chip.key
                        ? 'border-grimorio-gold/60 bg-accent text-grimorio-gold'
                        : 'border-grimorio-iron text-muted-foreground hover:text-foreground',
                    )}
                  >
                    {chip.label}
                  </button>
                )}
              </For>
            </div>
          </div>

          <Show
            when={stowed().length > 0}
            fallback={
              <p class="py-4 text-center text-xs text-muted-foreground">
                {query().trim() !== '' || filter() !== 'all'
                  ? 'Nenhum item para esse filtro.'
                  : 'Mochila vazia. Use "+ Catálogo" para adicionar.'}
              </p>
            }
          >
            <div class="grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-6">
              <For each={stowed()}>
                {(item) => (
                  <BagTile
                    item={item}
                    proficient={isItemProficient(props.character, item)}
                    onOpen={() => setOpenItemId(item.id)}
                  />
                )}
              </For>
            </div>
          </Show>
        </div>
      </div>

      <Show when={openItem()}>
        {(item) => (
          <BagItemSheet
            item={item()}
            proficient={isItemProficient(props.character, item())}
            open
            onOpenChange={(next) => {
              if (!next) setOpenItemId(null)
            }}
            actions={actions()}
          />
        )}
      </Show>
    </Panel>
  )
}
