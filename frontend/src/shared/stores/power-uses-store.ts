import { createStore, produce } from 'solid-js/store'
import type { PowerUse } from '@/shared/api/api'

/**
 * Quantas vezes cada poder limitado ("1/cena", "1/dia") já foi USADO, por
 * personagem.
 *
 * O DONO É O SERVIDOR (ALE-222). Isto vivia em `localStorage`, e o comentário
 * que estava aqui dizia o contrário: *"Local-only like conditionals-store: the
 * book limit is table-trust, not server state"*. A decisão do dono em
 * 2026-08-22 foi "o servidor mantém estado, ponto final", e a frase antiga fica
 * registrada em vez de apagada — o histórico não deve mentir sobre o que já se
 * pensou.
 *
 * Leitura SÍNCRONA e escrita OTIMISTA, pelo mesmo motivo do situacional: `used`
 * é lido de dentro de memos, e uma Promise ali tornaria a ficha inteira
 * assíncrona por causa de um contador.
 */

export type PowerUseScope = 'scene' | 'day'
export type PowerUseCounts = { scene: Record<string, number>; day: Record<string, number> }
type UsesByCharacter = Record<string, PowerUseCounts>

/** Quem avisa o servidor que gastou mais um. Injetado para o teste espiar. */
export type PowerUseWriter = (
  characterId: number,
  powerId: string,
  scope: PowerUseScope,
) => Promise<unknown>

export type PowerUsesStore = {
  used: (characterId: number, powerId: string) => { scene: number; day: number }
  bump: (characterId: number, powerId: string, scope: PowerUseScope) => void
  /**
   * Zera o que a CENA leva — só o CACHE. Quem chama já esperou o `/end-scene`
   * do servidor, que limpou lá; repetir a chamada aqui seria a mesma decisão
   * escrita em dois lugares, e um dia elas discordariam.
   */
  resetScene: (characterId: number) => void
  /** Idem para o dia — o `/end-day` já limpou os dois escopos no servidor. */
  resetDay: (characterId: number) => void
  hydrate: (characterId: number, uses: readonly PowerUse[]) => void
}

const SEM_USOS: PowerUseCounts = { scene: {}, day: {} }

export function createPowerUsesStore(
  write: PowerUseWriter,
  onError: (erro: unknown) => void = () => {},
): PowerUsesStore {
  const [uses, setUses] = createStore<UsesByCharacter>({})

  const contas = (characterId: number): PowerUseCounts => uses[String(characterId)] ?? SEM_USOS

  return {
    used: (characterId, powerId) => {
      const c = contas(characterId)
      return { scene: c.scene[powerId] ?? 0, day: c.day[powerId] ?? 0 }
    },

    hydrate: (characterId, lista) =>
      setUses(
        produce((draft) => {
          const contagem: PowerUseCounts = { scene: {}, day: {} }
          for (const u of lista) contagem[u.scope][u.powerId] = u.used
          draft[String(characterId)] = contagem
        }),
      ),

    // Soma UM, e é isso que vai no fio — nunca o total. Dois cliques rápidos
    // mandando "agora são 3" gravariam 3 duas vezes e perderiam um uso; quem
    // incrementa é o `ON CONFLICT` do servidor.
    bump: (characterId, powerId, scope) => {
      const chave = String(characterId)
      const antes = contas(characterId)[scope][powerId] ?? 0
      setUses(
        produce((draft) => {
          const atual = draft[chave] ?? { scene: {}, day: {} }
          atual[scope][powerId] = (atual[scope][powerId] ?? 0) + 1
          draft[chave] = atual
        }),
      )
      void write(characterId, powerId, scope).catch((erro) => {
        setUses(
          produce((draft) => {
            const atual = draft[chave] ?? { scene: {}, day: {} }
            atual[scope][powerId] = antes
            draft[chave] = atual
          }),
        )
        onError(erro)
      })
    },

    resetScene: (characterId) =>
      setUses(
        produce((draft) => {
          const atual = draft[String(characterId)]
          if (atual) atual.scene = {}
        }),
      ),

    resetDay: (characterId) =>
      setUses(
        produce((draft) => {
          draft[String(characterId)] = { scene: {}, day: {} }
        }),
      ),
  }
}
