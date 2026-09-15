import { readFileSync } from 'node:fs'

/**
 * O REGISTRO DAS CENAS MEDIDAS POR APARÊNCIA (ALE-320).
 *
 * Contraste e tipografia só o navegador responde — converter oklch para sRGB e
 * saber em que tamanho a Cinzel foi DESENHADA são perguntas que nenhuma outra
 * camada tem como fazer. Por isso a medição roda aqui.
 *
 * O que mudou é o REGIME. Ela morava enumerada: dezoito endereços escritos um a
 * um, catorze cópias do mesmo caso espalhadas pelos `describe` de um spec de
 * cenas só, e nada cobrando a cena que nascesse amanhã. O
 * `CLAUDE.md` chama isso de remendo desde a ALE-252 e nomeia o conserto: um
 * guarda que FORÇA a varredura, em vez de uma lista que alguém lembra de manter.
 *
 * # O registro não mora aqui, e a razão foi MEDIDA
 *
 * Ele é `engine-go/serve/web/appearance_scenes.json`, e este arquivo só o lê. Duas
 * razões, e a segunda mordeu:
 *
 * 1. Quais cenas existem e em que endereço é fato do APP, e o app é Go. O e2e é
 *    consumidor.
 * 2. O `go test` só invalida cache por arquivo DO MÓDULO. Com o registro aqui,
 *    a sabotagem que tirava o leitor da lista passava `ok (cached)` — o guarda
 *    afirmando sobre um repositório que ele não tinha lido.
 *
 * Quem cobra é o `convention.TestEveryPageSceneIsMeasuredForAppearance`: ele
 * deriva do código quem desenha página inteira (quem chama `WritePage`) e exige
 * uma entrada para cada, FALHANDO COM O NOME da que faltar. Ele achou o LEITOR
 * na primeira execução — página desde a ALE-264, nunca uma medição sequer.
 *
 * Acrescentar cena ao app é acrescentar uma linha no JSON. Não fazer isso é
 * vermelho em Go, e não uma tela sem guarda em silêncio.
 */
export type MeasuredVisit = {
  /** o endereço que o medidor abre */
  address: string
  /**
   * quem entra. `none` é a porta: ela só existe para quem NÃO entrou, e visitá-la
   * autenticado mediria o desvio, não a cena.
   */
  session: 'gm' | 'player' | 'none'
  /**
   * a cena pode não existir NESTA bancada, e a ausência é legítima. Hoje é só o
   * leitor: sem `LIVRO_PDF` a rota devolve 404 de propósito, e o caso PULA com
   * motivo visível em vez de reprovar. Pular é diferente de medir, e a diferença
   * aparece no relatório — que é o contrário de um verde silencioso.
   */
  mayBeAbsent?: boolean
}

export type MeasuredScene = {
  /**
   * PLURAL porque uma cena desenha telas diferentes, e duas delas para PAPÉIS
   * diferentes. A Mesa é o caso: mestre e jogador recebem metades distintas do
   * mesmo endereço, e medir só uma mede metade (ALE-276).
   */
  visits: MeasuredVisit[]
}

const REGISTRY = new URL('../../../engine-go/serve/web/appearance_scenes.json', import.meta.url)

export const MEASURED_SCENES: Record<string, MeasuredScene> = JSON.parse(
  readFileSync(REGISTRY, 'utf8'),
)

/** O arquivo de sessão de cada papel, na forma que o Playwright pede. */
export const SESSION_FILE: Record<MeasuredScene['session'], string | undefined> = {
  gm: '.auth/user.json',
  player: '.auth/player.json',
  none: undefined,
}
