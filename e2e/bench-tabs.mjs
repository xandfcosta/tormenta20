import { chromium } from '@playwright/test'

const BASE = process.argv[2]
const LABEL = process.argv[3]
const TABS = ['Mochila', 'Proficiências', 'Efeitos', 'Poderes', 'Perícias']

const browser = await chromium.launch()
const ctx = await browser.newContext({ storageState: '.auth/user.json', viewport: { width: 1600, height: 1000 } })
const page = await ctx.newPage()

// Personagem 1 = Tanque Placas Nv10 (do mestre).
await page.goto(`${BASE}/characters/1?tab=expertises`, { waitUntil: 'networkidle' })
await page.waitForTimeout(2500)

// Bloqueio de main thread por troca de bloco: é isso que o usuário sente.
await page.evaluate(() => {
  window.__long = []
  new PerformanceObserver((l) => {
    for (const e of l.getEntries()) window.__long.push(Math.round(e.duration))
  }).observe({ entryTypes: ['longtask'] })
})

const perTab = []
for (const nome of TABS) {
  await page.evaluate(() => { window.__long = [] })
  const alvo = page.getByRole('tab', { name: new RegExp(nome, 'i') }).first()
  if (await alvo.count() === 0) { perTab.push([nome, 'aba não achada']); continue }
  const t0 = Date.now()
  await alvo.click()
  await page.waitForTimeout(700)
  const longs = await page.evaluate(() => window.__long)
  const bloqueio = longs.reduce((a, b) => a + b, 0)
  perTab.push([nome, `clique→estável ${Date.now() - t0}ms | main-thread travada ${bloqueio}ms ${longs.length ? `(${longs.join('+')})` : '(nenhuma tarefa longa)'}`])
}
console.log(`\n### ${LABEL} — ${BASE}`)
for (const [n, r] of perTab) console.log(`  ${n.padEnd(15)} ${r}`)
await browser.close()
