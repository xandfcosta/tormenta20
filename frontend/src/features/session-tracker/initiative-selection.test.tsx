import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { InitiativeEntry, SessionRealtime } from '@/shared/realtime/realtime'
import { InitiativeCard } from './initiative-card'

/** Named fake for the session socket — o rastreador só lê estado aqui. */
class FakeRealtime {
  // Os espiões vivem na INSTÂNCIA: criados dentro do `asRealtime()` eles nasciam
  // de novo a cada chamada, e o teste inspecionava um objeto diferente do que a
  // tela usou.
  readonly updateEntry = vi.fn()
  readonly removeEntry = vi.fn()
  readonly deltaVitals = vi.fn()

  constructor(private readonly entries: InitiativeEntry[]) {}

  asRealtime(): SessionRealtime {
    return {
      state: () => ({ initiative: this.entries, round: 1, turnIndex: 0 }) as ReturnType<
        SessionRealtime['state']
      >,
      isConnected: () => true,
      error: () => null,
      hasPersistenceWarning: () => false,
      present: () => [],
      addEntry: vi.fn(),
      updateEntry: this.updateEntry,
      removeEntry: this.removeEntry,
      nextTurn: vi.fn(),
      resetInitiative: vi.fn(),
      populateParty: vi.fn(),
      deltaVitals: this.deltaVitals,
      applyEffect: vi.fn(),
      rest: vi.fn(),
    } as unknown as SessionRealtime
  }
}

const HEROI = {
  id: 'e1',
  label: 'Paladino Sagrado',
  initiative: 14,
  type: 'character',
  characterId: 15,
  hpCurrent: 57,
  hpMax: 95,
} as unknown as InitiativeEntry

const OGRO = {
  id: 'e2',
  label: 'Ogro',
  initiative: 22,
  type: 'npc',
  hpCurrent: 130,
  hpMax: 130,
} as unknown as InitiativeEntry

function renderCard(onSelect?: (entryId: string) => void, selectedId?: string | null) {
  const { user } = renderCardWithRt(onSelect, selectedId)
  return user
}

function renderCardWithRt(
  onSelect?: (entryId: string) => void,
  selectedId?: string | null,
  isGm = true,
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const rt = new FakeRealtime([OGRO, HEROI])
  const view = render(() => (
    <QueryClientProvider client={client}>
      <InitiativeCard
        rt={rt.asRealtime()}
        isGm={isGm}
        myCharacterIds={new Set<number>()}
        onSelect={onSelect}
        selectedId={selectedId}
      />
    </QueryClientProvider>
  ))
  return { ...view, rt, user: userEvent.setup() }
}

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

describe('seleção na iniciativa', () => {
  // Alcançar a ficha de um jogador no meio de um turno era: sair da sessão →
  // aba Membros → um link de 29×16 px → voltar (ALE-122).
  it('o nome do combatente abre o combatente', async () => {
    const onSelect = vi.fn()
    const user = renderCard(onSelect)

    await user.click(screen.getByRole('button', { name: 'Paladino Sagrado' }))

    expect(onSelect).toHaveBeenCalledWith('e1')
  })

  // A regra tem de valer para a linha inteira: metade das linhas quebrando o
  // "clico no nome, abre o combatente" em silêncio é pior que não ter a regra.
  it('vale para NPC também', async () => {
    const onSelect = vi.fn()
    const user = renderCard(onSelect)

    await user.click(screen.getByRole('button', { name: 'Ogro' }))

    expect(onSelect).toHaveBeenCalledWith('e2')
  })

  it('marca quem está aberto', () => {
    renderCard(vi.fn(), 'e1')

    expect(screen.getByRole('button', { name: 'Paladino Sagrado' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })

  // A view do JOGADOR usa o mesmo rastreador e não tem painel para abrir: ali o
  // nome não pode virar um botão que não leva a lugar nenhum.
  it('sem painel, o nome não é botão', () => {
    renderCard(undefined)

    expect(screen.queryByRole('button', { name: 'Paladino Sagrado' })).not.toBeInTheDocument()
    expect(screen.getByText('Paladino Sagrado')).toBeInTheDocument()
  })
})

/**
 * "Adicionar grupo" entra com iniciativa 0 e, até aqui, o único conserto era
 * REMOVER e adicionar de novo — perdendo PV e condições no caminho. O
 * `initiative-update` existia no cliente e nunca era chamado por ninguém.
 */
describe('corrigir a iniciativa de quem já está na lista', () => {
  it('o mestre reescreve o número e o servidor recebe a correção', async () => {
    const { rt, user } = renderCardWithRt()

    await user.click(screen.getByRole('button', { name: 'Iniciativa de Ogro' }))
    const campo = screen.getByRole('spinbutton', { name: 'Iniciativa de Ogro' })
    await user.clear(campo)
    await user.type(campo, '17')
    await user.click(screen.getByRole('button', { name: 'Salvar' }))

    expect(rt.updateEntry).toHaveBeenCalledWith('e2', { initiative: 17 })
  })

  it('cancelar não manda nada', async () => {
    const { rt, user } = renderCardWithRt()

    await user.click(screen.getByRole('button', { name: 'Iniciativa de Ogro' }))
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(rt.updateEntry).not.toHaveBeenCalled()
  })

  // Para o JOGADOR o número é texto: reordenar a mesa é do mestre.
  it('o jogador não reordena a mesa', () => {
    renderCardWithRt(undefined, null, false)

    expect(screen.queryByRole('button', { name: 'Iniciativa de Ogro' })).not.toBeInTheDocument()
  })
})
