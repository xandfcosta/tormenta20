import { Hourglass, Moon } from 'lucide-solid'
import { createSignal } from 'solid-js'
import type { RestCondition, SessionRealtime } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Select } from '@/shared/ui/select'

const REST_OPTIONS: { value: RestCondition; label: string }[] = [
  { value: 'ruim', label: 'Ruim (½ nível)' },
  { value: 'normal', label: 'Normal (nível)' },
  { value: 'confortavel', label: 'Confortável (2×)' },
  { value: 'luxuosa', label: 'Luxuosa (3×)' },
]

/**
 * Descanso de cena e de dia como AÇÕES RÁPIDAS da faixa do turno (ALE-122).
 *
 * O descanso de cena não precisa de nada além do clique, então acontece no
 * clique. O de dia precisa da qualidade que a p105 usa para calcular a
 * recuperação, e só aí abre o diálogo — pedir dois passos para o que tem um
 * seria cobrar por um dado que o motor não usa.
 *
 * Estavam num painel de duas linhas dentro do menu da sessão: correto para o
 * que se faz raramente, errado para o que o mestre faz ao fim de toda cena.
 */
export function RestControls(props: { rt: SessionRealtime }) {
  const [open, setOpen] = createSignal(false)
  const [condition, setCondition] = createSignal<RestCondition>('normal')

  const restDay = () => {
    props.rt.rest('day', condition())
    setOpen(false)
  }

  return (
    <>
      <Button
        size="sm"
        variant="outline"
        disabled={!props.rt.isConnected()}
        onClick={() => props.rt.rest('scene')}
      >
        <Hourglass aria-hidden="true" class="size-4" />
        Descanso de cena
      </Button>

      <Button
        size="sm"
        variant="outline"
        disabled={!props.rt.isConnected()}
        onClick={() => setOpen(true)}
      >
        <Moon aria-hidden="true" class="size-4" />
        Descanso de dia
      </Button>

      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class="max-w-sm">
          <DialogHeader>
            <DialogTitle>Descanso de dia</DialogTitle>
            <DialogDescription>
              A qualidade do descanso decide quanto o grupo recupera de PV e PM (p105).
            </DialogDescription>
          </DialogHeader>
          <Select
            aria-label="Qualidade do descanso"
            options={REST_OPTIONS}
            value={REST_OPTIONS.find((option) => option.value === condition()) ?? null}
            onChange={(option) => setCondition(option?.value ?? 'normal')}
          />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="button" disabled={!props.rt.isConnected()} onClick={restDay}>
              Descansar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
