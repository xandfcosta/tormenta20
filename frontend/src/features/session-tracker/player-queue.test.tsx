import { render, screen } from '@solidjs/testing-library'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import type { InitiativeEntry, SessionRealtime } from '@/shared/realtime/realtime'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { InitiativeCard } from './initiative-card'
import { fakeConditionals } from '@/shared/test/play-stores'

/**
 * A FILA DO JOGADOR É LEITURA (ALE-213).
 *
 * "Ele pode ver a ordem, e só." A tela nunca ofereceu ação sobre linha ALHEIA —
 * e o servidor recusa (`assertVitalsEditable`, provado em `realtime_authz_test.go`).
 * O que existia era ação sobre a linha DELE, e ela saiu: o PV do personagem já
 * se mexe na ficha, que é a superfície ao lado, e duas portas para o mesmo
 * número faziam a Mesa parecer um painel de mestre menor.
 *
 * Este arquivo existe porque o caso nunca tinha sido montado. O
 * `initiative-selection.test.tsx` passa `myCharacterIds` VAZIO em todos os
 * casos, então "o jogador com um personagem na fila" — o único estado em que a
 * coluna de ações aparecia para ele — não era exercido por teste nenhum, e a
 * remoção dela passou verde sem ninguém notar.
 */

const MEU = {
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
  hpCurrent: 40,
  hpMax: 60,
} as unknown as InitiativeEntry

/** Os verbos da coluna de ações, pelo nome acessível que cada um anuncia. */
const VERBOS = [
  /^Curar /,
  /^Ferir /,
  /^Editar PV/,
  /^Ocultar PV/,
  /^Revelar PV/,
  /^Remover /,
] as const

class FakeRealtime {
  readonly deltaVitals = vi.fn()
  readonly removeEntry = vi.fn()
  readonly rollSelfInitiative = vi.fn()

  asRealtime(): SessionRealtime {
    return {
      state: () => ({
        initiative: [OGRO, MEU],
        round: 1,
        turnIndex: 0,
        sceneActive: true,
      }),
      isConnected: () => true,
      error: () => null,
      hasPersistenceWarning: () => false,
      present: () => [],
      deltaVitals: this.deltaVitals,
      removeEntry: this.removeEntry,
      rollSelfInitiative: this.rollSelfInitiative,
      updateEntry: vi.fn(),
      addEntry: vi.fn(),
      nextTurn: vi.fn(),
      resetInitiative: vi.fn(),
      populateParty: vi.fn(),
      applyEffect: vi.fn(),
      rest: vi.fn(),
    } as unknown as SessionRealtime
  }
}

function renderQueue(isGm: boolean) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  // A ficha entra no cache para o botão de registrar iniciativa montar de
  // verdade em vez de ficar no esqueleto — senão o teste do jogador mediria uma
  // tela que ainda não terminou de carregar.
  client.setQueryData(
    characterQueryOptions(15).queryKey,
    makeCharacter({
      id: 15,
      name: 'Paladino Sagrado',
      // A perícia com um bônus NÃO-ZERO é o que dá dentes ao teste do d20.
      // A ficha padrão vem sem perícia nenhuma, o bônus cai em zero, e aí total
      // e d20 são o mesmo número — o teste passava verde com a tela somando o
      // total antes de mandar, que é exatamente o defeito que ele mira.
      // Nível 3 + destreza 2 do fixture, e o motor faz a conta.
      expertises: [{ name: 'Iniciativa', attribute: 'dexterity', trained: false, custom: false }],
    }),
  )
  const rt = new FakeRealtime()
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={fakeConditionals()}>
        <InitiativeCard rt={rt.asRealtime()} isGm={isGm} myCharacterIds={new Set([15])} />
      </ConditionalsProvider>
    </QueryClientProvider>
  ))
  return { rt, user: userEvent.setup() }
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

describe('a fila do jogador', () => {
  it('não oferece ação nenhuma, nem sobre o personagem DELE', () => {
    renderQueue(false)

    // Com `myCharacterIds` contendo o 15, esta é exatamente a linha que oferecia
    // curar, ferir e ajustar antes desta issue.
    for (const verbo of VERBOS) {
      expect(
        screen.queryByRole('button', { name: verbo }),
        `${verbo} continua na fila do jogador`,
      ).not.toBeInTheDocument()
    }
  })

  it('mas continua mostrando a ORDEM e os vitais', () => {
    renderQueue(false)

    // A metade que importa: tirar a coluna não pode ter tirado a fila junto.
    expect(screen.getByText('Ogro')).toBeInTheDocument()
    expect(screen.getByText('Paladino Sagrado')).toBeInTheDocument()
    // Os NÚMEROS continuam: o jogador vê quanto cada um aguenta, que é metade
    // do que a fila serve para responder. Tirar a coluna de ações não podia
    // levar a barra junto.
    expect(screen.getByRole('progressbar', { name: 'PV 57 de 95' })).toBeInTheDocument()
    expect(screen.getByRole('progressbar', { name: 'PV 40 de 60' })).toBeInTheDocument()
  })

  it('o mestre continua com a fileira inteira', () => {
    renderQueue(true)

    // Sem esta metade, um cartão quebrado passaria verde nas duas de cima.
    for (const verbo of [/^Curar /, /^Ferir /, /^Remover /]) {
      // Uma por LINHA: a coluna do mestre vale para as duas.
      expect(screen.getAllByRole('button', { name: verbo })).toHaveLength(2)
    }
  })
})

describe('registrar a própria iniciativa', () => {
  it('manda o D20 para o servidor, nunca o total', async () => {
    const { rt, user } = renderQueue(false)

    await user.click(screen.getByRole('button', { name: 'Registrar iniciativa' }))
    const campo = await screen.findByLabelText('Seu d20')
    await user.clear(campo)
    await user.type(campo, '14')
    // O total PREVISTO aparece, que é o que a issue pede — e é também o que
    // impede este teste de ficar vácuo: com um bônus zero, 14 e o total seriam
    // o mesmo número e a asserção de baixo não distinguiria nada.
    expect(screen.getByText('17')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Registrar' }))

    // 14 CRU. Quem soma o bônus da perícia é o Go: um total pronto saindo daqui
    // seria a tela decidindo regra do livro, livre para divergir do motor.
    expect(rt.rollSelfInitiative).toHaveBeenCalledWith(15, 14)
  })

  // O campo é digitado e o `NumberInput` não apara o que se digita. O servidor
  // recusa fora de 1..20 — e o cliente não escuta o `exception` do socket, então
  // a recusa sumiria em silêncio: o jogador clicaria e a tela não mudaria.
  it('com o d20 fora da faixa, Registrar fica travado e a tela diz por quê', async () => {
    const { rt, user } = renderQueue(false)

    await user.click(screen.getByRole('button', { name: 'Registrar iniciativa' }))
    const campo = await screen.findByLabelText('Seu d20')
    await user.clear(campo)
    await user.type(campo, '133')

    expect(screen.getByText('O d20 vai de 1 a 20.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Registrar' })).toBeDisabled()
    expect(rt.rollSelfInitiative).not.toHaveBeenCalled()
  })

  it('cancelar não manda nada', async () => {
    const { rt, user } = renderQueue(false)

    await user.click(screen.getByRole('button', { name: 'Registrar iniciativa' }))
    await screen.findByLabelText('Seu d20')
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(rt.rollSelfInitiative).not.toHaveBeenCalled()
  })
})
