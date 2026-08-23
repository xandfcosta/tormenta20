import { expect, test } from '@playwright/test'
import { abreAFila, acionaOCiclo, garanteACena, labelsNaFila } from './support/gm-scene'

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
/**
 * Esta spec dirige a sessão 5, e o `session.spec.ts` dirige a 4 — as duas são da
 * MESMA campanha (o jogador da seed é membro dela), e é essa separação que as
 * deixa correr em paralelo no CI sem se atropelar. Antes as duas escreviam na
 * sessão 4, e o sintoma não parecia corrida: era um número que não batia ou um
 * clique que caía no botão errado.
 *
 * SERIAL por dentro pelo mesmo motivo de sempre: estes testes escrevem uns
 * sobre os outros.
 */
test.describe.configure({ mode: 'serial' })

test.describe('Sessão ao vivo — dois clientes', () => {
  test('o que o mestre adiciona aparece na tela do jogador', async ({ browser }) => {
    const eco = `Eco de teste ${Date.now()}`
    const mestre = await browser.newContext({ storageState: '.auth/user.json' })
    const jogador = await browser.newContext({ storageState: '.auth/player.json' })
    const telaDoMestre = await mestre.newPage()
    const telaDoJogador = await jogador.newPage()

    try {
      await telaDoMestre.goto('/campaigns/1/sessions/5')
      await telaDoJogador.goto('/campaigns/1/sessions/5')
      // Os DOIS conectados antes de agir: sem isso o teste mede uma corrida.
      await expect(telaDoMestre.getByRole('status', { name: 'Conectado' })).toBeVisible()
      await expect(telaDoJogador.getByRole('status', { name: 'Conectado' })).toBeVisible()
      // A cena do jogador tem TRÊS superfícies e abre na ficha dele (ALE-129):
      // a iniciativa mora na Mesa, e é lá que ele olha para acompanhar o combate.
      await telaDoJogador.getByRole('button', { name: /Mesa/ }).click()
      // Sem CENA o servidor não manda fila nenhuma para a mesa (ALE-210), e o
      // eco chegaria só à tela do mestre — este teste passaria a medir a trava
      // em vez do broadcast.
      await garanteACena(telaDoMestre)

      // A fila do mestre é GAVETA desde a ALE-198, e é lá que mora a forma de
      // adicionar. O formulário dentro dela nasce fechado desde que se mediu o
      // custo dele em altura (ALE-122): um clique em "+ Combatente" o abre.
      const fila = await abreAFila(telaDoMestre)
      await fila.getByRole('button', { name: 'Combatente' }).click()
      await telaDoMestre.getByLabel('Nome').fill(eco)
      await telaDoMestre.getByRole('button', { name: 'Adicionar', exact: true }).click()

      // A tela do jogador não recarrega: o combatente chega pelo socket.
      await expect(telaDoJogador.getByText(eco)).toBeVisible()

      await fila.getByRole('button', { name: `Remover ${eco}` }).click()
      await expect(telaDoJogador.getByText(eco)).toBeHidden()
    } finally {
      await mestre.close()
      await jogador.close()
    }
  })

  /**
   * A CENA como cortina (ALE-210): o mestre desliga e a fila SOME da mesa,
   * enquanto continua inteira na tela dele.
   *
   * É e2e pela mesma razão que os dois vizinhos, e o guia da casa nomeia
   * exatamente esta: a trava vive em `redactForPlayers`, que alimenta as SALAS
   * POR PAPEL — o mestre recebe o estado cheio e o jogador uma cópia redigida,
   * pelo mesmo socket, no mesmo instante. Um handler que emitisse a cópia certa
   * para a sala errada (ou o estado cheio para as duas) passaria por todo teste
   * de unidade dos dois lados: o Go prova que a função redige, o vitest prova
   * que a tela desenha o que recebe, e nenhum dos dois prova que a mensagem foi
   * para a sala certa.
   *
   * As duas metades importam. Sumir da mesa é a regra; CONTINUAR na tela do
   * mestre é o que separa "redigi o broadcast" de "apaguei a fila" — e a
   * segunda leitura passaria verde afirmando só a primeira.
   */
  test('encerrar a cena tira a fila da mesa sem tirá-la do mestre', async ({ browser }) => {
    const eco = `Cortina de teste ${Date.now()}`
    const mestre = await browser.newContext({ storageState: '.auth/user.json' })
    const jogador = await browser.newContext({ storageState: '.auth/player.json' })
    const telaDoMestre = await mestre.newPage()
    const telaDoJogador = await jogador.newPage()

    try {
      await telaDoMestre.goto('/campaigns/1/sessions/5')
      await telaDoJogador.goto('/campaigns/1/sessions/5')
      await expect(telaDoMestre.getByRole('status', { name: 'Conectado' })).toBeVisible()
      await expect(telaDoJogador.getByRole('status', { name: 'Conectado' })).toBeVisible()
      await telaDoJogador.getByRole('button', { name: /Mesa/ }).click()
      await garanteACena(telaDoMestre)

      const fila = await abreAFila(telaDoMestre)
      await fila.getByRole('button', { name: 'Combatente' }).click()
      await telaDoMestre.getByLabel('Nome').fill(eco)
      await telaDoMestre.getByRole('button', { name: 'Adicionar', exact: true }).click()
      await expect(telaDoJogador.getByText(eco)).toBeVisible()

      // A gaveta é modal abaixo de 1280 e cobriria o pé do trilho.
      await telaDoMestre.keyboard.press('Escape')
      await expect(fila).toBeHidden()
      await acionaOCiclo(telaDoMestre, 'Encerrar cena', 'Encerrar a cena?')

      await expect(telaDoJogador.getByText(eco)).toBeHidden()
      await expect.poll(async () => labelsNaFila(telaDoMestre)).toContain(eco)

      // E volta pelo mesmo caminho: a fila estava guardada o tempo todo.
      await garanteACena(telaDoMestre)
      await expect(telaDoJogador.getByText(eco)).toBeVisible()

      const limpando = await abreAFila(telaDoMestre)
      await limpando.getByRole('button', { name: `Remover ${eco}` }).click()
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
      await telaDoMestre.goto('/campaigns/1/sessions/5')
      await telaDoJogador.goto('/campaigns/1/sessions/5')
      await expect(telaDoMestre.getByRole('status', { name: 'Conectado' })).toBeVisible()
      await expect(telaDoJogador.getByRole('status', { name: 'Conectado' })).toBeVisible()
      // O jogador vai para a superfície do tabuleiro — é onde ele o vê agora.
      await telaDoJogador.getByRole('button', { name: /Tabuleiro/ }).click()

      // O teste traz o PRÓPRIO combatente: a iniciativa da seed do CI está VAZIA,
      // e "trazer a iniciativa" para um tabuleiro sem ninguém não põe peça
      // nenhuma — o teste passava aqui e falhava lá, que foi exatamente o que
      // aconteceu no primeiro push (ALE-124).
      const fila = await abreAFila(telaDoMestre)
      await fila.getByRole('button', { name: 'Combatente' }).click()
      await telaDoMestre.getByLabel('Nome').fill(figurante)
      await telaDoMestre.getByRole('button', { name: 'Adicionar', exact: true }).click()
      // Escopado à LINHA dentro da gaveta, e não a "qualquer texto com esse
      // nome": desde a ALE-184 o avanço de turno anuncia o próximo combatente
      // pelo NOME, então o mesmo texto aparece na lista e dentro do botão.
      await expect(fila.getByRole('button', { name: figurante, exact: true })).toBeVisible()

      // O tabuleiro é a superfície PERMANENTE desde a ALE-198: ele já está na
      // tela, e o que precisa sair da frente é a gaveta da fila.
      await telaDoMestre.keyboard.press('Escape')
      await expect(fila).toBeHidden()
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
      // "Todos" porque o figurante é um NPC digitado à mão, e o padrão do
      // diálogo traz só os jogadores (ALE-204).
      await telaDoMestre.getByRole('button', { name: 'Trazer a iniciativa' }).click()
      const quemVem = telaDoMestre.getByRole('dialog')
      await quemVem.getByRole('button', { name: 'Todos' }).click()
      await quemVem.getByRole('button', { name: /^Trazer \d/ }).click()
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

      // Sai como entrou: o combatente do teste volta para fora da iniciativa.
      // Tirar alguém da fila é trabalho NA fila, e ela mora na gaveta desde a
      // ALE-198 — o tabuleiro continua na tela por baixo.
      const paraLimpar = await abreAFila(telaDoMestre)
      await paraLimpar.getByRole('button', { name: `Remover ${figurante}` }).click()
      // Escopado à LINHA, pelo mesmo motivo da asserção lá em cima: o avanço de
      // turno anuncia o próximo combatente pelo nome (ALE-184), então esperar
      // que o TEXTO suma casaria também com o rótulo do botão.
      await expect(paraLimpar.getByRole('button', { name: figurante, exact: true })).toBeHidden()
    } finally {
      await mestre.close()
      await jogador.close()
    }
  })
})
