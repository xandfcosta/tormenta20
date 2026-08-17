import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import { describe, expect, it } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { createConditionalsStore } from '@/shared/stores/conditionals-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { CombatantBand } from './combatant-band'

function renderBand() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={createConditionalsStore(new FakeStorage())}>
        <CombatantBand
          character={makeCharacter({ name: 'Paladino Sagrado', level: 9 })}
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
