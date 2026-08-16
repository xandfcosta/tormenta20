import { type JSX, createSignal } from 'solid-js'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { Select } from '@/shared/ui/select'
import { TERRAIN_IDS, TERRAIN_LABEL } from './board-terrain'

const TERRAIN_OPTIONS = TERRAIN_IDS.map((value) => ({ value, label: TERRAIN_LABEL[value] }))

/**
 * O mestre abre um tabuleiro (ALE-124).
 *
 * Pede o LUGAR antes da grade porque é uma cena que ele está montando, não uma
 * planilha: "Taverna do Javali" é o que a mesa vai ler no topo. O tamanho vem em
 * QUADRADOS, e o diálogo diz a conversão do livro (1 quadrado = 1,5m, p236) — a
 * ficha fala em metros, e sem a conversão à vista o mestre teria de fazê-la de
 * cabeça toda vez.
 *
 * Abrir tabuleiro NÃO inicia combate: a cena de interpretação também tem
 * posição, e essa ortogonalidade é o que faz a taverna existir.
 *
 * @example <OpenBoardDialog onOpen={rt.openBoard} trigger={(open) => <Button …/>} />
 */
export function OpenBoardDialog(props: {
  onOpen: (place: string, terrain: string) => void
  trigger: (open: () => void) => JSX.Element
}) {
  const [open, setOpen] = createSignal(false)
  const [place, setPlace] = createSignal('')
  const [terrain, setTerrain] = createSignal(TERRAIN_IDS[0])

  const confirm = () => {
    props.onOpen(place().trim() || 'Cena', terrain())
    setOpen(false)
  }

  return (
    <>
      {props.trigger(() => setOpen(true))}
      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class="max-w-sm">
          <DialogHeader>
            <DialogTitle>Abrir tabuleiro</DialogTitle>
            <DialogDescription>
              O tabuleiro não tem bordas: cada quadrado é 1,5m (p236) e a cena cresce para onde
              a mesa for. O deslocamento padrão de 9m anda 6 quadrados (p106).
            </DialogDescription>
          </DialogHeader>

          <div class="space-y-3">
            <div class="space-y-1">
              <label class="text-xs uppercase tracking-wide text-muted-foreground" for="board-place">
                Lugar
              </label>
              <Input
                id="board-place"
                value={place()}
                placeholder="Taverna do Javali"
                onInput={(event) => setPlace(event.currentTarget.value)}
              />
            </div>

            {/* "Cenário" e não "Lugar": o campo de cima já se chama Lugar, e dois
                controles com o mesmo nome acessível no mesmo diálogo deixam o
                leitor de tela (e o teste) sem saber qual é qual. */}
            <Select
              aria-label="Cenário"
              options={TERRAIN_OPTIONS}
              value={TERRAIN_OPTIONS.find((option) => option.value === terrain()) ?? null}
              onChange={(option) => setTerrain(option?.value ?? TERRAIN_IDS[0])}
            />
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="button" onClick={confirm}>
              Abrir
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
