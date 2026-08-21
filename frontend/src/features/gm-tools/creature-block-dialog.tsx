import { useQueryClient } from '@tanstack/solid-query'
import { type JSX, Show, createSignal, onMount } from 'solid-js'
import { createStore, unwrap } from 'solid-js/store'
import type { CampaignCreature, CreatureBlock } from '@/shared/api/creature-types'
import { blankCreatureBlock } from '@/shared/api/creature-types'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { Input } from '@/shared/ui/input'
import { CreatureBlockForm } from './creature-block-form'
import { creatureActions } from './creature-mutations'

/**
 * Onde o mestre escreve o bloco de criatura (ALE-137) — a resposta à queixa que
 * abriu a issue: *"o mestre acaba tendo que imaginar, ou inventar, ou anotar em
 * algum lugar os itens, PM, perícias"*.
 *
 * Um diálogo, e não campos soltos na linha da iniciativa, porque a lista se
 * recria a cada broadcast do socket: um input vivo ali perderia o foco a cada
 * tecla (a mesma razão pela qual a iniciativa se edita por diálogo desde a
 * ALE-122).
 *
 * O erro aparece INLINE e não como toast: o toast disparado de dentro de um
 * modal não é anunciado, porque o modal marca os irmãos como `aria-hidden` e a
 * região do sonner é irmã.
 *
 * @example <CreatureBlockDialog campaignId={1} creature={vilao} trigger={…} />
 */
export function CreatureBlockDialog(props: {
  campaignId: number
  /** Ausente = criar do zero. */
  creature?: CampaignCreature
  /** Bloco de partida ao criar — o "editar este ogro" copia o verbete. */
  seed?: { name: string; block: CreatureBlock }
  onSaved?: (creature: CampaignCreature) => void
  /**
   * Ausente = o diálogo foi montado JÁ para ser usado e abre sozinho — é o
   * caminho do "NPC completo", onde a decisão de abrir já foi tomada na forma
   * de adicionar. Com gatilho, quem abre é o clique.
   */
  trigger?: (open: () => void) => JSX.Element
  /** Fechado sem salvar. Quem montou precisa saber para desmontar. */
  onDismiss?: () => void
}) {
  const queryClient = useQueryClient()
  const [open, setOpen] = createSignal(false)
  const [name, setName] = createSignal('')
  const [block, setBlock] = createStore<CreatureBlock>(blankCreatureBlock())
  const [failure, setFailure] = createSignal<string | null>(null)
  const [saving, setSaving] = createSignal(false)

  // Cada abertura recomeça do que está no banco (ou da semente): sem isto, o
  // diálogo mostraria o rascunho abandonado da vez anterior.
  const start = () => {
    const source = props.creature ?? props.seed
    setName(source?.name ?? '')
    setBlock(structuredClone(source ? source.block : blankCreatureBlock()))
    setFailure(null)
    setOpen(true)
  }

  const save = async () => {
    setSaving(true)
    setFailure(null)
    try {
      const actions = creatureActions(queryClient, props.campaignId)
      const input = { name: name(), block: unwrap(block) }
      const saved = props.creature
        ? await actions.update(props.creature.id, input)
        : await actions.create(input)
      props.onSaved?.(saved)
      setOpen(false)
    } catch (error) {
      // A mensagem do servidor diz o campo e o valor recusado ("tipo \"dragao\"
      // is not one of the book's creature types"), então ela vale mais que um
      // "falhou" nosso.
      setFailure(error instanceof Error ? error.message : 'Não foi possível salvar a criatura')
    } finally {
      setSaving(false)
    }
  }

  // Sem gatilho, abrir é a razão de ele existir na árvore.
  onMount(() => {
    if (!props.trigger) start()
  })

  const close = () => {
    setOpen(false)
    props.onDismiss?.()
  }

  return (
    <Dialog open={open()} onOpenChange={(next) => (next ? start() : close())}>
      {/* Sem `DialogTrigger`: ele envolve o filho num `span role="button"`, e
          o resultado eram DOIS botões com o mesmo nome acessível em volta do
          mesmo clique. O diálogo já é controlado, então quem abre é o `start`
          que a própria função de gatilho recebe. */}
      <Show when={props.trigger}>{(trigger) => trigger()(start)}</Show>
      <DialogContent class="max-h-[85vh] w-[calc(100vw-1.5rem)] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle class="font-heading uppercase tracking-wide text-grimorio-gold">
            {props.creature ? 'Editar criatura' : 'Nova criatura'}
          </DialogTitle>
        </DialogHeader>

        <div class="space-y-4">
          <div class="space-y-1">
            <label
              for="creature-name"
              class="block text-3xs uppercase tracking-widest text-muted-foreground"
            >
              Nome
            </label>
            <Input
              id="creature-name"
              value={name()}
              maxLength={60}
              placeholder="Chefe bandido"
              onInput={(event) => setName(event.currentTarget.value)}
            />
          </div>

          <CreatureBlockForm block={block} setBlock={setBlock} />
        </div>

        <Show when={failure()}>{(message) => <DialogInlineError message={message()} />}</Show>

        <DialogFooter>
          <Button type="button" variant="ghost" onClick={close}>
            Cancelar
          </Button>
          <Button type="button" disabled={saving() || name().trim() === ''} onClick={save}>
            {saving() ? 'Salvando…' : 'Salvar'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
