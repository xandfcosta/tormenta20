import { type Accessor, createEffect } from 'solid-js'
import type { InitiativeEntry } from '@/shared/realtime/realtime'

/**
 * As duas animações que EXPLICAM o que chegou pelo socket (ALE-174).
 *
 * O problema que elas resolvem não é falta de polimento — a auditoria mediu 60
 * fps cravado em todas as cenas. É que numa lista de nove combatentes um
 * número troca sozinho e ninguém viu QUEM sangrou, nem para onde o holofote
 * andou.
 *
 * **Por que WAAPI e não animação CSS de mount.** Medido nesta lista: o estado
 * do socket chega como objeto novo a cada sync e o `For` reconcilia por
 * REFERÊNCIA, então as linhas remontam inteiras — carimbei os seis botões de
 * uma lista e ZERO sobreviveram a um único sync. Uma animação de entrada
 * replayaria em TODAS as linhas a cada ponto de vida que muda em qualquer uma
 * delas, que é o mecanismo da ALE-97. Disparar por DIFERENÇA de valor é imune
 * a isso, porque quem decide não é o nascimento do nó.
 *
 * **O gate de movimento reduzido não vem de graça.** A regra global do
 * `index.css` mata animação DECLARATIVA; `el.animate` passa por baixo dela.
 * Quem chama tem de perguntar ao `createPrefersReducedMotion` primeiro — é por
 * isso que estas funções não consultam a media query sozinhas: elas rodam
 * dentro de um efeito que já tem o acessor à mão, e uma consulta escondida
 * aqui dentro seria uma dependência reativa invisível.
 *
 * Nenhuma delas pede `will-change`: são disparos únicos, o compositor promove
 * sozinho, e `will-change` permanente em nove linhas só queimaria memória de
 * camada.
 */

/** Ambiente sem WAAPI (o jsdom da suíte) responde nada em vez de explodir. */
function podeAnimar(alvo: Element | null | undefined): alvo is Element {
  return !!alvo && typeof (alvo as HTMLElement).animate === 'function'
}

/**
 * Pisca a linha que acabou de tomar dano ou cura.
 *
 * A cor diz o SINAL: vermelho de crítico para quem perdeu, dourado de vida
 * cheia para quem ganhou. Pinta por cima com um véu próprio em vez de mexer no
 * fundo da linha, porque `opacity` mora no compositor e `background-color`
 * não — e porque a linha da vez já tem fundo dourado, que um flash dourado
 * apagaria.
 *
 * O véu nasce e morre com a animação: guardá-lo no DOM significaria nove nós
 * permanentes esperando um evento raro.
 *
 * @example piscarVital(linha, { curou: false })
 */
export function piscarVital(alvo: Element | null | undefined, opcoes: { curou: boolean }): void {
  if (!podeAnimar(alvo)) return
  const veu = document.createElement('div')
  veu.setAttribute('aria-hidden', 'true')
  veu.style.cssText = [
    'position:absolute',
    'inset:0',
    'border-radius:inherit',
    'pointer-events:none',
    `background:var(${opcoes.curou ? '--hp-full' : '--hp-critical'})`,
  ].join(';')
  alvo.appendChild(veu)

  const animacao = veu.animate([{ opacity: 0.45 }, { opacity: 0 }], {
    duration: 380,
    easing: 'ease-out',
  })
  animacao.finished.then(() => veu.remove()).catch(() => veu.remove())
}

/**
 * Pulso único na linha que ENTRA na vez.
 *
 * Escala e brilho, 250ms, uma vez só: o anel dourado sozinho teleporta de uma
 * linha para outra e a mesa não vê o holofote andar. Só `transform` e
 * `box-shadow`, e a escala é de 1.5% — o suficiente para o olho pegar o
 * movimento sem empurrar as linhas vizinhas.
 *
 * @example pulsarVez(linhaDaVez)
 */
export function pulsarVez(alvo: Element | null | undefined): void {
  if (!podeAnimar(alvo)) return
  alvo.animate(
    [
      { transform: 'scale(1)', boxShadow: '0 0 0 0 transparent' },
      {
        transform: 'scale(1.015)',
        boxShadow: '0 0 14px 2px color-mix(in oklch, var(--grimorio-gold) 45%, transparent)',
        offset: 0.4,
      },
      { transform: 'scale(1)', boxShadow: '0 0 0 0 transparent' },
    ],
    { duration: 250, easing: 'ease-out' },
  )
}

/**
 * Liga as duas animações à lista, disparando por DIFERENÇA de valor.
 *
 * Nasce UMA vez no corpo do cartão, e isso é a condição para funcionar: as
 * linhas remontam a cada sync, então um efeito criado DENTRO da linha
 * esqueceria o valor anterior toda vez e nunca veria diferença nenhuma. Quem
 * lembra tem de morar onde não remonta.
 *
 * Acha a linha por `data-entry-id` porque o nó daquela entry é outro depois do
 * sync — guardar uma referência seria guardar um nó morto.
 *
 * A primeira passada só CARIMBA o estado e não anima: sem isso, entrar na cena
 * (ou reconectar) piscaria a lista inteira, que é custo sem explicação — a
 * mesma razão pela qual a cena da sessão não tem animação de entrada.
 *
 * @example createTurnJuice({ palco: () => lista, entradas: () => rt.state().initiative, vez: () => rt.state().turnIndex, parado })
 */
export function createTurnJuice(opcoes: {
  palco: () => HTMLElement | undefined
  entradas: () => InitiativeEntry[]
  vez: () => number
  parado: Accessor<boolean>
}): void {
  const pvAnterior = new Map<string, number>()
  let vezAnterior: number | undefined
  const linha = (id: string) =>
    opcoes.palco()?.querySelector(`[data-entry-id="${CSS.escape(id)}"]`) ?? undefined

  createEffect(() => {
    for (const entrada of opcoes.entradas()) {
      const agora = entrada.hpCurrent
      if (agora === undefined) continue
      const antes = pvAnterior.get(entrada.id)
      pvAnterior.set(entrada.id, agora)
      if (antes === undefined || antes === agora || opcoes.parado()) continue
      piscarVital(linha(entrada.id), { curou: agora > antes })
    }
  })

  createEffect(() => {
    const vez = opcoes.vez()
    const daVez = opcoes.entradas()[vez]
    const primeira = vezAnterior === undefined
    if (vez === vezAnterior) return
    vezAnterior = vez
    if (primeira || !daVez || opcoes.parado()) return
    const alvo = linha(daVez.id)
    alvo?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    pulsarVez(alvo)
  })
}
