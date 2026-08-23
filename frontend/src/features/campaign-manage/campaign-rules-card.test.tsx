import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/shared/api/api'
import { CampaignRulesCard } from './campaign-rules-card'

/**
 * Os interruptores das regras opcionais (ALE-221).
 *
 * O que este arquivo protege é o CONTRATO com o servidor — que o clique manda o
 * conjunto inteiro e certo — e o que a tela anuncia. A consequência da regra
 * (sobrecarga, penalidades) é do motor e se prova lá; repeti-la aqui seria uma
 * terceira cópia da mesma verdade.
 */

function renderCard(ignoredRules: string[] = []) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const invalidate = vi.spyOn(client, 'invalidateQueries').mockResolvedValue()
  render(() => (
    <QueryClientProvider client={client}>
      <CampaignRulesCard campaignId={7} ignoredRules={ignoredRules} />
    </QueryClientProvider>
  ))
  return { user: userEvent.setup(), invalidate }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('as regras da campanha', () => {
  // O padrão do livro é a regra VALENDO, e a tela precisa dizer isso sem o
  // mestre ter de deduzir do silêncio.
  it('a campanha sem nada desligado mostra a regra em vigor', () => {
    renderCard()

    const botao = screen.getByRole('button', { name: 'Limites de carga' })
    expect(botao).toHaveAttribute('aria-pressed', 'true')
    expect(botao).toHaveTextContent('Em vigor')
  })

  it('a campanha que desligou mostra a regra desligada', () => {
    renderCard(['carga'])

    const botao = screen.getByRole('button', { name: 'Limites de carga' })
    expect(botao).toHaveAttribute('aria-pressed', 'false')
    expect(botao).toHaveTextContent('Desligada')
  })

  it('desligar manda o identificador da regra', async () => {
    const replaceRules = vi.spyOn(api.campaigns, 'replaceRules').mockResolvedValue({
      ignoredRules: ['carga'],
    })
    const { user } = renderCard()

    await user.click(screen.getByRole('button', { name: 'Limites de carga' }))

    expect(replaceRules).toHaveBeenCalledWith(7, ['carga'])
  })

  // Religar é a MESMA rota com a lista sem a regra: é substituição, não delta.
  // Mandar `['carga']` de novo aqui deixaria a regra desligada para sempre.
  it('religar manda a lista sem a regra', async () => {
    const replaceRules = vi.spyOn(api.campaigns, 'replaceRules').mockResolvedValue({
      ignoredRules: [],
    })
    const { user } = renderCard(['carga'])

    await user.click(screen.getByRole('button', { name: 'Limites de carga' }))

    expect(replaceRules).toHaveBeenCalledWith(7, [])
  })

  /**
   * A invalidação das FICHAS é a metade que ninguém vê e que quebra calada: a
   * regra vive na campanha e é aplicada na ficha, então sem derrubar
   * `['characters']` a ficha aberta ao lado continua mostrando a penalidade que
   * a mesa acabou de dispensar.
   */
  it('mexer na regra derruba o cache das FICHAS, e não só o da campanha', async () => {
    vi.spyOn(api.campaigns, 'replaceRules').mockResolvedValue({ ignoredRules: ['carga'] })
    const { user, invalidate } = renderCard()

    await user.click(screen.getByRole('button', { name: 'Limites de carga' }))

    await waitFor(() => {
      expect(invalidate).toHaveBeenCalledWith({ queryKey: ['characters'] })
    })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['campaigns', 7] })
  })

  // A falha vai INLINE e não por toast: este painel também é montado dentro de
  // um overlay na cena da sessão, e um toast disparado de dentro de modal não é
  // anunciado (o Kobalte marca os irmãos com `aria-hidden`).
  it('a recusa do servidor aparece no painel, anunciada', async () => {
    vi.spyOn(api.campaigns, 'replaceRules').mockRejectedValue(new Error('rede'))
    const { user } = renderCard()

    await user.click(screen.getByRole('button', { name: 'Limites de carga' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/Não foi possível salvar a regra/)
  })
})
