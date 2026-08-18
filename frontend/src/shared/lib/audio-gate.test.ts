import { describe, expect, it } from 'vitest'
import { AudioGate, type GatedAudioContext, type GestureTarget } from './audio-gate'

/** Named fake for Web Audio — reports the state the browser would report. */
class FakeAudioContext implements GatedAudioContext {
  state: AudioContextState
  resumes = 0
  constructor(bornAs: AudioContextState) {
    this.state = bornAs
  }
  resume(): Promise<void> {
    this.resumes++
    // The spec flips the state synchronously inside resume(); Chrome only
    // allows it when a gesture is in flight, which is exactly when we call it.
    this.state = 'running'
    return Promise.resolve()
  }
}

/** Named fake for `window` — the test fires the gesture by hand. */
class FakeGestures implements GestureTarget {
  private readonly listeners = new Map<string, (() => void)[]>()
  addEventListener(type: string, listener: () => void, _options?: { capture?: boolean }): void {
    this.listeners.set(type, [...(this.listeners.get(type) ?? []), listener])
  }
  fire(type: string): void {
    for (const listener of this.listeners.get(type) ?? []) listener()
  }
}

function world(bornAs: AudioContextState = 'running') {
  const gestures = new FakeGestures()
  const opened: FakeAudioContext[] = []
  const gate = new AudioGate<FakeAudioContext>(() => {
    const ctx = new FakeAudioContext(bornAs)
    opened.push(ctx)
    return ctx
  }, gestures)
  return { gate, gestures, opened }
}

describe('AudioGate', () => {
  // O defeito da ALE-165: o contexto nascia no mount da cena, ficava `suspended`
  // com `currentTime` congelado em 0, e todo cue agendado ali morria em silêncio
  // (o Chrome ainda logava "The AudioContext was not allowed to start").
  it('não constrói o áudio antes do primeiro gesto — e não entrega cue nenhum', () => {
    const { gate, opened } = world()

    expect(gate.ready()).toBeNull()
    expect(opened).toHaveLength(0)
  })

  it('o primeiro gesto constrói o contexto, e é dentro dele que o áudio nasce tocando', () => {
    const { gate, gestures, opened } = world()

    gestures.fire('pointerdown')

    expect(opened).toHaveLength(1)
    expect(gate.ready()).toBe(opened[0])
  })

  // Nem todo browser deixa o contexto nascer tocando; quando ele nasce parado,
  // é o resume() DE DENTRO do gesto que o solta.
  it('retoma o contexto que nasce parado, no mesmo gesto', () => {
    const { gate, gestures, opened } = world('suspended')

    gestures.fire('keydown')

    expect(opened[0]?.resumes).toBe(1)
    expect(gate.ready()).toBe(opened[0])
  })

  // O celular estaciona o contexto quando a aba vai para segundo plano; o gesto
  // seguinte tem de revivê-lo, e não abrir um segundo.
  it('revive o contexto estacionado sem abrir outro', () => {
    const { gate, gestures, opened } = world()
    gestures.fire('pointerdown')
    const ctx = opened[0]
    if (!ctx) throw new Error('esperado um contexto aberto pelo primeiro gesto')
    ctx.state = 'suspended'

    expect(gate.ready()).toBeNull()
    gestures.fire('pointerdown')

    expect(opened).toHaveLength(1)
    expect(gate.ready()).toBe(ctx)
  })
})

/**
 * O evento tem de ser ouvido na CAPTURA. Medido no browser: clicar numa área
 * neutra armava o áudio e clicar num CONTROLE não — o componente chama
 * `stopPropagation` no `pointerdown`, e na fase de borbulha o evento morre
 * antes de chegar à janela. Quem só clica em botões nunca teria som (ALE-185).
 *
 * Este caso usa DOM de verdade porque a regra é sobre PROPAGAÇÃO, e um alvo
 * falso teria de reimplementar exatamente a coisa que está sendo testada.
 */
describe('AudioGate no DOM real', () => {
  it('arma mesmo quando o controle clicado engole o evento', () => {
    const botao = document.createElement('button')
    document.body.append(botao)
    botao.addEventListener('pointerdown', (event) => event.stopPropagation())
    const gate = new AudioGate(() => new FakeAudioContext('running'), window)

    botao.dispatchEvent(new Event('pointerdown', { bubbles: true }))

    expect(gate.ready()).not.toBeNull()
    botao.remove()
  })
})
