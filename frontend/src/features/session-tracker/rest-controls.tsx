import { createSignal } from 'solid-js'
import type { RestCondition, SessionRealtime } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { Select } from '@/shared/ui/select'

const REST_OPTIONS: { value: RestCondition; label: string }[] = [
  { value: 'ruim', label: 'Ruim (½ nível)' },
  { value: 'normal', label: 'Normal (nível)' },
  { value: 'confortavel', label: 'Confortável (2×)' },
  { value: 'luxuosa', label: 'Luxuosa (3×)' },
]

/**
 * Descanso de cena e de dia, com a qualidade que a p105 usa para calcular a
 * recuperação.
 *
 * Vivia no cabeçalho do rastreador, onde ocupava DUAS linhas na frente do que
 * muda a cada turno — e é ação de uma vez por sessão. Agora mora no menu da
 * sessão, junto do resto do que se faz raramente (ALE-122).
 */
export function RestControls(props: { rt: SessionRealtime }) {
  const [condition, setCondition] = createSignal<RestCondition>('normal')

  return (
    <section class="space-y-2 rounded-sm border border-grimorio-iron p-3">
      <h3 class="font-heading text-sm uppercase tracking-wide text-grimorio-gold">Descanso</h3>
      <div class="flex flex-wrap items-center gap-2">
        <Button
          size="sm"
          variant="outline"
          disabled={!props.rt.isConnected()}
          onClick={() => props.rt.rest('scene')}
        >
          Descanso de cena
        </Button>
        <Select
          aria-label="Qualidade do descanso"
          size="sm"
          class="w-[150px]"
          options={REST_OPTIONS}
          value={REST_OPTIONS.find((option) => option.value === condition()) ?? null}
          onChange={(option) => setCondition(option?.value ?? 'normal')}
        />
        <Button
          size="sm"
          variant="outline"
          disabled={!props.rt.isConnected()}
          onClick={() => props.rt.rest('day', condition())}
        >
          Descanso de dia
        </Button>
      </div>
    </section>
  )
}
