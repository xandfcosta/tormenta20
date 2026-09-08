import { casaDoEstilo, deslizaAPeca } from '@/lib/token-move'

/**
 * A ILHA DA MESA — o que anima quando o estado chega pelo fio (ALE-174).
 *
 * # Por que ela nasceu, e o que a ausência dela escondia
 *
 * A ALE-174 nasceu de uma auditoria cujo veredito contraria a intuição e vale
 * repetir: *"o CSS de animação deste app é disciplinado e barato; nenhuma
 * animação existente compromete um frame"*. O problema nunca foi polir o que
 * existe — é o que falta: numa lista de nove combatentes um número troca sozinho
 * e **ninguém viu quem sangrou**; uma peça é confirmada e ela TELEPORTA.
 *
 * As duas primeiras respostas (a piscada do vital e o pulso de quem entra na
 * vez) já estavam escritas em `lib/turn-juice`, com testes de olho na folha de
 * especificação... e **um importador só: o `grimorio.ts`**. Elas eram
 * demonstráveis por botão e não aconteciam na sessão, porque a Mesa não tinha
 * módulo próprio — o `vite.piloto.config.ts` tinha quatro entradas e nenhuma
 * era esta. Isto é a quinta.
 *
 * **Entrada própria e não `cena.ts`**, pela mesma razão escrita naquele arquivo:
 * o `cena.js` carrega em TODA página, e pôr observador de tabuleiro nele seria
 * mandar código de mapa para quem abriu a ficha de um personagem.
 *
 * # O gate de movimento reduzido decide UMA vez, aqui
 *
 * A regra global do CSS mata animação declarativa; `el.animate` passa por baixo
 * dela. Quem chama é que tem de perguntar — e a pergunta é feita neste ponto e
 * não dentro de cada animação, que é o mesmo desenho da folha de especificação:
 * um ponto só decide, e é ele que se conferem ligando a preferência no sistema.
 */

const MOVIMENTO_REDUZIDO = '(prefers-reduced-motion: reduce)'

/**
 * Liga o deslize das peças ao remendo do servidor.
 *
 * O observador é de ATRIBUTO com valor antigo, e é isso que dá o delta: quando o
 * movimento é confirmado, o `style` da peça é reescrito no lugar e a mutação
 * chega com `--col:3` de um lado e `--col:7` do outro. Ver `token-move` para por
 * que isso é imune ao zoom e o que acontece se o morph mudar de comportamento.
 *
 * Ele observa `document.body` e não o tabuleiro: a Mesa REMENDA regiões
 * inteiras, então um observador preso ao `#tabuleiro` de agora ficaria pendurado
 * num nó que o próximo remendo descarta — e a lista de mutações voltaria vazia,
 * que é indistinguível de "nada mudou". O `body` nunca é remendado, e o próprio
 * guia da casa registra essa armadilha.
 *
 * O CUSTO DISSO FOI MEDIDO, porque a issue nasceu de uma auditoria que achou 60
 * fps cravado e seria tolo regredir isso para consertá-la: o pan reescreve o
 * `style` da cena a cada movimento do ponteiro, e cada uma dessas mutações passa
 * por aqui. São **0,1µs por mutação** — mil escritas de `style` somam 0,1ms,
 * contra os 16,7ms de um quadro. O filtro de atributo é o que compra isso: sem
 * ele o observador acordaria em toda mudança de classe da Mesa.
 */
function ligaODeslizeDasPecas(): void {
  const parado = window.matchMedia(MOVIMENTO_REDUZIDO)

  new MutationObserver((registros) => {
    if (parado.matches) return
    for (const registro of registros) {
      if (registro.attributeName !== 'style') continue
      const alvo = registro.target as Element
      if (!alvo.classList?.contains('tabuleiro-peca')) continue
      // O FANTASMA do arrasto é uma peça também, e ele nasce e morre a cada
      // gesto: animá-lo seria desenhar um deslize por cima da prévia que o
      // dedo está conduzindo.
      if (alvo.classList.contains('tabuleiro-peca-fantasma')) continue

      const de = casaDoEstilo(registro.oldValue)
      const para = casaDoEstilo(alvo.getAttribute('style'))
      if (!de || !para) continue

      // O lado da casa vem do COMPUTADO e não do atributo: `--quadrado` mora na
      // cena e a peça o herda, então lê-lo do `style` dela devolveria vazio.
      const quadrado = Number.parseFloat(getComputedStyle(alvo).getPropertyValue('--quadrado'))
      if (!Number.isFinite(quadrado) || quadrado <= 0) continue

      deslizaAPeca(alvo, de, para, quadrado)
    }
  }).observe(document.body, {
    subtree: true,
    attributes: true,
    attributeOldValue: true,
    attributeFilter: ['style'],
  })
}

ligaODeslizeDasPecas()
