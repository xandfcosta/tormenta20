import { render, screen } from '@solidjs/testing-library'
import { createSignal } from 'solid-js'
import { describe, expect, it } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { CharacterStage } from './character-stage'

const HEROI = makeCharacter({ id: 1, name: 'Thal, o Errante' })
const OUTRO = makeCharacter({ id: 2, name: 'Tanque Placas Nv10' })

/**
 * Renders the stage with the selected character and the defense behind signals,
 * so a test can swap either one the way the roster does.
 */
function renderStage() {
  const [selected, setSelected] = createSignal(HEROI)
  const [defense, setDefense] = createSignal<number | null>(null)
  render(() => (
    <CharacterStage
      selected={selected()}
      prev={null}
      next={OUTRO}
      direction={1}
      defense={defense()}
      onStep={() => {}}
      onOpen={() => {}}
      onDossier={() => {}}
      dossierOpen={false}
    />
  ))
  const portrait = () => screen.getByRole('button', { name: /^Abrir ficha de/ })
  return { portrait, setSelected, setDefense }
}

describe('CharacterStage', () => {
  it('mostra o personagem selecionado', () => {
    const { portrait } = renderStage()

    expect(portrait()).toHaveAccessibleName('Abrir ficha de Thal, o Errante')
  })

  /**
   * ALE-97. The slide/zoom is an `animate-in` class, which fires on MOUNT and
   * only on mount. Reusing the node — what a non-`keyed` Show does when one
   * character replaces another — leaves the animation silent, and the roster
   * switched characters with no transition at all. jsdom has no animation
   * timeline, so what's asserted is the thing that makes the animation possible:
   * the portrait must be a NEW element after a selection change.
   */
  it('reconstrói o retrato ao trocar de personagem (senão a animação não toca)', () => {
    const { portrait, setSelected } = renderStage()
    const antes = portrait()

    setSelected(OUTRO)

    expect(portrait()).toHaveAccessibleName('Abrir ficha de Tanque Placas Nv10')
    expect(portrait()).not.toBe(antes)
  })

  /** The mirror of the rule: the computed sheet landing is NOT a selection
   *  change, and re-playing the entrance every time DEF arrives would flash. */
  it('não reconstrói o retrato quando só a DEF chega', () => {
    const { portrait, setDefense } = renderStage()
    const antes = portrait()

    setDefense(22)

    expect(screen.getByText('22')).toBeInTheDocument()
    expect(portrait()).toBe(antes)
  })
})
