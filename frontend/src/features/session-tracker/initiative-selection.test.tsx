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
  readonly addEntry = vi.fn()
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
      addEntry: this.addEntry,
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
  entries: InitiativeEntry[] = [OGRO, HEROI],
  turnControls?: boolean,
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const rt = new FakeRealtime(entries)
  const view = render(() => (
    <QueryClientProvider client={client}>
      <InitiativeCard
        rt={rt.asRealtime()}
        isGm={isGm}
        myCharacterIds={new Set<number>()}
        onSelect={onSelect}
        selectedId={selectedId}
        turnControls={turnControls}
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

/**
 * NPC criado à mão nascia SEM vida, e a linha dele mostrava +/−/✎ que não tinham
 * em que mexer — controles inertes, que prometem o que não fazem (ALE-122).
 */
describe('adicionar combatente à mão', () => {
  it('com PV preenchido, o combatente entra com vida cheia', async () => {
    const { rt, user } = renderCardWithRt()

    await user.type(screen.getByLabelText('Nome'), 'Capanga')
    const pv = screen.getByRole('spinbutton', { name: 'PV' })
    await user.clear(pv)
    await user.type(pv, '18')
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    expect(rt.addEntry).toHaveBeenCalledWith(
      expect.objectContaining({ label: 'Capanga', hpCurrent: 18, hpMax: 18 }),
    )
  })

  // Zero é "sem vida registrada", não "morto": um capanga anônimo não precisa de
  // PV, e uma barra 0/0 diria que ele já caiu.
  it('sem PV, a entrada vai sem vida nenhuma', async () => {
    const { rt, user } = renderCardWithRt()

    await user.type(screen.getByLabelText('Nome'), 'Figurante')
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    const enviado = rt.addEntry.mock.calls[0][0]
    expect(enviado).toMatchObject({ label: 'Figurante' })
    expect(enviado).not.toHaveProperty('hpCurrent')
    expect(enviado).not.toHaveProperty('hpMax')
  })
})


/**
 * PV de monstro é decisão do MESTRE, linha a linha (ALE-122). Saber que o ogro
 * está com 12 de 130 muda o que a mesa faz no turno seguinte — e às vezes o
 * mestre quer exatamente que saibam. O servidor apaga os números na cópia do
 * jogador; o que se prova aqui é o controle e a marca.
 */
describe('ocultar os PV de uma linha', () => {
  it('o mestre fecha o olho e o servidor recebe a marca', async () => {
    const { rt, user } = renderCardWithRt()

    await user.click(screen.getByRole('button', { name: 'Ocultar PV de Ogro' }))

    expect(rt.updateEntry).toHaveBeenCalledWith('e2', { hpHidden: true })
  })

  it('já oculto, o mesmo botão revela', async () => {
    const oculto = { ...OGRO, hpHidden: true } as unknown as InitiativeEntry
    const { rt, user } = renderCardWithRt(undefined, null, true, [oculto, HEROI])

    await user.click(screen.getByRole('button', { name: 'Revelar PV de Ogro' }))

    expect(rt.updateEntry).toHaveBeenCalledWith('e2', { hpHidden: false })
  })

  // "Sem barra" e "escondido" precisam ser coisas DIFERENTES na tela do jogador:
  // silêncio faria o jogador supor que o monstro está inteiro.
  it('o jogador vê a marca no lugar dos números', () => {
    // Como chega do servidor depois da redação: a flag fica, os PV somem.
    const redigido = { ...OGRO, hpCurrent: undefined, hpMax: undefined, hpHidden: true }
    renderCardWithRt(undefined, null, false, [redigido as unknown as InitiativeEntry, HEROI])

    expect(screen.getByText('PV oculto')).toBeInTheDocument()
    expect(screen.queryByText('130')).not.toBeInTheDocument()
  })

  // Quem não decide o que a mesa vê não pode ter o controle na linha.
  it('o jogador não tem o olho', () => {
    renderCardWithRt(undefined, null, false)

    expect(screen.queryByRole('button', { name: /(Ocultar|Revelar) PV de Ogro/ })).not.toBeInTheDocument()
  })
})


/**
 * Altura é recurso escasso: no celular deitado (844×390) a soma de cabeçalho,
 * faixa de turno e seletor de região já come metade da tela antes de a lista
 * começar. O que dava para cortar sem decisão de produto era a REPETIÇÃO — a
 * faixa de turno fixa já diz a rodada, e o card dizia de novo a poucos pixels.
 */
describe('o card não repete o que a faixa de turno já diz', () => {
  it('com a faixa na cena, a rodada não sai duas vezes', () => {
    renderCardWithRt(undefined, null, true, [OGRO, HEROI], false)

    expect(screen.queryByText(/Rodada/)).not.toBeInTheDocument()
  })

  // Sem faixa (a view do jogador usa o card sozinho), a rodada continua sendo
  // do card: cortar nos dois casos deixaria a tela sem dizer em que rodada está.
  it('sem a faixa, o card continua dizendo a rodada', () => {
    renderCardWithRt(undefined, null, false)

    expect(screen.getByText(/Rodada 1/)).toBeInTheDocument()
  })
})
