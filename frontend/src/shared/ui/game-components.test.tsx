import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { Swords } from 'lucide-solid'
import { describe, expect, it, vi } from 'vitest'
import { BackgroundTexture } from './background-texture'
import { CharacterPortrait } from './character-portrait'
import { GameMenuButton } from './game-menu-button'
import { Input } from './input'
import { Label } from './label'
import { SceneTitle } from './scene-title'

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

// Os blocos `Badge` e `Skeleton` saíram inteiros na ALE-187.
//
// O do Badge afirmava o `data-variant`, e — conferido — nenhum CSS, e2e ou
// teste o consome; diferente do `BackgroundTexture` abaixo, o Badge não
// DECLARA esse atributo como contrato, ele só é carregado junto.
//
// Os do Skeleton contavam quantos cards a prop pediu: é um `For` sobre um
// array do tamanho da prop, que o typechecker já garante, e não há defeito na
// história deles — o comentário que morava ali defendia COMO contar (pelo
// slot, não pela classe), não SE contar.

describe('SceneTitle', () => {
  it('é h1 por padrão', () => {
    render(() => <SceneTitle>Tormenta 20</SceneTitle>)
    expect(screen.getByRole('heading', { level: 1, name: 'Tormenta 20' })).toBeInTheDocument()
  })

  it('vira h2 quando a cena já tem um h1', () => {
    render(() => <SceneTitle as="h2">Campanhas</SceneTitle>)
    expect(screen.getByRole('heading', { level: 2, name: 'Campanhas' })).toBeInTheDocument()
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
    render(() => <GameMenuButton active>Campanhas</GameMenuButton>)
    expect(screen.getByRole('button')).toHaveAttribute('aria-current', 'page')
  })

  it('sem active, não deixa aria-current pendurado', () => {
    render(() => <GameMenuButton>Campanhas</GameMenuButton>)
    expect(screen.getByRole('button')).not.toHaveAttribute('aria-current')
  })

  // 'rende o ícone quando passado' saiu na ALE-187: `querySelector('svg')` é
  // forma de DOM. O que importa do ícone — não roubar o nome acessível — é o
  // caso abaixo, e esse fica.

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


  // As três variações de INICIAIS saíram na ALE-187: elas re-testavam o
  // `initials` do `shared/lib` através do retrato, e a função tem dono em
  // `entities/campaign/emblem.test.ts`. Sobrou o caso de cima, que prova o
  // que este componente promete — que o retrato MOSTRA iniciais.

  it('tinge com o hue quando informado', () => {
    const { container } = render(() => <CharacterPortrait name="Arsenal" size="lg" hue={210} />)
    expect(container.firstElementChild?.getAttribute('style')).toContain('210')
  })



  it('é decorativo: as iniciais não entram no nome acessível da linha', () => {
    const { container } = render(() => <CharacterPortrait name="Arsenal" size="sm" />)
    expect(container.firstElementChild).toHaveAttribute('aria-hidden', 'true')
  })
})
