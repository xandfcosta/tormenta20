import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from '@tanstack/react-router'
import { render, screen, type RenderResult } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it } from 'vitest'
import type { CampaignMember } from '@/shared/api/api'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { MembersCard } from './members-card'

const CAMPAIGN_ID = 7

function fakeMember(overrides: Partial<CampaignMember> = {}): CampaignMember {
  const name = overrides.character?.name ?? 'Herói'
  return {
    id: overrides.id ?? 1,
    campaignId: CAMPAIGN_ID,
    characterId: overrides.characterId ?? 1,
    role: overrides.role ?? 'player',
    addedAt: '',
    character: {
      id: 1,
      ownerId: 1,
      name,
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

// Seeds the members cache and mounts inside a router (the plates are <Link>s).
function renderRoster(members: CampaignMember[], isGm: boolean): RenderResult {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  qc.setQueryData(campaignMembersQueryOptions(CAMPAIGN_ID).queryKey, members)
  const node: ReactNode = <MembersCard campaignId={CAMPAIGN_ID} isGm={isGm} />
  const rootRoute = createRootRoute({ component: () => <>{node}</> })
  const router = createRouter({
    routeTree: rootRoute,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  return render(
    <QueryClientProvider client={qc}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
}

describe('MembersCard — party roster', () => {
  it('lists every hero with class/level and crowns the GM', async () => {
    renderRoster(
      [
        fakeMember({
          id: 1,
          characterId: 1,
          role: 'player',
          character: {
            ...fakeMember().character!,
            name: 'Aventureiro',
            classes: [{ className: 'Bárbaro', level: 4 }],
          },
        }),
        fakeMember({ id: 2, characterId: 2, role: 'gm', character: { ...fakeMember().character!, id: 2, name: 'Mestra' } }),
      ],
      false,
    )
    expect(await screen.findByText('Aventureiro')).toBeInTheDocument()
    expect(screen.getByText('Mestra')).toBeInTheDocument()
    expect(screen.getByText('Mestre')).toBeInTheDocument()
    expect(screen.getByText('Jogador')).toBeInTheDocument()
    expect(screen.getByText('Bárbaro 4')).toBeInTheDocument()
  })

  it('shows the remove control + invite only for the GM', async () => {
    renderRoster([fakeMember({ character: { ...fakeMember().character!, name: 'Alvo' } })], true)
    expect(await screen.findByLabelText('Remover Alvo')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Convite/ })).toBeInTheDocument()
  })

  it('hides the remove control for a player', async () => {
    renderRoster([fakeMember({ character: { ...fakeMember().character!, name: 'Alvo' } })], false)
    await screen.findByText('Alvo')
    expect(screen.queryByLabelText('Remover Alvo')).not.toBeInTheDocument()
  })

  it('renders an empty muster when there are no members', async () => {
    renderRoster([], false)
    expect(
      await screen.findByText('Nenhum personagem inscrito ainda.'),
    ).toBeInTheDocument()
  })
})
