/**
 * Encounter difficulty math — Cap 7 p282, the rule for the effective ND of a
 * GROUP of monsters:
 *
 *   ND < 1 → groupNd = monsterNd × quantidade
 *            (quatro de ND 1/4 = ND 1; dois de ND 1/2 = ND 1)
 *   ND ≥ 1 → groupNd = monsterNd + 2 a cada dobra da quantidade
 *            (dois de ND 1 = ND 3; quatro de ND 5 = ND 9)
 *
 * `Math.log2` estende a regra para dobras não-inteiras, então um grupo de 3
 * cai entre 1× e 2×. Quem quiser um ND inteiro arredonda o resultado.
 */
export function computeGroupNd(monsterNd: number, quantity: number): number {
  if (quantity <= 0) return 0
  if (monsterNd < 1) return monsterNd * quantity
  if (quantity === 1) return monsterNd
  return monsterNd + 2 * Math.log2(quantity)
}

/**
 * XP de tesouro derivado do ND (Cap 8 p326). Reimplementado aqui em vez de
 * importado do `t20-data` de propósito: lá a função mora no módulo `bestiary`,
 * e importá-la puxaria o array `BESTIARY` inteiro de volta pro bundle
 * (`Object.freeze` não tree-shaka) — o front não embarca dado de catálogo.
 */
export function xpForNd(nd: number): number {
  return Math.round(nd * 1000)
}
