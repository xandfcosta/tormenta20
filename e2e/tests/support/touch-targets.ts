import type { Page } from '@playwright/test'

/**
 * A MEDIÇÃO DO WCAG 2.5.8 (Target Size — Minimum, AA), e ela não é "menor que
 * 24×24".
 *
 * O critério tem EXCEÇÃO DE ESPAÇAMENTO, e ignorá-la responde outra pergunta:
 * um alvo pequeno PASSA se um círculo de 24px de diâmetro centrado nele não
 * cruzar outro alvo nem o círculo de outro alvo pequeno. Uma linha de 139×20
 * sozinha numa faixa passa; duas empilhadas sem folga reprovam.
 *
 * A EXCEÇÃO DO EQUIVALENTE também é medida, e nesta ficha ela é mecânica em vez
 * de julgamento: um alvo pequeno passa se OUTRO alvo que dispara **o mesmo
 * comando** cumpre os 24px. É o caso de cada perícia — o número (`size-11`,
 * 44×44) e o nome (139×20) fazem os dois `$detail = <chave>`, e clicar em
 * qualquer um abre a mesma decomposição.
 *
 * Contar só o tamanho infla o número com alvos que a norma aceita, e isso não é
 * detalhe de purista: a ALE-177 nasceu de uma contagem assim, propondo esconder
 * a derivação das perícias para comprar espaço. **Eram 58 pequenos e quatro
 * reprovando** — o conserto teria mexido na densidade da tela mais densa do app
 * por 54 alvos que a norma já aceita.
 *
 * # O que este medidor NÃO vê
 *
 * A exceção do `inline` (alvo dentro de uma linha de texto corrido) não é
 * medida: ela não ocorre nesta ficha, e implementá-la exigiria decidir o que
 * conta como "texto corrido".
 *
 * E ALVO DENTRO DE ALVO seria falso positivo: o círculo do filho cruza a caixa
 * do pai por construção. Não acontece hoje — nenhum dos 136 alvos da ficha
 * aninha outro —, e o dia em que acontecer o guarda acusa um defeito que não
 * existe. Quem for depurar uma falha estranha olhe isto primeiro.
 *
 * Devolve `medidos` junto com as falhas, sempre: lista vazia e seletor que não
 * casa com nada se parecem no terminal.
 */
export type MeasuredTarget = {
  nome: string
  larg: number
  alt: number
  /** A assinatura que AGRUPA — a classe própria do nó, que é o que se conserta. */
  familia: string
  /** Reprova por tamanho, sem folga de espaçamento E sem equivalente que passe. */
  reprova: boolean
}

export async function touchTargets(page: Page): Promise<{ medidos: number; alvos: MeasuredTarget[] }> {
  return page.evaluate(() => {
    const SELETOR =
      'a[href],button:not([disabled]),input:not([type=hidden]),select,textarea,summary,[tabindex]:not([tabindex="-1"])'
    const visivel = (e: Element) => {
      const r = e.getBoundingClientRect()
      if (r.width <= 0 || r.height <= 0) return false
      const s = getComputedStyle(e)
      return s.visibility !== 'hidden' && s.display !== 'none'
    }
    const alvos = [...document.querySelectorAll(SELETOR)].filter(visivel)
    const caixas = alvos.map((e) => e.getBoundingClientRect())
    const pequeno = (r: DOMRect) => r.width < 24 || r.height < 24

    // O círculo de 24px da exceção: raio 12 em volta do CENTRO da caixa.
    const centro = (r: DOMRect) => ({ x: r.left + r.width / 2, y: r.top + r.height / 2 })
    const cruzaCaixa = (c: { x: number; y: number }, r: DOMRect) => {
      const px = Math.max(r.left, Math.min(c.x, r.right))
      const py = Math.max(r.top, Math.min(c.y, r.bottom))
      return Math.hypot(c.x - px, c.y - py) < 12
    }
    const cruzaCirculo = (a: { x: number; y: number }, b: { x: number; y: number }) =>
      Math.hypot(a.x - b.x, a.y - b.y) < 24

    // O COMANDO que o alvo dispara, para achar o equivalente. `data-on:click` é
    // o gesto do Datastar; `href` cobre os links.
    const comandoDe = (e: Element) =>
      e.getAttribute('data-on:click') ?? e.getAttribute('href') ?? ''
    const comandos = alvos.map(comandoDe)
    const temEquivalenteQuePassa = (i: number) => {
      const meu = comandos[i]
      if (!meu) return false
      return alvos.some((_, j) => j !== i && comandos[j] === meu && !pequeno(caixas[j]))
    }

    const nomeDe = (e: Element) =>
      (e.getAttribute('aria-label') ?? e.getAttribute('title') ?? e.textContent ?? '').trim().replace(/\s+/g, ' ').slice(0, 40)

    const medidos: MeasuredTarget[] = []
    for (let i = 0; i < alvos.length; i++) {
      const r = caixas[i]
      if (!pequeno(r)) continue
      const c = centro(r)
      let colide = false
      for (let j = 0; j < alvos.length && !colide; j++) {
        if (j === i) continue
        // O círculo não pode cruzar OUTRO ALVO...
        if (cruzaCaixa(c, caixas[j])) colide = true
        // ...nem o círculo de outro alvo PEQUENO.
        else if (pequeno(caixas[j]) && cruzaCirculo(c, centro(caixas[j]))) colide = true
      }
      const equivalente = temEquivalenteQuePassa(i)
      medidos.push({
        nome: nomeDe(alvos[i]),
        larg: Math.round(r.width),
        alt: Math.round(r.height),
        familia: (alvos[i].getAttribute('class') ?? '').split(/\s+/).slice(0, 3).join(' ') || alvos[i].tagName.toLowerCase(),
        reprova: colide && !equivalente,
      })
    }
    return { medidos: alvos.length, alvos: medidos }
  })
}
