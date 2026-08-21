import { render, screen } from '@solidjs/testing-library'
import { FakeStorage } from '@/shared/test/fake-storage'
import { describe, expect, it } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import type { Character } from '@/shared/api/api'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { createConditionalsStore } from '@/shared/stores/conditionals-store'
import { EffectsCountBadge } from './effects-count-badge'


function renderBadge(char: Character) {
  render(() => (
    <ConditionalsProvider store={createConditionalsStore(new FakeStorage())}>
      <EffectsCountBadge character={char} />
    </ConditionalsProvider>
  ))
}

describe('EffectsCountBadge', () => {
  it('conta o que a aba mostra', () => {
    renderBadge(makeCharacter({ activeConditions: '["caido","cego"]' }))
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  // Regressão: o rótulo morava num `aria-label` de <span>, que não tem papel
  // para carregá-lo — o leitor de tela anunciava só "2".
  // O plural concorda, e o teste antigo afirmava "1 efeitos ativos" — ele tinha
  // congelado o defeito. Só ficou visível quando o número passou a ser lido
  // junto do rótulo (ALE-173, P6).
  it('anuncia o que o número significa, com o plural certo', () => {
    renderBadge(makeCharacter({ activeConditions: '["caido"]' }))
    expect(screen.getByText('1 efeito ativo')).toBeInTheDocument()
    expect(screen.getByText('1')).toHaveAttribute('aria-hidden', 'true')
  })

  // Zero com condicional disponível é informação: a aba TEM o que oferecer e
  // nada está ligado. É por isso que a pílula fica, em tom apagado.
  it('mostra 0 apagado quando há o que ligar e nada ligado', () => {
    renderBadge(makeCharacter({ classes: [{ className: 'Bardo', level: 3 }] }))
    expect(screen.getByText('0')).toHaveClass('text-muted-foreground')
  })

  it('sem nada a oferecer, não pinta pílula', () => {
    renderBadge(makeCharacter({ classes: [{ className: 'Guerreiro', level: 3 }] }))
    expect(screen.queryByText('0')).not.toBeInTheDocument()
  })
})
