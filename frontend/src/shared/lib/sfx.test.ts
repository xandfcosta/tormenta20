import { FakeStorage } from '@/shared/test/fake-storage'
import { describe, expect, it } from 'vitest'
import { createUiStore } from '@/shared/stores/ui-store'
import { createSfx, createSfxToggle } from './sfx'
import type { SfxName, SfxPlayer } from './sfx-player'

/** Named fake for the audio backend — no Web Audio in the tests. */
class FakeSfxPlayer implements SfxPlayer {
  readonly played: SfxName[] = []
  readonly volumes: number[] = []
  play(name: SfxName, volume: number): void {
    this.played.push(name)
    this.volumes.push(volume)
  }
}


function world(coarsePointer = false) {
  const player = new FakeSfxPlayer()
  const ui = createUiStore(new FakeStorage())
  return { player, ui, sfx: createSfx(ui, () => player, () => coarsePointer) }
}

describe('createSfx', () => {
  // Sound is off by default so the app never surprises anyone with audio.
  it('fica mudo enquanto o som está desligado', () => {
    const { player, sfx } = world()
    sfx('select')
    expect(player.played).toEqual([])
  })

  it('toca depois que o jogador liga o som', () => {
    const { player, ui, sfx } = world()
    ui.setSfx(true)
    sfx('select')
    sfx('hover')
    expect(player.played).toEqual(['select', 'hover'])
  })

  // The gate is read at call time, not captured — a mid-session toggle takes
  // effect without rewiring every caller.
  it('respeita o desligar no meio da sessão', () => {
    const { player, ui, sfx } = world()
    ui.setSfx(true)
    sfx('open')
    ui.setSfx(false)
    sfx('back')
    expect(player.played).toEqual(['open'])
  })

  // No celular o `hover` dispara no MESMO toque que dispara o `select`, e a
  // pressão única vira som duplo (ALE-165).
  it('cala o hover em tela de toque, sem calar o select', () => {
    const { player, ui, sfx } = world(true)
    ui.setSfx(true)
    sfx('hover')
    sfx('select')
    expect(player.played).toEqual(['select'])
  })
})

describe('createSfxToggle', () => {
  // O clique que liga o som é o gesto que destrava o áudio: é o primeiro
  // instante em que uma confirmação pode ser OUVIDA, e o silêncio nesse
  // instante é o que fazia o jogador achar que o som não funcionava.
  it('ligar o som responde com um cue', () => {
    const { player, ui, sfx } = world()
    createSfxToggle(ui, sfx)()
    expect(ui.sfx()).toBe(true)
    expect(player.played).toEqual(['select'])
  })

  it('desligar o som não toca nada', () => {
    const { player, ui, sfx } = world()
    ui.setSfx(true)
    createSfxToggle(ui, sfx)()
    expect(ui.sfx()).toBe(false)
    expect(player.played).toEqual([])
  })
})

describe('volume', () => {
  // O store guarda 0–100 (é o que o slider mostra) e o player quer 0–1: mandar
  // 70 onde se espera 0,7 multiplicaria o ganho por setenta (ALE-180).
  it('converte a porcentagem do store na escala do player', () => {
    const { player, ui, sfx } = world()
    ui.setSfx(true)
    ui.setVolume(70)
    sfx('turn')
    expect(player.volumes).toEqual([0.7])
  })

  it('volume zero chega como zero, e o player é quem cala', () => {
    const { player, ui, sfx } = world()
    ui.setSfx(true)
    ui.setVolume(0)
    sfx('turn')
    expect(player.volumes).toEqual([0])
  })
})
