import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { characterQueryOptions } from '@/entities/character/queries'
import * as api from '@/shared/api/api'
import type { Character } from '@/shared/api/api'
import { VitalRows } from './vital-rows'

/**
 * A FIAÇÃO dos controles de vitais — e só ela.
 *
 * A regra já é provada em `vital-mutations.test.ts`, com nove casos: pintura
 * otimista, rajada de cliques virando UMA requisição, rollback quando a rede
 * cai, e a resposta do servidor como palavra final. Este arquivo não repete
 * nada disso. Ele prende o que ficava sem dono entre a regra e a tela: que o
 * botão da linha de Vida chama o caminho de DANO, e não o de escrita comum.
 *
 * A distinção não é detalhe. Baixar PV passa pelo pool de PV temporários no
 * servidor (uma requisição atômica, `applyDamage`); subir é cura comum
 * (`updateVitals`). Trocar um pelo outro na fiação não quebra tipo nenhum — as
 * duas assinaturas aceitam um número — e o sintoma seria PV temporário sendo
 * ignorado no meio do combate, sem erro em lugar nenhum.
 *
 * Veio de um e2e (ALE-187) que abria o Hub, o elenco, a ficha, clicava, esperava
 * a resposta da API e RECARREGAVA a página para provar que a escrita persistiu.
 * Duas coisas estavam erradas ali: a persistência é do servidor e tem oito
 * testes de handler em Go, e o e2e já havia esperado a resposta `ok` antes de
 * recarregar — então o reload só reprovava o que o Go já prova. O que sobrava de
 * único era esta fiação, e ela não precisa de browser.
 */
const CHARACTER_ID = 1

function personagem(overrides: Partial<Character> = {}): Character {
  return {
    id: CHARACTER_ID,
    name: 'Tanque',
    level: 10,
    hpMax: 100,
    hpCurrent: 50,
    mpMax: 20,
    mpCurrent: 10,
    items: [],
    activeEffects: [],
    ...overrides,
  } as Character
}

function montar(char: Character = personagem()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(CHARACTER_ID).queryKey, char)
  const view = render(() => (
    <QueryClientProvider client={client}>
      <VitalRows character={char} />
    </QueryClientProvider>
  ))
  return { ...view, client, user: userEvent.setup() }
}

describe('VitalRows — a fiação dos botões', () => {
  it('reduzir Vida vai pelo caminho do DANO, que respeita o PV temporário', async () => {
    const dano = vi
      .spyOn(api.api.characters, 'applyDamage')
      // A forma REAL do `ApplyDamageResult`: o `drained` diz o que o dano tirou
      // do pool de PV temporários. Minha primeira versão devolvia
      // `{hpCurrent, mpCurrent}`, que o servidor nunca devolve — o teste PASSOU
      // e deixou uma rejeição não tratada (`result.drained is not iterable`),
      // que o vitest avisa poder virar falso positivo. Mock com forma inventada
      // é teste medindo um servidor que não existe.
      .mockResolvedValue({ hpCurrent: 49, tempHpRemaining: 0, drained: [] })
    const vitais = vi.spyOn(api.api.characters, 'updateVitals')
    const { user } = montar()

    await user.click(await screen.findByRole('button', { name: /^Reduzir Vida/ }))

    expect(dano).toHaveBeenCalledWith(CHARACTER_ID, 1)
    // A ausência é metade da garantia: o caminho comum NÃO pode ser usado para
    // baixar PV, senão o pool temporário é pulado em silêncio.
    expect(vitais).not.toHaveBeenCalled()
  })

  // A PINTURA otimista não se afirma aqui, e a tentativa ensinou por quê: este
  // componente recebe o personagem por PROP, e quem o realimenta do cache é a
  // página. Um teste da pintura montado assim precisaria imitar o pai — mediria
  // o arreio, não o app — e a garantia já tem dono: o `vital-mutations.test.ts`
  // prova "pinta na hora e só manda uma vez no fim da rajada" sem DOM nenhum.

})
