import { createStore, produce } from 'solid-js/store'
import type { StancePayment } from '@/shared/api/api'

/**
 * O que o jogador PAGOU para entrar numa postura, por personagem. `steps` são
 * os passos do seletor de PM extra (Fúria p40: +1 PM por +1 de bônus a cada 5
 * níveis).
 *
 * O DONO É O SERVIDOR (ALE-222). Isto vivia em `localStorage`, e o comentário
 * daqui dizia *"local-only like conditionals-store… this only remembers what
 * was paid, so ending a stance never refunds"*. A decisão do dono em
 * 2026-08-22 foi "o servidor mantém estado, ponto final"; a razão de EXISTIR
 * continua a mesma — sair da postura não pode devolver PM —, só o dono mudou.
 *
 * Ele guarda o PREÇO, não se a postura está ligada: quem diz isso é o
 * situacional de mesmo nome, no [[conditionals-store]].
 */

export type StanceActivationRecord = { steps: number; pmPaid: number }
type RecordsByCharacter = Record<string, Record<string, StanceActivationRecord>>

/** Quem leva o pagamento ao servidor. Injetado para o teste espiar. */
export type StanceWriter = {
  set: (characterId: number, flag: string, record: StanceActivationRecord) => Promise<unknown>
  clear: (characterId: number, flag: string) => Promise<unknown>
}

export type StanceActivationStore = {
  /** O que foi pago por uma postura ativa, ou undefined se nunca foi. */
  paidFor: (characterId: number, flag: string) => StanceActivationRecord | undefined
  logActivation: (characterId: number, flag: string, record: StanceActivationRecord) => void
  clearActivation: (characterId: number, flag: string) => void
  hydrate: (characterId: number, stances: readonly StancePayment[]) => void
}

export function createStanceActivationStore(
  write: StanceWriter,
  onError: (erro: unknown) => void = () => {},
): StanceActivationStore {
  const [records, setRecords] = createStore<RecordsByCharacter>({})

  /** Escreve local, manda, e volta ao instantâneo se o servidor recusar. */
  const edit = (
    characterId: number,
    flag: string,
    mutate: (draft: RecordsByCharacter) => void,
    enviar: () => Promise<unknown>,
  ) => {
    const chave = String(characterId)
    const antes = records[chave]?.[flag]
    setRecords(produce(mutate))
    void enviar().catch((erro) => {
      setRecords(
        produce((draft) => {
          const doPersonagem = draft[chave] ?? {}
          if (antes) doPersonagem[flag] = antes
          else delete doPersonagem[flag]
          draft[chave] = doPersonagem
        }),
      )
      onError(erro)
    })
  }

  return {
    paidFor: (characterId, flag) => records[String(characterId)]?.[flag],

    hydrate: (characterId, stances) =>
      setRecords(
        produce((draft) => {
          const porFlag: Record<string, StanceActivationRecord> = {}
          for (const s of stances) porFlag[s.flag] = { steps: s.steps, pmPaid: s.pmPaid }
          draft[String(characterId)] = porFlag
        }),
      ),

    logActivation: (characterId, flag, record) =>
      edit(
        characterId,
        flag,
        (draft) => {
          const chave = String(characterId)
          draft[chave] = { ...(draft[chave] ?? {}), [flag]: record }
        },
        () => write.set(characterId, flag, record),
      ),

    clearActivation: (characterId, flag) =>
      edit(
        characterId,
        flag,
        (draft) => {
          const doPersonagem = draft[String(characterId)]
          if (doPersonagem) delete doPersonagem[flag]
        },
        () => write.clear(characterId, flag),
      ),
  }
}
