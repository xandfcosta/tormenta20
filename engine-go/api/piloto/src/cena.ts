/**
 * O módulo que as páginas do Datastar carregam (ALE-231).
 *
 * Ele existe por causa de uma medição, e a medição derruba a premissa que
 * sustentava a ideia de "ilha" para telas ricas: `spatial-nav.ts` (151 linhas)
 * e `sfx-player.ts` (46) não importam `solid-js` NENHUMA vez, e o
 * `scene-nav.ts` (387) importava duas. A gramática de teclado do app inteiro —
 * regiões declaradas em `data-nav-region`, foco movido por geometria,
 * travessia de borda — sempre foi código DOM lendo o próprio DOM. E um DOM
 * vindo do servidor é um DOM.
 *
 * Este arquivo é COLA, não implementação: ele importa os mesmos fontes que a
 * SPA importa e os liga ao HTML do servidor. É a mesma forma do
 * `piloto.src.css`, que importa o `index.css` da SPA — uma fonte, dois
 * consumidores. Duplicar 584 linhas seria a armadilha de divergência que esta
 * migração já viu duas vezes (a casca e os botões).
 *
 * O que ele expõe em `window.cena` é o que as expressões do Datastar chamam.
 * Nada de estado da aplicação passa por aqui: som e volume são preferência
 * DESTE aparelho, e tela cheia é do navegador.
 */
import { attachSceneNav } from '@/lib/scene-nav'
import { type SfxName, createSfxPlayer } from '@/lib/sfx-player'
import { STORAGE_KEY, persistUi, readStoredSfx, readStoredVolume } from '@/lib/ui-store'

type Preferencias = { som: boolean; volume: number }

function lidas(): Preferencias {
  const bruto = globalThis.localStorage?.getItem(STORAGE_KEY) ?? null
  return { som: readStoredSfx(bruto), volume: readStoredVolume(bruto) }
}

const prefs = lidas()
const tocador = createSfxPlayer()

/**
 * Toca um cue, respeitando a preferência.
 *
 * O `hover` é engolido em ponteiro grosso porque o toque dispara `hover` e
 * `select` no MESMO gesto, e a pessoa ouve duas vezes — regra que veio do
 * `sfx.ts` da SPA e não é reinventada aqui.
 */
function cue(nome: SfxName): void {
  if (!prefs.som) return
  if (nome === 'hover' && window.matchMedia?.('(pointer: coarse)').matches) return
  tocador.play(nome, prefs.volume / 100)
}

/**
 * Posiciona um popover nativo abaixo/acima do gatilho.
 *
 * A Popover API põe o elemento na TOP LAYER, então ele sai do fluxo e do
 * contexto de posicionamento do pai — `position: absolute` no pai deixa de
 * alcançá-lo. O CSS anchor positioning resolveria isso declarativamente, mas
 * ainda não é universal; dez linhas de `getBoundingClientRect` funcionam em
 * todo navegador que tem `popover`, que é a linha de base desde 2024.
 *
 * O que NÃO se escreve aqui é o que importa: foco, dispensa por clique fora,
 * `Esc` e camada vêm do navegador. Era isso que o Popover do Kobalte trazia.
 */
function ancora(painel: HTMLElement): void {
  const gatilho = document.querySelector<HTMLElement>(`[popovertarget="${painel.id}"]`)
  if (!gatilho) return
  const g = gatilho.getBoundingClientRect()

  // Ancorado pelo BOTTOM quando abre para cima, e não por `top - altura`: no
  // `beforetoggle` o painel ainda está oculto, então `offsetHeight` é ZERO —
  // medi e ele posicionou 46px fora do lugar com um palpite de altura. Ancorar
  // pela borda de baixo dispensa a altura, e aí não há palpite.
  //
  // "Para cima" é decidido pela METADE DA TELA e não por caber: caber também
  // exige a altura. É a heurística que todo menu de rodapé usa, e o rodapé do
  // Hub está sempre no pé.
  const paraCima = g.top > window.innerHeight / 2
  const margem = 8
  // `auto` e NÃO string vazia. O estilo de agente de usuário do popover é
  // `inset: 0`, então limpar a propriedade devolve o `top: 0` DELE — e com
  // `top` e `bottom` ambos definidos e altura automática a caixa fica
  // sobre-restringida: o `top` vence e o `bottom` é ignorado em silêncio.
  // Medido: o painel encostava no alto da tela com `bottom: 112px` aplicado.
  painel.style.top = paraCima ? 'auto' : `${Math.round(g.bottom + margem)}px`
  painel.style.bottom = paraCima ? `${Math.round(window.innerHeight - g.top + margem)}px` : 'auto'
  // Preso à janela: num telefone o gatilho pode estar perto da borda direita e
  // o painel é mais largo que ele.
  const largura = painel.offsetWidth || 224
  painel.style.left = `${Math.round(Math.max(margem, Math.min(g.left, window.innerWidth - largura - margem)))}px`
}

/** A superfície que as expressões do Datastar chamam. */
const cena = {
  /** Alterna o som e devolve o novo estado, para o sinal da tela acompanhar. */
  som(): boolean {
    prefs.som = !prefs.som
    persistUi({ sfx: prefs.som, volume: prefs.volume })
    // Ligar responde com um cue: o clique é o gesto que DESTRAVA o áudio, e
    // silêncio logo depois de ligar o som é o que fez gente achar que estava
    // quebrado (ALE-165).
    if (prefs.som) cue('select')
    return prefs.som
  },
  volume(pct: number): number {
    prefs.volume = Math.min(100, Math.max(0, Math.round(pct)))
    persistUi({ sfx: prefs.som, volume: prefs.volume })
    return prefs.volume
  },
  telaCheia(): boolean {
    if (document.fullscreenElement) {
      void document.exitFullscreen()
      return false
    }
    void document.documentElement.requestFullscreen?.()
    return true
  },
  temTelaCheia(): boolean {
    return typeof document.documentElement.requestFullscreen === 'function'
  },
  /** Lidas uma vez no `data-init` para os rótulos do menu nascerem certos. */
  preferencias(): Preferencias {
    return { ...prefs }
  },
  cue,
}

declare global {
  interface Window {
    cena: typeof cena
  }
}
window.cena = cena

// ── ligação com o HTML do servidor ───────────────────────────────────────────

const cascaDaCena = () => document.querySelector<HTMLElement>('[data-slot="scene-shell"]')

attachSceneNav({
  root: cascaDaCena,
  // `data-voltar` na casca diz para onde o Esc leva. Sem ele, Esc não faz nada
  // — é o caso do Hub, que é a cena raiz e não tem para onde voltar.
  onEscape: () => {
    const destino = cascaDaCena()?.dataset.voltar
    if (destino) window.location.href = destino
  },
  sfx: cue,
})

// Os cues de ponteiro por DELEGAÇÃO, num ouvinte só: o HTML vem do servidor e
// pode ser remendado a qualquer tique, e ouvinte preso a nó morre no remendo.
document.addEventListener(
  'pointerover',
  (e) => {
    if ((e.target as Element | null)?.closest?.('[data-cue-hover]')) cue('hover')
  },
  true,
)
document.addEventListener('click', (e) => {
  if ((e.target as Element | null)?.closest?.('[data-cue-select]')) cue('select')
})

// Na CAPTURA, e isso não é preferência: `beforetoggle` NÃO BORBULHA. Um
// ouvinte no `document` sem captura nunca dispara — medido, o popover abria no
// canto superior esquerdo. A fase de captura desce do root até o alvo mesmo
// para evento que não sobe.
document.addEventListener(
  'beforetoggle',
  (e) => {
    const alvo = e.target as HTMLElement
    if (alvo?.matches?.('[popover]') && (e as ToggleEvent).newState === 'open') ancora(alvo)
  },
  true,
)
