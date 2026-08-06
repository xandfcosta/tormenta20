import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SPELL_CATALOG } from '@tormenta20/t20-data'
import type { Character, CharacterSpell } from '@/shared/api/api'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import { SpellRow } from './spell-row'

// Named fake — full Character shape with neutral defaults; tests override
// only what the row under test reads (classes, level, mp, spells).
function fakeCharacter(overrides: Partial<Character> = {}): Character {
  return {
    id: 1,
    ownerId: 1,
    name: 'Teste Conjurador',
    origin: 'Acólito',
    god: null,
    godPower: '',
    tibar: 0,
    level: 4,
    hpMax: 20,
    hpCurrent: 20,
    mpMax: 12,
    mpCurrent: 12,
    strength: 0,
    dexterity: 1,
    constitution: 1,
    intelligence: 3,
    wisdom: 1,
    charisma: 0,
    size: 'Médio',
    displacement: 9,
    proficiencies: '[]',
    raceAbilityChoices: '[]',
    activeConditions: '[]',
    raceAttributeChoices: '{}',
    secondaryRaceChoices: '[]',
    originChoices: '[]',
    classPowers: '[]',
    classChoices: '{}',
    powerChoices: '{}',
    createdAt: '',
    updatedAt: '',
    races: [{ race: 'Humano' }],
    classes: [{ className: 'Arcanista', level: 4 }],
    expertises: [],
    items: [],
    activeEffects: [],
    spells: [],
    ...overrides,
  }
}

const CIRCLE1_SPELL = Object.values(SPELL_CATALOG).find(
  (s) => s.circle === 1 && s.classes.includes('Arcanista'),
)
const CIRCLE5_SPELL = Object.values(SPELL_CATALOG).find(
  (s) => s.circle === 5 && s.classes.includes('Arcanista'),
)

function learnedRow(spellId: string): CharacterSpell {
  return { id: 1, catalogSpellId: spellId, prepared: true, learnedAt: '' }
}

function renderWithQuery(ui: React.ReactElement) {
  return render(
    <QueryClientProvider client={new QueryClient()}>{ui}</QueryClientProvider>,
  )
}

describe('SpellRow — Conjurar sempre visível', () => {
  it('mostra o botão Conjurar no cabeçalho sem expandir a linha', () => {
    const spell = CIRCLE1_SPELL!
    const character = fakeCharacter({ spells: [learnedRow(spell.id)] })
    renderWithQuery(
      <SpellRow
        spell={spell}
        character={character}
        casterClasses={['Arcanista']}
        learned={character.spells[0]}
        spellCdByAttribute={computedSheetFor(character).spellCdByAttribute}
        />,
    )
    // Collapsed: expanded-only content is absent, but the cast trigger shows.
    expect(screen.queryByText('Execução')).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: `Conjurar ${spell.name}` }),
    ).toBeEnabled()
  })

  it('clicar em Conjurar abre o diálogo de conjuração direto', () => {
    const spell = CIRCLE1_SPELL!
    const character = fakeCharacter({ spells: [learnedRow(spell.id)] })
    renderWithQuery(
      <SpellRow
        spell={spell}
        character={character}
        casterClasses={['Arcanista']}
        learned={character.spells[0]}
        spellCdByAttribute={computedSheetFor(character).spellCdByAttribute}
        />,
    )
    fireEvent.click(
      screen.getByRole('button', { name: `Conjurar ${spell.name}` }),
    )
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Custo total')).toBeInTheDocument()
  })

  it('não mostra Conjurar quando a magia não é aprendida', () => {
    const spell = CIRCLE1_SPELL!
    const character = fakeCharacter()
    renderWithQuery(
      <SpellRow
        spell={spell}
        character={character}
        casterClasses={['Arcanista']}
        learned={null}
        spellCdByAttribute={computedSheetFor(character).spellCdByAttribute}
        />,
    )
    expect(
      screen.queryByRole('button', { name: `Conjurar ${spell.name}` }),
    ).not.toBeInTheDocument()
  })

  it('desabilita Conjurar quando o círculo excede o máximo conjurável', () => {
    const spell = CIRCLE5_SPELL!
    const character = fakeCharacter({
      level: 1,
      classes: [{ className: 'Arcanista', level: 1 }],
      spells: [learnedRow(spell.id)],
    })
    renderWithQuery(
      <SpellRow
        spell={spell}
        character={character}
        casterClasses={['Arcanista']}
        learned={character.spells[0]}
        spellCdByAttribute={computedSheetFor(character).spellCdByAttribute}
        />,
    )
    expect(
      screen.getByRole('button', { name: `Conjurar ${spell.name}` }),
    ).toBeDisabled()
  })

  it('expandir a linha continua funcionando após o novo cabeçalho', () => {
    const spell = CIRCLE1_SPELL!
    const character = fakeCharacter({ spells: [learnedRow(spell.id)] })
    renderWithQuery(
      <SpellRow
        spell={spell}
        character={character}
        casterClasses={['Arcanista']}
        learned={character.spells[0]}
        spellCdByAttribute={computedSheetFor(character).spellCdByAttribute}
        />,
    )
    // Radix's DialogTrigger (Conjurar) also exposes aria-expanded, so pick
    // the header toggle by excluding the trigger's accessible name.
    fireEvent.click(
      screen.getByRole('button', {
        expanded: false,
        name: (name) => !name.startsWith('Conjurar'),
      }),
    )
    expect(screen.getByText('Execução')).toBeInTheDocument()
  })
})
