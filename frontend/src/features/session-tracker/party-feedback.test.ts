import { createRoot, createSignal } from 'solid-js'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import { createPartyFeedback } from './party-feedback'

const entrada = (id: string) => ({ id, label: id }) as unknown as InitiativeEntry

/**
 * `createEffect` do Solid roda DEPOIS do corpo, então o teste tem de ceder um
 * microtique entre mexer no sinal e conferir o que foi dito — medir na mesma
 * linha mede o estado anterior.
 */
async function naRaiz(corpo: () => Promise<void>): Promise<void> {
  let descartar = () => {}
  // O corpo é chamado DENTRO da raiz: até o primeiro `await` ele roda síncrono,
  // que é quando o `createPartyFeedback` registra o efeito e o `onCleanup`.
  // Chamá-lo de fora deixaria os dois sem dono.
  const feito = createRoot((dispose) => {
    descartar = dispose
    return corpo()
  })
  await feito
  descartar()
}

/**
 * O que "Adicionar grupo" trouxe (ALE-135). A regra é: o resultado vem do
 * BROADCAST seguinte, não de uma resposta — `populateParty` é um `send` sem
 * ack —, e quando nada entra pode não haver broadcast nenhum, que é o caso
 * "já está tudo aqui".
 *
 * O botão continuar clicável é de propósito (o servidor é idempotente); o que
 * faltava era a tela dizer o que aconteceu.
 */
describe('createPartyFeedback', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('conta quantos entraram quando o estado chega', async () => {
    await naRaiz(async () => {
      const [lista, setLista] = createSignal([entrada('a')])
      const ditos: string[] = []
      const anunciar = createPartyFeedback(lista, (m) => ditos.push(m))

      await Promise.resolve()
      anunciar()
      setLista([entrada('a'), entrada('b'), entrada('c')])

      await Promise.resolve()
      expect(ditos).toEqual(['2 personagens entraram na iniciativa'])
    })
  })

  it('fala no singular quando entrou um só', async () => {
    await naRaiz(async () => {
      const [lista, setLista] = createSignal([entrada('a')])
      const ditos: string[] = []
      const anunciar = createPartyFeedback(lista, (m) => ditos.push(m))

      await Promise.resolve()
      anunciar()
      setLista([entrada('a'), entrada('b')])

      await Promise.resolve()
      expect(ditos).toEqual(['1 personagem entrou na iniciativa'])
    })
  })

  // O caso que o dono viu: clicar de novo com todo mundo já dentro. Não há
  // broadcast, e sem a espera a tela ficaria muda — que era o defeito.
  it('sem ninguém novo, diz que o grupo já está lá', async () => {
    await naRaiz(async () => {
      const [lista] = createSignal([entrada('a')])
      const ditos: string[] = []
      const anunciar = createPartyFeedback(lista, (m) => ditos.push(m))

      anunciar()
      expect(ditos).toEqual([])

      vi.advanceTimersByTime(2000)
      expect(ditos).toEqual(['O grupo já está na iniciativa'])
    })
  })

  it('não fala sozinho: mudança de lista sem clique é silêncio', async () => {
    await naRaiz(async () => {
      const [lista, setLista] = createSignal([entrada('a')])
      const ditos: string[] = []
      createPartyFeedback(lista, (m) => ditos.push(m))

      // Alguém entrou pela mão de outro mestre, ou saiu — não é resposta a
      // nada que ESTE cliente pediu.
      setLista([entrada('a'), entrada('b')])
      await Promise.resolve()
      vi.advanceTimersByTime(2000)

      expect(ditos).toEqual([])
    })
  })
})
