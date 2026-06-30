/**
 * Perigos Ambientais — PDF Cap 7 (Aventura, p317-319) + Cap 6 (Mestre,
 * Clima p267, Terrenos p268-269).
 *
 * T20 differences from D&D-style hazards:
 *  - **Afogamento NÃO é categoria separada** — fica sob SUFOCAMENTO
 *    (segurar respiração + Fortitude).
 *  - **Queimadura NÃO é categoria separada** — fica sob FOGO ("em
 *    chamas" condition).
 *  - **Queda usa 1d6 por 1,5m** (max 40d6 a 60m), NÃO 1d6 por 3m.
 *  - **Tormenta** é perigo ambiental único de T20 com progressão de
 *    insanidade (frustrada → esmorecida → confusa → insana-NPC).
 */

export type HazardCategory =
  | 'sufocamento'
  | 'fome-sede'
  | 'temperatura'
  | 'queda'
  | 'fogo'
  | 'meio-ambiente'

export type HazardSaveType = 'fortitude' | 'reflexos' | 'vontade' | 'none'

export type EnvironmentalHazard = {
  id: string
  name: string
  category: HazardCategory
  triggerCondition: string
  damagePerInterval: string
  saveType: HazardSaveType
  /** null quando não há save, ou quando o CD escala dinamicamente. */
  saveCd: number | null
  saveCdProgression: string
  recoveryCondition: string
  effect: string
  bookPage: number
}

// ─── Constantes Quantitativas ────────────────────────────────────────
/** PDF p319: 1d6 por 1,5m. */
export const QUEDA_DAMAGE_PER_INTERVAL_METERS = 1.5
/** PDF p319: dano máximo é 40d6 (queda de 60m). */
export const QUEDA_MAX_DAMAGE_DICE = 40
/** PDF p319: cair em água reduz dano em 4d6 (equivalente a -6m). */
export const QUEDA_WATER_REDUCTION_DICE = 4

/** PDF p319: prender respiração = (1 + Constituição) rodadas. */
export const BREATH_HOLD_ROUNDS_BASE = 1
/** PDF p319: CD inicial Fortitude para Sufocamento, Fome/Sede, Sono. */
export const SUFOCAMENTO_BASE_CD = 15
/** PDF p319: CD Fumaça começa em 10. */
export const FUMACA_BASE_CD = 10
/** PDF p319: Tormenta CD Vontade inicial = 25. */
export const TORMENTA_BASE_CD = 25

/**
 * Calcula dano de queda em dados (d6) para uma queda de `meters`
 * metros. Aplica cap em 40d6 (PDF p319) e redução opcional se cair em
 * água (-4d6).
 */
export function fallDamageDice(
  meters: number,
  intoWater: boolean = false,
): number {
  if (meters <= 0) return 0
  const raw = Math.floor(meters / QUEDA_DAMAGE_PER_INTERVAL_METERS)
  const capped = Math.min(raw, QUEDA_MAX_DAMAGE_DICE)
  const reduced = intoWater
    ? Math.max(0, capped - QUEDA_WATER_REDUCTION_DICE)
    : capped
  return reduced
}

/**
 * Rodadas que o personagem consegue prender a respiração antes de
 * iniciar testes de Fortitude. PDF p319: `1 + Constituição`.
 */
export function breathHoldRounds(constituicaoMod: number): number {
  return BREATH_HOLD_ROUNDS_BASE + constituicaoMod
}

/**
 * CD escalonado pela rodada/dia (PDF padrão: CD-base +1 por teste
 * anterior). Útil para Sufocamento, Fome/Sede, Sono, Calor/Frio.
 */
export function escalatingFortitudeCd(
  baseCd: number,
  previousTests: number,
): number {
  return baseCd + previousTests
}

export const ENVIRONMENTAL_HAZARDS: readonly EnvironmentalHazard[] =
  Object.freeze([
    {
      id: 'acido',
      name: 'Ácido',
      category: 'meio-ambiente',
      triggerCondition: 'exposto a ácido corrosivo (respingo, imersão)',
      damagePerInterval:
        '1d6 por rodada de exposição; 10d6 por rodada se imersão total (persiste 1 rodada após sair)',
      saveType: 'none',
      saveCd: null,
      saveCdProgression: 'sem teste de resistência',
      recoveryCondition:
        'sair do contato com o ácido (dano persiste 1 rodada após imersão)',
      effect:
        'Ácidos corrosivos causam 1d6 por rodada de exposição. Imersão total: 10d6 por rodada e dano persiste 1 rodada após sair. Ex.: 2 rodadas imerso + sair = 30d6.',
      bookPage: 317,
    },
    {
      id: 'areia-movedica',
      name: 'Areia Movediça',
      category: 'meio-ambiente',
      triggerCondition:
        'caminhar sobre quadrado de areia movediça (6m de lado) em pântano ou deserto',
      damagePerInterval:
        'sem dano direto; consequência é afundar e sufocar (vide Sufocamento)',
      saveType: 'reflexos',
      saveCd: 25,
      saveCdProgression:
        'Sobrevivência CD 25 para notar antes de entrar; Atletismo CD 25 (ação completa) para escapar; condição cumulativa por falhas',
      recoveryCondition:
        'Atletismo CD 25 ou ajuda de aliado externo (galho/vara/corda)',
      effect:
        'Sobrevivência CD 25 para notar antes de entrar. Entrar = agarrado. Submerge se passar uma rodada inteira agarrado (Sufocamento). Atletismo CD 25 (ação completa) para escapar. Falhar por 5+ piora condição de cansaço.',
      bookPage: 317,
    },
    {
      id: 'calor-frio-extremo',
      name: 'Calor e Frio',
      category: 'temperatura',
      triggerCondition:
        'clima muito quente (>50°C) ou muito frio (<-10°C); extremos: >60°C ou <-20°C',
      damagePerInterval:
        '1d6 por falha (fogo se calor, frio se frio); 1×/dia em quente/frio normal, 1×/minuto em extremo',
      saveType: 'fortitude',
      saveCd: SUFOCAMENTO_BASE_CD,
      saveCdProgression: 'CD 15 inicial, +1 por teste anterior',
      recoveryCondition: 'dano só cura após sair do clima',
      effect:
        'Em clima muito quente (>50°C) ou frio (<-10°C): Fortitude/dia (CD 15 +1 por teste anterior). Falhar = 1d6 fogo (calor) ou frio (frio). Em extremos (>60°C ou <-20°C) teste é feito por minuto. Dano só cura após sair do clima.',
      bookPage: 267,
    },
    {
      id: 'fogo',
      name: 'Fogo',
      category: 'fogo',
      triggerCondition:
        'exposto a fogo natural ou mágico contínuo (não a explosões instantâneas)',
      damagePerInterval: '1d6 de fogo no início de cada turno em chamas',
      saveType: 'reflexos',
      saveCd: 15,
      saveCdProgression: 'CD 15 fixa (não escalona)',
      recoveryCondition:
        'ação padrão para apagar com as mãos; imersão em água também apaga',
      effect:
        'Reflexos CD 15. Falhar = em chamas; sofre 1d6 fogo no início de cada turno. Ação padrão para apagar com as mãos, ou imergir em água. Efeitos instantâneos (Bola de Fogo, Explosão de Chamas) não duram para incendiar.',
      bookPage: 319,
    },
    {
      id: 'fome-sede',
      name: 'Fome e Sede',
      category: 'fome-sede',
      triggerCondition:
        'um dia sem comida ou água; depois disso teste diário',
      damagePerInterval:
        'sem dano de PV; condições progressivas (fatigado → exausto → inconsciente → morte na 4ª falha)',
      saveType: 'fortitude',
      saveCd: SUFOCAMENTO_BASE_CD,
      saveCdProgression: 'CD 15 +1 por teste anterior',
      recoveryCondition:
        'apenas comendo e bebendo; efeito tipo Metabolismo (constructos/mortos-vivos imunes)',
      effect:
        '1 dia sem comida/água sem teste. Depois, Fortitude/dia (CD 15 +1 por teste anterior). 1ª falha = fatigado. 2ª = exausto. 3ª = inconsciente. 4ª = letal. Recuperação apenas comendo/bebendo.',
      bookPage: 319,
    },
    {
      id: 'fumaca',
      name: 'Fumaça',
      category: 'sufocamento',
      triggerCondition: 'imerso em fumaça densa (casa em chamas)',
      damagePerInterval:
        '1d6 PV após duas falhas seguidas; também perde turno engasgando',
      saveType: 'fortitude',
      saveCd: FUMACA_BASE_CD,
      saveCdProgression: 'CD 10 +1 por teste anterior',
      recoveryCondition: 'sair da área de fumaça densa',
      effect:
        'Fortitude no início de cada turno (CD 10 +1 por teste anterior). Falhar = perde turno tossindo. Duas falhas seguidas = 1d6 dano. Fumaça densa fornece camuflagem leve. Tipo Metabolismo.',
      bookPage: 319,
    },
    {
      id: 'lava',
      name: 'Lava',
      category: 'fogo',
      triggerCondition:
        'contato com lava, magma ou material incandescente (ex: metal derretido)',
      damagePerInterval:
        '2d6 fogo por rodada de exposição; 20d6 se imersão total (persiste 1 rodada após sair)',
      saveType: 'none',
      saveCd: null,
      saveCdProgression: 'sem teste de resistência',
      recoveryCondition:
        'sair do contato (dano persiste 1 rodada após imersão)',
      effect:
        'Exposição direta: 2d6 fogo por rodada. Imersão total: 20d6 por rodada, persiste 1 rodada após sair. Ex.: 2 rodadas imerso + sair = 60d6.',
      bookPage: 319,
    },
    {
      id: 'queda',
      name: 'Queda',
      category: 'queda',
      triggerCondition: 'cair de altura',
      damagePerInterval:
        '1d6 de impacto por 1,5m de queda, até máximo de 40d6 (60m)',
      saveType: 'none',
      saveCd: null,
      saveCdProgression:
        'sem teste para a queda em si; cair em água reduz dano em 4d6 (-6m equivalentes)',
      recoveryCondition: 'cair em água reduz dano em 4d6',
      effect:
        '1d6 de impacto por 1,5m caídos, máximo 40d6 (60m). Cair em água reduz dano em 4d6. Objeto pesado caindo: 1d6 por 1,5m; objeto muito pesado dobra dano.',
      bookPage: 319,
    },
    {
      id: 'sufocamento',
      name: 'Sufocamento',
      category: 'sufocamento',
      triggerCondition:
        'submerso em água, soterrado, ou ambiente sem ar respirável',
      damagePerInterval: '1d6 PV por rodada após primeira falha',
      saveType: 'fortitude',
      saveCd: SUFOCAMENTO_BASE_CD,
      saveCdProgression: 'CD 15 +1 por teste anterior',
      recoveryCondition:
        'voltar a respirar ar fresco interrompe o ciclo; senão, morre',
      effect:
        'Pode prender respiração (1 + Constituição) rodadas. Ex.: Con 2 = 3 rodadas. Depois, Fortitude/rodada (CD 15 +1 por teste anterior). Falhar = inconsciente, perde 1d6 PV por rodada até respirar ou morrer. Tipo Metabolismo.',
      bookPage: 319,
    },
    {
      id: 'sono',
      name: 'Sono (Privação)',
      category: 'fome-sede',
      triggerCondition: 'mais de uma noite sem dormir',
      damagePerInterval:
        'sem dano de PV; condições progressivas (fatigado → exausto → inconsciente)',
      saveType: 'fortitude',
      saveCd: SUFOCAMENTO_BASE_CD,
      saveCdProgression: 'CD 15 +1 por teste anterior',
      recoveryCondition:
        'dormir pelo menos 8 horas; uma noite sem dormir tolerada mas sem recuperar PV/PM',
      effect:
        '1 noite sem dormir: não recupera PV/PM, mas sem condições. Depois, Fortitude/dia (CD 15 +1 por teste anterior). 1ª = fatigado. 2ª = exausto. 3ª = inconsciente até dormir 8h.',
      bookPage: 319,
    },
    {
      id: 'tormenta',
      name: 'Tormenta',
      category: 'meio-ambiente',
      triggerCondition: 'entrar em área de Tormenta',
      damagePerInterval:
        'sem dano de PV direto; progressão de insanidade (esmorecida → confusa → insana/NPC)',
      saveType: 'vontade',
      saveCd: TORMENTA_BASE_CD,
      saveCdProgression: 'CD 25 +2 por dia anterior consecutivo na área',
      recoveryCondition:
        'sair da área interrompe progressão diária; lefeu são imunes',
      effect:
        'Entrar = frustrada automaticamente. No início de cada dia, Vontade (CD 25 +2 por dia anterior). Falhar: já frustrada → esmorecida; esmorecida → confusa; confusa → insana permanente (PJ vira NPC). Custos PM +2 na área; itens mágicos perdem 1 encantamento; recuperação PV/PM por descanso reduzida à metade. Lefeu imunes.',
      bookPage: 319,
    },
    {
      id: 'altitude',
      name: 'Altitude',
      category: 'temperatura',
      triggerCondition:
        'falta de oxigênio em grandes altitudes (cume de montanha)',
      damagePerInterval: 'sem dano direto; fatigado/exausto progressivo',
      saveType: 'fortitude',
      saveCd: SUFOCAMENTO_BASE_CD,
      saveCdProgression: 'CD 15 +1 por teste anterior',
      recoveryCondition:
        'descer da altitude; já fatigado vira exausto se falhar novamente',
      effect:
        'No cume de uma montanha: Fortitude (CD 15 +1 por teste anterior). Falhar = fatigado até descer. Já fatigado = exausto.',
      bookPage: 268,
    },
    {
      id: 'rio-congelado',
      name: 'Gelo e Rio Congelado',
      category: 'temperatura',
      triggerCondition: 'andar sobre gelo ou rio congelado',
      damagePerInterval:
        '1d6 frio por rodada submerso após gelo quebrar; 1d6 impacto em queda por falha de Acrobacia',
      saveType: 'reflexos',
      saveCd: 15,
      saveCdProgression:
        'Acrobacia CD 15 (ou igual ao dano sofrido) ao correr/investir/sofrer dano; rolar 1 no d4 ao cair quebra o gelo',
      recoveryCondition:
        'sair de baixo do buraco (ação de movimento); abrir buraco no gelo requer 10 pts impacto ou fogo',
      effect:
        'Andar normal: sem teste. Correr/investir/sofrer dano em gelo: Acrobacia (CD 15 ou igual ao dano sofrido). Falhar = cai e desliza 1d4 × 1,5m. Em rio congelado: rolar 1 no d4 = gelo quebra, sofre 1d6 frio por rodada submerso.',
      bookPage: 269,
    },
    {
      id: 'escuridao',
      name: 'Escuridão',
      category: 'meio-ambiente',
      triggerCondition:
        'penumbra (leve) ou breu/subterrâneos longe da entrada (total)',
      damagePerInterval: 'sem dano direto; fornece camuflagem',
      saveType: 'none',
      saveCd: null,
      saveCdProgression: 'sem teste',
      recoveryCondition:
        'fonte de luz; visão na penumbra (leve) ou visão no escuro (total) negam',
      effect:
        'Escuridão Leve: noite enluarada, postes de cidade. Fornece camuflagem leve. Escuridão Total: breu sem estrelas/luar, câmara fechada. Fornece camuflagem total.',
      bookPage: 318,
    },
  ])

const byId = new Map(ENVIRONMENTAL_HAZARDS.map((h) => [h.id, h]))

export function hazardById(id: string): EnvironmentalHazard | undefined {
  return byId.get(id)
}

export function hazardsByCategory(
  category: HazardCategory,
): readonly EnvironmentalHazard[] {
  return ENVIRONMENTAL_HAZARDS.filter((h) => h.category === category)
}
