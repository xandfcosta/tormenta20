import {
  type ConditionalsStore,
  createConditionalsStore,
} from '@/shared/stores/conditionals-store'
import { type PowerUsesStore, createPowerUsesStore } from '@/shared/stores/power-uses-store'
import {
  type StanceActivationStore,
  createStanceActivationStore,
} from '@/shared/stores/stance-activation-store'

/**
 * Os três stores de estado de jogo com o servidor MUDO, para teste (ALE-222).
 *
 * Substituem o `new FakeStorage()` que os testes passavam quando esses stores
 * viviam em `localStorage`. O que se falseia mudou de LUGAR de armazenamento
 * para DESTINO da escrita, e a diferença importa: o cache local continua real,
 * então o teste ainda exercita a pintura otimista de verdade — só o que iria
 * pelo fio é que não vai.
 *
 * Quem quiser espiar o que foi enviado passa o próprio writer em vez de usar
 * estes.
 *
 * @example <ConditionalsProvider store={fakeConditionals()}>
 */
export function fakeConditionals(): ConditionalsStore {
  return createConditionalsStore(async () => {})
}

export function fakePowerUses(): PowerUsesStore {
  return createPowerUsesStore(async () => {})
}

export function fakeStances(): StanceActivationStore {
  return createStanceActivationStore({ set: async () => {}, clear: async () => {} })
}
