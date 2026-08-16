import { render, screen } from '@solidjs/testing-library'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { AccountInvite } from '@/shared/api/api'
import { OpenInvitesPanel } from './open-invites-panel'

/**
 * O prazo que o admin LÊ tem de bater com o que ele acabou de prometer ao
 * jogador. O bug que motivou o arredondamento — um convite recém-criado, com
 * 7 dias menos alguns segundos, anunciando "6 dias" — estava contado no
 * docstring da produção e não tinha teste nenhum: a próxima pessoa que
 * "simplificasse" para `Math.floor` traria o defeito de volta em silêncio.
 */
const AGORA = new Date('2026-08-16T12:00:00.000Z')

function invite(expiresAt: string, token = 'abcdef123456'): AccountInvite {
  return { token, expiresAt }
}

function renderPanel(...invites: AccountInvite[]) {
  vi.setSystemTime(AGORA)
  return render(() => <OpenInvitesPanel invites={invites} onCopy={vi.fn()} />)
}

afterEach(() => vi.useRealTimers())

describe('OpenInvitesPanel — o prazo do convite', () => {
  it('sete dias menos alguns segundos ainda são "7 dias", não 6', () => {
    vi.useFakeTimers()
    const quaseSete = new Date(AGORA.getTime() + 7 * 86_400_000 - 3_000).toISOString()

    renderPanel(invite(quaseSete))

    expect(screen.getByText(/Expira em 7 dias\./)).toBeInTheDocument()
  })

  it('singular quando falta um dia', () => {
    vi.useFakeTimers()
    renderPanel(invite(new Date(AGORA.getTime() + 86_400_000).toISOString()))

    expect(screen.getByText(/Expira em 1 dia\./)).toBeInTheDocument()
  })

  // Abaixo de um dia o admin precisa da escala de HORAS: "0 dias" não diz se dá
  // tempo de mandar a mensagem.
  it('menos de um dia vira horas', () => {
    vi.useFakeTimers()
    renderPanel(invite(new Date(AGORA.getTime() + 5 * 3_600_000).toISOString()))

    expect(screen.getByText(/Expira em 5 horas\./)).toBeInTheDocument()
  })

  it('prestes a vencer não vira "0 horas"', () => {
    vi.useFakeTimers()
    renderPanel(invite(new Date(AGORA.getTime() + 60_000).toISOString()))

    expect(screen.getByText(/Expira em 1 hora\./)).toBeInTheDocument()
  })
})
