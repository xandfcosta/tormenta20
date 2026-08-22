import { useQuery } from '@tanstack/solid-query'
import { Dices } from 'lucide-solid'
import { Show, createSignal } from 'solid-js'
import { computedSheetFor, expertiseFromSheet } from '@/entities/character/computed-sheet'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character } from '@/shared/api/api'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Label } from '@/shared/ui/label'
import { NumberInput } from '@/shared/ui/number-input'
import { rollD20 } from '@/shared/lib/dice'
import { Skeleton } from '@/shared/ui/skeleton'
import { settledQuery } from '@/shared/lib/settled-query'

/** Um d20 é um d20: o campo é digitado, e um dedo escorregado põe 133 na frente
 *  da fila. O servidor recusa fora daqui; a tela evita chegar lá. */
const D20_MIN = 1
const D20_MAX = 20

/**
 * O jogador REGISTRA a própria iniciativa (ALE-213).
 *
 * Um caminho só, com a escolha por dentro: as duas mesas existem — a que rola
 * dado de verdade na mesa e digita o que caiu, e a que deixa o app rolar. Dois
 * botões lado a lado seriam duas portas para uma decisão que se toma uma vez
 * por cena; aqui o `🎲 Rolar` preenche o mesmo campo que o dedo preencheria, e
 * dá para ver o bônus antes de decidir.
 *
 * **O total é PREVISÃO, não verdade.** Quem soma é o Go (`initiative-self` manda
 * o d20 e nada mais), porque o bônus da perícia é regra do livro e uma soma
 * escrita aqui seria livre para divergir do motor. O número da tela vem da MESMA
 * `ComputeSheetV2`, só que compilada para WASM — é antecipação, no sentido que a
 * casa dá à palavra, e não uma segunda autoridade.
 *
 * @example <InitiativeRollButton characterId={7} rt={rt} />
 */
export function InitiativeRollButton(props: { characterId: number; rt: SessionRealtime }) {
  const character = useQuery(() => characterQueryOptions(props.characterId))

  return (
    <Show when={settledQuery(character)} fallback={<Skeleton class="h-8 w-44" />}>
      {(data) => <RegisterDialog character={data()} rt={props.rt} />}
    </Show>
  )
}

function RegisterDialog(props: { character: Character; rt: SessionRealtime }) {
  const conditionals = useConditionals()
  const [open, setOpen] = createSignal(false)
  const [d20, setD20] = createSignal(10)

  const bonus = () => {
    const sheet = computedSheetFor(props.character, conditionals.active(props.character.id))
    return expertiseFromSheet(sheet, 'Iniciativa')?.total ?? 0
  }
  const previsto = () => d20() + bonus()
  /**
   * O campo é DIGITADO, e o `NumberInput` não apara o que se digita — `min`/`max`
   * governam só os botões de passo. Limpar o campo dá 0, e um dedo escorregado
   * dá 133.
   *
   * A tela não oferece o que o servidor recusaria, que é a regra da casa. E aqui
   * ela não é só cortesia: o cliente NÃO escuta o `exception` do socket, então
   * uma recusa do servidor sumiria em silêncio e o jogador clicaria "Registrar"
   * olhando para uma tela que não muda. (O buraco do `exception` é anterior a
   * esta issue e vale para toda mutação — está registrado na ALE-213.)
   */
  const valido = () => Number.isInteger(d20()) && d20() >= D20_MIN && d20() <= D20_MAX

  const registrar = () => {
    props.rt.rollSelfInitiative(props.character.id, d20())
    setOpen(false)
  }

  return (
    <>
      <Button
        type="button"
        size="sm"
        variant="secondary"
        class="gap-1.5"
        disabled={!props.rt.isConnected()}
        onClick={() => setOpen(true)}
      >
        <Dices aria-hidden="true" class="size-4" />
        Registrar iniciativa
      </Button>

      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class="max-w-sm">
          <DialogHeader>
            <DialogTitle>Registrar iniciativa</DialogTitle>
            <DialogDescription>{props.character.name}</DialogDescription>
          </DialogHeader>

          <div class="space-y-3">
            <div class="flex items-end gap-2">
              {/* `for`+`id` e não o `label` por fora: o `NumberInput` é
                  componente da casa, e o biome não sabe que há um controle
                  dentro dele (`noLabelWithoutControl`). */}
              <div class="min-w-0 flex-1 space-y-1">
                <Label for="d20-registrado">Seu d20</Label>
                <NumberInput
                  id="d20-registrado"
                  value={d20()}
                  onChange={setD20}
                  min={D20_MIN}
                  max={D20_MAX}
                  spinnerLabel="d20"
                />
              </div>
              <Button
                type="button"
                variant="outline"
                class="gap-1.5"
                onClick={() => setD20(rollD20())}
              >
                <Dices aria-hidden="true" class="size-4" />
                Rolar
              </Button>
            </div>

            <Show when={!valido()}>
              <p class="text-sm text-destructive">O d20 vai de {D20_MIN} a {D20_MAX}.</p>
            </Show>

            <dl class="space-y-1 border-t border-grimorio-iron pt-3 text-sm">
              <div class="flex items-baseline justify-between gap-2">
                <dt class="text-muted-foreground">Iniciativa (perícia)</dt>
                <dd class="font-mono tabular-nums">
                  {bonus() >= 0 ? '+' : ''}
                  {bonus()}
                </dd>
              </div>
              <div class="flex items-baseline justify-between gap-2">
                <dt class="font-medium">Total</dt>
                <dd class="font-mono text-lg tabular-nums text-grimorio-gold">{previsto()}</dd>
              </div>
            </dl>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button
              type="button"
              disabled={!props.rt.isConnected() || !valido()}
              onClick={registrar}
            >
              Registrar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
