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
    const figurante = `Peça de teste ${Date.now()}`
    const mestre = await browser.newContext({ storageState: '.auth/user.json' })
    const jogador = await browser.newContext({ storageState: '.auth/player.json' })
    const telaDoMestre = await mestre.newPage()
    const telaDoJogador = await jogador.newPage()

    try {
      await telaDoMestre.goto('/campaigns/1/sessions/4')
      await telaDoJogador.goto('/campaigns/1/sessions/4')
      await expect(telaDoMestre.getByRole('status', { name: 'Conectado' })).toBeVisible()
      await expect(telaDoJogador.getByRole('status', { name: 'Conectado' })).toBeVisible()

      // O teste traz o PRÓPRIO combatente: a iniciativa da seed do CI está VAZIA,
      // e "trazer a iniciativa" para um tabuleiro sem ninguém não põe peça
      // nenhuma — o teste passava aqui e falhava lá, que foi exatamente o que
      // aconteceu no primeiro push (ALE-124).
      await telaDoMestre.getByRole('button', { name: 'Combatente' }).click()
      await telaDoMestre.getByLabel('Nome').fill(figurante)
      await telaDoMestre.getByRole('button', { name: 'Adicionar', exact: true }).click()
      await expect(telaDoMestre.getByText(figurante)).toBeVisible()

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

      await expect(telaDoJogador.getByRole('grid', { name: new RegExp(lugar) })).toBeVisible()

      // E as peças também: "trazer a iniciativa" é o mestre montando a cena, e a
      // peça chega na tela do jogador com o nome de quem ela representa.
      await telaDoMestre.getByRole('button', { name: 'Trazer a iniciativa' }).click()
      await expect(
        telaDoMestre.getByRole('button', { name: new RegExp(`^${figurante}, coluna`) }),
      ).toBeVisible()

      // O mestre PÕE a peça na origem antes de o jogador olhar, e isso não é
      // conveniência de teste: a janela do jogador é pequena (o rail tem ~350px)
      // e o plano é INFINITO, então "a peça está na tela dele" depende de ONDE
      // ela está. Na origem é determinístico nos dois ambientes — e de quebra
      // isto passa a provar o movimento atravessando as duas telas (ALE-124).
      await telaDoMestre
        .getByRole('button', { name: new RegExp(`^${figurante}, coluna`) })
        .click()
      // `exact` porque "Coluna 0, linha 0" é SUBSTRING de "Zumbi 1, coluna 0,
      // linha 0" — a peça e o quadrado dela colidem no nome acessível. E o
      // destino é (0,-1), um quadrado VAZIO: um clique num quadrado ocupado cai
      // na peça que está por cima, não no quadrado.
      await telaDoMestre
        .getByRole('button', { name: 'Coluna 0, linha -1', exact: true })
        .click()

      await expect(
        telaDoJogador.getByRole('button', { name: `${figurante}, coluna 0, linha -1` }),
      ).toBeVisible()

      await telaDoMestre.getByRole('button', { name: 'Encerrar o tabuleiro' }).click()
      await telaDoMestre.getByRole('dialog').getByRole('button', { name: 'Encerrar' }).click()
      await expect(telaDoJogador.getByRole('grid', { name: new RegExp(lugar) })).toBeHidden()

      // Sai como entrou: o combatente do teste volta para fora da iniciativa. Não
      // precisa trocar de região para isso — nesta largura a iniciativa é a
      // espinha e fica sempre na tela; só a segunda coluna alterna (ALE-130).
      await telaDoMestre.getByRole('button', { name: `Remover ${figurante}` }).click()
      await expect(telaDoMestre.getByText(figurante)).toBeHidden()
    } finally {
      await mestre.close()
      await jogador.close()
    }
  })
})
