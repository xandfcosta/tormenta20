import { describe, expect, it } from 'vitest'
import { type NotesView, cabeLadoALado, efetivo, readNotesView } from './notes-view'

/**
 * As duas regras do modo das notas (ALE-139): o que sobrevive ao recarregar, e
 * o que a largura consegue mostrar.
 */
describe('readNotesView', () => {
  it('devolve o modo guardado', () => {
    expect(readNotesView('ler')).toBe('ler')
    expect(readNotesView('escrever')).toBe('escrever')
  })

  /** Um valor estranho não pode deixar a aba sem arranjo nenhum. */
  it('cai no lado a lado quando o guardado não presta', () => {
    expect(readNotesView(null)).toBe('duplo')
    expect(readNotesView('paisagem')).toBe('duplo')
    expect(readNotesView('')).toBe('duplo')
  })

  /** O padrão é o arranjo de hoje: ninguém perde o que já usava. */
  it('sem nada guardado, é o lado a lado de sempre', () => {
    expect(readNotesView(null)).toBe('duplo')
  })
})

describe('efetivo — o que a largura deixa mostrar', () => {
  it('lado a lado num contêiner estreito vira escrever', () => {
    expect(efetivo('duplo', false)).toBe('escrever')
  })

  /**
   * Vira ESCREVER e não "ler": quem estava no modo duplo estava escrevendo e
   * lendo ao mesmo tempo, e tirar o editor de quem digita é pior que tirar a
   * prévia.
   */
  it('não vira ler', () => {
    expect(efetivo('duplo', false)).not.toBe('ler')
  })

  it('uma escolha explícita não é contrariada pela largura', () => {
    for (const modo of ['escrever', 'ler'] as NotesView[]) {
      expect(efetivo(modo, false)).toBe(modo)
      expect(efetivo(modo, true)).toBe(modo)
    }
  })

  it('cabendo, o lado a lado é o lado a lado', () => {
    expect(efetivo('duplo', true)).toBe('duplo')
  })
})

describe('cabeLadoALado', () => {
  it('não cabe numa região estreita', () => {
    expect(cabeLadoALado(400)).toBe(false)
  })

  it('cabe a partir do ponto onde o arranjo já vivia', () => {
    expect(cabeLadoALado(672)).toBe(true)
    expect(cabeLadoALado(1200)).toBe(true)
  })

  /**
   * Antes da primeira medição a resposta tem de ser o arranjo de sempre: dizer
   * "não cabe" faria a aba piscar do editor sozinho para o lado a lado no
   * primeiro quadro depois da medição.
   */
  it('enquanto não mediu, responde o arranjo de sempre', () => {
    expect(cabeLadoALado(0)).toBe(true)
  })
})
