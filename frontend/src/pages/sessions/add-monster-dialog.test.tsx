import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { createSignal } from 'solid-js'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Monster } from '@/shared/api/catalog-types'
import { AddMonsterDialog, type EntradaAjustada } from './add-monster-dialog'

/**
 * O diálogo que LÊ e AJUSTA a criatura antes de ela entrar (ALE-208).
 *
 * O que este arquivo protege é a TRADUÇÃO entre o que o mestre ajusta e o que
 * sai para a mesa — é ali que um erro põe um ogro de 45 PV onde ele quis 20, ou
 * três ogros com iniciativas diferentes.
 *
 * O clique no card da LISTA não é testado aqui: a `VirtualList` mede zero em
 * jsdom e nenhuma linha renderiza, então um teste desses passaria verde sobre
 * qualquer coisa. Ele está no e2e (`session.spec.ts`), que é onde há leiaute.
 */

// A forma do verbete é a do catálogo, copiada do `monster-detail.test.tsx` —
// um verbete PARCIAL rebenta longe da causa, dentro do bloco de estatísticas.
const OGRO = {
  id: 'ogro',
  name: 'Ogro',
  nd: 2,
  tipo: 'humanoide',
  size: 'grande',
  hp: 45,
  defesa: 16,
  forca: 4,
  destreza: 1,
  constituicao: 3,
  inteligencia: -2,
  sabedoria: 0,
  carisma: -1,
  fortitude: 8,
  reflexos: 3,
  vontade: 1,
  deslocamento: '9m',
  attacks: [{ name: 'Clava', attackBonus: 8, damage: '1d10+6' }],
  specialAbilities: ['Faro.'],
  iniciativa: 1,
  percepcao: 2,
  skills: [],
  equipamento: 'Clava',
  tesouro: 'Padrão',
  bookPage: 248,
} as unknown as Monster

const GOBLIN = { ...OGRO, id: 'goblin', name: 'Goblin', hp: 9 } as unknown as Monster

function renderDialog(inicial: Monster = OGRO) {
  const adicionados: { monster: Monster; ajuste: EntradaAjustada }[] = []
  // Sinal DE VERDADE, e não um valor fixo: trocar a criatura sem desmontar é o
  // que exercita o `keyed` do `Show`. Com um valor fixo o teste da troca
  // remontaria a árvore e passaria verde sobre o bug que ele existe para pegar.
  const [aberta, setAberta] = createSignal<Monster | null>(inicial)
  render(() => (
    <AddMonsterDialog
      monster={aberta()}
      onAdd={(m, ajuste) => adicionados.push({ monster: m, ajuste })}
      onClose={() => setAberta(null)}
    />
  ))
  return { adicionados, user: userEvent.setup(), abre: setAberta }
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
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('abrir a criatura', () => {
  it('mostra o bloco do livro e nasce com os números dele', async () => {
    renderDialog()

    expect(await screen.findByRole('dialog')).toHaveTextContent('Ogro')
    expect(screen.getByLabelText('PV')).toHaveValue(45)
    expect(screen.getByLabelText('Quantas')).toHaveValue(1)
  })

  it('o que o mestre ajusta é o que sai', async () => {
    const { adicionados, user } = renderDialog()

    const pv = await screen.findByLabelText('PV')
    await user.clear(pv)
    await user.type(pv, '20')
    const iniciativa = screen.getByLabelText('Iniciativa')
    await user.clear(iniciativa)
    await user.type(iniciativa, '17')
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    await waitFor(() => expect(adicionados).toHaveLength(1))
    expect(adicionados[0].ajuste).toEqual({ hp: 20, initiative: 17, quantidade: 1 })
  })

  // A iniciativa nasce ROLADA para quem não jogou o dado na mesa, e o botão
  // rola no MESMO campo — quem já jogou digita por cima.
  it('a iniciativa nasce rolada, entre 1 e 20', async () => {
    renderDialog()

    const valor = (await screen.findByLabelText('Iniciativa')) as HTMLInputElement
    const rolado = Number(valor.value)
    expect(rolado).toBeGreaterThanOrEqual(1)
    expect(rolado).toBeLessThanOrEqual(20)
  })

  // O `max` do `NumberInput` governa o SPINNER; DIGITAR passa direto por ele
  // (ALE-236), e quem prende é este diálogo. Sem isso, teclar `99` mandava 99
  // criaturas para a fila de uma vez — uma chamada de rede por cópia.
  it('digitar acima do teto de cópias não passa', async () => {
    const { adicionados, user } = renderDialog()

    const quantas = await screen.findByLabelText('Quantas')
    await user.type(quantas, '{backspace}99')
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))

    await waitFor(() => expect(adicionados).toHaveLength(1))
    expect(adicionados[0].ajuste.quantidade).toBe(12)
  })

  it('cancelar não manda nada', async () => {
    const { adicionados, user } = renderDialog()

    await user.click(await screen.findByRole('button', { name: 'Cancelar' }))

    expect(adicionados).toHaveLength(0)
  })

  // Com mais de uma, a tela DIZ que elas entram juntas e que o servidor numera
  // — senão o mestre não tem como saber o que vai aparecer na fila.
  it('com mais de uma, explica que entram juntas e numeradas', async () => {
    const { user } = renderDialog()

    const quantas = await screen.findByLabelText('Quantas')
    // `{backspace}` e não `clear()`: `input[type=number]` não tem API de
    // seleção (ALE-236). Este teste passava com `clear()` porque o campo
    // emitia `Number('')` === 0 e o dígito colava num "0" — hoje o campo
    // esvazia de verdade, e o gesto de verdade é apagar e digitar.
    await user.type(quantas, '{backspace}3')

    expect(await screen.findByText(/As 3 entram juntas, com a mesma iniciativa/)).toBeInTheDocument()
    expect(screen.getByText(/Ogro, Ogro 2/)).toBeInTheDocument()
  })
})

/**
 * O diálogo é o DONO do rascunho: trocar de criatura recomeça dos números do
 * livro. Sem isto o PV que o mestre baixou num ogro reaparece no goblin, e ele
 * não tem como saber que carregou.
 *
 * Em Solid isso não é automático — um `Show` sem `keyed`, ou com `keyed` e a
 * função filha SEM parâmetro declarado, reusa o nó e os campos ficam de pé.
 */
describe('trocar de criatura', () => {
  it('recomeça dos números do livro', async () => {
    const { user, abre } = renderDialog()
    const pv = await screen.findByLabelText('PV')
    await user.clear(pv)
    await user.type(pv, '20')
    expect(screen.getByLabelText('PV')).toHaveValue(20)

    // SEM desmontar: é assim que acontece na tela — fecha o ogro, abre o goblin.
    abre(GOBLIN)

    await waitFor(() => expect(screen.getByLabelText('PV')).toHaveValue(9))
  })

  // A mesma criatura reaberta também recomeça: o mestre que baixou o PV de um
  // ogro e voltou para pegar o segundo não quer o primeiro ferido de novo.
  it('reabrir a MESMA criatura também recomeça', async () => {
    const { user, abre } = renderDialog()
    const pv = await screen.findByLabelText('PV')
    await user.clear(pv)
    await user.type(pv, '20')

    abre(null)
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    abre(OGRO)

    await waitFor(() => expect(screen.getByLabelText('PV')).toHaveValue(45))
  })
})
