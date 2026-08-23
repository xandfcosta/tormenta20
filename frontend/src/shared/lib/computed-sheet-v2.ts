/**
 * A ficha derivada que o motor Go devolve pela fronteira WASM.
 *
 * Estes tipos eram declarados AQUI, à mão, com o pedido de "keep the two in
 * lockstep" no cabeçalho — sincronia manual entre uma struct Go e um tipo TS que
 * nada verificava: uma divergência não dá erro de compilação, dá número errado
 * na tela. Agora eles são GERADOS das structs (`engine-go/cmd/tsgen`) e um teste
 * no Go falha se alguém mudar a struct sem regenerar (ALE-108).
 *
 * O módulo continua existindo como a porta que os consumidores já conhecem — o
 * caminho de import não mudou junto com a autoria.
 */
export type {
  AttributeBreakdown,
  BreakdownContribution,
  ComputedSheetV2,
  DefenseBreakdown,
  ExpertiseBreakdown,
  LoadBreakdown,
  RdBreakdown,
  SourceAmount,
  TempHpBreakdown,
  TotalContribs,
  ValueBreakdown,
  WeaponCard,
} from '@/shared/api/engine-types'
