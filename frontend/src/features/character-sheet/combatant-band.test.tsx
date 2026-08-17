import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { createConditionalsStore } from '@/shared/stores/conditionals-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { CombatantBand } from './combatant-band'

function renderBand(activeConditions?: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={createConditionalsStore(new FakeStorage())}>
        <CombatantBand
          character={makeCharacter({
            name: 'Paladino Sagrado',
            level: 9,
            ...(activeConditions ? { activeConditions } : {}),
          })}
          actions={<button type="button">Fechar o combatente</button>}
        />
      </ConditionalsProvider>
    </QueryClientProvider>
  ))
}

/**
 * A faixa do combatente é uma DECISÃO de escopo do dono, não um estilo: nome,
 * nível, PV/PM com ajuste, Defesa e condições — e nada mais, porque tudo o que
 * ela mostra o mestre paga em altura na região onde a ficha precisa caber
 * (ALE-145). Um teste que só afirmasse o que ela mostra deixaria a lista
 * crescer de volta em silêncio, que é exatamente como o cartão inteiro foi
 * parar ali.
 */
describe('CombatantBand', () => {
  it('mostra o básico que o mestre toca por turno', () => {
    renderBand()

    expect(screen.getByRole('heading', { name: 'Paladino Sagrado' })).toBeInTheDocument()
    expect(screen.getByText('Nv 9')).toBeInTheDocument()
    // Defesa é chip com nome acessível: o glifo de escudo sozinho não se lê.
    expect(screen.getByRole('img', { name: /^Defesa \d+$/ })).toBeInTheDocument()
    expect(screen.getByRole('progressbar', { name: 'Vida' })).toBeInTheDocument()
    expect(screen.getByRole('progressbar', { name: 'Mana' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Reduzir Vida/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Aumentar Mana/ })).toBeInTheDocument()
    // "Você está caído" é o que um mestre mais declara: fica a um clique daqui,
    // e não a dois pela aba Efeitos (decisão herdada da ALE-122).
    expect(screen.getByLabelText('Aplicar condição')).toBeInTheDocument()
  })

  it('NÃO traz ataques, resistências nem atributos — eles moram na aba Combate', () => {
    renderBand()

    expect(screen.queryByText('Atq CaC')).not.toBeInTheDocument()
    expect(screen.queryByText('Atq Dist')).not.toBeInTheDocument()
    expect(screen.queryByText('Fort')).not.toBeInTheDocument()
    expect(screen.queryByText('INT')).not.toBeInTheDocument()
  })

  /**
   * A ALE-147: as condições eram o único conteúdo da faixa que crescia sozinho
   * durante o combate, e cresciam sem limite. Com duas ativas o nome do
   * combatente já virava "AI" e as ações iam para fora da tela.
   *
   * O que este teste protege é o LIMITE, que é o que impede o crescimento —
   * a largura resultante é assunto do browser (e da asserção em `session.spec`).
   */
  it('acima do limite, os chips param de crescer e o resto vai para o popover', async () => {
    renderBand('["abalado","agarrado","cego","atordoado"]')
    const user = userEvent.setup()

    // Dois chips na fileira, e o gatilho diz o TOTAL — nada fica escondido sem
    // aviso de que existe.
    expect(screen.getAllByRole('button', { name: /^Remover condição/ })).toHaveLength(2)
    const gatilho = screen.getByRole('button', { name: 'Ver as 4 condições ativas' })

    // E as quatro continuam removíveis por dentro dele.
    await user.click(gatilho)
    const popover = await screen.findByRole('dialog')
    expect(within(popover).getAllByRole('button', { name: /^Remover condição/ })).toHaveLength(4)
  })

  it('com condições ativas, nome e ações continuam na faixa', () => {
    renderBand('["abalado","agarrado","cego","atordoado"]')

    expect(screen.getByRole('heading', { name: 'Paladino Sagrado' })).toBeInTheDocument()
    expect(screen.getByText('Nv 9')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Fechar o combatente' })).toBeInTheDocument()
    expect(screen.getByLabelText('Aplicar condição')).toBeInTheDocument()
  })

  it('o nível sobrevive ao truncamento do nome', () => {
    render(() => {
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
      return (
        <QueryClientProvider client={client}>
          <ConditionalsProvider store={createConditionalsStore(new FakeStorage())}>
            <CombatantBand
              character={makeCharacter({
                name: 'Paladino Sagrado de Khalmyr, o Inquebrantável da Aliança',
                level: 12,
              })}
            />
          </ConditionalsProvider>
        </QueryClientProvider>
      )
    })

    // O nível é IRMÃO do nome, não filho: dentro do `truncate` ele era a
    // primeira coisa a sumir, e a 390px a faixa mostrava "Paladino S…" sem
    // nível — um dos quatro itens que o dono pediu.
    const nivel = screen.getByText('Nv 12')
    expect(nivel).toBeInTheDocument()
    expect(nivel.closest('h2')).toBeNull()
  })
})
