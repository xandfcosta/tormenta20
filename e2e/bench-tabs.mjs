import { chromium } from '@playwright/test'

/**
 * Cost of switching a character-sheet block, on whichever front is running.
 * Built for the React→Solid comparison (ALE-73): same character, same clicks,
 * same numbers on both.
 *
 * Two metrics, because they answer different questions:
 *
 *   painted  — click → the frame that shows the new block, in full precision.
 *              This is what the player experiences as "it responded".
 *   blocked  — total main-thread time spent in tasks over 50ms (the Long Tasks
 *              API only reports above that). While one runs, the page cannot
 *              paint or react to input, so this is the JANK, not the latency.
 *              A high `painted` with zero `blocked` means the work was spread
 *              across frames and the app stayed responsive throughout.
 *
 * `painted` is measured with a double rAF: the first fires before the frame is
 * committed, the second after the browser has painted it.
 *
 * Usage: node bench-tabs.mjs http://localhost:5174 "SOLID"
 */
const BASE = process.argv[2]
const LABEL = process.argv[3] ?? BASE
const TABS = ['Mochila', 'Proficiências', 'Efeitos', 'Poderes', 'Perícias']
const RUNS = 3

const browser = await chromium.launch()
const ctx = await browser.newContext({
  storageState: '.auth/user.json',
  viewport: { width: 1600, height: 1000 },
})
const page = await ctx.newPage()

// Personagem 1 = Tanque Placas Nv10 (do seed).
await page.goto(`${BASE}/characters/1?tab=expertises`, { waitUntil: 'networkidle' })
await page.waitForTimeout(2500)

await page.evaluate(() => {
  window.__bench = { long: [], painted: null }
  new PerformanceObserver((list) => {
    for (const entry of list.getEntries()) window.__bench.long.push(Math.round(entry.duration))
  }).observe({ entryTypes: ['longtask'] })

  // Armed before each click; resolves on the frame AFTER the one the click
  // produced, which is the first frame the user could actually see.
  window.__armPaintProbe = () => {
    window.__bench.painted = null
    document.addEventListener(
      'click',
      () => {
        const t0 = performance.now()
        requestAnimationFrame(() =>
          requestAnimationFrame(() => {
            window.__bench.painted = performance.now() - t0
          }),
        )
      },
      { capture: true, once: true },
    )
  }
})

const median = (xs) => [...xs].sort((a, b) => a - b)[Math.floor(xs.length / 2)]
const rows = []

for (const name of TABS) {
  const tab = page.getByRole('tab', { name: new RegExp(name, 'i') }).first()
  if ((await tab.count()) === 0) {
    rows.push([name, 'aba não encontrada'])
    continue
  }

  // Bounce off a DIFFERENT block between runs: bouncing off the target itself
  // makes the measured click a no-op, and the run reports the cost of nothing.
  const bounce = name === 'Perícias' ? 'Mochila' : 'Perícias'

  const painted = []
  const blocked = []
  for (let run = 0; run < RUNS; run++) {
    await page.getByRole('tab', { name: new RegExp(bounce, 'i') }).first().click()
    await page.waitForTimeout(400)

    await page.evaluate(() => {
      window.__bench.long = []
      window.__armPaintProbe()
    })
    await tab.click()
    await page.waitForTimeout(600)

    const result = await page.evaluate(() => window.__bench)
    painted.push(result.painted == null ? Number.NaN : Math.round(result.painted))
    blocked.push(result.long.reduce((a, b) => a + b, 0))
  }

  const p = median(painted.filter((x) => !Number.isNaN(x)))
  const b = median(blocked)
  rows.push([
    name,
    `pintou em ${p ?? '?'}ms · travou ${b}ms${b === 0 ? ' (nada acima de 50ms)' : ''}`,
  ])
}

console.log(`\n### ${LABEL} — ${BASE}   (mediana de ${RUNS} trocas)`)
for (const [name, result] of rows) console.log(`  ${name.padEnd(15)} ${result}`)
await browser.close()
