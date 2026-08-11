import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import type { CampaignMember } from '@/shared/api/api'
import { renderWithRouter } from '@/shared/test/render-with-router'
import { MembersCard, MemberPlate } from './members-card'

const CAMPAIGN_ID = 7

function fakeMember(overrides: Partial<CampaignMember> = {}): CampaignMember {
  return {
    id: 1,
    campaignId: CAMPAIGN_ID,
    characterId: 1,
    role: 'player',
    addedAt: '2026-01-01T00:00:00.000Z',
    character: {
      id: 1,
      ownerId: 1,
      name: 'Alvo',
      level: 3,
      hpCurrent: 20,
      hpMax: 20,
      mpCurrent: 6,
      mpMax: 6,
      classes: [{ className: 'Guerreiro', level: 3 }],
    },
    ...overrides,
  }
}

/** Seeds the members cache so the roster paints without a network round-trip. */
function renderRoster(members: CampaignMember[], isGm: boolean) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Number.POSITIVE_INFINITY } },
  })
  client.setQueryData(campaignMembersQueryOptions(CAMPAIGN_ID).queryKey, members)
  renderWithRouter(() => (
    <QueryClientProvider client={client}>
      <MembersCard campaignId={CAMPAIGN_ID} isGm={isGm} />
    </QueryClientProvider>
  ))
}

beforeEach(() => {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: false,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
})

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('MembersCard — escrita do mestre', () => {
  it('mostra remover e convite para o mestre', async () => {
    renderRoster([fakeMember()], true)

    expect(await screen.findByLabelText('Remover Alvo')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Convite/ })).toBeInTheDocument()
  })

  // A trava de verdade é do servidor; isto é só UX — mas oferecer ao jogador um
  // botão que sempre falha é pior que não oferecer.
  it('esconde remover e convite do jogador', async () => {
    renderRoster([fakeMember()], false)

    expect(await screen.findByText('Alvo')).toBeInTheDocument()
    expect(screen.queryByLabelText('Remover Alvo')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Convite/ })).not.toBeInTheDocument()
  })
})

describe('MemberPlate', () => {
  it('remove o membro no clique', async () => {
    const onRemove = vi.fn().mockResolvedValue(undefined)
    renderWithRouter(() => (
      <MemberPlate member={fakeMember()} canRemove onRemove={onRemove} />
    ))

    await userEvent.setup().click(await screen.findByLabelText('Remover Alvo'))

    expect(onRemove).toHaveBeenCalledOnce()
  })

  // O cursor de setas anda pelo grid de heróis; parar em "remover" no caminho
  // seria um passo a um Enter de distância de perder alguém da mesa.
  it('deixa o botão de remover fora do cursor de setas', async () => {
    renderWithRouter(() => (
      <MemberPlate member={fakeMember()} canRemove onRemove={vi.fn()} />
    ))

    expect(await screen.findByLabelText('Remover Alvo')).toHaveAttribute('data-nav-skip')
  })
})
