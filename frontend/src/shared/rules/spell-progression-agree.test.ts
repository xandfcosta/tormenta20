import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { SPELL_PROGRESSION } from './class-spellcasting'

/**
 * A tabela de progressão de círculo diz a MESMA coisa nos dois lados
 * (ALE-272, fatia 6).
 *
 * Ela vivia só aqui, e por isso o limite de círculo dos aprimoramentos era só de
 * interface: o servidor não tinha como perguntar em que nível cada classe
 * destrava cada círculo, então o `validateAugments` aceitava qualquer
 * `requiresCircle` — 126 dos 486 aprimoramentos do catálogo. A tabela foi para o
 * `classes.json`, e agora as duas pontas leem a mesma coisa.
 *
 * Ela foi **movida, não retranscrita**: nenhum número foi lido do livro na
 * mudança. É exatamente isso que este teste garante — que a cópia é fiel. Uma
 * auditoria contra o livro é outro trabalho, e este teste não a substitui nem
 * finge substituir.
 *
 * Ele MORRE COM A SPA: quando `frontend/` sair, o catálogo fica sendo a única
 * cópia e não há o que comparar.
 */
describe('a progressão de círculo por classe', () => {
  const classes: { name: string; spellcasting?: Record<string, unknown> }[] = JSON.parse(
    readFileSync(
      resolve(__dirname, '../../../../engine-go/catalog/data/classes.json'),
      'utf8',
    ),
  )

  it('tem no catálogo as mesmas cinco classes conjuradoras', () => {
    const doCatalogo = classes.filter((c) => c.spellcasting).map((c) => c.name).sort()
    expect(doCatalogo).toEqual(Object.keys(SPELL_PROGRESSION).sort())
  })

  it('destrava cada círculo no mesmo nível dos dois lados', () => {
    let conferidas = 0
    for (const [nome, doTs] of Object.entries(SPELL_PROGRESSION)) {
      const noCatalogo = classes.find((c) => c.name === nome)?.spellcasting
      expect(noCatalogo, `a classe ${nome} não tem \`spellcasting\` no catálogo`).toBeDefined()

      // As CHAVES do JSON são strings ("2"), e as do TS são números — comparar
      // os objetos crus daria falso negativo em toda classe.
      const unlockDoTs = Object.fromEntries(
        Object.entries(doTs.unlockLevel).map(([circulo, nivel]) => [String(circulo), nivel]),
      )
      expect(noCatalogo?.unlockLevel, `unlockLevel de ${nome}`).toEqual(unlockDoTs)
      expect(noCatalogo?.maxCircle, `maxCircle de ${nome}`).toBe(doTs.maxCircle)
      expect(noCatalogo?.list, `list de ${nome}`).toBe(doTs.list)
      expect(noCatalogo?.attribute, `attribute de ${nome}`).toBe(doTs.attribute)
      expect(noCatalogo?.schoolRestriction, `schoolRestriction de ${nome}`).toBe(
        doTs.schoolRestriction,
      )
      expect(noCatalogo?.startingSpellsKnown, `startingSpellsKnown de ${nome}`).toBe(
        doTs.startingSpellsKnown,
      )
      expect(noCatalogo?.learnCadence, `learnCadence de ${nome}`).toBe(doTs.learnCadence)
      conferidas++
    }
    // CONTROLE: sem ele, um `SPELL_PROGRESSION` que virasse vazio faria o laço
    // não rodar nenhuma vez e o teste passar afirmando nada.
    expect(conferidas, 'o laço não conferiu classe nenhuma').toBe(5)
  })
})
