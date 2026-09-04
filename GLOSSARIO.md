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

**O kit de interface não está aqui, e é de propósito.** `Button`, `TextField`,
`PanelFrame`, `ScrollBox` e a casca vivem em `web/ui` com nomes em inglês
(ALE-278, fatia 4) porque nenhum deles é termo do DOMÍNIO — não há conceito da
mesa por trás de um botão. Este glossário governa a linguagem do Tormenta 20 e
do app; o vocabulário de widget segue a regra de idioma do `CLAUDE.md` e mais
nada. O que continua aqui é o que a mesa fala: **elo**, **verbete**, **livro** —
e o `eloParaOAcervo` segue com o nome dele porque ficou no `api`, junto do
catálogo que ele consulta.

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
| **livro** | `…DoLivro` | ~~manual~~, ~~PDF~~ (na tela) | O Tormenta 20 impresso, e a AUTORIDADE das regras: `criaturasDoLivro`, `magiaDoLivro`, `condicaoDoLivro` são entradas dele. Quando o ARQUIVO importa — o PDF que o servidor entrega em `/livro` —, diga **PDF do livro**: quem serve é a mesa, e serve só se o dono configurar `LIVRO_PDF` (ALE-264). |
| **perícia** | `pericias` | ~~habilidade~~, ~~skill~~ | Uma das 29 do livro (T20 p115, Tabela 2-1). Ganhou catálogo na ALE-264: existia como lista de nome e atributo dentro do `options.json`, sem página e sem as DUAS regras que o livro imprime ao lado do nome — **só treinada** (sem treinamento nem se rola) e **penalidade de armadura**. As três com penalidade são as mesmas que o motor já conhecia, e há teste costurando as duas fontes. **Não confundir com poder**: perícia se ROLA, poder se TEM. |
| **perícia inventada** | `customExpertise` | ~~ofício~~ na TELA | A perícia que o jogador cria e o livro não tem — o saber de um ferreiro, a arte de um marinheiro. Nasce TREINADA (inventar um saber e não tê-lo treinado não significa nada) e é a única que se REMOVE da ficha; a coluna `custom` do banco é quem decide, e não uma lista de nomes no código. **A tela diz "nova perícia" e NUNCA "ofício"** — e isto é colisão de verdade, não preferência: **Ofício É uma das 29 do livro** (p115), então chamar a inventada de ofício poria duas coisas diferentes com o mesmo nome na MESMA lista. O identificador pode falar de ofício, porque identificador não é o que a pessoa lê (ALE-272). |
| **gravação falhando** | `SaveFailed` | ~~sujo~~, ~~dirty~~ (na tela) | A última tentativa de escrever a mesa no disco não deu certo, e a mesa continua rodando de MEMÓRIA. É estado e não notícia: vale enquanto durar, então a tela o LÊ do store a cada quadro em vez de esperar um aviso que pode se perder (ALE-288). `Dirty` é o nome do campo dentro dos dois stores e fica; o que atravessa a porta e o que a pessoa lê é "gravação falhando". **Só o mestre vê** — é ele quem pode parar a sessão e chamar alguém; para o jogador seria um aviso sobre o qual ele não pode fazer nada. |
| **carga** | `carga`, `LoadBreakdown` | ~~peso~~, ~~encumbrance~~ | Quanto o personagem está carregando, em ESPAÇOS — o T20 não pesa nada em quilos (p141). O limite é `10 + 2×Força`, passar dele é **sobrecarga** (−2 em Acrobacia, Furtividade e Ladinagem e −3m de deslocamento), e acima do DOBRO o livro diz que não dá para carregar. A conta inteira é do motor desde a ALE-215; a tela só a escreve. **A palavra está tomada**: `uso` de poder não é carga (ver **uso**). |
| **espaço** | `slots` | ~~peso~~, ~~slot~~ (na tela) | A unidade da carga. Conta de meio em meio — uma adaga ocupa 1, um bálsamo 0,5 —, e cada 1.000 moedas ocupam um espaço. `slots` é o nome do campo desde sempre e fica; o que a pessoa lê é "espaço". |
| **tibar** | `tibar`, `T$` | ~~ouro~~, ~~moeda~~, ~~PO~~ | O dinheiro de Arton (p140). Na tela sai como **T$**. Ele é a MESMA coisa que a carga conta em moedas, e é um campo só na ficha: a Forja preenche, a Mochila mexe. São três gestos e não um — receber, gastar e corrigir (ALE-224) —, e o saldo nunca fica negativo, porque dívida viraria carga de moeda negativa. |
| **melhoria** | `improvements` | ~~upgrade~~, ~~encantamento~~ | O que se forja num item para ele fazer mais: Certeira, Cruel, Reforçada (p165). São até quatro no mesmo item, e cada uma só cabe numa FAMÍLIA — arma, armadura, escudo, vestuário. Quem recusa a que não cabe é o servidor (`aMelhoriaCabeNoItem`, ALE-272); a lista do diálogo é conveniência sobre a mesma regra. |
| **material** | `material` | ~~liga~~, ~~matéria-prima~~ | O material especial de que o item é feito — aço-rubi, mitral, adamante (p166). É UM por item, ao contrário da melhoria, e obedece à mesma regra de família. |
| **equipado** | `equipped` | ~~equipar~~ como estado, ~~slot de corpo~~ | Onde o item está: **empunhado** (uma ou duas mãos), **vestido**, ou **guardado** — que é o vazio da coluna. Os tetos são dois e são do livro: no máximo duas mãos ocupadas e quatro vestidos (p141). **Não há casa de corpo** no T20: nada de elmo, botas e anel numa boneca — a tira da Mochila desenha os dois TETOS, e é só isso que o livro tem. |
| **poder** | `classPowers`, `powers` | ~~habilidade~~ (como escolha), ~~talento~~, ~~feat~~ | O que o personagem TEM e a ficha lista na aba Poderes. Vem de cinco procedências — raça, origem, classe automática, poder de classe escolhido e poder geral — e a mesa não distingue nenhuma delas na hora de usar. **Poder se TEM, perícia se ROLA.** Uma VAGA de poder abre por nível a partir do 2º (p33), e "você sempre pode substituir um poder de classe por um poder geral" — por isso as duas listas viram uma no diálogo de escolher. |
| **postura** | `stance` | ~~modo~~, ~~forma~~ | O poder que se LIGA e DURA: a Fúria do bárbaro (p40), a Inspiração do bardo (p44). Entrar custa PM e pode ter DEGRAUS que o nível na classe abre; sair não devolve nada, e é por isso que o preço pago fica gravado (`character_stances`, ALE-222). **Ligar mora nos Poderes** — é lá que o PM é cobrado —, e a aba Efeitos mostra a que está em curso e a encerra. |
| **ativação** | `activations.json` | ~~ação~~ (como registro) | O registro que diz o que se pode FAZER com um poder: o tipo (`instant`, `stance`, `passive`, `triggered-passive`), a ação que o uso consome, o PM e o limite. Um poder sem entrada nele é passiva silenciosa — aparece na lista e não oferece botão. |
| **pendência** | `pendencia` | ~~aviso~~, ~~erro~~ | Uma escolha que ainda falta fazer — distribuir o bônus de atributo da raça, os dois benefícios da origem, os poderes das vagas abertas, o caminho, o devoto. Ela NÃO é erro: a ficha existe para ser preenchida aos poucos, e a forja promete por escrito "dá para criar assim e terminar na ficha" (ALE-169). |
| **forja** | `forge` | ~~criação~~, ~~assistente~~, ~~wizard~~ | A cena onde um personagem NASCE: nome, raça, classe, origem, o equipamento inicial e os atributos. Ela é curta de propósito (decisão do dono, ALE-272): dá o que a ficha não sabe dar e para por aí — poderes, benefícios de origem, treino de perícia e escolhas de raça se terminam na ficha, que é o que a **pendência** aponta. O herói nasce ao fim da primeira tela e já é personagem de verdade; **incompleto não é inválido**. |
| **kit inicial** | `startingKit` | ~~equipamento de classe~~, ~~inventário inicial~~ | O equipamento com que o herói nasce (p140). É UM só para todas as classes — o livro não tem lista por classe —, parametrizado pelas proficiências dela: mochila, saco de dormir e traje de viajante para todo mundo; uma arma simples; uma marcial se a classe for proficiente; armadura leve à escolha, brunea se ela usa pesadas, nenhuma para o arcanista; escudo leve se ela usa escudo. Vem JUNTO com a **bolsa inicial**, e não no lugar dela. |
| **bolsa inicial** | `startingMoney` | ~~ouro inicial~~, ~~verba~~ | Os tibares com que o herói nasce (Tabela 3-1, p140): no nível 1 são **T$ 4d6**, rolados no nascimento; do 2º ao 20º é um valor fixo da tabela. Depois disso quem mexe é a Mochila — ver **tibar**. |
| **proficiência** | `proficiencies` | ~~treinamento~~, ~~habilitação~~ | Saber usar uma categoria de arma, armadura ou escudo. São SETE — armas simples, marciais, exóticas e de fogo (T20 p142), armaduras leves e pesadas, e escudos (p148) — e a classe concede as dela numa linha só ("Proficiências. Armas marciais e escudos.", p36–83). **Não confundir com perícia**: perícia se ROLA e tem graus de treinamento; proficiência se TEM ou não se tem, e o que falta vira PENALIDADE (−5 no ataque, p142; a penalidade da armadura nas perícias de Força e Destreza, p148). **Quem sabe usar armadura pesada sabe usar a leve, e isso o livro NÃO diz** — é decisão de produto, e está escrita e presa por teste em `web/sheetui/proficiencias.go`. |
| **escola de magia** | `escolas-de-magia` | ~~tipo de magia~~, ~~classe de magia~~ | A família de uma magia — **Abjuração**, **Evocação**, **Ilusão** (T20 p172, oito). O livro imprime a abreviatura ao lado ("Abjuração (Abjur)") e as tabelas dele usam a forma curta. Ganhou catálogo na ALE-264: a magia guardava `school: "evocacao"`, isso decidia o filtro, e o nome não aparecia em cartão nenhum. **Escola conta como tipo de efeito** — é o livro que diz —, então as duas famílias se tocam. |
| **grimório** | `spellbook` | ~~livro de magias~~, ~~repertório~~ | As magias que a ficha SABE. É o título da aba Magias, e a palavra vem do livro. **Colide com a identidade visual do app** (`scene-grimorio`, "Grimório de Arton") e com as abas de catálogo do mestre — ver a colisão C8. |
| **magia preparada** | `prepared` | ~~memorizada~~, ~~equipada~~ | A magia que o conjurador escolheu para hoje, e o interruptor só existe para quem PREPARA: Clérigo, Druida e o Arcanista do caminho mago (p171). Para os outros a magia se conjura direto, e um botão de preparar seria um controle que não significa nada. |
| **círculo** | `circle` | ~~nível da magia~~, ~~grau~~ | A potência da magia, de **Truque** (0) ao **5º** (p170). **Nunca "nível"**: nível é do personagem, e o livro usa as duas palavras em frases vizinhas — o círculo que um personagem alcança é função do nível dele, e trocar as palavras torna a frase indecifrável. A tela escreve "Truque" e "3º", como a mesa fala. |
| **aprimoramento** | `augment` | ~~upgrade~~, ~~melhoria~~ | O que se paga a mais para a magia fazer mais (p170). Custa PM sobre o custo base, e alguns só existem a partir de um círculo — esse aparece TRANCADO em vez de sumir, porque quem não alcança merece saber que existe. Quem recusa é o servidor (`validateAugments`); o cadeado é UX. |
| **tipo de efeito** | `tipos-de-efeito` | ~~tag~~, ~~categoria~~ | A família de um efeito — **Medo**, **Mental**, **Movimento**, **Metabolismo** (T20 p228). O livro a imprime em itálico no fim da condição ("Abalado … *Medo*."), e ela é o que decide imunidade: "criatura com imunidade a medo não será afetada por efeitos do tipo medo". No dado a condição a guarda como CHAVE (`tags: ["cansaco"]`, sem acento); na tela sai o nome do livro. Ganhou catálogo próprio na ALE-264 porque não havia para onde o elo apontar. |
| **elo** | `eloParaOAcervo` | ~~link~~ (no código) | A palavra que leva ao verbete dela: a tag da condição leva ao tipo de efeito, "Agrava para Apavorado" leva à condição, o poder concedido leva ao poder. **Não é o botão do livro** — o elo anda DENTRO do acervo, o botão sai para o PDF. Os dois usam o mesmo sublinhado pontilhado de propósito: são o mesmo gesto ("leva para outro lugar") e desenhá-los diferente ensinaria que não são. O clique MOSTRA o verbete numa caixa sobre a cena; o `href` (`?aba=X&entrada=<id>`) fica para o ctrl+clique e o link copiado. |
| **entrada** | `?entrada=<id>` | ~~item~~, ~~registro~~ | O endereço de UM verbete: a aba dele mostrando só ele. **Não é busca** — `?busca=Medo` procura o termo nos oito catálogos e mostra os grupos; `?entrada=medo` é o Medo. A diferença nasceu de um defeito de UX: o elo endereçava por busca, e clicar num conceito caía numa lista onde ele era o quinto grupo. |
| **leitor** | `leitor` | ~~visualizador~~, ~~viewer~~ | A cena que abre o PDF do livro na página do verbete e o DESTACA (ALE-264). **Não é o visualizador do navegador**, que continua a um clique de distância ("abrir fora") — a distinção importa porque os dois existem lado a lado: o leitor mostra uma página por vez com o termo marcado, o visualizador tem busca, miniaturas e impressão. Ele existe por medição: o Chrome ignora `#search=` e transfere o arquivo inteiro (85 MiB) para abrir uma página; o leitor destaca e custou 1 MiB. |
| **página do livro** | `BookPage` | ~~folha~~ | O número IMPRESSO no rodapé, que é o que o catálogo grava e o que a ficha mostra ("p289"). **Não é a página do ARQUIVO**, que é a que `#page=N` conta — ver `abertura` abaixo. |
| **abertura** | `LivroAbertura` | ~~offset~~, ~~deslocamento~~ | Quantas páginas o arquivo tem ANTES da página impressa 1: 6 no PDF da casa, medido pelo rodapé. `deslocamento` está proibido porque em T20 é o quanto uma criatura anda. |
| **tela cheia** | `api/piloto/src/cena.ts` | ~~fullscreen~~ (na tela) | O gesto do menu do Hub que estica a janela ATUAL pela Fullscreen API. Some quando a aba fecha, e o iPhone não o tem. Continua sendo a saída de quem não instalou (ALE-118). |

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
| **NPC** | `type: 'npc'` | ~~PC~~ | O PAPEL de uma linha na fila: não é ficha de jogador. Ortogonal a verbete/bloco. **Na tela o par é `Ficha` / `NPC`** — decisão do dono, 2026-08-24, ao tirar o `PC` proibido dos DOIS apps. O oposto de NPC na fila é **ficha** e não "personagem": é a pergunta que a tabela de colisões faz do `type === 'character'` (linha abaixo), e "Personagem" não caberia no selo de uma linha de 390px. Cuidado ao mexer: `Ficha` é 2,5× mais largo que o `PC` que estava lá, e esse selo JÁ transbordou uma vez, na SPA — os guardas de transbordo do `piloto-transbordo.spec.ts` são obrigatórios. **O endereço mudou duas vezes** (ALE-284): a linha citava o `initiative-card.tsx:428`, que morreu com a SPA, e mandava obedecer um `session.spec.ts` que não existe mais — quem obedecesse procuraria a garantia num arquivo apagado. |
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

**C3 — RESOLVIDA pela morte da SPA, e não por conserto (ALE-284).** Aqui morava
"`RestScope` mente na metade dos usos", sobre a assinatura
`applyEffect(entryId, spellId, scope?: RestScope)` do `realtime.ts`, onde `scope`
era *duração de magia* e não escopo de descanso. **`RestScope` tem hoje ZERO
ocorrências no repositório**: o tipo saiu inteiro com a SPA (ALE-272), e o
`applyEffect` que sobreviveu é um handler Go de outra forma. A linha fica como
lápide porque uma colisão que some sem decisão é a que mais tenta voltar — se um
dia um `scope` de duração e um de descanso se encontrarem de novo, é um
`EffectScope` próprio, que era o conserto já escolhido.

**C4 — TRÊS predicados vestidos de "jogador", e eram quatro.** Eles respondem a
**perguntas diferentes** e por isso não se unificam; o que falta é o nome dizer
qual:

| Predicado | Pergunta que responde | Onde |
|---|---|---|
| `entry.Type == "character"` | esta linha é ficha ou é NPC? | `board_state.go:468`, `tokens.go:61`, `view.go:281` |
| `member.role === 'player'` | a PESSOA dona é jogador ou mestre? | `listPlayerCombatants` |
| "é meu?" | o personagem é de quem está olhando? | `mesaRoster`, em `routes.go` |

**O QUARTO se juntou ao primeiro, e é preciso dizer como** (ALE-284): ele era
`token.kind === 'character'`, "esta peça se desenha como PC?", e vivia no
`board-view.tsx`. O campo continua no fio — `BoardToken.Kind` é `"character" |
"npc" | "object"` —, escrito em `board_body.go` e, ao pôr no mapa, a partir do
próprio `entry.Type`, que é o PRIMEIRO predicado. **Medido: nada em produção o
LÊ**, só teste. Ou seja, não é que a pergunta tenha sido respondida; é que quem a
fazia morreu e o campo ficou. Quem for desenhar a peça de novo decide se a
pergunta volta a ser dela ou se `Kind` é redundante com `Type`.

Decisão do dono, 2026-08-22 (ALE-204): o atalho "trazer os jogadores" usa
`type === 'character'`, porque é o **mesmo** predicado com que o servidor escolhe
o lado do mapa — o atalho põe as peças exatamente na fileira do grupo. O efeito
colateral é conhecido e aceito: um PC do mestre que esteja na fila vem junto.

**C5 — RESOLVIDA (ALE-284).** Aqui morava "o README ainda usa três palavras para
campanha", citando "as dos outros aparecem nas **Crônicas**" e "abre e edita
qualquer **mesa**". As duas saíram — a primeira num passe anterior que não
atualizou esta linha, a segunda nesta issue, junto com "passa as **mesas** dela
para você". O README continua usando `mesa` oito vezes, e todas são a metonímia
que a seção A permite: a mesa na LAN, quem senta à mesa, a mesa no ar. Nenhuma é
entidade.

> O número das colisões NÃO é reaproveitado quando uma sai. `C1`, `C4`, `C6`,
> `C7` e `C8` são citados por número de dentro do código (`sheet/play_state.go`,
> `tokens.go`, `vitals_rules.go`) e do README — renumerar quebraria
> cinco referências em silêncio, que é o defeito que esta issue veio consertar.

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

**C8 — `grimório` é o app, é o catálogo do mestre e é a aba do jogador.** A
casca visual inteira se chama assim (`scene-grimorio`, `grimorio-gold`,
"— Grimório de Arton —"), as abas de catálogo do mestre são "as abas do
grimório" (ver **aba**), e desde a ALE-272 a aba Magias da ficha tem
**Grimório** por título — que é o sentido do LIVRO, e o mais estreito dos três.
Nenhum é errado e nenhum se conserta por palpite: o primeiro é a marca do
produto, o terceiro é a palavra da mesa. O identificador é que não pode ser só
`grimorio` — os desta fatia são `spellbookPanel`, `learnedSpellRow`,
`spellbookPanelOf`.

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
nenhum, que é a raiz de metade desta lista.

O exemplo que abria esta seção era o `board-region.tsx`, onde conviviam
`selectedToken` e `linhasNoMapa`. **Ele morreu com a SPA e a frase ficou**
(ALE-284), o que é a própria costura falhando: o arquivo que ilustrava a mistura
sumiu e ninguém releu a linha que o citava. O exemplo vivo é o
`board_view.go`, e a mistura lá não é entre arquivos — é dentro
de um identificador só: `tabuleiroView`, `pecaDoTabuleiro`, `movimentoView` e
`tabuleiroViewOf` colam raiz portuguesa em sufixo inglês, na mesma linha.

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

---

## G. As grafias inglesas que a ALE-282 assentou

A varredura dos nomes de teste traduziu 773 nomes de uma vez, e vinte e poucos
termos da mesa **não tinham grafia inglesa nenhuma** — nem na coluna "no código",
nem em nenhum identificador do repositório. Traduzir na hora, e duas vezes, é
como um conceito vira dois (`place` e `location` para o mesmo `lugar`), então a
escolha ficou escrita aqui.

**Isto não é um renome pedido.** A coluna "no código" das seções acima continua
sendo o passivo: `cortina`, `lente`, `traco` seguem com o nome que têm nos
arquivos onde já estão. O que a tabela abaixo governa é o nome NOVO — e o nome de
teste foi o primeiro lugar onde ele precisou existir.

| termo | grafia inglesa | onde ela já aparece |
|---|---|---|
| **fila** (de iniciativa) | `tracker` | já estava na seção B; a varredura só a usou |
| **buscador do livro** | `finder` | `web/finder` (ALE-278) — e NÃO `search`, que é o casamento e a pontuação, nem `busca`, que é o filtro da cena |
| **elo** | `crossref` | `web/bookui` (ALE-278). **Não `link`**, que a seção A já proíbe: o elo anda DENTRO do acervo e o botão do livro SAI para o PDF, e chamar os dois de link apagaria a distinção que a linha do verbete existe para fazer |
| **a interface que sabe do livro** | `bookui` | `web/bookui` (ALE-278) — o kit que conhece o domínio, ao lado do `web/ui`, que não pode conhecer |
| **a Mesa do Mestre** (o trilho e as paradas dele) | `master` | `web/master` (ALE-278). A grafia **já existia e já tinha uma rival**: `routes.MasterBestiary` de um lado e `GMToolRoutes` do outro, duas palavras para a mesma tela — que é exatamente como um conceito vira dois. `master` ganha porque é a que está no ENDEREÇO, e endereço é o que alguém cola no chat da mesa. O `gm` da seção A continua valendo para o PAPEL (`role === 'gm'`): o mestre é uma pessoa, a Mesa do Mestre é uma tela |
| **encontro** | `encounter` | `web/master` (ALE-278) — a ferramenta que monta o grupo de criaturas e diz o ND que ele resulta |
| **improviso** | `improv` | `web/master` (ALE-278) — as quatro tabelas que o mestre rola na hora: ruína, perseguição, recompensa e ideias. Não `improvisation`, que é comprido sem comprar precisão |
| **verbete** | `entry` | `routes.Entry` já usava. Ele traduz também **entrada**, e isso NÃO é colisão nova: `?entrada=<id>` é o endereço de UM verbete, então as duas palavras são o mesmo conceito visto do lado do dado e do lado do endereço |
| **a cena que lista os personagens** | `characters` | `web/characters` (ALE-278) — o PLURAL, porque a cena é a lista de quem uma pessoa tem. A do singular é a ficha, logo abaixo |
| **a ficha** (a cena, não o dado) | `sheetui` | `web/sheetui` (ALE-278) — a cena de `/personagens/{id}`. O sufixo existe porque `sheet` já é a FORMA do dado (`CharacterDTO`, `Compute`), e a cena o lê 148 vezes em 20 arquivos: sem o sufixo, cada um desses arquivos carregaria um apelido. Ver a colisão registrada abaixo |
| **a cena das campanhas** | `campaigns` | `web/campaigns` (ALE-278) — o PLURAL, como o `web/characters`, e ela cobre a lista, a campanha aberta, a folha em branco e a carta de entrar. **O plural é o que evita o apelido**: as regras de campanha são o pacote `campaign` (singular), e a cena o importa sem ambiguidade. É o mesmo problema que o `sheetui` resolve com sufixo — quando a cena e a camada falam do mesmo conceito, uma das duas precisa de outra palavra |
| **a Mesa** (a cena) | `table` | `web/table` (ALE-278) — a cena de `/mesa/{campanha}/{sessao}`: o palco, a fila, o elenco, as notas, a cortina e o tabuleiro inteiro. **O nome não foi escolhido agora**: `routes.Table` já monta este endereço desde que o `web/routes` nasceu, e `joinTable` já é "sentar à mesa" no hospedeiro — a palavra inglesa estava assentada e o que faltava era o pacote. Ela NÃO precisa de sufixo como o `sheetui`, porque não há uma camada `table` de dado para colidir: o estado ao vivo é `aovivo` e o mapa é `tabuleiro`. **Cuidado com a outra acepção**: `table` de banco de dados aparece só no `cmd/seed`, que é outro pacote e não é domínio |
| **leitor** (a cena que abre o PDF) | `reader` | `web/reader` (ALE-278), e `routes.Reader` já usava. **Não é o visualizador do navegador** — a linha da seção A faz essa distinção e ela continua valendo |
| **abertura** | `opening` | `web/reader` (ALE-278). O `bookui.BookAddress.Abertura` fica com o nome que tem: renomeá-lo move todo chamador, e a regra do §F diz que o nome CHAMADO segue o que está lá. Então uma linha do leitor cola os dois (`Opening: livro.Abertura`), o que é a costura acontecendo à vista em vez de escondida |
| **acervo** (cenas guardadas) | `archive` | `TestTheArchiveSaysWhichSceneIsOnTheTable` |
| **acervo** (de poderes, de catálogo) | `collection` | `TestTheCollectionJoinsTheFiveOrigins` |
| **cortina** | `curtain` | `curtained` já existia no fio |
| **lente** | `lens` | `TestTheLensSaysHowManyVanished` |
| **palco** | `stage` | `TestTheStageHasTheTwoPartsThatAnimate` |
| **trilho** | `rail` | `TestTheRailOffersEveryStop` |
| **trilha** | `trail` | `TestTheTrailSlugsAreUnique` |
| **régua** | `ruler` | `TestTheRulerHasNoDirection` |
| **gabarito** | `template` | `TestTheTemplateRefusesAShapeTheBookDoesNotHave` |
| **traço** | `stroke` | `TestTheStrokeHasNoGap` |
| **pincel** | `brush` | `TestTheBrushIsIdempotentForEachKind` |
| **borracha** | `eraser` | `TestTheEraserClearsTheWholeSquare` |
| **fantasma** | `ghost` | `TestTheGhostMarksTheOriginWithTheTokenMonogram` |
| **seta** (do movimento) | `arrow` | `TestTheArrowBendsAtTheStopsAndEndsAtTheDestinationEdge` |
| **janela** (do mapa) | `viewport` | `TestNoLayerReadsThePointWithoutAddingTheViewport` |
| **chão** | `ground` | `TestEveryOfferedGroundCanBePainted` |
| **espécie** (de terreno) | `kind` | `AreaKind` já usava `Kind` |
| **ferramenta** (do mapa) | `tool` | `TestEachToolHasAKeyOfItsOwn` |
| **mostrar à mesa** / puxão | `pull` | `TestThePullReachesWhoNeverChoseATab` |
| **perna** (do caminho) | `leg` | `TestEachLegGetsItsOwnLabel` |
| **laço** | `lasso` | `TestTheLassoCatchesTheTokenByItsBody` |
| **porta** (a cena de entrar) | `door` | `TestTheDoorRefusesAWrongPasswordWithoutOpeningASession` |
| **crachá** | `badge` | `TestTheTypeBadgeTogglesWithoutNavigating` |
| **dossiê** | `dossier` | `TestTheDossierRespectsTheLimit` |
| **monograma** | `monogram` | `TestTheMonogramComesFromTheKindAndTheNumberBecomesASeal` |
| **matiz** | `hue` | `TestTheNameHueMatchesTheJs` |
| **sobrecarga** | `overload` | `TestOverloadPenalizesDisplacementAndArmorExpertises` |
| **mesa** (as pessoas conectadas) | `table` | `TableMember`, `TableRoute`, `TestTheTableDoesNotLeakHiddenHp` |
| **bolsa inicial** | `purse` (na frase) | `startingMoney` continua sendo o campo |

**Três colisões que esta tabela ABRE, e que ficam registradas em vez de
resolvidas por palpite:**

- **`door` e `port`** são a mesma palavra em português. A cena de entrar é
  `door`; a interface que uma cena declara para o hospedeiro (ALE-278) é `port`.
  Quem escrever "porta" num comentário diz de qual está falando.
- **`arrow` e `wire`** são os dois `fio` — a seta do movimento tem `Fio` no
  código (seção C) e o formato de fio do SSE também. A varredura usou `arrow`
  para o desenho e `wire` para o protocolo.
- **`sheet` é o DADO e `sheetui` é a CENA.** A primeira versão desta linha dizia
  que a cena se chamaria `web/sheet` e que "quem importar os dois escreve o
  apelido" — decisão tomada com uma estimativa errada de ONDE os dois se
  encontram.
  **Medido depois:** a cena lê o pacote de dado **148 vezes em 20 arquivos**. O
  encontro é DENTRO da cena, não nas duas linhas do `api` que montam a rota, e
  o apelido teria caído em vinte arquivos com um nome (`data.`) que não está
  neste glossário.
  Com o número na mesa, o sufixo entrega a mesma grafia nas chamadas
  (`sheetui.Routes`) sem apelido nenhum e sem pasta divergindo do pacote. **A
  palavra da mesa não foi trocada** — `ficha` continua sendo `sheet`; o que
  ganhou sufixo foi a TELA, que é a camada, e não o conceito.
  A lição, para a próxima colisão: **conte os usos antes de escolher onde o
  apelido dói.** O `web/campaigns` tinha resolvido isso de graça com o plural, e
  aqui o plural não servia porque a ficha é uma.
- **`archive` e `collection`** são os dois `acervo`, e a colisão é anterior a
  esta issue: o acervo de CENAS guardadas e o acervo de PODERES não têm nada em
  comum além da palavra. Se um dia um deles for renomeado na tela, é o segundo —
  "acervo" é a palavra da mesa para o primeiro.
