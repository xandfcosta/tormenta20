import { expect, test } from '@playwright/test'
import { VIEWPORTS } from './support/viewports'

/**
 * A cena da campanha vista por um JOGADOR, não pelo mestre (ALE-24). Todo o
 * resto da suíte roda como o mestre da seed pelo storageState compartilhado;
 * este spec sai disso e entra como o jogador que é membro da campanha 1 e não
 * é dono de nada nela.
 *
 * As três asserções de AUSÊNCIA que moravam aqui (rail sem Config, nenhuma ação
 * de dono, `?tab=config` na mão) saíram na ALE-144: a trava de verdade é do
 * servidor e está em `api/authz_http_test.go`, e a metade de UX mudou de casa
 * na ALE-255: com a crônica virando cena do servidor, quem prova que o jogador
 * pedindo `?tab=config` cai na visão geral é o
 * `TestJogadorPedindoConfigCaiParaAVisaoGeral`, em `api/piloto_cronica_test.go`.
 * Botão ausente nunca foi prova de trava, e a regra da casa é que cada garantia
 * fique na camada mais barata que a sustenta — que aqui ficou mais barata
 * ainda, porque a aba deixou de existir no servidor em vez de ser escondida.
 *
 * E o que sobrava aqui — "o jogador ENTRA na sessão ao vivo" — saiu na ALE-269,
 * junto com o `openSection` e o `CAMPAIGN` que só ele usava: é o passo de CAÇAR
 * ÓRFÃOS da virada acontecendo em miniatura. A razão do caso está escrita no
 * lugar dele, logo abaixo.
 */
test.use({ storageState: '.auth/player.json' })

test.describe('Campanha vista pelo jogador', () => {
  // O jogador PERDE ações de escrita, não a mesa: ele continua lendo a campanha
  // e entrando na sessão ao vivo.
  // 'o jogador ainda lê a campanha e entra na sessão ao vivo' saiu na ALE-187:
  // eram três `toBeVisible` de texto, nada que o jsdom não veja. O vizinho
  // abaixo FICA, e a diferença entre os dois é exatamente o critério da casa.

  /**
   * O TESTE DA ALE-96 MORREU DUAS VEZES, E A SEGUNDA FOI A VIRADA (ALE-269).
   *
   * Ele media que a partida não desaparecia enquanto a ficha do jogador estava
   * em voo: ler `.data` de uma query PENDENTE suspende, e o boundary mais
   * próximo é o `Suspense` que o solid-router põe em todo route match — então a
   * cena inteira era desanexada por uma requisição que nem era dela.
   *
   * A PRIMEIRA morte foi a crônica virar página do SERVIDOR (ALE-255): sem route
   * match não há Suspense, e o caso foi reapontado para a sessão ao vivo, que
   * ainda era da SPA. O comentário de então já dizia o que fazer se ela também
   * saísse: *"mantê-lo apontado para a cena nova o transformaria num teste que
   * passa por não ter o que medir — que é pior que teste nenhum, porque parece
   * cobertura."*
   *
   * A SEGUNDA morte é agora. Entrar numa sessão passou a ser a Mesa em Datastar,
   * que chega RENDERIZADA — não existe janela entre "o shell pintou" e "os dados
   * chegaram", que é a janela inteira que este caso media. Reapontá-lo seria
   * fazer exatamente o que ele mesmo proibiu.
   *
   * A técnica era boa e o histórico a guarda
   * (`git show a94ddc5:e2e/tests/campaign-player.spec.ts`): a resposta ficava
   * SEGURA em vez de só atrasada, para a asserção ser sobre um estado e não
   * sobre uma corrida. Vale para o dia em que houver de novo uma tela que
   * carrega em duas etapas.
   */
})

/**
 * O HUD do jogador DENTRO da sessão (ALE-127).
 *
 * O dono mandou um print em que as caixas de estatística não fechavam a mesma
 * linha de base e "ATQ DIST" quebrava em duas linhas enquanto as vizinhas não —
 * o HUD estava espremido no rail de 22rem que a sessão dava a ele. A ALE-129
 * tirou o rail e deu a tela inteira à ficha, e com isso o desalinhamento sumiu.
 *
 * Este teste existe para ele não VOLTAR, e é da classe de asserção que faltava
 * na suíte: RELAÇÃO entre caixas irmãs. "A página não rola" nunca veria isto.
 */
test('as caixas do HUD fecham a mesma linha em todo formato', async ({ page }) => {
  await page.goto('/campaigns/1/sessions/4')
  await expect(page.getByRole('button', { name: /Minha ficha/ })).toBeVisible()

  for (const viewport of VIEWPORTS) {
    await page.setViewportSize({ width: viewport.width, height: viewport.height })
    const medida = await page.evaluate(() => {
      // As caixas trazem rótulo e número colados: "Defesa16", "Atq Dist+11".
      const padrao = /^(Defesa|Atq CaC|Atq Dist|Fort|Refl|Vont)[+\-−]?\d/
      const caixas = [...document.querySelectorAll('button')]
        .filter((b) => padrao.test((b.textContent ?? '').trim()))
        .map((b) => ({
          alto: Math.round(b.getBoundingClientRect().height),
          rotulo: Math.round(b.querySelector('span')?.getBoundingClientRect().height ?? 0),
        }))
        .filter((c) => c.alto > 0) // no telefone o HUD largo dá lugar à seção Vitais
      if (caixas.length === 0) return null
      const alturas = caixas.map((c) => c.alto)
      const rotulos = caixas.map((c) => c.rotulo)
      return {
        desvioAltura: Math.max(...alturas) - Math.min(...alturas),
        rotuloQuebrou: Math.max(...rotulos) > Math.min(...rotulos),
      }
    })
    if (medida === null) continue
    expect(medida.desvioAltura, `${viewport.name}: as caixas do HUD têm alturas diferentes`).toBe(0)
    expect(medida.rotuloQuebrou, `${viewport.name}: um rótulo quebrou e os outros não`).toBe(false)
  }
})
