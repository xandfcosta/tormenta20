import { type JSX, createSignal } from 'solid-js'
import type { BoardToken } from '@/shared/realtime/realtime'
import { boardFootprint } from '@/shared/lib/engine-wasm'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'

/** Os tamanhos que o livro usa para espaço ocupado (T20 p107, Tab. 1-21). O
 *  número de quadrados NÃO é escrito aqui: quem responde é o motor. */
const TAMANHOS = ['Médio', 'Grande', 'Enorme', 'Colossal'] as const

/** Os três papéis de uma peça. Objeto é a porta, o baú, o barril — cenário que
 *  ocupa espaço e não tem turno. */
const TIPOS: { valor: BoardToken['kind']; rotulo: string }[] = [
  { valor: 'npc', rotulo: 'NPC' },
  { valor: 'character', rotulo: 'Personagem' },
  { valor: 'object', rotulo: 'Objeto' },
]

/**
 * Onde o mestre cria ou edita uma peça (ALE-178).
 *
 * Diálogo, e não campos soltos na barra: a lista de peças se recria a cada
 * broadcast do socket, e um input vivo ali perderia o foco a cada tecla — a
 * mesma razão pela qual a iniciativa se edita por diálogo desde a ALE-122.
 *
 * @example <TokenDialog onSave={salvar} trigger={(abrir) => <Button onClick={abrir}>Peça</Button>} />
 */
export function TokenDialog(props: {
  /** Ausente = criar do zero. */
  token?: BoardToken
  onSave: (patch: { label: string; kind: BoardToken['kind']; footprint: number }) => void
  trigger: (open: () => void) => JSX.Element
}) {
  const [open, setOpen] = createSignal(false)
  const [label, setLabel] = createSignal('')
  const [kind, setKind] = createSignal<BoardToken['kind']>('npc')
  const [tamanho, setTamanho] = createSignal<(typeof TAMANHOS)[number]>('Médio')

  // Cada abertura recomeça do que está no estado: sem isto, o diálogo mostraria
  // o rascunho abandonado da vez anterior.
  const start = () => {
    setLabel(props.token?.label ?? '')
    setKind(props.token?.kind ?? 'npc')
    setTamanho(nomeDoTamanho(props.token?.footprint ?? 1))
    setOpen(true)
  }

  const save = () => {
    // O lado do quadrado sai do MOTOR (T20 p107): escrever 1/2/3/6 aqui seria
    // uma segunda tabela do livro, e a primeira já existe no Go.
    props.onSave({ label: label().trim(), kind: kind(), footprint: boardFootprint(tamanho()) })
    setOpen(false)
  }

  return (
    <Dialog open={open()} onOpenChange={(next) => (next ? start() : setOpen(false))}>
      {props.trigger(start)}
      <DialogContent class="max-w-sm">
        <DialogHeader>
          <DialogTitle class="font-heading uppercase tracking-wide text-grimorio-gold">
            {props.token ? 'Editar peça' : 'Nova peça'}
          </DialogTitle>
        </DialogHeader>

        <div class="space-y-3">
          <div class="space-y-1">
            <label for="token-label" class="block text-[10px] uppercase tracking-widest text-muted-foreground">
              Nome
            </label>
            <Input
              id="token-label"
              value={label()}
              maxLength={40}
              placeholder="Porta emperrada"
              onInput={(event) => setLabel(event.currentTarget.value)}
            />
          </div>

          <fieldset class="space-y-1">
            <legend class="text-[10px] uppercase tracking-widest text-muted-foreground">Tipo</legend>
            <div class="flex gap-1">
              {TIPOS.map((tipo) => (
                <Button
                  type="button"
                  size="sm"
                  variant={kind() === tipo.valor ? 'default' : 'secondary'}
                  aria-pressed={kind() === tipo.valor}
                  onClick={() => setKind(tipo.valor)}
                >
                  {tipo.rotulo}
                </Button>
              ))}
            </div>
          </fieldset>

          <fieldset class="space-y-1">
            <legend class="text-[10px] uppercase tracking-widest text-muted-foreground">
              Tamanho
            </legend>
            <div class="flex flex-wrap gap-1">
              {TAMANHOS.map((nome) => (
                <Button
                  type="button"
                  size="sm"
                  variant={tamanho() === nome ? 'default' : 'secondary'}
                  aria-pressed={tamanho() === nome}
                  onClick={() => setTamanho(nome)}
                >
                  {nome}
                </Button>
              ))}
            </div>
          </fieldset>
        </div>

        <DialogFooter>
          <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
            Cancelar
          </Button>
          <Button type="button" disabled={label().trim() === ''} onClick={save}>
            Salvar
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** O nome do tamanho a partir do lado em quadrados — o caminho inverso do
 *  motor, para o diálogo abrir marcando o que a peça já é. */
function nomeDoTamanho(footprint: number): (typeof TAMANHOS)[number] {
  return TAMANHOS.find((nome) => boardFootprint(nome) === footprint) ?? 'Médio'
}
