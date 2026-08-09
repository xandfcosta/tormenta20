import { renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useUiStore } from '@/shared/stores/ui-store'
import { useSfx } from './use-sfx'

// Named fake for the audio backend — the player module is the I/O seam.
const { playSpy } = vi.hoisted(() => ({ playSpy: vi.fn() }))
vi.mock('./sfx-player', () => ({ createSfxPlayer: () => ({ play: playSpy }) }))

afterEach(() => {
  vi.clearAllMocks()
  useUiStore.setState({ sfx: false })
})

describe('useSfx', () => {
  it('plays the requested cue when sound is enabled', () => {
    useUiStore.setState({ sfx: true })
    const { result } = renderHook(() => useSfx())
    result.current('select')
    expect(playSpy).toHaveBeenCalledWith('select')
  })

  it('stays silent when sound is disabled', () => {
    useUiStore.setState({ sfx: false })
    const { result } = renderHook(() => useSfx())
    result.current('hover')
    expect(playSpy).not.toHaveBeenCalled()
  })
})
