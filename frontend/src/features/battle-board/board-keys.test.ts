import { describe, expect, it } from 'vitest'
import { boardKeyAction } from './board-keys'

/**
 * A tabela de teclas do tabuleiro (ALE-194).
 *
 * O que se prova aqui é a REGRA que a integração não alcança: quais teclas são
 * NOSSAS. Um atalho que rouba `Ctrl+−` tira do usuário o zoom do navegador para
 * dar um zoom que ele não pediu, e isso não aparece no teste da cena — lá o
 * evento chega sem modificador.
 */
describe('as teclas do tabuleiro', () => {
  const tecla = (key: string, extras: Partial<KeyboardEvent> = {}) =>
    boardKeyAction({ key, ctrlKey: false, metaKey: false, altKey: false, ...extras } as KeyboardEvent)

  it('a seta vira um passo de UM quadrado, que é a unidade do estado', () => {
    expect(tecla('ArrowRight')).toEqual({ kind: 'step', dx: 1, dy: 0 })
    expect(tecla('ArrowUp')).toEqual({ kind: 'step', dx: 0, dy: -1 })
  })

  // Com modificador a tecla é do BROWSER: Ctrl+Home vai ao topo do documento e
  // Ctrl+− diminui a página.
  it('com modificador, a tecla não é nossa', () => {
    expect(tecla('ArrowRight', { ctrlKey: true })).toBeNull()
    expect(tecla('Home', { metaKey: true })).toBeNull()
    // O Alt+seta do Roll20 move um PIXEL, e aqui isso não existe: o estado é em
    // quadrados inteiros (T20 p236), e meio quadrado não é uma posição.
    expect(tecla('ArrowUp', { altKey: true })).toBeNull()
  })

  // No ABNT2 e no US o `+` exige Shift: aceitar só a tecla exata seria exigir
  // duas mãos para aproximar.
  it('o `=` aproxima como o `+`', () => {
    expect(tecla('=')).toEqual(tecla('+'))
    expect(tecla('-')).toEqual({ kind: 'zoom', deltaPx: -8 })
  })

  it('o que não é nosso passa adiante', () => {
    expect(tecla('a')).toBeNull()
    expect(tecla('Enter')).toBeNull()
  })
})
