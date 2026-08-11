import { expect, test } from '@playwright/test'

const CAMPAIGN = '/campaigns/1' // Snapshot Test ALE-33 (seed)

test.describe('Detalhe da campanha', () => {
  test('troca de aba mostra o roster de membros', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=visao`)
    await expect(
      page.getByRole('heading', { name: /Snapshot Test ALE-33/i }),
    ).toBeVisible()

    await page.getByRole('tab', { name: 'Membros' }).click()
    await expect(page).toHaveURL(/tab=membros/)
    await expect(page.getByText('Tanque Placas Nv10')).toBeVisible()
    // Party roster from the rich seed (1 GM + 4 players).
    await expect(page.getByText('Recruta Nv1 Simples')).toBeVisible()
  })
})

/**
 * The GM's write block (ALE-79). Every case stops SHORT of a destructive write:
 * the seed is shared with the other specs, so editing is opened and cancelled,
 * and deleting only gets as far as the confirmation.
 */
test.describe('Escrita do mestre', () => {
  test('a aba Config traz o ledger e a zona de perigo', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=config`)

    await expect(page.getByText('Zona de perigo')).toBeVisible()
    await expect(
      page.getByRole('button', { name: /Excluir campanha/ }),
    ).toBeVisible()
  })

  test('editar abre o formulário com os valores atuais', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=config`)

    await page.getByRole('button', { name: 'Editar' }).click()

    await expect(page.getByLabel('Nome')).toHaveValue(/Snapshot Test ALE-33/)
    await expect(page.getByLabel('Descrição')).toBeVisible()
    await page.getByRole('button', { name: 'Cancelar' }).click()
    await expect(page.getByLabel('Nome')).toBeHidden()
  })

  test('excluir exige confirmação antes de qualquer coisa', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=config`)

    await page.getByRole('button', { name: /Excluir campanha/ }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toContainText('não pode ser desfeita')
    await dialog.getByRole('button', { name: 'Cancelar' }).click()
    await expect(page).toHaveURL(/tab=config/)
  })

  test('o convite gera um link de entrar na mesa', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=membros`)

    await page.getByRole('button', { name: /Convite/ }).click()
    await page.getByRole('button', { name: 'Gerar link' }).click()

    await expect(page.getByLabel('Link de convite')).toHaveValue(/\/join\/.+/)
  })
})

/**
 * Criar campanha e entrar por convite (ALE-80). The create case writes for
 * real, so it DELETES what it made — the seed is shared with every other spec
 * and a run that leaves campaigns behind poisons the next one.
 */
test.describe('Abrir e fechar uma crônica', () => {
  test('criar leva direto para a nova crônica, e excluir traz de volta', async ({
    page,
  }) => {
    const name = `E2E Descartável ${Date.now()}`
    await page.goto('/campaigns/new')

    await page.getByLabel('Nome').fill(name)
    await page.getByLabel('Descrição').fill('Criada e excluída pelo E2E.')
    await page.getByRole('button', { name: 'Abrir crônica' }).click()

    // Landed on the new chronicle's own page.
    await expect(page).toHaveURL(/\/campaigns\/\d+/)
    await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible()

    // Clean up through the UI, which also exercises the ALE-79 delete path.
    await page.goto(`${new URL(page.url()).pathname}?tab=config`)
    await page.getByRole('button', { name: /Excluir campanha/ }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Excluir' }).click()
    await expect(page).toHaveURL(/\/campaigns$/)
  })

  test('nome em branco não cria nada', async ({ page }) => {
    await page.goto('/campaigns/new')

    await page.getByRole('button', { name: 'Abrir crônica' }).click()

    await expect(page.getByText('Nome é obrigatório')).toBeVisible()
    await expect(page).toHaveURL(/\/campaigns\/new/)
  })
})

test.describe('Entrar por convite', () => {
  test('o link de convite abre a carta com o token', async ({ page }) => {
    await page.goto('/join/um-token-qualquer')

    // The /join/$token shim hands off to the real form.
    await expect(page).toHaveURL(/\/campaigns\/join\?token=um-token-qualquer/)
    await expect(page.getByRole('heading', { name: 'Entrar na mesa' })).toBeVisible()
  })

  // Um token morto tem que DIZER que morreu; sem isso o jogador só vê um botão
  // desabilitado e não sabe que precisa pedir outro link.
  test('convite expirado explica o que fazer', async ({ page }) => {
    await page.goto('/campaigns/join?token=token-que-nao-existe')

    await expect(page.getByText(/Convite inválido ou expirado/)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Entrar na mesa' })).toBeDisabled()
  })

  test('sem convite, pede o número da mesa e o herói', async ({ page }) => {
    await page.goto('/campaigns/join')

    await expect(page.getByLabel('Número da campanha')).toBeVisible()
    await expect(page.getByText('Qual herói entra na mesa?')).toBeVisible()
    // Nada escolhido ainda → não dá para entrar.
    await expect(page.getByRole('button', { name: 'Entrar na mesa' })).toBeDisabled()
  })
})

// The scene must FILL the width at every form factor — no horizontal body
// scroll. This is the deterministic version of the manual 6-resolution pass.
const VIEWPORTS = [
  { name: 'desktop', width: 1920, height: 1080 },
  { name: 'laptop', width: 1440, height: 900 },
  { name: 'tablet-landscape', width: 1024, height: 768 },
  { name: 'tablet-portrait', width: 768, height: 1024 },
  { name: 'mobile-landscape', width: 844, height: 390 },
  { name: 'mobile-portrait', width: 390, height: 844 },
]

// Every campaign leaf of the grimório, since they share the TomePage surface —
// a regression in it would otherwise only be caught on whichever one we spot-checked.
const SCENES = [
  { name: 'detalhe', path: `${CAMPAIGN}?tab=membros`, heading: /Snapshot Test ALE-33/i },
  { name: 'nova', path: '/campaigns/new', heading: /Abrir nova crônica/i },
  { name: 'convite', path: '/campaigns/join', heading: /Entrar na mesa/i },
]

test.describe('Campanha — responsivo (preenche a tela, sem overflow horizontal)', () => {
  for (const scene of SCENES) {
    for (const vp of VIEWPORTS) {
      test(`${scene.name}: sem scroll horizontal @ ${vp.name} (${vp.width}×${vp.height})`, async ({
        page,
      }) => {
        await page.setViewportSize({ width: vp.width, height: vp.height })
        await page.goto(scene.path)
        await expect(page.getByRole('heading', { name: scene.heading })).toBeVisible()
        const overflow = await page.evaluate(
          () =>
            document.documentElement.scrollWidth -
            document.documentElement.clientWidth,
        )
        expect(overflow, 'a página não deve rolar horizontalmente').toBeLessThanOrEqual(1)
      })
    }
  }
})
