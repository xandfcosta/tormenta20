import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from '@tanstack/react-router'
import { render, screen, type RenderResult } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import type { CampaignMember, Session } from '@/shared/api/api'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { CampaignOverview } from './campaign-overview'

const CAMPAIGN_ID = 5

function fakeMember(id: number, role: CampaignMember['role']): CampaignMember {
  return {
    id,
    campaignId: CAMPAIGN_ID,
    characterId: id,
    role,
    addedAt: '',
    character: {
      id,
      ownerId: 1,
      name: `Herói ${id}`,
      level: 2,
      hpCurrent: 10,
      hpMax: 10,
      mpCurrent: 3,
      mpMax: 3,
      classes: [{ className: 'Ladino', level: 2 }],
    },
  }
}

function fakeSession(sessionNumber: number, status: Session['status']): Session {
  return {
    id: sessionNumber,
    campaignId: CAMPAIGN_ID,
    title: null,
    sessionNumber,
    notes: null,
    status,
    startedAt: null,
    endedAt: null,
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '',
  }
}

function renderOverview(
  members: CampaignMember[],
  sessions: Session[],
  onGoToTab: (tab: 'sessoes' | 'membros') => void,
): RenderResult {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  qc.setQueryData(campaignMembersQueryOptions(CAMPAIGN_ID).queryKey, members)
  qc.setQueryData(campaignSessionsQueryOptions(CAMPAIGN_ID).queryKey, sessions)
  const node: ReactNode = (
    <CampaignOverview campaignId={CAMPAIGN_ID} isGm={false} onGoToTab={onGoToTab} />
  )
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

describe('CampaignOverview — dashboard', () => {
  it('counts heroes (players) and links into the party section', async () => {
    const goToTab = vi.fn()
    renderOverview(
      [fakeMember(1, 'gm'), fakeMember(2, 'player'), fakeMember(3, 'player')],
      [fakeSession(1, 'ended')],
      goToTab,
    )
    // 2 players → "Heróis" sigil reads 2; the muster link counts all 3 members.
    expect(await screen.findByText('Heróis')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /Ver 3 membros/ }))
    expect(goToTab).toHaveBeenCalledWith('membros')
  })

  it('links the recent chronicle to the full log', async () => {
    const goToTab = vi.fn()
    renderOverview([], [fakeSession(1, 'active')], goToTab)
    await userEvent.click(await screen.findByRole('button', { name: /Ver todas/ }))
    expect(goToTab).toHaveBeenCalledWith('sessoes')
  })
})
