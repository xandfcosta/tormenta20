import { FakeStorage } from '@/shared/test/fake-storage'
import { describe, expect, it } from 'vitest'
import { createUiStore } from '@/shared/stores/ui-store'
import { createSfx } from './sfx'
import type { SfxName, SfxPlayer } from './sfx-player'

/** Named fake for the audio backend — no Web Audio in the tests. */
class FakeSfxPlayer implements SfxPlayer {
  readonly played: SfxName[] = []
  play(name: SfxName): void {
    this.played.push(name)
  }
}


function world() {
  const player = new FakeSfxPlayer()
  const ui = createUiStore(new FakeStorage())
  return { player, ui, sfx: createSfx(ui, () => player) }
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
})
