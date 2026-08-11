import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Campaign, CampaignMember, Session } from '@/shared/api/api'
import {
  CampaignTome,
  type CampaignTomeProps,
  campaignTabs,
  isCampaignTab,
  sectionLayout,
} from './campaign-tome'
import { memberName, sortRoster } from './members-card'

describe('sectionLayout', () => {
  // The nav driver reads this off the DOM to decide whether arrows move in 2-D
  // or 1-D — it has to match what the eye sees in each section.
  it('grid nas seções em grade, coluna na linha do tempo e no formulário', () => {
    expect(sectionLayout('visao')).toBe('grid')
    expect(sectionLayout('membros')).toBe('grid')
    expect(sectionLayout('sessoes')).toBe('column')
    expect(sectionLayout('config')).toBe('column')
  })
})

describe('isCampaignTab', () => {
  it('aceita as seções conhecidas', () => {
    for (const tab of campaignTabs(true)) expect(isCampaignTab(tab)).toBe(true)
  })

  // A URL é a fonte da verdade e vem do usuário: `?tab=lixo` não pode quebrar.
  it('rejeita qualquer outra coisa vinda da URL', () => {
    expect(isCampaignTab('lixo')).toBe(false)
    expect(isCampaignTab(undefined)).toBe(false)
  })
})

describe('campaignTabs', () => {
  it('o mestre percorre as quatro seções, na ordem do rail', () => {
    expect(campaignTabs(true)).toEqual(['visao', 'sessoes', 'membros', 'config'])
  })

  // Os bumpers (PgUp/PgDn) andam por esta lista: se Config entrasse nela, o
  // jogador cairia numa seção que o rail dele não tem.
  it('o jogador não tem Config no ciclo', () => {
    expect(campaignTabs(false)).toEqual(['visao', 'sessoes', 'membros'])
  })
})

function member(id: number, role: 'gm' | 'player', name?: string): CampaignMember {
  return {
    id,
    campaignId: 1,
    characterId: id * 10,
    role,
    addedAt: '2026-01-01T00:00:00.000Z',
    ...(name
      ? {
          character: {
            id: id * 10,
            ownerId: 1,
            name,
            level: 5,
            hpCurrent: 30,
            hpMax: 30,
            mpCurrent: 10,
            mpMax: 10,
            classes: [{ className: 'Guerreiro', level: 5 }],
          },
        }
      : {}),
  }
}

describe('sortRoster', () => {
  it('o mestre lidera a formação', () => {
    const roster = sortRoster([member(1, 'player'), member(2, 'gm'), member(3, 'player')])
    expect(roster.map((m) => m.role)).toEqual(['gm', 'player', 'player'])
  })

  it('mantém a ordem de chegada dentro do mesmo papel', () => {
    const roster = sortRoster([member(1, 'player'), member(2, 'player')])
    expect(roster.map((m) => m.id)).toEqual([1, 2])
  })

  it('não muta a lista da query', () => {
    const original = [member(1, 'player'), member(2, 'gm')]
    sortRoster(original)
    expect(original.map((m) => m.role)).toEqual(['player', 'gm'])
  })
})

describe('memberName', () => {
  it('usa o nome do personagem', () => {
    expect(memberName(member(1, 'player', 'Tanque Placas'))).toBe('Tanque Placas')
  })

  // O membro pode vir sem o personagem embutido (payload enxuto); mostrar
  // "undefined" na mesa seria pior que um marcador.
  it('cai num marcador quando o personagem não veio', () => {
    expect(memberName(member(4, 'player'))).toBe('Personagem 40')
  })
})

// --- render ------------------------------------------------------------------

const campaign: Campaign = {
  id: 1,
  ownerId: 1,
  name: 'Snapshot Test',
  description: 'Uma mesa-vitrine.',
  createdAt: '2026-01-01T00:00:00.000Z',
  updatedAt: '2026-01-01T00:00:00.000Z',
  role: 'gm',
}

const liveSession: Session = {
  id: 4,
  campaignId: 1,
  title: null,
  sessionNumber: 4,
  notes: null,
  status: 'active',
  startedAt: '2026-01-01T00:00:00.000Z',
  endedAt: null,
  createdAt: '2026-01-01T00:00:00.000Z',
  updatedAt: '2026-01-01T00:00:00.000Z',
}

function renderTome(overrides: Partial<CampaignTomeProps> = {}): CampaignTomeProps {
  const props: CampaignTomeProps = {
    campaign,
    campaignId: 1,
    isGm: true,
    activeSession: undefined,
    playerCount: 4,
    current: 'visao',
    onTab: vi.fn(),
    onResume: vi.fn(),
    ...overrides,
  }
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(() => (
    <QueryClientProvider client={client}>
      <CampaignTome {...props} />
    </QueryClientProvider>
  ))
  return props
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

describe('CampaignTome', () => {
  it('mostra o nome da crônica e a contagem de heróis', () => {
    renderTome()
    expect(screen.getByRole('heading', { name: 'Snapshot Test', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('4 heróis')).toBeInTheDocument()
  })

  it('singulariza um herói só', () => {
    renderTome({ playerCount: 1 })
    expect(screen.getByText('1 herói')).toBeInTheDocument()
  })

  it('sem sessão ativa, diz isso em vez de oferecer ação', () => {
    renderTome()
    expect(screen.getByText('Nenhuma sessão ativa.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Continuar a sessão/ })).not.toBeInTheDocument()
  })

  it('com sessão ao vivo, oferece continuar', () => {
    const props = renderTome({ activeSession: liveSession })
    const button = screen.getByRole('button', { name: /Continuar a sessão/ })
    expect(screen.getByText('Sessão 4 em andamento')).toBeInTheDocument()
    button.click()
    expect(props.onResume).toHaveBeenCalledOnce()
  })

  // O driver de navegação lê essas marcas do DOM; sem elas as setas não veem
  // as regiões e o teclado morre em silêncio.
  it('declara as regiões que o scene-nav usa', () => {
    renderTome()
    const rail = document.querySelector('[data-nav-region="rail"]')
    const content = document.querySelector('[data-nav-region="content"]')
    expect(document.querySelector('[data-nav-region="header"]')).toBeInTheDocument()
    expect(rail).toHaveAttribute('data-nav-edge-right', 'content')
    expect(content).toHaveAttribute('data-nav-edge-left', 'rail')
    expect(content).toHaveAttribute('data-nav-layout', 'grid')
  })

  it('o layout da região de conteúdo acompanha a seção', () => {
    renderTome({ current: 'sessoes' })
    expect(document.querySelector('[data-nav-region="content"]')).toHaveAttribute(
      'data-nav-layout',
      'column',
    )
  })

  it('escolher uma seção avisa o chamador (que escreve na URL)', () => {
    const props = renderTome()
    screen.getByRole('tab', { name: 'Membros' }).click()
    expect(props.onTab).toHaveBeenCalledWith('membros')
  })

  it('o mestre tem a seção Config no rail', () => {
    renderTome()
    expect(screen.getByRole('tab', { name: 'Config' })).toBeInTheDocument()
  })

  it('o jogador não vê Config no rail', () => {
    renderTome({ isGm: false })
    expect(screen.queryByRole('tab', { name: 'Config' })).not.toBeInTheDocument()
  })

  it('a seção Config traz o ledger e a zona de perigo', () => {
    renderTome({ current: 'config' })
    expect(screen.getByRole('heading', { name: 'Snapshot Test', level: 2 })).toBeInTheDocument()
    expect(screen.getByText('Zona de perigo')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Excluir campanha/ })).toBeInTheDocument()
  })
})
