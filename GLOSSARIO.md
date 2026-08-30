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
| **buscador do livro** | `buscador` | ~~paleta~~, ~~palette~~, ~~busca global~~ | A caixa que o ⌃K abre em qualquer cena e que procura nas 1.072 entradas do livro de uma vez (ALE-264). **Não é a busca da cena**, que é o campo que o `/` foca e que filtra a lista da tela atual — uma procura no LIVRO, a outra estreita o que já está na frente. O sinal se chama `buscador` justamente para não colidir com o `busca` das cenas, que vive no mesmo documento. |
| **livro** | `…DoLivro` | ~~manual~~, ~~PDF~~ (na tela) | O Tormenta 20 impresso, e a AUTORIDADE das regras: `criaturasDoLivro`, `magiaDoLivro`, `condicaoDoLivro` são entradas dele. Quando o ARQUIVO importa — o PDF que o servidor entrega em `/piloto/livro` —, diga **PDF do livro**: quem serve é a mesa, e serve só se o dono configurar `LIVRO_PDF` (ALE-264). |
| **perícia** | `pericias` | ~~habilidade~~, ~~skill~~ | Uma das 29 do livro (T20 p115, Tabela 2-1). Ganhou catálogo na ALE-264: existia como lista de nome e atributo dentro do `options.json`, sem página e sem as DUAS regras que o livro imprime ao lado do nome — **só treinada** (sem treinamento nem se rola) e **penalidade de armadura**. As três com penalidade são as mesmas que o motor já conhecia, e há teste costurando as duas fontes. **Não confundir com poder**: perícia se ROLA, poder se TEM. |
| **proficiência** | `proficiencies` | ~~treinamento~~, ~~habilitação~~ | Saber usar uma categoria de arma, armadura ou escudo. São SETE — armas simples, marciais, exóticas e de fogo (T20 p142), armaduras leves e pesadas, e escudos (p148) — e a classe concede as dela numa linha só ("Proficiências. Armas marciais e escudos.", p36–83). **Não confundir com perícia**: perícia se ROLA e tem graus de treinamento; proficiência se TEM ou não se tem, e o que falta vira PENALIDADE (−5 no ataque, p142; a penalidade da armadura nas perícias de Força e Destreza, p148). **Quem sabe usar armadura pesada sabe usar a leve, e isso o livro NÃO diz** — é decisão de produto, e está escrita e presa por teste em `piloto_ficha_proficiencias.go`. |
| **escola de magia** | `escolas-de-magia` | ~~tipo de magia~~, ~~classe de magia~~ | A família de uma magia — **Abjuração**, **Evocação**, **Ilusão** (T20 p172, oito). O livro imprime a abreviatura ao lado ("Abjuração (Abjur)") e as tabelas dele usam a forma curta. Ganhou catálogo na ALE-264: a magia guardava `school: "evocacao"`, isso decidia o filtro, e o nome não aparecia em cartão nenhum. **Escola conta como tipo de efeito** — é o livro que diz —, então as duas famílias se tocam. |
| **tipo de efeito** | `tipos-de-efeito` | ~~tag~~, ~~categoria~~ | A família de um efeito — **Medo**, **Mental**, **Movimento**, **Metabolismo** (T20 p228). O livro a imprime em itálico no fim da condição ("Abalado … *Medo*."), e ela é o que decide imunidade: "criatura com imunidade a medo não será afetada por efeitos do tipo medo". No dado a condição a guarda como CHAVE (`tags: ["cansaco"]`, sem acento); na tela sai o nome do livro. Ganhou catálogo próprio na ALE-264 porque não havia para onde o elo apontar. |
| **elo** | `eloParaOAcervo` | ~~link~~ (no código) | A palavra que leva ao verbete dela: a tag da condição leva ao tipo de efeito, "Agrava para Apavorado" leva à condição, o poder concedido leva ao poder. **Não é o botão do livro** — o elo anda DENTRO do acervo, o botão sai para o PDF. Os dois usam o mesmo sublinhado pontilhado de propósito: são o mesmo gesto ("leva para outro lugar") e desenhá-los diferente ensinaria que não são. O clique MOSTRA o verbete numa caixa sobre a cena; o `href` (`?aba=X&entrada=<id>`) fica para o ctrl+clique e o link copiado. |
| **entrada** | `?entrada=<id>` | ~~item~~, ~~registro~~ | O endereço de UM verbete: a aba dele mostrando só ele. **Não é busca** — `?busca=Medo` procura o termo nos oito catálogos e mostra os grupos; `?entrada=medo` é o Medo. A diferença nasceu de um defeito de UX: o elo endereçava por busca, e clicar num conceito caía numa lista onde ele era o quinto grupo. |
| **leitor** | `leitor` | ~~visualizador~~, ~~viewer~~ | A cena que abre o PDF do livro na página do verbete e o DESTACA (ALE-264). **Não é o visualizador do navegador**, que continua a um clique de distância ("abrir fora") — a distinção importa porque os dois existem lado a lado: o leitor mostra uma página por vez com o termo marcado, o visualizador tem busca, miniaturas e impressão. Ele existe por medição: o Chrome ignora `#search=` e transfere o arquivo inteiro (85 MiB) para abrir uma página; o leitor destaca e custou 1 MiB. |
| **página do livro** | `BookPage` | ~~folha~~ | O número IMPRESSO no rodapé, que é o que o catálogo grava e o que a ficha mostra ("p289"). **Não é a página do ARQUIVO**, que é a que `#page=N` conta — ver `abertura` abaixo. |
| **abertura** | `LivroAbertura` | ~~offset~~, ~~deslocamento~~ | Quantas páginas o arquivo tem ANTES da página impressa 1: 6 no PDF da casa, medido pelo rodapé. `deslocamento` está proibido porque em T20 é o quanto uma criatura anda. |
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
| **tabuleiro** | `board` | — | Uma superfície aberta na mesa agora. **São vários** desde a ALE-205 — o grupo se separou e a cripta não pode custar a taverna —, com teto de oito por sessão. Cada um tem a própria **cortina**, as próprias peças e o próprio terreno; o que NÃO é dele é a **janela**, que é de quem olha. Era "a superfície aberta na mesa agora", no singular, e a palavra "a" era o schema falando: `session_boards` tinha `sessionId INTEGER PRIMARY KEY`. |
| **aba** (do tabuleiro) | `abaDoTabuleiro`, `tabuleiro-aba` | ~~guia~~, ~~painel~~ | Uma das fichas que trocam qual **tabuleiro** a pessoa está olhando, no painel do canto superior esquerdo. A barra só nasce a partir de DOIS abertos. **A aba ativa é de cada PESSOA, não da mesa** — o jogador que desceu na cripta abre a aba da cripta porque quer —, e ela mora no SERVIDOR (`asAbasEscolhidas`), pelo mesmo motivo da lente: o stream não pergunta nada a ninguém. A exceção é **mostrar à mesa**, a linha abaixo. **Colide com as abas do grimório** (`?aba=X`, que escolhem catálogo) **e com as do editor de NPC** (Números / Ataques / Perícias): são o mesmo desenho de tela e coisas diferentes, então o identificador nunca é só `aba` — foi por isso que o gesto virou `trocaDeTabuleiro` e não `escolheAAba`, que já existia. |
| **mostrar à mesa** | `Puxa`, `mostraAMesaEstaAba` | ~~forçar~~, ~~travar~~ | O "parem tudo e olhem isto": o mestre traz TODA a mesa para a **aba** que ele está olhando, e traz também para a superfície do Tabuleiro — quem estava na superfície Mesa não veria nada sem isso. É um EMPURRÃO e não uma trava (decisão do dono, 2026-08-30): assim que a pessoa escolhe qualquer aba ela volta a decidir sozinha, e a tira que a avisa carrega a saída ("Voltar para a Taverna", ou "Continuar aqui" para quem já estava lá). `~~forçar~~` está proibido porque foi a palavra da issue enquanto o desenho era prender: ele descreve o que a feature NÃO faz. Alcança quem nunca escolheu aba nenhuma porque é um CONTADOR da sessão, e não uma escrita na escolha de cada um. |
| **peça** | `BoardToken` | ~~token~~ (na tela) | A representação de alguém no mapa. **Não é o combatente** — a ponte é o `entryId`, e uma peça pode existir sem linha na fila (a porta, o baú). |
| **pôr no mapa** | `poeNoMapa`, `Populate` | ~~trazer o grupo~~ (para o tabuleiro), ~~povoar~~ | Fazer nascer uma **peça** para um **combatente** da fila que ainda não tem uma. Idempotente: quem já está no mapa não duplica. **Não é "+ Adicionar grupo"**, que é o gesto da FILA — aquele põe os personagens do grupo na iniciativa e não toca no tabuleiro. Os dois têm o mesmo verbo na boca do mestre e destinos diferentes, e a palavra precisou de linha porque `trazOGrupo` já estava tomado pela fila quando o tabuleiro precisou do gesto. O padrão é o SEGURO (ALE-204): o atalho põe só as **fichas**, e trazer a fila inteira exige escolher — ela inclui o vilão montado para aparecer no terceiro turno. |
| **lugar** | `BoardPlace` | ~~cena guardada~~ | O mapa GUARDADO no acervo da campanha, com as peças onde ficaram. **A IDENTIDADE DELE É O NOME** dentro da campanha, e não o id: é assim que o `Archive` decide se sobrescreve ou cria, então encerrar a taverna duas vezes produz UMA taverna — e é por isso que o acervo sabe dizer qual lugar está aberto agora numa **aba** (ALE-205, fatia 3), casando o nome com as cenas na mesa. A consequência, para ninguém a redescobrir: uma cena aberta DO ZERO com o nome de um lugar guardado é tratada como aquele lugar, que é a mesma conta que o arquivamento fará quando ela fechar. **Reabrir não troca mais nada de lugar**: ele acrescenta uma aba, e a cena que estava na mesa continua onde estava. |
| **fantasma** | `Fantasma`, `tabuleiro-peca-fantasma` | ~~sombra~~, ~~origem~~ (como coisa na tela) | A peça TRANSPARENTE na casa de onde ela saiu, enquanto o **movimento** é proposta (ALE-203, item 4). Existe porque a peça deixou de ficar lá: ela é desenhada onde foi SOLTA, e o começo do caminho precisava de quem o marcasse. É a peça inteira — monograma e selo —, porque a pergunta que ela responde é *qual* dos três zumbis está a caminho. Some no **confirmar** e no cancelar, junto com a **seta**. |
| **seta** (do movimento) | `Fio`, `tabuleiro-movimento-fio` | ~~régua~~ (esta é da ferramenta), ~~losango~~, ~~destino~~ (o marcador, que SAIU) | A linha dourada com ponta que liga o **fantasma** à peça, dobrando em cada **parada**. Divide a gramática com a régua — mesmo dourado, mesma espessura — e o que a distingue é ter SENTIDO: a régua mede entre dois lugares, o movimento vai de um para o outro. Não é a **trilha**: a trilha são as casas que custaram (a diagonal do livro, p238, está no desenho dela) e a seta é o gesto, onde a pessoa clicou. |
| **trilha** | `Trilha`, `tabuleiro-trilha` | ~~caminho~~ (como coisa na tela), ~~rastro~~ | As casas por onde a peça PASSA no movimento proposto, pintadas de dourado translúcido. É a CONTA do custo virada desenho. Ver **seta**, que é a outra metade e diz outra coisa. |
| **parada** | `Stops`, `tabuleiro-parada` | ~~ponto~~, ~~vértice~~ | A casa onde a pessoa CLICOU ao montar o movimento. As INTERMEDIÁRIAS viram um pingo sobre a **trilha**; as duas pontas não, porque a primeira já é o **fantasma** e a última já é a peça. É a lista que faz "Desfazer parada" ter o que desfazer — o caminho sozinho não a deixa deduzir, porque um trecho legítimo já dobra na diagonal. |
| **ferramenta** (do mapa) | `$ferramenta`, `ferramentaDoMapa` | ~~pincel~~ (como sinal), ~~modo~~ | O que o dedo no tabuleiro FAZ. São dez, cada uma com um número de atalho fixo: mover (1), arrastar a **janela** (2), régua (3), gabarito (4), marcar (5), os quatro pincéis de terreno (6–9) e a borracha (0). **É um sinal só e o valor É a ferramenta**, então a exclusão é por construção — não há como duas estarem ligadas. O sinal chamava-se `$pincel` enquanto terreno era a única, e o nome ficou errado no dia em que MARCAR entrou. **Colide com a seção "Ferramentas" do trilho do mestre** (encontros, improviso), que é outra coisa: aquela é onde se vai, esta é o que o clique faz. A BORRACHA já foi um MODO do terreno (`$apagando`) que invertia o pincel na mão, e isso produziu o defeito de apagar a espécie errada em silêncio: virou ferramenta própria, e limpa a casa inteira (ALE-203). |
| **marcador** | `BoardMarker` | — | O ponto apontado no mapa (ALE-195). Nasce escondido. |
| **terreno** | — | ~~obstáculo~~ | O que o QUADRADO faz com quem está nele ou atrás dele. É a FAMÍLIA, e tem quatro espécies: **difícil**, **cobertura**, **camuflagem** e **elevado** (T20 p238, Tabela 5-3; os acidentes que os produzem estão em p267-268). Nasceu significando só "difícil", que era a única espécie implementada — a palavra foi promovida quando as outras três chegaram. **Não confundir com o CHÃO do lugar**, que é a aparência, nem com a **janela**, que é o recorte. Não tem identificador próprio de propósito: cada espécie tem o seu, e um campo `terreno` genérico convidaria a guardar as quatro numa string. |
| **difícil** | `Difficult` | ~~lento~~, ~~pesado~~ | Espécie de terreno: entrar custa o dobro (p238). É a ÚNICA que o motor consome hoje — ela entra no custo do movimento. |
| **cobertura** | `Cover` | ~~proteção~~, ~~abrigo~~ | Espécie de terreno: quem está nela recebe **+5 na Defesa** (p238). Trincheira, árvore estreita, a lateral de uma carroça. A cobertura TOTAL do livro (o alvo não pode ser atacado) não é pintável — ela vem de parede, que é ausência de quadrado. |
| **camuflagem** | `Concealment` | ~~ocultação~~, ~~névoa~~ | Espécie de terreno: quem está nela dá **20% de chance de falha** ao atacante (p238). Folhagens, moitas. A camuflagem TOTAL (50%) é escuridão, que é do lugar inteiro e não de um quadrado. |
| **elevado** | `Elevated` | ~~alto~~, ~~plataforma~~ | Espécie de terreno: quem ataca DE LÁ recebe **+2 no ataque** (p238). É a única espécie que beneficia quem está nela em vez de proteger — por isso o desenho dela não pode se parecer com o das outras três. |
| **chão** | `BoardState.Terrain` | ~~cenário~~ | A APARÊNCIA do lugar — pedra, taverna, floresta, ermo, cripta, papel. É o que o `.chao-*` pinta. **Não é o terreno difícil** da linha acima, que é regra de movimento e não pintura: um é como a cena se PARECE, o outro é quanto custa atravessá-la. `~~cenário~~` está proibido porque em T20 cenário é ARTON, o mundo da campanha — a SPA usa essa palavra no rótulo do diálogo (ALE-124) e o piloto não a herda. O identificador do fio continua `terrain` e não se conserta por palpite: renomeá-lo mexe no acervo de lugares gravado. |
| **cortina** | `curtained` | ~~oculto~~, ~~privado~~, ~~rascunho~~ | O tabuleiro EXISTE para o mestre e a mesa vê uma cortina no lugar dele (ALE-202). É durante a sessão, com a mesa presente — montar a taverna enquanto eles olham a cripta. **Não é o rascunho de lugar**, que é preparação FORA da sessão, sem ninguém conectado: tempos diferentes, gestos diferentes, e a decisão do dono foi que os dois convivem. Também **não é** "o mestre ainda não abriu um tabuleiro", que continua significando exatamente isso. A cortina é DESENHADA: o jogador sabe que vem cena, sem ver qual. |
| **janela** | `$vistax`, `$vistay`, `--vista-*` | ~~moldura~~, ~~extenso~~, ~~viewport~~ | O RECORTE que cada pessoa está olhando, em pixels do plano, e ele é do NAVEGADOR: nunca vai ao servidor, e por isso o remendo do SSE não o perde e duas pessoas olham lugares diferentes do mesmo tabuleiro. Com o zoom (`--quadrado`) ele é todo o enquadramento. **Não é "o tamanho do tabuleiro"**, que não existe: o plano é infinito e o servidor manda o que EXISTE, em coordenada absoluta. `~~moldura~~` está proibida porque foi a palavra do que a janela substituiu — o retângulo que o SERVIDOR desenhava, com margem e piso de 20×14 (ALE-263). Ela CRESCIA ao pintar perto da borda, e o mesmo ponto da tela virava outro quadrado entre dois cliques; saiu inteira na ALE-203, com o tipo `Moldura` e o `MolduraDe`. Quem escrever "moldura" hoje está falando de um conceito que o app não tem mais. |

## D. Os monstros do mestre

| Canônico | Identificador | Proibidos | O que é |
|---|---|---|---|
| **verbete** | `MonsterID` | — | A entrada IMUTÁVEL do bestiário do livro. Por extensão, qualquer entrada de catálogo que o mestre CONSULTA — a condição, a magia, o tipo de efeito. A ALE-264 desfez a divisão que separava o bestiário dos outros: "o bestiário conta como catálogo", e o trilho passou a ter uma seção **Ferramentas** (o que o mestre FAZ: encontros, improviso) e uma **Catálogos** (o que ele consulta, nove). |
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

Nome de pacote é identificador, então o glossário manda nele — e pela regra do
§F identificador novo é em inglês. `aovivo/` e `plataforma/` são anteriores à
regra e ficam: renomear pacote move todo import do repositório, que é o preço
mais alto da lista por ganho puramente estético.

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

O domínio é pt-BR e o código era misto — e isso não estava escrito em lugar
nenhum, que é a raiz de metade desta lista. Em `board-region.tsx` convivem
`selectedToken` e `linhasNoMapa`, e `board` e `cena` denotam o mesmo tipo no
mesmo arquivo.

A regra, daqui para frente (a completa está em
[CLAUDE.md § Idioma](CLAUDE.md#idioma); aqui fica o que é do glossário):

- **A tela fala pt-BR.** Sem exceção, incluindo `aria-label` e `title` (o Kobalte
  injeta rótulos em inglês — passe o seu, mesmo quando a palavra está escrita).
- **O identificador fala inglês.** Variável, função, tipo, campo, arquivo — e a
  fronteira, que já era: tabela, campo JSON, evento SSE, rota HTTP.
- **O que muda é a GRAFIA, não o conceito.** A coluna "no código" registra como
  o termo está escrito HOJE, e boa parte dela ainda está em pt-BR — é o passivo,
  não o alvo. Ao escrever um identificador novo para um termo que já tem
  tradução em uso, use a que já está em uso; ao traduzir um termo pela primeira
  vez, **anote a tradução na linha dele antes de escrever o código** — traduzir
  na hora, duas vezes, é como um conceito vira dois (`place` e `location` para o
  mesmo `lugar`). Termo sem tradução — `tormenta`, `goblinoide` — fica como
  está: é nome próprio.
- **Um arquivo não mistura os dois nomes do MESMO conceito.** O código antigo
  não é varrido (a razão está no CLAUDE.md), então as duas grafias convivem por
  um tempo: num arquivo que já chama `cena`, não introduza `scene` ao lado — o
  nome NOVO segue a regra, o nome CHAMADO segue o que está lá.

---

Termo novo, ou conceito que ainda não está aqui: **escreva a linha antes de
escrever o código.** Nomear depois é como este arquivo nasceu.
