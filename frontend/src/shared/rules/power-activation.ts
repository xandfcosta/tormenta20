import type { ActivationScaling } from '@/shared/api/catalog-types'

/**
 * A regra de ESCALA de uma ativação — quantos degraus o poder já alcançou no
 * nível dado. Único pedaço do `power-activation.ts` do `t20-data` que sobreviveu
 * ao desligamento: o resto era o CATÁLOGO de ativações, que hoje vem por HTTP
 * (ALE-109).
 */
/**
 * Passos extras que o nível NA CLASSE concede (p40: multiclasse conta só os
 * níveis de Bárbaro para a Fúria).
 *
 * @example maxStepsForLevel({ firstStepLevel: 5, stepEveryLevels: 5, ... }, 10) // 2
 */
export function maxStepsForLevel(
  scaling: ActivationScaling,
  classLevel: number,
): number {
  if (classLevel < scaling.firstStepLevel) return 0
  return 1 + Math.floor((classLevel - scaling.firstStepLevel) / scaling.stepEveryLevels)
}
