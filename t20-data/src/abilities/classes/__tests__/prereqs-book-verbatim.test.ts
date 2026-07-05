import { describe, expect, it } from 'vitest'
import { CLASS_POWERS_CATALOG } from '../index'

/**
 * Prerequisitos verbatim do livro para poderes onde audit encontrou
 * ou pode encontrar bugs. Cada spec cita PDF page para navegação.
 *
 * Regression: audit 2026-07-04 flagou Bárbaro Destruidor sem attr('strength',1)
 * embora o livro (p41) exija For 1. Guerreiro Destruidor tinha o prereq;
 * apenas Bárbaro faltava.
 */

function findPower(className: string, name: string) {
  return CLASS_POWERS_CATALOG.find(
    (p) => p.className === className && p.name === name,
  )
}

describe('Bárbaro Destruidor (p41) exige For 1', () => {
  const power = findPower('Bárbaro', 'Destruidor')

  it('poder existe no catálogo', () => {
    expect(power).toBeDefined()
  })

  it('prerequisites inclui attribute strength ≥ 1', () => {
    const prereqs = power?.prerequisites ?? []
    const attrPrereq = prereqs.find(
      (p) => p.kind === 'attribute' && p.attr === 'strength',
    )
    expect(attrPrereq).toBeDefined()
    if (attrPrereq?.kind === 'attribute') {
      expect(attrPrereq.min).toBe(1)
    }
  })
})

describe('Guerreiro Destruidor (p65) exige For 1 — verifica que a fix Bárbaro não regride Guerreiro', () => {
  const power = findPower('Guerreiro', 'Destruidor')

  it('poder existe', () => {
    expect(power).toBeDefined()
  })

  it('prerequisites inclui attribute strength ≥ 1', () => {
    const prereqs = power?.prerequisites ?? []
    const attrPrereq = prereqs.find(
      (p) => p.kind === 'attribute' && p.attr === 'strength',
    )
    expect(attrPrereq).toBeDefined()
  })
})
