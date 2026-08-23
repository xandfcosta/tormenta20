import { createEffect } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { useConditionals } from './conditionals-context'
import { usePowerUses } from './power-uses-context'
import { useStanceActivations } from './stance-activation-context'

/**
 * Enche os três stores de estado de jogo com o que veio NA FICHA (ALE-222).
 *
 * Existe porque os stores são indexados por personagem e vivem acima do
 * roteador, enquanto a ficha chega por query: alguém precisa fazer a ponte, e
 * fazê-la em três `createEffect` espalhados pelos consumidores significaria três
 * lugares para esquecer.
 *
 * A hidratação é DESTRUTIVA de propósito — ela substitui o cache em vez de
 * mesclar. O servidor é o dono; mesclar deixaria um situacional que outra aba
 * desligou sobreviver aqui, e a ficha recomputaria contra um conjunto que não
 * existe em lugar nenhum.
 *
 * @example createPlayStateHydration(() => character())
 */
export function createPlayStateHydration(character: () => Character | undefined): void {
  const conditionals = useConditionals()
  const powerUses = usePowerUses()
  const stances = useStanceActivations()

  createEffect(() => {
    const ficha = character()
    if (!ficha) return
    // Campos ausentes viram vazio em vez de derrubar a ficha: uma resposta de
    // servidor antigo (ou um fixture de teste incompleto) não deve tirar a tela
    // do ar por causa de um contador.
    conditionals.hydrate(ficha.id, ficha.conditionals ?? [])
    powerUses.hydrate(ficha.id, ficha.powerUses ?? [])
    stances.hydrate(ficha.id, ficha.stances ?? [])
  })
}
