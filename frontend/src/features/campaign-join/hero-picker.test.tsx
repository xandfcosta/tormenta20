import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Character } from '@/shared/api/api'
import { HeroPicker } from './hero-picker'
import { InviteLetter } from './invite-letter'

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

function hero(id: number, name: string, classes: { className: string; level: number }[]): Character {
  return {
    id,
    name,
    level: classes[0]?.level ?? 1,
    classes,
    ownerId: 1,
    origin: '',
    god: null,
    godPower: '',
    tibar: 0,
    hpMax: 20,
    hpCurrent: 20,
    mpMax: 5,
    mpCurrent: 5,
    strength: 10,
    dexterity: 10,
    constitution: 10,
    intelligence: 10,
    wisdom: 10,
    charisma: 10,
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
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    races: [],
    expertises: [],
    items: [],
    activeEffects: [],
    spells: [],
    conditionals: [],
    powerUses: [],
    stances: [],
  }
}

const ROSTER = [
  hero(1, 'Tanque Placas', [{ className: 'Guerreiro', level: 10 }]),
  hero(2, 'Arcanista Erudito', [{ className: 'Arcanista', level: 9 }]),
]

describe('HeroPicker', () => {
  it('mostra cada herói com a linha de classe/nível', () => {
    render(() => <HeroPicker characters={ROSTER} selectedId={null} onSelect={vi.fn()} />)

    expect(screen.getByRole('button', { name: /Tanque Placas/ })).toBeInTheDocument()
    expect(screen.getByText('Guerreiro 10')).toBeInTheDocument()
    expect(screen.getByText('Arcanista 9')).toBeInTheDocument()
  })

  it('avisa qual herói foi escolhido', async () => {
    const onSelect = vi.fn()
    render(() => <HeroPicker characters={ROSTER} selectedId={null} onSelect={onSelect} />)

    await userEvent.setup().click(screen.getByRole('button', { name: /Arcanista Erudito/ }))

    expect(onSelect).toHaveBeenCalledWith(2)
  })

  // A borda dourada é a única marca visual do escolhido; sem aria-pressed
  // um leitor de tela não saberia qual está selecionado.
  it('anuncia a seleção com aria-pressed', () => {
    render(() => <HeroPicker characters={ROSTER} selectedId={2} onSelect={vi.fn()} />)

    expect(screen.getByRole('button', { name: /Arcanista Erudito/ })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    expect(screen.getByRole('button', { name: /Tanque Placas/ })).toHaveAttribute(
      'aria-pressed',
      'false',
    )
  })

  it('cai no nível quando o herói não tem classes', () => {
    render(() => <HeroPicker characters={[hero(3, 'Recruta', [])]} selectedId={null} onSelect={vi.fn()} />)

    expect(screen.getByText('Nv 1')).toBeInTheDocument()
  })
})

describe('InviteLetter', () => {
  it('nomeia a mesa para a qual o jogador foi chamado', () => {
    render(() => <InviteLetter loading={false} invalid={false} campaignName="Snapshot Test" />)

    expect(screen.getByText('Snapshot Test')).toBeInTheDocument()
  })

  // Um token morto tem que DIZER que morreu — senão o jogador fica olhando o
  // botão desabilitado sem entender o motivo.
  it('explica o convite morto e o que fazer', () => {
    render(() => <InviteLetter loading={false} invalid campaignName={undefined} />)

    expect(screen.getByText(/Convite inválido ou expirado/)).toBeInTheDocument()
    expect(screen.getByText(/Peça um novo link ao mestre/)).toBeInTheDocument()
  })

  it('enquanto resolve, não afirma nada sobre a mesa', () => {
    render(() => <InviteLetter loading invalid={false} campaignName={undefined} />)

    expect(screen.queryByText(/Você foi convidado/)).not.toBeInTheDocument()
    expect(screen.queryByText(/inválido/)).not.toBeInTheDocument()
  })
})
