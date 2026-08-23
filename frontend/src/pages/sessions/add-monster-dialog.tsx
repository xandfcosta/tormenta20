import { Dices } from 'lucide-solid'
import { Show, createSignal } from 'solid-js'
import type { Monster } from '@/shared/api/catalog-types'
import { MonsterDetail } from '@/features/gm-tools/monster-detail'
import { rollD20 } from '@/shared/lib/dice'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Label } from '@/shared/ui/label'
import { NumberInput } from '@/shared/ui/number-input'

/** O que o mestre ajustou antes de a criatura entrar. */
export type EntradaAjustada = { hp: number; initiative: number; quantidade: number }

/** Teto de cópias num gesto. Não é regra do livro: é o que separa "quatro
 *  goblins" de um zero a mais que enche a fila e o mestre tem de desfazer
 *  linha por linha. A fila tem teto próprio no servidor, e ele responde. */
const MAX_COPIAS = 12

/** O `min`/`max` do `NumberInput` governa o SPINNER; digitar passa direto por
 *  ele, e quem prende é o chamador (ALE-236). Sem isto, teclar `99` em
 *  "Quantas" mandava 99 criaturas para a fila de uma vez. */
function preso(valor: number, min: number, max: number): number {
  if (!Number.isFinite(valor)) return min
  return Math.min(max, Math.max(min, Math.round(valor)))
}

/**
 * LER a criatura e, se for o caso, AJUSTÁ-LA antes de ela entrar (ALE-208).
 *
 * Clicar num card do bestiário jogava a criatura direto na iniciativa ao vivo,
 * com PV do livro e um d20 cru — não havia caminho para só olhar. E o que
 * entrava era quase sempre errado: o mestre quer o ogro com menos PV, ou a
 * iniciativa que ele acabou de rolar nos dados de verdade em cima da mesa.
 * Ajustar depois custa achar a linha na fila e editar campo por campo.
 *
 * UM diálogo para os dois gestos, e não um "ler" e um "adicionar" separados: a
 * linha da lista fica com um alvo de toque só — o que importa no telefone — e
 * some o problema de dois botões precisarem de nomes acessíveis distintos na
 * mesma linha. Quem só queria ler abre, lê e sai pelo Esc; nada entra.
 *
 * O DIÁLOGO É O DONO DO RASCUNHO: os campos nascem do bloco do livro a cada
 * abertura. Sem isso, o PV que o mestre baixou para um ogro reapareceria no
 * próximo, e ele não tem como saber que carregou.
 *
 * @example <AddMonsterDialog monster={escolhido()} onAdd={adiciona} onClose={fecha} />
 */
export function AddMonsterDialog(props: {
  /** A criatura aberta, ou `null` com o diálogo fechado. */
  monster: Monster | null
  onAdd: (monster: Monster, ajuste: EntradaAjustada) => void
  onClose: () => void
}) {
  return (
    // `keyed` com parâmetro declarado: sem o `(m) =>` o Solid não reconstrói o
    // bloco, e os campos do ogro anterior ficariam de pé sobre a criatura nova
    // (armadilha do guia do front). A chave é o próprio verbete.
    <Dialog open={props.monster !== null} onOpenChange={(open) => !open && props.onClose()}>
      <DialogContent class="dialogo-de-ficha flex w-full max-w-3xl flex-col gap-0 overflow-hidden p-0 sm:max-w-3xl">
        <Show when={props.monster} keyed>
          {(monster) => (
            <Corpo monster={monster} onAdd={props.onAdd} onClose={props.onClose} />
          )}
        </Show>
      </DialogContent>
    </Dialog>
  )
}

function Corpo(props: {
  monster: Monster
  onAdd: (monster: Monster, ajuste: EntradaAjustada) => void
  onClose: () => void
}) {
  const [hp, setHp] = createSignal(props.monster.hp)
  const [initiative, setInitiative] = createSignal(rollD20())
  const [quantidade, setQuantidade] = createSignal(1)

  const confirmar = () => {
    props.onAdd(props.monster, { hp: hp(), initiative: initiative(), quantidade: quantidade() })
    props.onClose()
  }

  return (
    <>
      <DialogHeader class="shrink-0 border-b border-grimorio-iron px-3 py-2 sm:px-4">
        <DialogTitle class="font-heading uppercase tracking-wide text-grimorio-gold">
          {props.monster.name}
        </DialogTitle>
      </DialogHeader>

      {/* O bloco do livro rola; a faixa de ajuste fica ancorada embaixo, porque
          é ela que o mestre veio usar quando não veio só ler. */}
      <div class="min-h-0 flex-1 overflow-y-auto p-3 sm:p-4">
        <MonsterDetail monster={props.monster} />
      </div>

      <div class="shrink-0 space-y-3 border-t border-grimorio-iron p-3 sm:p-4">
        <div class="flex flex-wrap items-end gap-3">
          <div class="min-w-24 flex-1 space-y-1">
            <Label for="add-monster-hp">PV</Label>
            <NumberInput
              id="add-monster-hp"
              value={hp()}
              onChange={(pv) => setHp(preso(pv, 1, 9999))}
              min={1}
              step={1}
              spinnerLabel="PV"
            />
          </div>
          <div class="min-w-32 flex-1 space-y-1">
            <Label for="add-monster-init">Iniciativa</Label>
            <div class="flex items-center gap-1">
              <NumberInput
                id="add-monster-init"
                value={initiative()}
                onChange={(valor) => setInitiative(preso(valor, 1, 99))}
                min={1}
                step={1}
                spinnerLabel="iniciativa"
              />
              {/* Rola no MESMO campo em vez de rolar e gravar: o mestre que já
                  jogou o dado na mesa digita, e quem não jogou usa este. */}
              <Button
                type="button"
                variant="outline"
                size="icon"
                aria-label="Rolar a iniciativa"
                onClick={() => setInitiative(rollD20())}
              >
                <Dices aria-hidden="true" class="size-4" />
              </Button>
            </div>
          </div>
          <div class="min-w-20 flex-1 space-y-1">
            <Label for="add-monster-qtd">Quantas</Label>
            <NumberInput
              id="add-monster-qtd"
              value={quantidade()}
              onChange={(quantas) => setQuantidade(preso(quantas, 1, MAX_COPIAS))}
              min={1}
              max={MAX_COPIAS}
              step={1}
              spinnerLabel="cópias"
            />
          </div>
        </div>

        {/* Com mais de uma, a iniciativa é UMA para o grupo — é o que a mesa
            faz com um bando de goblins, e rolar uma por cópia daria uma fila
            de sete linhas intercaladas que ninguém pediu. */}
        <Show when={quantidade() > 1}>
          <p class="text-xs text-muted-foreground">
            As {quantidade()} entram juntas, com a mesma iniciativa. O servidor numera:{' '}
            {props.monster.name}, {props.monster.name} 2, …
          </p>
        </Show>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={props.onClose}>
            Cancelar
          </Button>
          <Button type="button" onClick={confirmar}>
            Adicionar
          </Button>
        </DialogFooter>
      </div>
    </>
  )
}
