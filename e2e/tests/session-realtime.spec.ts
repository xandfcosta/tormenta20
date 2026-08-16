import { expect, test } from '@playwright/test'

/**
 * O realtime com DOIS clientes na mesma mesa — o mecanismo que só um browser
 * (dois, na verdade) testemunha, e a única justificativa de e2e que o guia da
 * casa nomeia e que a suíte ainda não gastava.
 *
 * O que existia provava que UM cliente aperta a mão com o gateway. Ninguém
 * verificava o que a sessão ao vivo existe para fazer: o que o mestre faz
 * aparecer na tela do jogador sem ninguém recarregar nada. Um handler que
 * emitisse para o socket em vez de para a SALA passaria por todos os outros
 * testes — o mestre veria a própria ação e o jogador não veria nada.
 *
 * Escreve na seed e limpa atrás de si: o combatente tem nome único e sai no
 * fim, então a iniciativa termina como começou.
 */
test.describe('Sessão ao vivo — dois clientes', () => {
  test('o que o mestre adiciona aparece na tela do jogador', async ({ browser }) => {
    const eco = `Eco de teste ${Date.now()}`
    const mestre = await browser.newContext({ storageState: '.auth/user.json' })
    const jogador = await browser.newContext({ storageState: '.auth/player.json' })
    const telaDoMestre = await mestre.newPage()
    const telaDoJogador = await jogador.newPage()

    try {
      await telaDoMestre.goto('/campaigns/1/sessions/4')
      await telaDoJogador.goto('/campaigns/1/sessions/4')
      // Os DOIS conectados antes de agir: sem isso o teste mede uma corrida.
      await expect(telaDoMestre.getByRole('status', { name: 'Conectado' })).toBeVisible()
      await expect(telaDoJogador.getByRole('status', { name: 'Conectado' })).toBeVisible()

      // O formulário nasce fechado desde que se mediu o custo dele em altura
      // (ALE-122): um clique no topo o abre.
      await telaDoMestre.getByRole('button', { name: 'Combatente' }).click()
      await telaDoMestre.getByLabel('Nome').fill(eco)
      await telaDoMestre.getByRole('button', { name: 'Adicionar', exact: true }).click()

      // A tela do jogador não recarrega: o combatente chega pelo socket.
      await expect(telaDoJogador.getByText(eco)).toBeVisible()

      await telaDoMestre.getByRole('button', { name: `Remover ${eco}` }).click()
      await expect(telaDoJogador.getByText(eco)).toBeHidden()
    } finally {
      await mestre.close()
      await jogador.close()
    }
  })

  /**
   * O tabuleiro atravessa a mesa (ALE-124). É e2e pelo mesmo motivo do teste
   * acima e por um a mais: o estado do tabuleiro sai para SALAS POR PAPEL, e um
   * emit para o socket em vez da sala do jogador passaria por todo teste de
   * unidade — o mestre veria a própria grade e o jogador veria nada.
   *
   * Limpa atrás de si: encerra o tabuleiro no fim, então a sessão da seed
   * termina como começou.
   */
  test('o tabuleiro que o mestre abre aparece na tela do jogador', async ({ browser }) => {
    const lugar = `Cripta de teste ${Date.now()}`
    const mestre = await browser.newContext({ storageState: '.auth/user.json' })
    const jogador = await browser.newContext({ storageState: '.auth/player.json' })
    const telaDoMestre = await mestre.newPage()
    const telaDoJogador = await jogador.newPage()

    try {
      await telaDoMestre.goto('/campaigns/1/sessions/4')
      await telaDoJogador.goto('/campaigns/1/sessions/4')
      await expect(telaDoMestre.getByRole('status', { name: 'Conectado' })).toBeVisible()
      await expect(telaDoJogador.getByRole('status', { name: 'Conectado' })).toBeVisible()

      // Desktop Chrome do Playwright é 1280 de largura: abaixo de 1536 a cena do
      // mestre mostra uma região por vez, então o tabuleiro precisa ser escolhido
      // antes de existir na tela (ALE-124).
      await telaDoMestre.getByRole('button', { name: 'tabuleiro', exact: true }).click()
      // O teste cuida do próprio terreno: a sessão da seed pode ter ficado com um
      // tabuleiro aberto de outra rodada (ou de alguém usando o app em dev), e
      // esperar pelo estado vazio faria a suíte falhar por causa do ambiente.
      const encerrar = telaDoMestre.getByRole('button', { name: 'Encerrar o tabuleiro' })
      if (await encerrar.isVisible()) {
        await encerrar.click()
        await telaDoMestre.getByRole('dialog').getByRole('button', { name: 'Encerrar' }).click()
      }
      await telaDoMestre.getByRole('button', { name: 'Abrir tabuleiro' }).click()
      await telaDoMestre.locator('#board-place').fill(lugar)
      await telaDoMestre.getByRole('dialog').getByRole('button', { name: 'Abrir' }).click()
      await expect(telaDoMestre.getByRole('grid', { name: new RegExp(lugar) })).toBeVisible()

      // Sem recarregar nada: a grade chega pelo socket, na sala do JOGADOR.
      await expect(telaDoJogador.getByRole('grid', { name: new RegExp(lugar) })).toBeVisible()

      // E as peças também: "trazer a iniciativa" é o mestre montando a cena.
      await telaDoMestre.getByRole('button', { name: 'Trazer a iniciativa' }).click()
      await expect(telaDoJogador.locator('button[aria-label*=", coluna "]').first()).toBeVisible()

      await telaDoMestre.getByRole('button', { name: 'Encerrar o tabuleiro' }).click()
      await telaDoMestre.getByRole('dialog').getByRole('button', { name: 'Encerrar' }).click()
      await expect(telaDoJogador.getByRole('grid', { name: new RegExp(lugar) })).toBeHidden()
    } finally {
      await mestre.close()
      await jogador.close()
    }
  })
})
