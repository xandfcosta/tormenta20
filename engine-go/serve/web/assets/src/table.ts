import { squareFromStyle, slideTheToken } from '@/lib/token-move'
import { piscarVital, pulsarVez, emerge } from '@/lib/turn-juice'

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
 * módulo próprio — o `vite.config.ts` tinha quatro entradas e nenhuma
 * era esta. Isto é a quinta, e hoje ela liga as TRÊS.
 *
 * **Entrada própria e não `scene.ts`**, pela mesma razão escrita naquele arquivo:
 * o `scene.js` carrega em TODA página, e pôr observador de tabuleiro nele seria
 * mandar código de mapa para quem abriu a ficha de um personagem.
 *
 * # O gate de movimento reduzido decide UMA vez, aqui
 *
 * A regra global do CSS mata animação declarativa; `el.animate` passa por baixo
 * dela. Quem chama é que tem de perguntar — e a pergunta é feita neste ponto e
 * não dentro de cada animação, que é o mesmo desenho da folha de especificação:
 * um ponto só decide, e é ele que se conferem ligando a preferência no sistema.
 */

const REDUCED_MOTION = '(prefers-reduced-motion: reduce)'

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
function wireTheTokenSlide(still: MediaQueryList): void {
  new MutationObserver((records) => {
    if (still.matches) return
    for (const record of records) {
      if (record.attributeName !== 'style') continue
      const subject = record.target as Element
      if (!subject.classList?.contains('board-token')) continue
      // O FANTASMA do arrasto é uma peça também, e ele nasce e morre a cada
      // gesto: animá-lo seria desenhar um deslize por cima da prévia que o
      // dedo está conduzindo.
      if (subject.classList.contains('board-token-ghost')) continue

      const de = squareFromStyle(record.oldValue)
      const to = squareFromStyle(subject.getAttribute('style'))
      if (!de || !to) continue

      // O lado da casa vem do COMPUTADO e não do atributo: `--quadrado` mora na
      // cena e a peça o herda, então lê-lo do `style` dela devolveria vazio.
      const square = Number.parseFloat(getComputedStyle(subject).getPropertyValue('--quadrado'))
      if (!Number.isFinite(square) || square <= 0) continue

      slideTheToken(subject, de, to, square)
    }
  }).observe(document.body, {
    subtree: true,
    attributes: true,
    attributeOldValue: true,
    attributeFilter: ['style'],
  })
}

/**
 * A LINHA da fila que sangrou ou foi curada.
 *
 * O gatilho é o `aria-valuenow` da barra de vital, e a escolha dele é o achado
 * desta fatia: ele é o NÚMERO, medido chegando como `"40" → "39"` sobre o mesmo
 * nó. As outras coisas que mudam no mesmo remendo não servem — a largura em
 * porcentagem ARREDONDA (perder 1 de 300 não muda um por cento, e a linha não
 * piscaria), e o `aria-label` da linha é texto que precisaria ser interpretado.
 *
 * A COR diz o sinal, e quem a escolhe é a comparação: subiu é cura, desceu é
 * dano. Por isso o valor antigo é obrigatório — sem ele haveria um flash só,
 * e "levei 12" e "curei 12" ficariam iguais.
 *
 * Ela pinta a LINHA e não a barra: a pergunta que a issue registra é *"ninguém
 * viu QUEM sangrou"*, e quem responde é a linha inteira, com o nome dentro.
 *
 * O trilho de 80px fica de fora de propósito. Ele tem um filete de 4px e as
 * iniciais, sem número nenhum — a piscada ali não responderia a pergunta, e
 * pendurá-la num rótulo de texto ("… — PV 39 de 40") seria interpretar prosa
 * para descobrir o que um atributo já diz na gaveta.
 *
 * # A FICHA usa outro gatilho, e a diferença é honesta
 *
 * A ficha dentro da sessão também mostra PV e PM, e o P1 pede a piscada lá. Ela
 * não tem `aria-valuenow`: a barra dela é `aria-hidden` de propósito — *"um
 * leitor de tela não tem o que fazer com uma largura em porcento, e a fração já
 * diz tudo"* — e o número mora no TEXTO ao lado, `39/40`.
 *
 * Pôr `role="progressbar"` ali só para este módulo achar o nó faria a ficha
 * anunciar "PV 39 de 40" e "39/40" em seguida: duas vozes para o mesmo dado. O
 * gatilho é a mudança do TEXTO, que é o número de verdade, e o `data-vital` diz
 * apenas QUAL poço é — não repete o valor.
 */
function wireTheVitalBlink(stopped: MediaQueryList): void {
  new MutationObserver((records) => {
    if (stopped.matches) return
    for (const record of records) {
      if (record.attributeName !== 'aria-valuenow') continue
      const bar = record.target as Element
      const before = Number(record.oldValue)
      const now = Number(bar.getAttribute('aria-valuenow'))
      if (!Number.isFinite(before) || !Number.isFinite(now) || before === now) continue

      // A linha é quem tem o nome. Sem ela — uma barra fora de linha nenhuma —
      // não há o que piscar, e piscar a barra seria responder outra pergunta.
      const line = bar.closest('li')
      if (!line) continue

      // O QUADRO DE ESPERA PERDEU A RAZÃO QUE O TROUXE, e fica sem uma nova
      // medida (ALE-370). Aqui estava escrito que o véu é filho da linha e que
      // o morph o remove — isso deixou de ser verdade na ALE-322, quando o véu
      // passou a morar no `<body>`, fora do alcance do remendo.
      //
      // O que sobrou é uma suspeita, não uma medição: o `piscarVital` lê o
      // retângulo da linha para posicionar um véu `fixed`, e uma linha que mede
      // 0×0 o faz desistir em silêncio. MEDIDO: tirar este `rAF` deixa os
      // quatro guardas de `tracker-that-explains` VERDES — ou seja, nenhum
      // deles responde por ele. Tirá-lo é fatia própria, e ela começa por um
      // guarda que compare o retângulo lido aqui com o do quadro seguinte.
      //
      // O pulso da vez nunca precisou disto: ele anima a própria linha, e o
      // morph reusa esse nó em vez de trocá-lo.
      requestAnimationFrame(() => piscarVital(line, { curou: now > before }))
    }
  }).observe(document.body, {
    subtree: true,
    attributes: true,
    attributeOldValue: true,
    attributeFilter: ['aria-valuenow'],
  })
}

/** `"39/40"` → 39. Nulo quando a fração não é uma fração. */
function currentOfTheFraction(text: string | null): number | null {
  const m = /^\s*(-?\d+)\s*\//.exec(text ?? '')
  return m ? Number(m[1]) : null
}

/**
 * A PISCADA na ficha dentro da sessão.
 *
 * Mesmo efeito, outro gatilho: aqui o número é o TEXTO da fração, então o que se
 * observa é `characterData`. Ver o cabeçalho de `wireTheVitalBlink` para por
 * que não é um papel ARIA.
 *
 * Ela pinta a FILEIRA do vital — o `<div>` que tem o rótulo, a barra, a fração e
 * os passos —, que é o equivalente da linha da fila: a caixa que diz DE QUEM é o
 * número que mudou. Piscar só a fração seria piscar dois dígitos.
 */
function wireTheSheetBlink(still: MediaQueryList): void {
  new MutationObserver((records) => {
    if (still.matches) return
    for (const record of records) {
      const text = record.target
      const span = text.parentElement
      if (!span?.hasAttribute('data-vital')) continue

      const before = currentOfTheFraction(record.oldValue)
      const now = currentOfTheFraction(text.textContent)
      if (before === null || now === null || before === now) continue

      const row = span.parentElement
      if (!row) continue
      // O mesmo quadro de espera da fila, e com a mesma pendência: ver lá.
      requestAnimationFrame(() => piscarVital(row, { curou: now > before }))
    }
  }).observe(document.body, {
    subtree: true,
    characterData: true,
    characterDataOldValue: true,
  })
}

/**
 * O PULSO de quem entra na vez.
 *
 * O gatilho é o `aria-current`, que a linha já escrevia por acessibilidade
 * (ALE-212) — nenhum atributo novo foi inventado para animar. Isso não é
 * economia: um atributo que existe só para o JS achar o nó envelhece sozinho,
 * e este é lido por leitor de tela, então ele tem quem o defenda.
 *
 * **Só quando ele é "true"**, e não há guarda contra reescrita de valor igual —
 * o que é decisão MEDIDA e não descuido, porque ela é a primeira coisa que
 * alguém vai querer acrescentar aqui.
 *
 * O medo é razoável: a Mesa redesenha a região da fila inteira a cada dano de
 * qualquer um, então parecia que o `aria-current` da linha da vez seria
 * reescrito com o mesmo `"true"` a todo remendo, e o holofote viraria um
 * pisca-pisca. Escrevi a guarda antes de medir.
 *
 * Ela nunca protegeu nada: **o morph não toca atributo que já bate**, então
 * mutação de `aria-current` só chega quando a vez MUDA de verdade. Provado pelo
 * avesso — com a guarda removida do pacote e um guarda de e2e ferindo outro
 * combatente enquanto a vez estava parada, ZERO pulsos. Um `if` que nunca é
 * verdadeiro é dívida com cara de cuidado.
 *
 * E o caso oposto degrada bem: se o morph um dia TROCAR o nó da linha em vez de
 * reconciliá-lo, não há mutação de atributo nenhuma e o pulso simplesmente não
 * toca — perde-se a animação, nunca se ganha uma errada.
 *
 * # O que FALTA aqui, e por que não entrou junto
 *
 * A issue pede também `scrollIntoView({behavior:'smooth'})` na linha que entra
 * na vez — a metade que importa numa fila longa, onde o holofote pode andar
 * para fora da janela de rolagem.
 *
 * Ela não entrou porque **este repositório já mediu rolagem suave falhando em
 * silêncio**: o comentário do centralizar do tabuleiro registra que
 * `scroll-behavior: smooth` deixava o `scrollTop` em ZERO, sem erro em lugar
 * nenhum. E provar que esta instância CHEGA custa dois clientes — a lista rola
 * na gaveta, que é modal, então quem avança o turno não é quem a vê rolar.
 * Escrever a linha com um comentário dizendo "deve funcionar" seria dívida com
 * cara de entrega.
 */
function wireTheTurnPulse(still: MediaQueryList): void {
  new MutationObserver((records) => {
    if (still.matches) return
    for (const record of records) {
      if (record.attributeName !== 'aria-current') continue
      const row = record.target as Element
      if (row.getAttribute('aria-current') !== 'true') continue
      pulsarVez(row)
    }
  }).observe(document.body, {
    subtree: true,
    attributes: true,
    // Sem `attributeOldValue`: o valor antigo não é lido aqui. Ele estava
    // ligado enquanto existia a guarda contra repetição, e ficar depois dela
    // sair seria pedir ao navegador que guardasse uma string por mutação para
    // ninguém olhar.
    attributeFilter: ['aria-current'],
  })
}

/**
 * A CONDIÇÃO que foi aplicada (ALE-174, P4).
 *
 * É a única dos cinco disparos que é de MOUNT de verdade, e isso foi medido:
 * aplicar uma condição CRIA o `<ul>` dos crachás, que não existia com a lista
 * vazia. O que o morph REUSA — a linha de um combatente que entra na fila, o
 * P5 — não tem mount para prender, e por isso ficou de fora desta fatia com a
 * medição registrada na issue.
 *
 * Ele anima os crachás e não o `<ul>`: a lista é um contêiner sem tinta, e uma
 * segunda condição chega dentro de um `<ul>` que já existe — animar só o
 * contêiner faria a primeira aparecer e as seguintes não.
 */
function wireTheConditionEmerging(still: MediaQueryList): void {
  new MutationObserver((records) => {
    if (still.matches) return
    for (const record of records) {
      for (const no of record.addedNodes) {
        if (no.nodeType !== Node.ELEMENT_NODE) continue
        const element = no as Element
        // O crachá em si, ou a lista inteira chegando de uma vez.
        const badges = element.matches('[data-condicao]')
          ? [element]
          : [...element.querySelectorAll('[data-condicao]')]
        for (const badge of badges) emerge(badge)
      }
    }
  }).observe(document.body, { subtree: true, childList: true })
}

const STILL = window.matchMedia(REDUCED_MOTION)

wireTheTokenSlide(STILL)
wireTheVitalBlink(STILL)
wireTheSheetBlink(STILL)
wireTheTurnPulse(STILL)
wireTheConditionEmerging(STILL)
