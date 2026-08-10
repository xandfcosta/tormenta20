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
import type { Session } from '@/shared/api/api'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { SessionsCard } from './sessions-card'

const CAMPAIGN_ID = 3

function fakeSession(overrides: Partial<Session> = {}): Session {
  return {
    id: overrides.sessionNumber ?? 1,
    campaignId: CAMPAIGN_ID,
    title: null,
    sessionNumber: 1,
    notes: null,
    status: 'planned',
    startedAt: null,
    endedAt: null,
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '',
    ...overrides,
  }
}

// Seeds the sessions cache and mounts inside a router (entries are <Link>s).
function renderLog(sessions: Session[], isGm: boolean): RenderResult {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  qc.setQueryData(campaignSessionsQueryOptions(CAMPAIGN_ID).queryKey, sessions)
  const node: ReactNode = <SessionsCard campaignId={CAMPAIGN_ID} isGm={isGm} />
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

describe('SessionsCard — chronicle log', () => {
  it('marks the live session and offers Entrar', async () => {
    renderLog(
      [
        fakeSession({ sessionNumber: 1, status: 'ended' }),
        fakeSession({ sessionNumber: 2, status: 'active' }),
      ],
      false,
    )
    expect(await screen.findByText('Ao vivo')).toBeInTheDocument()
    expect(screen.getByText('Encerrada')).toBeInTheDocument()
    expect(screen.getByText('Entrar')).toBeInTheDocument()
  })

  it('shows the new-session action only for the GM', async () => {
    renderLog([fakeSession({ sessionNumber: 1, status: 'planned' })], true)
    expect(
      await screen.findByRole('button', { name: /Sessão 2/ }),
    ).toBeInTheDocument()
  })

  it('renders the empty log when there are no sessions', async () => {
    renderLog([], false)
    expect(
      await screen.findByText(/A crônica ainda não tem sessões/),
    ).toBeInTheDocument()
  })
})
