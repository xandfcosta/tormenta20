# Glossário — a linguagem ubíqua da mesa

Uma palavra por conceito, e um conceito por palavra. Este arquivo existe porque
o app tinha **onze** lugares onde isso não valia: `campanha` e `crônica` na mesma
tela, `cena` significando quatro coisas, três predicados diferentes vestidos da
palavra "jogador". Nada disso era descuido isolado — era a costura da linguagem
não estar escrita, então cada issue reabria a decisão e escolhia diferente.

**O que ele governa:** todo texto NOVO que o usuário lê e todo nome NOVO no
código. Prosa e identificadores que já existem não ficam errados
retroativamente — os renomes andam issue a issue, e a coluna "proibido" é o que
impede o décimo segundo caso.

**Como se lê uma linha:** o termo canônico é o que se escreve na tela; o
identificador é como ele se chama no código; os proibidos são sinônimos que
alguém já usou e que não voltam.

---

## A. O que existe na mesa

| Canônico | Identificador | Proibidos | O que é |
|---|---|---|---|
| **campanha** | `campaign` | ~~crônica~~ | O que o mestre abre e configura, e onde vivem as fichas, o elenco, os lugares e as sessões. Decisão do dono, 2026-08-22: "campanha" é a palavra do hobby e é a que o jogador procura; `crônica` sai das telas. |
| **sessão** | `session` | ~~partida~~ | Um encontro da campanha. Tem número, título e histórico; começa e encerra. |
| **mesa** | — | — | **As pessoas conectadas agora.** Metonímia, nunca entidade: "a mesa vê a fila" é sobre quem está olhando a tela. Nunca use "mesa" para campanha nem para sessão. |
| **ficha** | `character` | — | O personagem completo com atributos, perícias e poderes. |
| **personagem** | `character` | ~~PC~~, ~~herói~~ | A criatura que uma pessoa joga. |
| **jogador** | `member.role === 'player'` | — | **A PESSOA.** Papel no roster da campanha, e não um tipo de linha. |
| **mestre** | `role === 'gm'` | ~~GM~~ (na tela) | Quem conduz. `gm` continua no código e no fio. |
| **elenco** | `cast` | — | Quem existe na campanha fora da iniciativa — jogadores e NPCs do mestre (ALE-212). |
| **regra opcional** | `ignoredRules` | ~~regra da casa~~, ~~regra enforçada~~ | Regra do livro que o mestre pode desligar na campanha — o próprio T20 diz "O mestre pode ignorar essa regra" (p141). Na tela: **Regras da campanha**, e cada chave mostra a regra LIGADA. No código o campo nomeia o que está DESLIGADO, e isso é proposital: valor zero significa "tudo em vigor", que é o padrão do livro. Catálogo em ALE-221. |
| **app instalado** | `manifest.webmanifest` | ~~atalho~~, ~~PWA~~ (na tela) | O app aberto pela própria janela, sem barra de endereço, depois de "Adicionar à Tela de Início". **Não é a Tela cheia** — ver a colisão C7. `atalho` é proibido porque é o que se ganha SEM manifest servido por HTTPS: um ícone que abre uma aba comum, com a barra de volta. |
| **tela cheia** | `shared/lib/fullscreen.ts` | ~~fullscreen~~ (na tela) | O gesto do menu do Hub que estica a janela ATUAL pela Fullscreen API. Some quando a aba fecha, e o iPhone não o tem. Continua sendo a saída de quem não instalou (ALE-118). |

## B. O combate

| Canônico | Identificador | Proibidos | O que é |
|---|---|---|---|
| **fila** | `tracker`, `initiative` | ~~rastreador~~ | A ordem do combate. Na tela: "Fila do combate". |
| **combatente** | `InitiativeEntry` | ~~linha~~ (em texto visível) | Uma entrada na fila. "Linha" só vale falando do elemento de UI, nunca do objeto. |
| **iniciativa** | `initiative` | — | **Três coisas, e o contexto separa:** a perícia do livro, o VALOR rolado (`entry.initiative`) e, por extensão, a fila. Só a perícia é do livro — as outras duas são nossas e cedem quando houver ambiguidade. |
| **cena** | `SceneActive` | — | O estado que o mestre liga e desliga; enquanto ela dura, a mesa recebe a fila (ALE-210). **Ver a colisão C1.** |
| **rodada / vez** | `round` / `turnIndex` | ~~turno~~ para "de quem é a vez" | "Turno" é o que se GASTA (ação de movimento etc.); "vez" é de quem é. |
| **recuperação** | `rest` | — | O painel. As ações dentro dele são **descanso de cena** e **descanso de dia**, que é a palavra do livro (T20 p105) — a seção diz o que devolve, as ações dizem o que o livro chama. |
| **situacional** | `conditional` | ~~condição~~, ~~modificador opcional~~ | O modificador OPT-IN que o jogador liga na própria ficha e que muda o cálculo dela — Fúria, Ataque Poderoso, os homebrew. Na tela: "Situação — opt-in por contexto". **Não é condição** — ver C6. |
| **postura** | `stance` | — | O situacional que cobra PM para entrar. O que foi pago fica registrado, para sair não devolver. Na tela: "Posturas ativas". |
| **uso** | `powerUse` | ~~carga~~, ~~gasto~~ | Quantas vezes um poder "1/cena" ou "1/dia" já foi usado. ~~carga~~ está tomada: desde a ALE-215 ela é o peso que a mochila carrega. |

## C. O tabuleiro

| Canônico | Identificador | Proibidos | O que é |
|---|---|---|---|
| **tabuleiro** | `board` | — | A superfície aberta na mesa agora. |
| **peça** | `BoardToken` | ~~token~~ (na tela) | A representação de alguém no mapa. **Não é o combatente** — a ponte é o `entryId`, e uma peça pode existir sem linha na fila (a porta, o baú). |
| **lugar** | `BoardPlace` | ~~cena guardada~~ | O mapa GUARDADO no acervo da campanha, com as peças onde ficaram. |
| **marcador** | `BoardMarker` | — | O ponto apontado no mapa (ALE-195). Nasce escondido. |
| **terreno** | — | ~~obstáculo~~ | O que o QUADRADO faz com quem está nele ou atrás dele. É a FAMÍLIA, e tem quatro espécies: **difícil**, **cobertura**, **camuflagem** e **elevado** (T20 p238, Tabela 5-3; os acidentes que os produzem estão em p267-268). Nasceu significando só "difícil", que era a única espécie implementada — a palavra foi promovida quando as outras três chegaram. **Não confundir com o CHÃO do lugar**, que é a aparência, nem com a **moldura**, que é o recorte. Não tem identificador próprio de propósito: cada espécie tem o seu, e um campo `terreno` genérico convidaria a guardar as quatro numa string. |
| **difícil** | `Difficult` | ~~lento~~, ~~pesado~~ | Espécie de terreno: entrar custa o dobro (p238). É a ÚNICA que o motor consome hoje — ela entra no custo do movimento. |
| **cobertura** | `Cover` | ~~proteção~~, ~~abrigo~~ | Espécie de terreno: quem está nela recebe **+5 na Defesa** (p238). Trincheira, árvore estreita, a lateral de uma carroça. A cobertura TOTAL do livro (o alvo não pode ser atacado) não é pintável — ela vem de parede, que é ausência de quadrado. |
| **camuflagem** | `Concealment` | ~~ocultação~~, ~~névoa~~ | Espécie de terreno: quem está nela dá **20% de chance de falha** ao atacante (p238). Folhagens, moitas. A camuflagem TOTAL (50%) é escuridão, que é do lugar inteiro e não de um quadrado. |
| **elevado** | `Elevated` | ~~alto~~, ~~plataforma~~ | Espécie de terreno: quem ataca DE LÁ recebe **+2 no ataque** (p238). É a única espécie que beneficia quem está nela em vez de proteger — por isso o desenho dela não pode se parecer com o das outras três. |
| **chão** | `BoardState.Terrain` | ~~cenário~~ | A APARÊNCIA do lugar — pedra, taverna, floresta, ermo, cripta, papel. É o que o `.chao-*` pinta. **Não é o terreno difícil** da linha acima, que é regra de movimento e não pintura: um é como a cena se PARECE, o outro é quanto custa atravessá-la. `~~cenário~~` está proibido porque em T20 cenário é ARTON, o mundo da campanha — a SPA usa essa palavra no rótulo do diálogo (ALE-124) e o piloto não a herda. O identificador do fio continua `terrain` e não se conserta por palpite: renomeá-lo mexe no acervo de lugares gravado. |
| **cortina** | `curtained` | ~~oculto~~, ~~privado~~, ~~rascunho~~ | O tabuleiro EXISTE para o mestre e a mesa vê uma cortina no lugar dele (ALE-202). É durante a sessão, com a mesa presente — montar a taverna enquanto eles olham a cripta. **Não é o rascunho de lugar**, que é preparação FORA da sessão, sem ninguém conectado: tempos diferentes, gestos diferentes, e a decisão do dono foi que os dois convivem. Também **não é** "o mestre ainda não abriu um tabuleiro", que continua significando exatamente isso. A cortina é DESENHADA: o jogador sabe que vem cena, sem ver qual. |

## D. Os monstros do mestre

| Canônico | Identificador | Proibidos | O que é |
|---|---|---|---|
| **verbete** | `MonsterID` | — | A entrada IMUTÁVEL do bestiário do livro. |
| **bloco de criatura** | `CreatureID` | — | O bloco EDITÁVEL que o mestre escreveu, e que pertence à campanha (ALE-137). Uma linha tem verbete ou bloco, nunca os dois. |
| **NPC** | `type: 'npc'` | ~~PC~~ | O PAPEL de uma linha na fila: não é ficha de jogador. Ortogonal a verbete/bloco. **Na tela o par é `Ficha` / `NPC`** — decisão do dono, 2026-08-24, ao tirar o `PC` proibido dos DOIS apps. O oposto de NPC na fila é **ficha** e não "personagem": é a pergunta que a tabela de colisões faz do `type === 'character'` (linha abaixo), e "Personagem" não caberia no selo de uma linha de 390px. Cuidado ao mexer: `Ficha` é 2,5× mais largo que o `PC` que estava lá, e esse selo JÁ transbordou uma vez (`initiative-card.tsx:428`) — os guardas de transbordo do `session.spec.ts` são obrigatórios. |
| **criatura** | — | — | O guarda-chuva ("Adicionar criatura", "Buscar criatura"). Use quando as três acima não importam. |

---

## E. Colisões abertas — NÃO conserte por palpite

Estão aqui para que ninguém as "arrume" escolhendo sozinho. Cada uma precisa de
uma decisão do dono antes do renome.

**C1 — `cena` carrega quatro conceitos.**
1. o estado da sessão (`SceneActive`, `session-scene-start/end`);
2. a **duração de efeito** do livro (`scope: "scene"`, `DeleteEffectsByScope`);
3. o mapa guardado do tabuleiro (`board-place-scene`, `parseScene` → devolve um `BoardState`);
4. a casca visual da UI (`SceneShell`, `scene-grimorio`, `SceneContainerProvider`, `scene-nav`).

Consequência medida: `endScene` existe **três vezes** no pacote `api` —
`session_state.go` e `session_store.go` desligam a cena, e
`character_effects.go` limpa a duração "cena" de UMA ficha. Os três nomes
continuam, e o renome ainda espera decisão do dono.

O BUG que a colisão escondia está fechado (ALE-220): `onSceneEnd` não limpava
efeito nenhum, e quem limpava era só a Recuperação com `scope='scene'` — o
mestre encerrava a cena e a bênção que dura "cena" continuava na ficha. Agora os
dois gestos passam pelo MESMO `expirePartyScene`, e o livro é quem manda: "a
habilidade dura uma cena inteira, encerrando-se quando esse momento da história
acaba" (p227), e o início e o fim de uma cena são dados "pelo andamento da
história" (p11) — que é o que o mestre declara ao clicar em Encerrar cena.
**Era bug, não nome** — e o nome foi o que o escondeu.

**C2 — `place` é rótulo, entidade e marcador.** `BoardState.Place` é a *string*
com o nome da cena aberta; `BoardPlace{id, name, …}` é o lugar guardado, e o
rótulo dele chama-se `name`. Mesmo dado, dois campos com nomes diferentes. E
"Marcar um lugar" (`BoardMarker`) usa a palavra num terceiro sentido.

**C3 — `RestScope` mente na metade dos usos.** `applyEffect(entryId, spellId,
scope?: RestScope)` (`realtime.ts:256`): ali `scope` é *duração de magia*, regra
do livro, e não escopo de descanso. Conserto óbvio e sem UX: um `EffectScope`
próprio.

**C4 — quatro predicados vestidos de "jogador".** Eles respondem a **perguntas
diferentes** e por isso não se unificam; o que falta é o nome dizer qual:

| Predicado | Pergunta que responde | Onde |
|---|---|---|
| `entry.type === 'character'` | esta linha é ficha ou é NPC? | `board_state.go:423`, `populate-dialog.tsx:18` |
| `member.role === 'player'` | a PESSOA dona é jogador ou mestre? | `cast-panel.tsx:54`, `listPlayerCombatants` |
| `myCharacterIds.has(id)` | é meu? | `tracker-rules.ts` (`myCharacterIdsOf`) |
| `token.kind === 'character'` | esta peça se desenha como PC? | `board-view.tsx:571` |

Decisão do dono, 2026-08-22 (ALE-204): o atalho "trazer os jogadores" usa
`type === 'character'`, porque é o **mesmo** predicado com que o servidor escolhe
o lado do mapa — o atalho põe as peças exatamente na fileira do grupo. O efeito
colateral é conhecido e aceito: um PC do mestre que esteja na fila vem junto.

**C5 — o README ainda usa três palavras para campanha.** "as dos outros aparecem
nas **Crônicas**", "abre e edita qualquer **mesa**". Sai no próximo passe.

**C6 — `condition` e `conditional` são conceitos DIFERENTES a uma letra de
distância.** A **condição** é do livro (p394-395): Caído, Atordoado, Cego. Ela
chega de fora — o mestre aplica, a magia impõe — e mora na coluna JSON
`characters.activeConditions`. O **situacional** é escolha do jogador na própria
ficha: Fúria, Ataque Poderoso, os homebrew. Ele muda o cálculo da ficha e, desde
a ALE-222, mora na tabela `character_conditionals`.

Os dois ficam a uma letra de distância no autocomplete, e trocá-los significa
aplicar Fúria como se fosse Caído. Ao escrever query nova, confira o plural
inteiro.

E os dois guardam-se em FORMAS diferentes — coluna JSON de um lado, tabela do
outro — sem que isso signifique nada sobre os conceitos: é o sqlc, que não
enxerga `ALTER TABLE ADD COLUMN` de arquivo de migração novo (ALE-124), então
espelhar a coluna não era opção. Não leia a assimetria como intenção.

A tela não desambigua sozinha porque ela nem tenta: diz "Condições" de um lado
e "Situação" do outro, que é a separação certa. É só no código que os dois se
parecem.

**C7 — "tela cheia" e "app instalado" resolvem o mesmo incômodo por caminhos
diferentes, e a issue que os separou (ALE-118) quase os fundiu.** A tela cheia é
um GESTO sobre a janela de agora: o jogador clica no menu do Hub, o browser
estica, e fechar a aba desfaz. O app instalado é uma JANELA OUTRA, que o sistema
guarda com ícone próprio e abre sem barra nenhuma.

Não são etapas de uma mesma coisa e nenhum substitui o outro: o desktop e quem
não instalou vivem do primeiro, e o iPhone não tem o primeiro (o Safari não
expõe Fullscreen API para elemento), então lá o segundo é o único caminho.
Escrever "tela cheia" no texto de instalar — ou o contrário — manda metade da
mesa para o botão que o aparelho dela não tem.

---

## E-bis. Os contextos do servidor (ALE-254)

Nome de pacote é identificador, então o glossário manda nele. E a regra da
costura já responde qual idioma: **fronteira** é o que fala inglês — tabela,
campo JSON, evento SSE, rota HTTP. Nome de pacote interno não é fronteira, então
contexto de domínio entra em **pt-BR**.

| termo | no código | proibido | o que é |
| -- | -- | -- | -- |
| **ao vivo** | `aovivo/`, `live*` | ~~mesa~~ (como pacote), ~~tempo real~~ | **O REGIME: a sessão enquanto está acontecendo.** O que existe só enquanto há gente conectada — o estado da fila em memória, a entrega por SSE, a presença, e a autorização de quem está na sessão. Nomeia o regime e não as pessoas, o que o distingue de `mesa`, e não a linha do banco, o que o distingue de `sessão`. O código já dizia `mountLiveRoutes`, `liveAccess` e `liveCtx` antes de a palavra existir aqui. |
| **plataforma** | `plataforma/` | ~~util~~, ~~common~~, ~~shared~~ | **O que não é domínio nenhum**: responder e decodificar HTTP, validar corpo, ler config, negociar codificação. Existe para NÃO virar o saco onde tudo cabe — se um conceito do jogo entrar aqui, a fronteira está errada. Os nomes proibidos são os que convidam exatamente esse acúmulo. |

**Por que `mesa` NÃO serve para o pacote**, e vale ficar escrito porque a
tentação vai voltar: a linha da seção A diz que "mesa" é metonímia das pessoas
conectadas, **nunca entidade**, e nunca serve para campanha nem para sessão. Um
pacote guardando o estado da sessão faria as duas coisas proibidas de uma vez, e
criaria a terceira acepção da palavra que a linha existe para impedir.

---

## F. A costura PT/EN

O domínio é pt-BR e o código é misto — e isso não estava escrito em lugar
nenhum, que é a raiz de metade desta lista. Em `board-region.tsx` convivem
`selectedToken` e `linhasNoMapa`, e `board` e `cena` denotam o mesmo tipo no
mesmo arquivo.

A regra, daqui para frente:

- **A tela fala pt-BR.** Sem exceção, incluindo `aria-label` e `title` (o Kobalte
  injeta rótulos em inglês — passe o seu, mesmo quando a palavra está escrita).
- **A fronteira fala inglês**: nomes de tabela, campos JSON, eventos SSE e
  rotas HTTP. Trocá-los quebra clientes e migrações, e o ganho é estético.
- **Dentro de uma função, o idioma é o do conceito.** Nome de domínio que o
  glossário canoniza em pt-BR entra em pt-BR (`combatente`, `peça`, `lugar`);
  mecânica de programação fica em inglês (`index`, `next`, `pending`).
- **Um arquivo não mistura os dois nomes do MESMO conceito.** Se o arquivo já
  chama `board`, não introduza `cena` ao lado; siga o que está lá e anote a
  dívida.

---

Termo novo, ou conceito que ainda não está aqui: **escreva a linha antes de
escrever o código.** Nomear depois é como este arquivo nasceu.
