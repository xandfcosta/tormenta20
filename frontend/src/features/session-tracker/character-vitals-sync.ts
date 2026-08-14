import { useQueryClient } from '@tanstack/solid-query'
import { createEffect } from 'solid-js'
import type { SessionRealtime } from '@/shared/realtime/realtime'

/**
 * Faz a FICHA acompanhar o rastreador (ALE-122).
 *
 * Desde esta fatia o servidor grava os vitais do combate na linha do personagem,
 * então os dois números finalmente concordam no banco — mas a tela ainda não:
 * o card "Grupo" e a ficha aberta leem por query, e nada as invalida quando o
 * valor muda pelo socket. Sem isto, o mestre bate -5 e continua vendo o número
 * antigo a 300px de distância, que é a queixa original.
 *
 * Invalida SÓ quem mudou: comparar o snapshot anterior evita refazer as fichas
 * da mesa inteira a cada passagem de turno.
 *
 * @example createCharacterVitalsSync(rt, campaignId)
 */
export function createCharacterVitalsSync(rt: SessionRealtime, campaignId: () => number): void {
  const queryClient = useQueryClient()
  let previous = new Map<number, string>()

  createEffect(() => {
    let changed = false
    const current = new Map<number, string>()
    for (const entry of rt.state().initiative) {
      if (entry.characterId == null) continue
      current.set(entry.characterId, `${entry.hpCurrent ?? '-'}/${entry.mpCurrent ?? '-'}`)
    }
    for (const [characterId, vitals] of current) {
      const before = previous.get(characterId)
      // `before === undefined` é a primeira leitura da entrada, não uma mudança:
      // invalidar ali refetch tudo ao entrar na sessão, sem motivo.
      if (before !== undefined && before !== vitals) {
        void queryClient.invalidateQueries({ queryKey: ['characters', characterId] })
        // O card "Grupo" NÃO lê a ficha: os PV dele vêm do payload de MEMBROS
        // da campanha. Invalidar só `characters` deixava 52 na iniciativa e 57
        // no card, que é exatamente a queixa — descoberto medindo, não lendo.
        changed = true
      }
    }
    if (changed) {
      void queryClient.invalidateQueries({ queryKey: ['campaigns', campaignId(), 'members'] })
    }
    previous = current
  })
}
