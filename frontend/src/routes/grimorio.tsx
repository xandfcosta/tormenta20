import { createFileRoute } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'

/**
 * ISOLATED preview route for the "Grimório" theme (ALE-35). Not part of the
 * app — it scopes all theme tokens to a local `.grimorio` wrapper so nothing
 * global changes yet. Once approved here, the tokens get ported into
 * `index.css`. Delete this file after the theme lands.
 *
 * Open at /grimorio.
 */
export const Route = createFileRoute('/grimorio')({
  component: GrimorioPreview,
})

/** Proposed Grimório tokens (dark-first). oklch, high-contrast, AA body. */
const TOKENS: Record<string, string> = {
  '--g-bg': 'oklch(0.15 0.010 305)',
  '--g-bg-2': 'oklch(0.19 0.012 305)',
  '--g-panel': 'oklch(0.225 0.014 300)',
  '--g-panel-raised': 'oklch(0.27 0.016 300)',
  '--g-parchment': 'oklch(0.90 0.035 80)',
  '--g-parchment-ink': 'oklch(0.28 0.03 60)',
  '--g-fg': 'oklch(0.93 0.014 85)',
  '--g-fg-muted': 'oklch(0.74 0.020 85)',
  '--g-crimson': 'oklch(0.55 0.19 25)',
  '--g-crimson-bright': 'oklch(0.64 0.21 25)',
  '--g-purple': 'oklch(0.55 0.15 305)',
  '--g-gold': 'oklch(0.80 0.11 85)',
  '--g-iron': 'oklch(0.40 0.015 300)',
  '--g-iron-light': 'oklch(0.52 0.020 300)',
}

/** oklch string → sRGB [r,g,b] via a 1px canvas (browser does the conversion). */
function oklchToRgb(oklch: string): [number, number, number] {
  const c = document.createElement('canvas')
  c.width = c.height = 1
  const ctx = c.getContext('2d')
  if (!ctx) return [0, 0, 0]
  ctx.fillStyle = oklch
  ctx.fillRect(0, 0, 1, 1)
  const [r, g, b] = ctx.getImageData(0, 0, 1, 1).data
  return [r, g, b]
}

function relLuminance([r, g, b]: [number, number, number]): number {
  const lin = (v: number) => {
    const s = v / 255
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b)
}

function contrastRatio(fg: string, bg: string): number {
  const l1 = relLuminance(oklchToRgb(fg))
  const l2 = relLuminance(oklchToRgb(bg))
  const [hi, lo] = l1 > l2 ? [l1, l2] : [l2, l1]
  return (hi + 0.05) / (lo + 0.05)
}

type Pair = { label: string; fg: string; bg: string }
const CONTRAST_PAIRS: Pair[] = [
  { label: 'Corpo (fg / bg)', fg: '--g-fg', bg: '--g-bg' },
  { label: 'Corpo (fg / painel)', fg: '--g-fg', bg: '--g-panel' },
  { label: 'Secundário (muted / bg)', fg: '--g-fg-muted', bg: '--g-bg' },
  { label: 'Carmesim vivo / bg', fg: '--g-crimson-bright', bg: '--g-bg' },
  { label: 'Ouro / bg', fg: '--g-gold', bg: '--g-bg' },
  { label: 'Roxo / bg', fg: '--g-purple', bg: '--g-bg' },
  { label: 'Tinta / pergaminho', fg: '--g-parchment-ink', bg: '--g-parchment' },
]

function grade(ratio: number): { tag: string; color: string } {
  if (ratio >= 7) return { tag: 'AAA', color: 'var(--g-gold)' }
  if (ratio >= 4.5) return { tag: 'AA', color: 'oklch(0.7 0.17 145)' }
  if (ratio >= 3) return { tag: 'AA Large', color: 'oklch(0.75 0.15 70)' }
  return { tag: 'FALHA', color: 'var(--g-crimson-bright)' }
}

function GrimorioPreview() {
  const rootRef = useRef<HTMLDivElement>(null)
  const [ratios, setRatios] = useState<number[]>([])

  useEffect(() => {
    const el = rootRef.current
    if (!el) return
    const read = (name: string) =>
      getComputedStyle(el).getPropertyValue(name).trim()
    setRatios(CONTRAST_PAIRS.map((p) => contrastRatio(read(p.fg), read(p.bg))))
  }, [])

  const styleVars = TOKENS as React.CSSProperties

  return (
    <div ref={rootRef} className="grimorio" style={styleVars}>
      <style>{CSS}</style>

      <p className="note">
        Preview isolado do tema <b>Grimório</b> (ALE-35). Tokens escopados aqui,
        nada global mudou. Aprovando, porto pro <code>index.css</code>.
      </p>

      <MockHub />
      <Palette ratios={ratios} />
      <Typography />
      <Surfaces />
    </div>
  )
}

/** The ALE-38 Hub mockup, fully themed — the "does it feel like a game" test. */
function MockHub() {
  const items = [
    { label: 'Meus Heróis', arrow: false },
    { label: 'Crônicas', arrow: false },
    { label: 'Ferramentas do Mestre', arrow: false },
    { label: 'Continuar sessão', arrow: true },
  ]
  return (
    <section className="hub">
      <div className="hub-veil" aria-hidden />
      <div className="hub-inner">
        <h1 className="scene-title">TORMENTA&nbsp;20</h1>
        <p className="scene-kicker">— Grimório de Arton —</p>
        <nav className="menu">
          {items.map((it) => (
            <button key={it.label} type="button" className="menu-btn">
              <span className="menu-tick" aria-hidden>
                ▸
              </span>
              <span className="menu-label">{it.label}</span>
              {it.arrow && (
                <span className="menu-arrow" aria-hidden>
                  ►
                </span>
              )}
            </button>
          ))}
        </nav>
        <footer className="hub-foot">
          <span className="portrait" aria-hidden>
            A
          </span>
          <span className="hub-name">Alexandre</span>
          <span className="hub-actions">
            <button type="button" className="icon-btn" aria-label="Config">
              ⚙
            </button>
            <button type="button" className="icon-btn" aria-label="Sair">
              ⏻
            </button>
          </span>
        </footer>
      </div>
    </section>
  )
}

function Palette({ ratios }: { ratios: number[] }) {
  return (
    <section className="block">
      <h2 className="h2">Paleta &amp; contraste</h2>
      <div className="swatches">
        {Object.entries(TOKENS).map(([name, val]) => (
          <div key={name} className="swatch">
            <span
              className="chip"
              style={{ background: `var(${name})` }}
              aria-hidden
            />
            <code className="swatch-name">{name}</code>
            <code className="swatch-val">{val}</code>
          </div>
        ))}
      </div>
      <h3 className="h3">Razões de contraste (WCAG, calculadas ao vivo)</h3>
      <div className="contrast-list">
        {CONTRAST_PAIRS.map((p, i) => {
          const r = ratios[i] ?? 0
          const g = grade(r)
          return (
            <div
              key={p.label}
              className="contrast-row"
              style={{
                background: `var(${p.bg})`,
                color: `var(${p.fg})`,
              }}
            >
              <span>{p.label}</span>
              <span className="contrast-meta">
                <b>{r ? r.toFixed(2) : '—'}:1</b>
                <span className="grade" style={{ color: g.color }}>
                  {g.tag}
                </span>
              </span>
            </div>
          )
        })}
      </div>
    </section>
  )
}

function Typography() {
  return (
    <section className="block">
      <h2 className="h2">Tipografia</h2>
      <p className="display xl">Aventuras em Arton</p>
      <p className="display lg">A Deusa da Tormenta</p>
      <p className="body">
        Corpo em sans limpa (Inter) para legibilidade — títulos em Cinzel, corpo
        neutro. O texto longo nunca fica na fonte decorativa. Um clérigo de
        Khalmyr entrou na taverna e pediu por notícias das terras do norte.
      </p>
      <p className="muted">
        Texto secundário / metadados — nível, classe, origem.
      </p>
      <p className="mono">PV 42/42 · PM 18 · DEF 17 · d20+7</p>
    </section>
  )
}

function Surfaces() {
  return (
    <section className="block">
      <h2 className="h2">Superfícies emolduradas (prévia dos primitivos ALE-36)</h2>
      <div className="surfaces">
        <div className="framed">
          <h4 className="framed-title">Painel de Pedra</h4>
          <p className="body">
            Moldura de ferro com filete de ouro. Fundo de pedra escura. É o
            container padrão das cenas.
          </p>
        </div>
        <div className="framed parchment">
          <h4 className="framed-title ink">Pergaminho</h4>
          <p className="ink">
            Superfície de destaque — descrições, texto de magia, cartas. Tinta
            escura sobre pergaminho envelhecido, alto contraste.
          </p>
        </div>
      </div>
    </section>
  )
}

const CSS = `
.grimorio {
  --font-display: 'Cinzel', ui-serif, Georgia, serif;
  --font-body: 'Inter', ui-sans-serif, system-ui, sans-serif;
  --font-mono: 'JetBrains Mono', ui-monospace, monospace;
  min-height: 100%;
  background: var(--g-bg);
  color: var(--g-fg);
  font-family: var(--font-body);
  padding: 0 0 4rem;
}
.grimorio .note {
  font-size: 0.8rem; color: var(--g-fg-muted);
  padding: 0.6rem 1rem; border-bottom: 1px solid var(--g-iron);
  background: var(--g-bg-2);
}
.grimorio .note code, .grimorio code { font-family: var(--font-mono); }

/* --- Hub --- */
.grimorio .hub {
  position: relative; overflow: hidden;
  min-height: 82vh; display: grid; place-items: center;
  padding: 3rem 1.25rem;
  background:
    radial-gradient(120% 80% at 50% -10%, oklch(0.28 0.06 305 / 0.55), transparent 60%),
    radial-gradient(100% 60% at 50% 120%, oklch(0.30 0.12 25 / 0.30), transparent 55%),
    repeating-linear-gradient(115deg, oklch(1 0 0 / 0.015) 0 2px, transparent 2px 6px),
    var(--g-bg);
}
.grimorio .hub-veil {
  position: absolute; inset: 0; pointer-events: none;
  box-shadow: inset 0 0 240px 40px oklch(0 0 0 / 0.65);
}
.grimorio .hub-inner {
  position: relative; width: 100%; max-width: 30rem;
  display: flex; flex-direction: column; align-items: center; gap: 0.35rem;
}
.grimorio .scene-title {
  font-family: var(--font-display); font-weight: 700;
  font-size: clamp(2.2rem, 8vw, 3.4rem); letter-spacing: 0.18em;
  color: var(--g-fg);
  text-shadow: 0 0 24px oklch(0.55 0.15 305 / 0.5), 0 2px 2px oklch(0 0 0 / 0.6);
}
.grimorio .scene-kicker {
  font-family: var(--font-display); letter-spacing: 0.3em;
  font-size: 0.72rem; color: var(--g-gold); margin-bottom: 1.8rem;
  text-transform: uppercase;
}
.grimorio .menu {
  width: 100%; display: flex; flex-direction: column; gap: 0.6rem;
}
.grimorio .menu-btn {
  position: relative; display: flex; align-items: center; gap: 0.75rem;
  width: 100%; padding: 0.85rem 1.1rem; cursor: pointer;
  font-family: var(--font-display); font-size: 1.05rem; letter-spacing: 0.06em;
  color: var(--g-fg); text-align: left;
  background: linear-gradient(oklch(0.24 0.014 300 / 0.9), oklch(0.19 0.012 305 / 0.9));
  border: 1px solid var(--g-iron);
  border-left: 3px solid var(--g-iron-light);
  border-radius: 4px;
  transition: border-color .18s, box-shadow .18s, transform .12s, background .18s;
}
.grimorio .menu-btn:hover {
  border-color: var(--g-gold);
  border-left-color: var(--g-crimson-bright);
  box-shadow: 0 0 0 1px oklch(0.80 0.11 85 / 0.25), 0 0 22px oklch(0.55 0.19 25 / 0.28);
  transform: translateX(3px);
  background: linear-gradient(oklch(0.28 0.02 300 / 0.95), oklch(0.22 0.014 305 / 0.95));
}
.grimorio .menu-btn:focus-visible { outline: 2px solid var(--g-gold); outline-offset: 2px; }
.grimorio .menu-tick { color: var(--g-crimson-bright); font-size: 0.8rem; }
.grimorio .menu-label { flex: 1; }
.grimorio .menu-arrow { color: var(--g-gold); font-size: 0.85rem; }

.grimorio .hub-foot {
  display: flex; align-items: center; gap: 0.7rem; margin-top: 2.4rem;
  padding-top: 1.2rem; width: 100%; border-top: 1px solid var(--g-iron);
}
.grimorio .portrait {
  display: grid; place-items: center; width: 2.5rem; height: 2.5rem;
  border-radius: 50%; background: var(--g-panel-raised);
  border: 2px solid var(--g-gold); color: var(--g-gold);
  font-family: var(--font-display); font-weight: 700;
}
.grimorio .hub-name { flex: 1; font-family: var(--font-display); letter-spacing: 0.04em; }
.grimorio .hub-actions { display: flex; gap: 0.4rem; }
.grimorio .icon-btn {
  width: 2.2rem; height: 2.2rem; border-radius: 4px; cursor: pointer;
  background: var(--g-panel); border: 1px solid var(--g-iron); color: var(--g-fg-muted);
  transition: color .15s, border-color .15s;
}
.grimorio .icon-btn:hover { color: var(--g-crimson-bright); border-color: var(--g-crimson); }

/* --- generic blocks --- */
.grimorio .block { max-width: 60rem; margin: 0 auto; padding: 2.5rem 1.25rem 0; }
.grimorio .h2 {
  font-family: var(--font-display); font-size: 1.4rem; letter-spacing: 0.08em;
  color: var(--g-gold); margin-bottom: 1.1rem;
  border-bottom: 1px solid var(--g-iron); padding-bottom: 0.4rem;
}
.grimorio .h3 { font-family: var(--font-display); color: var(--g-fg); margin: 1.6rem 0 0.7rem; letter-spacing: 0.05em; }

/* swatches */
.grimorio .swatches { display: grid; grid-template-columns: repeat(auto-fill, minmax(13rem, 1fr)); gap: 0.6rem; }
.grimorio .swatch { display: flex; align-items: center; gap: 0.6rem; padding: 0.5rem; background: var(--g-panel); border: 1px solid var(--g-iron); border-radius: 4px; }
.grimorio .chip { width: 2rem; height: 2rem; border-radius: 4px; border: 1px solid var(--g-iron-light); flex: none; }
.grimorio .swatch-name { font-size: 0.72rem; color: var(--g-fg); }
.grimorio .swatch-val { font-size: 0.62rem; color: var(--g-fg-muted); margin-left: auto; }

/* contrast */
.grimorio .contrast-list { display: flex; flex-direction: column; gap: 0.4rem; }
.grimorio .contrast-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 0.7rem 1rem; border-radius: 4px; border: 1px solid var(--g-iron);
  font-size: 0.95rem;
}
.grimorio .contrast-meta { display: flex; align-items: center; gap: 0.7rem; font-family: var(--font-mono); font-size: 0.85rem; }
.grimorio .grade { font-family: var(--font-display); font-size: 0.7rem; letter-spacing: 0.1em; }

/* typography */
.grimorio .display { font-family: var(--font-display); color: var(--g-fg); line-height: 1.1; }
.grimorio .display.xl { font-size: clamp(2rem, 6vw, 3rem); font-weight: 700; letter-spacing: 0.04em; }
.grimorio .display.lg { font-size: 1.8rem; font-weight: 500; letter-spacing: 0.06em; color: var(--g-gold); margin-top: 0.4rem; }
.grimorio .body { font-family: var(--font-body); color: var(--g-fg); line-height: 1.6; margin-top: 1rem; max-width: 42rem; }
.grimorio .muted { color: var(--g-fg-muted); margin-top: 0.6rem; font-size: 0.9rem; }
.grimorio .mono { font-family: var(--font-mono); color: var(--g-fg); margin-top: 0.8rem; letter-spacing: 0.02em; }

/* surfaces */
.grimorio .surfaces { display: grid; gap: 1rem; grid-template-columns: repeat(auto-fit, minmax(16rem, 1fr)); }
.grimorio .framed {
  position: relative; padding: 1.3rem; border-radius: 5px;
  background: linear-gradient(var(--g-panel-raised), var(--g-panel));
  border: 1px solid var(--g-iron);
  box-shadow: inset 0 0 0 1px oklch(0.80 0.11 85 / 0.12), 0 6px 20px oklch(0 0 0 / 0.4);
}
.grimorio .framed-title { font-family: var(--font-display); color: var(--g-gold); letter-spacing: 0.06em; margin-bottom: 0.5rem; }
.grimorio .framed.parchment {
  background:
    radial-gradient(120% 120% at 30% 10%, oklch(0.93 0.03 85), oklch(0.86 0.04 75));
  border-color: oklch(0.55 0.06 70);
}
.grimorio .ink { color: var(--g-parchment-ink); line-height: 1.55; }
.grimorio .framed-title.ink { color: oklch(0.38 0.12 30); }
`
