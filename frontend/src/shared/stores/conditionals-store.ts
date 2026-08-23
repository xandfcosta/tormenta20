import { createStore, produce } from 'solid-js/store'

/**
 * Quais SITUACIONAIS o jogador ligou, por personagem — Fúria, Ataque Poderoso,
 * os homebrew. A ficha recomputa contra este conjunto.
 *
 * O DONO É O SERVIDOR (ALE-222). Até aqui isto vivia em `localStorage`, e o
 * comentário que estava nesta mesma linha dizia o contrário: *"It is a CLIENT
 * choice, not server state: the sheet recomputes against this set, so it lives
 * here and persists locally"*. A decisão do dono em 2026-08-22 foi "o servidor
 * mantém estado, ponto final", e a frase antiga fica registrada aqui em vez de
 * apagada — o histórico não deve mentir sobre o que já se pensou.
 *
 * O que NÃO mudou é onde a CONTA acontece: o WASM roda o mesmo motor Go, então
 * a ficha continua recomputando local em 0,28ms contra o conjunto que veio do
 * servidor. "O servidor é dono do estado" nunca obrigou o cálculo a ir junto —
 * o que passou a custar rede é só a escrita do toggle.
 *
 * Por isso a leitura continua SÍNCRONA: `active()` é chamado de dentro de
 * memos, e devolver Promise faria a ficha inteira virar assíncrona por causa de
 * um booleano. O cache local é enchido pela ficha (`hydrate`) e escrito
 * OTIMISTA: pinta primeiro, manda depois, desfaz se o servidor recusar.
 *
 * `situacional` NÃO é `condição`: aquele é este opt-in, esta é a do livro
 * (Caído, Atordoado) e mora em `character.activeConditions`. Ver a colisão C6
 * no GLOSSARIO.md.
 */

type ActiveByCharacter = Record<string, string[]>

/** Quem leva o conjunto novo ao servidor. Injetado para o teste poder espiar. */
export type ConditionalsWriter = (
  characterId: number,
  conditionals: string[],
) => Promise<unknown>

export type ConditionalsStore = {
  /** Os ids ligados de um personagem. Rastreia — leia dentro de um memo. */
  active: (characterId: number) => ReadonlySet<string>
  toggle: (characterId: number, id: string) => void
  /** Lote: liga ou desliga todos os `ids` de uma vez. */
  setMany: (characterId: number, ids: string[], value: boolean) => void
  clear: (characterId: number) => void
  /**
   * Enche o cache com o que veio na ficha. Chamado quando ela carrega — sem
   * isto o app abriria com tudo desligado e ligaria um instante depois,
   * piscando exatamente os números que o situacional muda.
   */
  hydrate: (characterId: number, conditionals: readonly string[]) => void
}

export function createConditionalsStore(
  write: ConditionalsWriter,
  onError: (erro: unknown) => void = () => {},
): ConditionalsStore {
  const [active, setActive] = createStore<ActiveByCharacter>({})

  /**
   * Escreve local, manda ao servidor e DESFAZ se ele recusar.
   *
   * O instantâneo é tirado antes de mutar e a volta é para ELE, não para "tire
   * o que eu acabei de pôr": entre a ida e a recusa o jogador pode ter ligado
   * outro situacional, e desfazer por operação inversa deixaria o conjunto num
   * estado que ninguém pediu.
   */
  const edit = (characterId: number, mutate: (draft: ActiveByCharacter) => void) => {
    const chave = String(characterId)
    const antes = [...(active[chave] ?? [])]
    setActive(produce(mutate))
    const depois = [...(active[chave] ?? [])]
    void write(characterId, depois).catch((erro) => {
      setActive(
        produce((draft) => {
          draft[chave] = antes
        }),
      )
      onError(erro)
    })
  }

  return {
    // Um Set NOVO a cada leitura: quem chama o passa para `computedSheetFor`,
    // que indexa o cache pelo CONTEÚDO do conjunto, e um Set compartilhado e
    // mutável deixaria um toggle passar por esse cache sem ser notado.
    active: (characterId) => new Set(active[String(characterId)] ?? []),

    hydrate: (characterId, conditionals) =>
      setActive(
        produce((draft) => {
          draft[String(characterId)] = [...conditionals]
        }),
      ),

    toggle: (characterId, id) =>
      edit(characterId, (draft) => {
        const chave = String(characterId)
        const atual = draft[chave] ?? []
        draft[chave] = atual.includes(id) ? atual.filter((x) => x !== id) : [...atual, id]
      }),

    setMany: (characterId, ids, value) =>
      edit(characterId, (draft) => {
        const chave = String(characterId)
        const atual = new Set(draft[chave] ?? [])
        for (const id of ids) {
          if (value) atual.add(id)
          else atual.delete(id)
        }
        draft[chave] = [...atual]
      }),

    clear: (characterId) =>
      edit(characterId, (draft) => {
        draft[String(characterId)] = []
      }),
  }
}
