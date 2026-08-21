import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import { ConsumeAction } from './consume-action'

afterEach(() => {
  document.body.innerHTML = ''
})

/** The catalogs are primed from t20-data in test-setup, so these are the real
 *  entries the app ships. */
function consumableOf(catalogId: string) {
  const consumable = getCatalogItem(catalogId)?.consumable
  if (!consumable) throw new Error(`Catálogo sem consumível: ${catalogId}`)
  return consumable
}

function renderAction(catalogId: string, onConsume: () => void) {
  render(() => (
    <ConsumeAction
      consumable={consumableOf(catalogId)}
      itemName="Bálsamo restaurador"
      onConsume={onConsume}
      trigger={(open) => (
        <button type="button" onClick={open}>
          Usar
        </button>
      )}
    />
  ))
  return userEvent.setup()
}

describe('ConsumeAction', () => {
  // Macarrão de Yuvalin dá +5 PV fixos: não há dado para rolar, então usar é um clique.
  it('consumível de ganho fixo aplica direto, sem diálogo', async () => {
    const onConsume = vi.fn()
    const user = renderAction('macarrao-de-yuvalin', onConsume)
    const usar = screen.getByRole('button', { name: 'Usar' })

    await user.click(usar)

    expect(onConsume).toHaveBeenCalledWith()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    // Tira o ponteiro do gatilho antes de acabar. SÓ este caminho — o de ganho
    // fixo — embrulha o botão num `Tooltip`, e o `openDelay` padrão do Kobalte
    // é de 700ms: o clique agenda a abertura, o teste termina em
    // milissegundos, e o timer dispara depois do ambiente jsdom ser desmontado,
    // com `window is not defined` que o vitest conta como falha do arquivo
    // INTEIRO. Foi assim que o CI ficou vermelho uma vez sem nada relacionado
    // ter mudado. Sair do gatilho cancela o agendamento.
    await user.unhover(usar)
  })

  it('consumível de dado pede o resultado e soma o bônus do dado', async () => {
    const onConsume = vi.fn()
    const user = renderAction('balsamo-restaurador', onConsume)

    await user.click(screen.getByRole('button', { name: 'Usar' }))
    expect(screen.getByText(/Role 2d4/)).toBeInTheDocument()

    await user.type(screen.getByLabelText(/Role 2d4/), '6')
    await user.click(screen.getByRole('button', { name: 'Aplicar' }))

    expect(onConsume).toHaveBeenCalledWith({ hpRolled: 6 })
  })

  // 2d4 nunca sai 9 — um 9 digitado é erro de dedo, e mandar isso ao backend
  // seria inventar cura.
  it('recusa um resultado que o dado não produz', async () => {
    const onConsume = vi.fn()
    const user = renderAction('balsamo-restaurador', onConsume)

    await user.click(screen.getByRole('button', { name: 'Usar' }))
    await user.type(screen.getByLabelText(/Role 2d4/), '9')
    await user.click(screen.getByRole('button', { name: 'Aplicar' }))

    expect(screen.getByText(/Fora do intervalo \(2–8\)/)).toBeInTheDocument()
    expect(onConsume).not.toHaveBeenCalled()
  })

  it('não aplica com o campo vazio', async () => {
    const onConsume = vi.fn()
    const user = renderAction('balsamo-restaurador', onConsume)

    await user.click(screen.getByRole('button', { name: 'Usar' }))
    await user.click(screen.getByRole('button', { name: 'Aplicar' }))

    expect(screen.getByText('Informe o resultado do dado')).toBeInTheDocument()
    expect(onConsume).not.toHaveBeenCalled()
  })
})
