import { render, screen } from '@solidjs/testing-library'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import type { CampaignMember } from '@/shared/api/types'
import type { InitiativeEntry, PresenceUser } from '@/shared/realtime/realtime'
import { SessionRail } from './session-rail'

/**
 * O TRILHO ESQUERDO com os dois blocos (ALE-211).
 *
 * O que este arquivo prova é a COMPOSIÇÃO: que os dois blocos convivem, que a
 * fila diz quem já agiu e que o elenco diz quem está na mesa. A conta de
 * quantos vizinhos cabem é regra e se prova em `rail-geometry.test.ts`.
 *
 * O RECORTE não se prova aqui, e isso é da natureza do jsdom: todo elemento
 * mede zero, então `cabemNaFila(0)` responde zero e a fila desenha todo mundo.
 * É a mesma cegueira que a ALE-139 documentou. Quem testemunha o recorte é o
 * e2e (`session.spec.ts`), onde a altura é real.
 */

const linha = (id: string, label: string): InitiativeEntry =>
  ({ id, label, initiative: 10, type: 'npc' }) as InitiativeEntry

const FILA = [linha('a', 'Arcanista'), linha('b', 'Guerreiro'), linha('c', 'Ogro')]

const membro = (characterId: number, name: string, ownerId: number): CampaignMember =>
  ({
    id: characterId,
    campaignId: 1,
    characterId,
    role: 'player',
    addedAt: '2026-01-01T00:00:00.000Z',
    character: {
      id: characterId,
      ownerId,
      name,
      level: 3,
      hpCurrent: 20,
      hpMax: 20,
      mpCurrent: 5,
      mpMax: 5,
      classes: [],
    },
  }) as CampaignMember

function renderRail(over: { turnIndex?: number; present?: PresenceUser[] } = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(campaignMembersQueryOptions(1).queryKey, [
    membro(12, 'Arcanista Erudito', 7),
    membro(15, 'Paladino Sagrado', 8),
  ])
  const onOpenCharacter = vi.fn()
  const onOpenQueue = vi.fn()
  const onOpenCast = vi.fn()
  render(() => (
    <QueryClientProvider client={client}>
      <SessionRail
        campaignId={1}
        entries={FILA}
        turnIndex={over.turnIndex ?? 1}
        activeEntryId={FILA[over.turnIndex ?? 1]?.id ?? null}
        present={over.present ?? []}
        connected
        sceneActive
        onOpenCombatant={vi.fn()}
        onOpenQueue={onOpenQueue}
        onOpenCast={onOpenCast}
        onOpenCharacter={onOpenCharacter}
        onEndScene={vi.fn()}
        onResetScene={vi.fn()}
      />
    </QueryClientProvider>
  ))
  return { onOpenCharacter, onOpenQueue, onOpenCast }
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

describe('o trilho esquerdo', () => {
  it('hospeda os DOIS blocos, cada um com seu jeito de abrir', () => {
    renderRail()

    expect(screen.getByRole('navigation', { name: 'Fila do combate' })).toBeInTheDocument()
    expect(screen.getByRole('navigation', { name: 'Elenco' })).toBeInTheDocument()
  })

  /**
   * Escurecer é o sinal para quem enxerga; o nome acessível é o sinal para quem
   * não. Sem ele, dois itens da fila soam idênticos e a informação que o bloco
   * existe para dar — quem já jogou — não chega.
   */
  it('a fila diz quem JÁ AGIU, e não só com tinta', () => {
    renderRail({ turnIndex: 1 })

    expect(screen.getByRole('button', { name: /^Abrir Arcanista.* — já agiu$/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^Abrir Guerreiro.* — na vez$/ })).toBeInTheDocument()
    // Quem ainda vai não ganha sufixo: o silêncio aqui É a informação.
    expect(screen.getByRole('button', { name: 'Abrir Ogro' })).toBeInTheDocument()
  })

  // Fora de combate ninguém agiu, e marcar alguém como tal seria inventar um
  // passado. É o estado em que o mestre acabou de iniciar a cena (ALE-210).
  it('sem vez de ninguém, ninguém já agiu', () => {
    renderRail({ turnIndex: -1 })

    expect(screen.queryByRole('button', { name: /já agiu/ })).not.toBeInTheDocument()
  })

  it('o elenco diz quem está na mesa agora', () => {
    renderRail({ present: [{ userId: 7, name: 'Alex', role: 'player' }] })

    expect(
      screen.getByRole('button', { name: 'Abrir a ficha de Arcanista Erudito — na mesa' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Abrir a ficha de Paladino Sagrado — fora da mesa' }),
    ).toBeInTheDocument()
  })

  it('cada bloco abre a gaveta dele', async () => {
    const { onOpenQueue, onOpenCast } = renderRail()

    screen.getByRole('button', { name: 'Abrir a iniciativa' }).click()
    screen.getByRole('button', { name: 'Abrir o elenco' }).click()

    expect(onOpenQueue).toHaveBeenCalledTimes(1)
    expect(onOpenCast).toHaveBeenCalledTimes(1)
  })
})
