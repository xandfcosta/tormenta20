import { render, screen } from '@solidjs/testing-library'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { campaignCreaturesQueryOptions } from '@/entities/creature/queries'
import type { CampaignMember } from '@/shared/api/types'
import type { CampaignCreature } from '@/shared/api/creature-types'
import { blankCreatureBlock } from '@/shared/api/creature-types'
import type { PresenceUser } from '@/shared/realtime/realtime'
import { CastPanel } from './cast-panel'

/**
 * O ELENCO (ALE-212).
 *
 * Iniciativa não é lista de combatentes: o elenco é onde o taverneiro que não
 * briga, o chefe da semana que vem e a ficha de um jogador fora do combate
 * existem. O que este teste protege é a composição — quem entra em cada lista,
 * quem acende como presente, e que criar um NPC aqui NÃO põe ninguém na fila.
 */

const membro = (
  characterId: number,
  name: string,
  ownerId: number,
  role: 'player' | 'gm' = 'player',
): CampaignMember =>
  ({
    id: characterId,
    campaignId: 1,
    characterId,
    role,
    addedAt: '2026-01-01T00:00:00.000Z',
    character: { id: characterId, ownerId, name, level: 3, hpCurrent: 20, hpMax: 20, mpCurrent: 5, mpMax: 5, classes: [{ className: 'Bardo', level: 3 }] },
  }) as CampaignMember

const criatura = (id: number, name: string, hp: number): CampaignCreature =>
  ({ id, campaignId: 1, name, block: { ...blankCreatureBlock(), hp } }) as CampaignCreature

const MEMBROS = [
  membro(12, 'Arcanista Erudito', 7),
  membro(15, 'Paladino Sagrado', 8),
  // O PC do próprio mestre: está no roster, mas com papel de GM.
  membro(99, 'Tanque Placas', 1, 'gm'),
]

function renderCast(present: PresenceUser[] = [], creatures = [criatura(3, 'Taverneiro Gordo', 30)]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(campaignMembersQueryOptions(1).queryKey, MEMBROS)
  client.setQueryData(campaignCreaturesQueryOptions(1).queryKey, creatures)
  const onOpenCharacter = vi.fn()
  render(() => (
    <QueryClientProvider client={client}>
      <CastPanel campaignId={1} present={present} onOpenCharacter={onOpenCharacter} />
    </QueryClientProvider>
  ))
  return { onOpenCharacter, user: userEvent.setup() }
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

describe('o elenco', () => {
  it('lista os PCs e os NPCs do mestre em seções separadas', () => {
    renderCast()

    expect(screen.getByText('Jogadores · 2')).toBeInTheDocument()
    expect(screen.getByText('Meus NPCs · 1')).toBeInTheDocument()
    expect(screen.getByText('Taverneiro Gordo')).toBeInTheDocument()
  })

  /**
   * O PC do mestre está no roster com papel de GM, e é EXATAMENTE o filtro que
   * o "Adicionar grupo" usa no servidor (`listPlayerCombatants` pula quem não é
   * `player`). Listá-lo sob "Jogadores" criaria uma linha que o botão de trazer
   * o grupo nunca traz — duas telas discordando sobre quem é o grupo.
   */
  it('o PC do próprio mestre não entra em Jogadores', () => {
    renderCast()

    expect(screen.queryByText('Tanque Placas')).not.toBeInTheDocument()
  })

  // A presença chega por USUÁRIO e o elenco lista PERSONAGEM; a ponte é o
  // `ownerId` do roster. O ponto é invisível para leitor de tela, então quem
  // carrega a resposta é o nome acessível.
  it('diz quem está na mesa agora', () => {
    renderCast([{ userId: 7, name: 'Alex', role: 'player' }])

    expect(
      screen.getByRole('button', { name: 'Abrir a ficha de Arcanista Erudito — na mesa' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Abrir a ficha de Paladino Sagrado — fora da mesa' }),
    ).toBeInTheDocument()
  })

  it('clicar num jogador pede a ficha dele', async () => {
    const { onOpenCharacter, user } = renderCast()

    await user.click(screen.getByRole('button', { name: /^Abrir a ficha de Arcanista Erudito/ }))

    expect(onOpenCharacter).toHaveBeenCalledWith(12)
  })

  /**
   * A razão de a issue existir, do lado do NPC: até aqui o bloco só nascia a
   * partir do "+ Combatente completo", ou seja, o mestre era obrigado a pôr o
   * vilão na FILA para poder escrevê-lo. Criar daqui não põe ninguém na mesa —
   * quem faz isso é o mestre, quando ele quiser (ALE-211).
   */
  it('criar um NPC abre o bloco em branco, sem tocar na iniciativa', async () => {
    const { user } = renderCast()

    await user.click(screen.getByRole('button', { name: 'Criar NPC' }))

    // O diálogo do bloco é o mesmo que a ALE-137 já tinha; o que muda é de onde
    // ele é chamado. Nome vazio = criação, não edição.
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    expect(screen.getByLabelText('Nome')).toHaveValue('')
  })

  it('apagar um NPC pergunta antes', async () => {
    const { user } = renderCast()

    await user.click(screen.getByRole('button', { name: 'Apagar Taverneiro Gordo' }))

    expect(await screen.findByRole('dialog')).toHaveTextContent('Apagar "Taverneiro Gordo"?')
  })

  it('sem NPC escrito, a lista diz o que ela é', () => {
    renderCast([], [])

    expect(screen.getByText(/só você vê/)).toBeInTheDocument()
  })
})
