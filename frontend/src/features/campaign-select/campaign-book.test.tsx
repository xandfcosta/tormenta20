import { render, screen } from '@solidjs/testing-library'
import { createSignal } from 'solid-js'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Campaign } from '@/shared/api/api'
import { CampaignBook } from './campaign-book'

function campaign(id: number): Campaign {
  return {
    id,
    ownerId: 1,
    name: `Crônica ${id}`,
    description: null,
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
  }
}

/** Desktop + motion allowed, so the leaf actually turns. */
beforeEach(() => {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: !media.includes('prefers-reduced-motion'),
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
})

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

const leaf = () => document.querySelector<HTMLElement>('.grimorio-leaf')

/**
 * jsdom runs no CSS, so the browser's animationend is simulated here. This
 * jsdom has no `AnimationEvent` constructor at all, so it is a plain Event with
 * `animationName` pinned on — the component filters on that name and would
 * ignore a nameless event.
 */
function endLeafAnimation(name = 'grimorio-leaf-turn') {
  const event = new Event('animationend', { bubbles: true })
  Object.defineProperty(event, 'animationName', { value: name })
  leaf()?.dispatchEvent(event)
}

function mountBook() {
  const [pick, setPick] = createSignal(campaign(1))
  render(() => (
    <CampaignBook
      campaign={pick()}
      isLive={false}
      orderIds={[1, 2, 3, 4]}
      onOpen={() => {}}
      onResume={() => {}}
    />
  ))
  return setPick
}

/** Mounts with `isLive` under the test's control, as the page's query is. */
function mountBookWithLive() {
  const [live, setLive] = createSignal(false)
  render(() => (
    <CampaignBook
      campaign={campaign(1)}
      isLive={live()}
      orderIds={[1]}
      onOpen={() => {}}
      onResume={() => {}}
    />
  ))
  const actionLabels = () =>
    screen
      .getAllByRole('button')
      .map((button) => button.textContent ?? '')
      .filter((text) => /Abrir crônica|Continuar a sessão/.test(text))
      .map((text) => (text.includes('Continuar') ? 'continuar' : 'abrir'))
  return { setLive, actionLabels }
}

describe('CampaignBook — virada de folha', () => {
  it('em repouso não há folha virando', () => {
    mountBook()
    expect(leaf()).toBeNull()
  })

  it('escolher outra crônica põe uma folha em movimento', () => {
    const setPick = mountBook()

    setPick(campaign(2))

    expect(leaf()).not.toBeNull()
  })

  it('terminada a animação, a folha sai da frente', () => {
    const setPick = mountBook()
    setPick(campaign(2))

    endLeafAnimation()

    expect(leaf()).toBeNull()
  })

  /**
   * Voltar uma crônica anima com o espelho (`--rev`), e o `animationend` chega
   * com OUTRO nome. O componente filtra por nome, e o teste só disparava o nome
   * da ida: apagar o braço do `-rev` da produção deixava tudo verde, e na tela a
   * folha da volta ficaria congelada sobre a lombada para sempre.
   */
  it('a folha da VOLTA também sai da frente quando a animação termina', () => {
    const setPick = mountBook()
    setPick(campaign(2))
    endLeafAnimation()
    setPick(campaign(1)) // de volta: dir = -1

    endLeafAnimation('grimorio-leaf-turn-rev')

    expect(leaf()).toBeNull()
  })

  /**
   * A regressão: com escolhas enfileiradas, a folha da virada seguinte tem de
   * ser um elemento NOVO. Reaproveitar o nó deixa a animação CSS já terminada
   * nele — ela não recomeça, o `animationend` nunca vem, e a máquina trava com
   * a folha congelada sobre a lombada (nada mais anima depois disso).
   */
  it('cada virada enfileirada ganha uma folha nova, e não o nó reaproveitado', () => {
    const setPick = mountBook()
    setPick(campaign(2))
    const first = leaf()
    expect(first).not.toBeNull()

    // Duas escolhas durante a animação: elas ficam na fila.
    setPick(campaign(3))
    setPick(campaign(4))
    endLeafAnimation()

    const second = leaf()
    expect(second).not.toBeNull()
    expect(second).not.toBe(first)
  })

  /**
   * ALE-96. A cena das Crônicas passou a pintar ANTES de saber se a mesa tem
   * sessão viva — o suspend que antes a segurava era o próprio defeito. Com
   * isso o `isLive` vira verdadeiro com a linha de ações já na tela, e a ordem
   * de ontem punha `Continuar a sessão` na frente: o `Abrir crônica` que o
   * jogador estava mirando pulava para a segunda posição e o clique caía no
   * botão vizinho. É a ALE-78 outra vez, e a ordem é o que a impede.
   *
   * Medido no browser antes do conserto: SEGURADO `["Abrir crônica"]`,
   * ASSENTADO `["Continuar a sessão", "Abrir crônica"]`.
   */
  it('a sessão viva chegando não tira o "Abrir crônica" de baixo do cursor', () => {
    const { setLive, actionLabels } = mountBookWithLive()
    expect(actionLabels()).toEqual(['abrir'])

    setLive(true)

    expect(actionLabels()).toEqual(['abrir', 'continuar'])
  })

  it('a fila drena até o fim em vez de travar na primeira virada', () => {
    const setPick = mountBook()
    setPick(campaign(2))
    setPick(campaign(3))
    setPick(campaign(4))

    // Uma animação por virada enfileirada; ao final não sobra folha nenhuma.
    endLeafAnimation()
    endLeafAnimation()
    endLeafAnimation()

    expect(leaf()).toBeNull()
  })
})
