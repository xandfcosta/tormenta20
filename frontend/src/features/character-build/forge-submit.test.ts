import { QueryClient } from '@tanstack/solid-query'
import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/shared/api/api'
import type { CreateCharacterInput } from '@/shared/api/types'
import { createCharacterDraftStore } from '@/shared/stores/character-draft-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { createForgeSubmit } from './forge-submit'

/** Named fake for the one write the Forja makes (CLAUDE.md: fakes, not stubs). */
class FakeCharacterWriter {
  readonly bodies: CreateCharacterInput[] = []
  private release: (() => void) | null = null

  constructor(private readonly outcome: 'ok' | ApiError | Error = 'ok') {}

  readonly create = async (input: CreateCharacterInput): Promise<{ id: number }> => {
    this.bodies.push(input)
    if (this.release) await new Promise<void>((resolve) => (this.release = resolve))
    if (this.outcome !== 'ok') throw this.outcome
    return { id: 7 }
  }

  /** Makes the next call hang until `finish()` — for testing the in-flight guard. */
  hold(): void {
    this.release = () => {}
  }

  finish(): void {
    this.release?.()
    this.release = null
  }
}

function makeForge(writer: FakeCharacterWriter) {
  const draft = createCharacterDraftStore(new FakeStorage())
  draft.setValue('name', 'Thal')
  draft.setValue('races', ['Humano'])
  draft.setValue('origin', 'Acólito')
  draft.setValue('classes', [{ className: 'Guerreiro', level: 1 }])
  const onCreated = vi.fn()
  const forge = createForgeSubmit({
    draft,
    queryClient: new QueryClient(),
    createCharacter: writer.create,
    onCreated,
  })
  return { draft, forge, onCreated }
}

describe('createForgeSubmit', () => {
  it('envia o rascunho e leva para a ficha criada', async () => {
    const writer = new FakeCharacterWriter()
    const { forge, onCreated } = makeForge(writer)

    await forge.create()

    expect(writer.bodies[0].name).toBe('Thal')
    expect(onCreated).toHaveBeenCalledWith(7)
  })

  it('só descarta o rascunho depois que o servidor aceitou', async () => {
    const writer = new FakeCharacterWriter(new ApiError(500, 'Could not create character'))
    const { draft, forge } = makeForge(writer)

    await forge.create()

    // Perder a ficha inteira porque o servidor caiu é o pior desfecho possível.
    expect(draft.values.name).toBe('Thal')
    expect(forge.error).toBe('Could not create character')
  })

  it('limpa o rascunho quando deu certo', async () => {
    const { draft, forge } = makeForge(new FakeCharacterWriter())

    await forge.create()

    expect(draft.values.name).toBe('')
    expect(draft.values.classes).toEqual([])
  })

  it('traduz uma falha inesperada em vez de vazar o erro cru', async () => {
    const { forge } = makeForge(new FakeCharacterWriter(new TypeError('fetch failed')))

    await forge.create()

    expect(forge.error).toMatch(/não foi possível forjar/i)
  })

  it('não forja dois personagens com um clique duplo', async () => {
    const writer = new FakeCharacterWriter()
    writer.hold()
    const { forge } = makeForge(writer)

    const first = forge.create()
    await forge.create()
    writer.finish()
    await first

    expect(writer.bodies).toHaveLength(1)
  })

  it('limpa o erro anterior ao tentar de novo', async () => {
    const failing = new FakeCharacterWriter(new ApiError(500, 'Could not create character'))
    const { forge } = makeForge(failing)
    await forge.create()
    expect(forge.error).not.toBeNull()

    const { forge: retry } = makeForge(new FakeCharacterWriter())
    await retry.create()

    expect(retry.error).toBeNull()
  })
})
