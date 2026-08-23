import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { PowerUsesProvider } from '@/shared/stores/power-uses-context'
import { StanceActivationProvider } from '@/shared/stores/stance-activation-context'
import { CharacterSheet } from './character-sheet'
import { fakeConditionals, fakePowerUses, fakeStances } from '@/shared/test/play-stores'

/**
 * As duas props que a ficha ganhou para caber numa COLUNA (ALE-122).
 *
 * O `matchMedia` aqui responde SEMPRE que casa — é o pior caso, o de uma janela
 * larga: é nele que a ficha escolheria o layout de duas colunas dentro de um
 * painel estreito e cortaria. Foi assim, numa janela de 1920, que 22 elementos
 * apareceram cortados num painel de 812px.
 */
beforeEach(() => {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: true,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
})

function renderSheet(props: { compact?: boolean; hudless?: boolean; tab?: string }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={fakeConditionals()}>
        <PowerUsesProvider store={fakePowerUses()}>
          <StanceActivationProvider store={fakeStances()}>
            <CharacterSheet
              character={makeCharacter({ name: 'Paladino Sagrado' })}
              tab="expertises"
              onTabChange={vi.fn()}
              {...props}
            />
          </StanceActivationProvider>
        </PowerUsesProvider>
      </ConditionalsProvider>
    </QueryClientProvider>
  ))
}

describe('CharacterSheet numa coluna', () => {
  // Os dois layouts têm as mesmas abas com os mesmos nomes ACESSÍVEIS —
  // contá-las não distingue um do outro (a primeira versão deste teste passou
  // verde sobre a implementação sabotada por isso). O que o usuário vê difere:
  // no largo o rótulo é TEXTO ao lado do ícone, no estreito a aba é só ícone.
  it('compact força o layout de um bloco por vez, mesmo em janela larga', () => {
    renderSheet({ compact: true })

    expect(screen.getByRole('tab', { name: 'Perícias' })).toHaveTextContent('')
  })

  it('sem compact, em janela larga, a aba mostra o rótulo escrito', () => {
    renderSheet({})

    expect(screen.getByRole('tab', { name: 'Perícias' })).toHaveTextContent('Perícias')
  })

  // Quem monta o painel já mostra o cartão de combate acima; dois HUDs seriam
  // dois lugares com o mesmo PV na mesma coluna — o defeito que originou a issue.
  it('hudless não repete o HUD', () => {
    renderSheet({ compact: true, hudless: true })

    expect(screen.queryByText('Vida')).not.toBeInTheDocument()
    expect(screen.queryByText('Mana')).not.toBeInTheDocument()
  })

  it('sem hudless o HUD continua lá', () => {
    renderSheet({ compact: true })

    expect(screen.getByText('Vida')).toBeInTheDocument()
  })

  /**
   * Defesa, ataques, resistências, atributos e fórmulas de arma não tinham aba
   * nenhuma: moravam só no `CharacterHud`, que os esconde abaixo de `md`. Num
   * telefone o jogador nunca via os próprios ataques, e tirá-los da faixa do
   * combatente sem lhes dar casa teria apagado a informação da tela do mestre
   * também (ALE-145).
   */
  it('a aba Combate dá casa aos números que saíram da faixa do combatente', () => {
    renderSheet({ compact: true, hudless: true, tab: 'combat' })

    expect(screen.getByText('Atq CaC')).toBeInTheDocument()
    expect(screen.getByText('Atq Dist')).toBeInTheDocument()
    expect(screen.getByText('Fort')).toBeInTheDocument()
    expect(screen.getByText('INT')).toBeInTheDocument()
  })
})
