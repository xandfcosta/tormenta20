import type { QueryClient } from '@tanstack/solid-query'
import { api } from '@/shared/api/api'
import { createCharacterWrite } from './character-write'

/** Grava o novo saldo em T$. Rejeita como qualquer escrita da ficha. */
export type SetTibar = (tibar: number) => Promise<void>

/**
 * A escrita do dinheiro, otimista como as outras da ficha. É um SALDO e não um
 * delta: o servidor recusa (negativo, absurdo) em vez de aparar, então pintar o
 * valor pedido nunca mostra um número que o servidor mudaria calado — a recusa
 * volta pelo rollback do `createCharacterWrite`.
 *
 * A pintura importa mais aqui do que num campo qualquer: o tibar é carga
 * (p141), então a barra da mochila se move junto com o número no mesmo quadro.
 *
 * @example const setTibar = tibarAction(queryClient, character.id)
 */
export function tibarAction(queryClient: QueryClient, characterId: number): SetTibar {
  const write = createCharacterWrite(queryClient, characterId)
  return (tibar) =>
    write(
      (previous) => ({ ...previous, tibar }),
      async () => {
        await api.characters.updateTibar(characterId, { tibar })
      },
    )
}
