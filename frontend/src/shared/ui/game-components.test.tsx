import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { Swords } from 'lucide-solid'
import { describe, expect, it, vi } from 'vitest'
import { BackgroundTexture } from './background-texture'
import { Badge } from './badge'
import { CharacterPortrait } from './character-portrait'
import { GameMenuButton } from './game-menu-button'
import { Input } from './input'
import { Label } from './label'
import { SceneTitle } from './scene-title'
import { SkeletonCardGrid, SkeletonRows } from './skeleton'

describe('Input', () => {
  it('associa por for/id e recebe texto', async () => {
    render(() => (
      <>
        <Label for="email">E-mail</Label>
        <Input id="email" />
      </>
    ))
    await userEvent.setup().type(screen.getByLabelText('E-mail'), 'mestre@t20.local')
    expect(screen.getByLabelText('E-mail')).toHaveValue('mestre@t20.local')
  })

  it('expõe aria-invalid pro anel de erro', () => {
    render(() => <Input aria-invalid={true} />)
    expect(screen.getByRole('textbox')).toHaveAttribute('aria-invalid', 'true')
  })
})

describe('Badge', () => {
  it('usa a variante pedida', () => {
    render(() => <Badge variant="outline">Nv 10</Badge>)
    expect(screen.getByText('Nv 10')).toHaveAttribute('data-variant', 'outline')
  })
})

describe('Skeleton', () => {
  // A promessa é a QUANTIDADE (o esqueleto tem de ocupar o lugar do que vem), e
  // ela é contada pelo slot, não por classe de estilo — `.rounded-sm.border`
  // quebrava em qualquer restyle legítimo e não dizia nada ao usuário.
  it('SkeletonCardGrid rende a quantidade pedida de cards', () => {
    const { container } = render(() => <SkeletonCardGrid count={5} />)
    expect(container.querySelectorAll('[data-slot=skeleton-card]')).toHaveLength(5)
  })

  it('SkeletonRows rende a quantidade pedida de linhas', () => {
    const { container } = render(() => <SkeletonRows count={2} />)
    expect(container.querySelectorAll('[data-slot=skeleton-row]')).toHaveLength(2)
  })
})

describe('SceneTitle', () => {
  it('é h1 por padrão', () => {
    render(() => <SceneTitle>Tormenta 20</SceneTitle>)
    expect(screen.getByRole('heading', { level: 1, name: 'Tormenta 20' })).toBeInTheDocument()
  })

  it('vira h2 quando a cena já tem um h1', () => {
    render(() => <SceneTitle as="h2">Crônicas</SceneTitle>)
    expect(screen.getByRole('heading', { level: 2, name: 'Crônicas' })).toBeInTheDocument()
  })

  it('mostra o kicker quando passado', () => {
    render(() => <SceneTitle kicker="— Grimório de Arton —">Tormenta 20</SceneTitle>)
    expect(screen.getByText('— Grimório de Arton —')).toBeInTheDocument()
  })

  it('sem kicker, não rende o parágrafo vazio', () => {
    const { container } = render(() => <SceneTitle>Tormenta 20</SceneTitle>)
    expect(container.querySelector('p')).toBeNull()
  })
})

describe('BackgroundTexture', () => {
  // Asserts the VALUE, not just the attribute: Solid renders a bare
  // `aria-hidden` as `aria-hidden=""`, which does not hide anything.
  it('é decorativa: escondida de leitores de tela', () => {
    const { container } = render(() => <BackgroundTexture />)
    expect(container.querySelector('[data-slot=background-texture]')).toHaveAttribute(
      'aria-hidden',
      'true',
    )
  })

  it('stone por padrão, parchment sob demanda', () => {
    const { container } = render(() => <BackgroundTexture variant="parchment" />)
    const texture = container.querySelector('[data-slot=background-texture]')
    expect(texture).toHaveAttribute('data-variant', 'parchment')
  })

  it('vignette é opt-in', () => {
    const { container } = render(() => <BackgroundTexture vignette />)
    expect(container.querySelector('[data-slot=background-texture]')).toHaveAttribute(
      'data-vignette',
      'true',
    )
  })
})

describe('GameMenuButton', () => {
  it('navega no clique', async () => {
    const onClick = vi.fn()
    render(() => <GameMenuButton onClick={onClick}>Meus Heróis</GameMenuButton>)
    await userEvent.setup().click(screen.getByRole('button', { name: 'Meus Heróis' }))
    expect(onClick).toHaveBeenCalledOnce()
  })

  it('marca o destino atual com aria-current', () => {
    render(() => <GameMenuButton active>Crônicas</GameMenuButton>)
    expect(screen.getByRole('button')).toHaveAttribute('aria-current', 'page')
  })

  it('sem active, não deixa aria-current pendurado', () => {
    render(() => <GameMenuButton>Crônicas</GameMenuButton>)
    expect(screen.getByRole('button')).not.toHaveAttribute('aria-current')
  })

  it('rende o ícone quando passado', () => {
    const { container } = render(() => <GameMenuButton icon={Swords}>Sessão</GameMenuButton>)
    expect(container.querySelector('svg')).toBeInTheDocument()
  })

  it('o nome acessível é só o rótulo — os enfeites são aria-hidden', () => {
    render(() => (
      <GameMenuButton hasNext icon={Swords}>
        Continuar
      </GameMenuButton>
    ))
    expect(screen.getByRole('button', { name: 'Continuar' })).toBeInTheDocument()
  })
})

describe('CharacterPortrait', () => {
  it('mostra as iniciais das duas primeiras palavras', () => {
    render(() => <CharacterPortrait name="Tanque Placas Nv10" size="sm" />)
    expect(screen.getByText('TP')).toBeInTheDocument()
  })

  it('cai pra ? quando o nome é vazio', () => {
    render(() => <CharacterPortrait name="   " size="sm" />)
    expect(screen.getByText('?')).toBeInTheDocument()
  })

  it('nome de uma palavra usa só a inicial', () => {
    render(() => <CharacterPortrait name="Arsenal" size="sm" />)
    expect(screen.getByText('A')).toBeInTheDocument()
  })

  it('tinge com o hue quando informado', () => {
    const { container } = render(() => <CharacterPortrait name="Arsenal" size="lg" hue={210} />)
    expect(container.firstElementChild?.getAttribute('style')).toContain('210')
  })



  it('é decorativo: as iniciais não entram no nome acessível da linha', () => {
    const { container } = render(() => <CharacterPortrait name="Arsenal" size="sm" />)
    expect(container.firstElementChild).toHaveAttribute('aria-hidden', 'true')
  })
})
