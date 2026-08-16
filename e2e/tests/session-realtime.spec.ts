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
})
