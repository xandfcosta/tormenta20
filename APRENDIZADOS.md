# Aprendizados

O que o repositório aprendeu errando, e que não é regra. A REGRA mora nos guias
(`CLAUDE.md` e `engine-go/CLAUDE.md`); aqui mora o CASO que a produziu, para quem
quiser entender por que ela existe.

**Este arquivo é revisado periodicamente com o dono.** A cada revisão, um
aprendizado ou vira regra num guia (e sai daqui), ou continua útil como caso, ou
envelheceu e sai. Ele não cresce para sempre.

Os guias não citam issue do Linear para explicar o porquê: o Linear tem limite de
issues, issues já foram removidas, e a citação vira ponteiro morto. Quando um
guia precisar do caso, ele aponta para cá.

Formato: um título `###` que diga a lição, a data em que entrou, e o caso em
poucas linhas, na parte do guia de onde ele saiu.

## Do guia da raiz

### A tela mentiu com a suíte verde (2026-10)

A decomposição da perícia mostrou números errados na tela por meses, com 193
casos de teste verdes e 75 guardas no ar. O dado estava certo e o defeito era
100% de apresentação, e nenhum passo do repositório mandava abrir a cena e
julgar. Daí o "O último passo é OLHAR" do guia.

### A manobra anunciava o vencedor (2026-10)

A manobra resolvia o teste oposto, dizia quem venceu e aplicava a condição
sozinha. Foi o primeiro caso da regra "o sistema informa, o mestre decide", e ela
nasceu depois: o trabalho já entregue a violava, e o conserto só veio quando o
dono mandou. Daí "achou contradição, pergunte ao dono" do guia.

### O instrumento mentiu com cara de resultado (2026-10)

Os casos por trás das linhas de "O INSTRUMENTO MENTE" do guia:

- Um `finally` que estourava substituiu o erro de verdade.
- Procurar uma linha no fim da saída (`| tail`) e não achar virou "o evento não aconteceu", quando o canal nem existia.
- Um `MutationObserver` num `addInitScript` observou um `body` ainda `null`, e "não houve mutação" era o nada.
- O `boundingBox` devolveu a caixa de um elemento coberto, e o clique acertou quem estava por cima.
- Um parser que descartava seletor desconhecido acusou 24 botões vivos como quebrados.
- Uma diferença de 2px era real, e a pergunta certa era o limite de contraste do WCAG 1.4.11.
- Fotos de tela "provaram" uma animação que o `screenshot()` tinha finalizado antes.
- Uma sonda armada antes do gesto contou a mão, e não a animação.
- Contar alvos `< 24px` inflou o defeito em 25×, por ignorar as duas exceções do WCAG 2.5.8 (espaçamento e equivalente).
- Medir tipografia pelo texto do código não juntou dois tokens em `class=` diferentes que a herança do CSS junta.
- Ler o computado "no instante certo" de uma transição só trocou o erro de lugar.
- A faixa dizia "Salvo" antes de qualquer digitação: o repouso era igual ao sucesso.
- O guarda "não desanexa a cena" passava no pior caso: a navegação tinha descartado o documento, e a lista de mutações voltava vazia.
- Sabotar a opacidade (`text-x/50`) não mudou nada no medidor, que ignora o alfa, e quase virou "o guarda está cego".

### A decomposição que somava 792 de 390 (2026-10)

Medir quanto cada camada come dos 390px de altura exigiu três instrumentos. O primeiro somou 792: subia a árvore somando irmãos e contou a textura de fundo (`absolute inset-0`) como camada empilhada. O segundo somou 412: contou o `gap` de um filho `display:none`. As duas tabelas tinham nomes e alturas plausíveis, e só a soma que não fechava denunciou. O terceiro fechou em 390 por construção, com uma coluna de sondas `elementFromPoint` de y=0 a y=389.

### Guardas que não visitavam (2026-10)

- Dois defeitos de contraste sobreviveram anos com o guarda no ar, porque ele nunca abria um popover nem entrava na cena de campanhas.
- A ficha atravessou duas fatias sem uma medição de contraste, porque o medidor era função privada de outro arquivo de teste.
- Um teste de arrasto com uma peça só não distinguia o gesto certo do errado: com uma peça, a primeira do DOM É a arrastada.
- As sete abas da ficha eram visitadas com um guerreiro, e metade do painel de Combate só existe para quem conjura.

### O número de guardas envelheceu três vezes (2026-10)

O guia listava os guardas de varredura um a um, com o total. O número ficou
obsoleto três vezes sem ninguém mexer na linha, e uma vez desceu, de 46 para 44,
porque o terreno que dois deles varriam foi apagado. Daí "o `grep` é a fonte".

### Os comentários que só cresciam (2026-10)

O guia mandava preservar todo comentário num refactor, "porque carrega intenção
e procedência". Somado a "cite a issue", cada conserto acrescentava prosa e nada
removia: medido, eram 766 blocos de 10 ou mais linhas no `engine-go`, 11.586
linhas ao todo, e 37% do `web/` era comentário. Uma mesma regra ("porta que
devolve tipo do hospedeiro não é porta") estava escrita em quatro `deps.go` e
no guia. Daí "comentário se RELÊ" e "não repita a regra do guia".

### Os guardas de idioma (2026-10)

Houve quatro guardas de idioma, com 1.620 linhas entre código e linhas de base.
Os três de lista de palavras portuguesas subcontavam em silêncio; o de nome de
arquivo, lista de permitidos, barrou numa sessão só `context`, `nested`,
`request`, `read` e `swallowed`. Foram apagados: idioma se confere no olho.

### O guia que era metade história (2026-10)

O `CLAUDE.md` da raiz tinha 847 linhas quando a regra "regra com guarda não
precisa de história" foi escrita, e 42% delas eram duas seções contando casos.
Duas passagens tinham virado mentira sem ninguém mexer nelas.

### Os renomes de pacote (2026-10)

O guia dizia que `aovivo/` e `plataforma/` ficavam em português porque renomear
pacote "custa caro"; o dono decidiu o contrário, e o renome para `live/` e
`platform/` custou 151 arquivos, todos provados pelo compilador. O de
`tabuleiro/` para `board/` custou 105 arquivos e 444 linhas, e o caro foi a
sombra: `board` é a variável mais comum do código da mesa. A cena da ficha virou
`sheetui` e não `sheet` porque ela lê o pacote de dado 148 vezes em 20 arquivos.

### Os rótulos em caixa alta (2026-10)

A SPA contou 208 rótulos em caixa alta escritos de 59 jeitos e achou três papéis
diferentes. As cenas em templ reinventaram dois deles em 106 sítios com 30
grafias antes de o kit ganhar as três receitas.

## Do guia do `engine-go`

### O caso de uso morava no domínio, e a contagem dele envelheceu (2026-10)

O `apply` do store dos tabuleiros — carrega o estado, chama a regra pura, grava —
é um caso de uso, e passou anos arquivado dentro de `domain/`. Quando os dois
stores mudaram para `app/`, as mutações que eles chamam (`AddToken`,
`AdvanceTurn` e as irmãs) deixaram de ser privadas e viraram a API dos pacotes
puros. O guia dizia "mais de 1.800 linhas" sem persistência no `domain/board`; o
pacote passou de 3.700 sem ninguém mexer na frase. A afirmação nunca foi a
contagem, e sim a ausência — que o `boundary_test.go` prende.

### A recusa da ficha que levava número de HTTP (2026-10)

O `applySpellBuffEffect` devolvia `(efeito, int, error)` e montava um erro de
campo com status 400 dentro da regra. Ele foi apagado, e é a razão de o
`character.Plays` poder ser exceção ao embrulho tipado sem ser exceção à regra
de não carregar número de HTTP.

### Dois motores concordaram no erro por meses (2026-10)

Quando havia dois motores (o Go e o compilado para o navegador), eles estiveram
perfeitamente de acordo durante meses enquanto AMBOS erravam a RD do Guerreiro,
que não existe no livro. Paridade prova concordância, nunca correção — e hoje o
oráculo é o Go descrevendo o Go, então a única rede é revisar o diff contra o
livro.

### Citação de página "verificada" saiu errada (2026-10)

Onze citações de página erradas foram corrigidas no repositório, e três delas
foram escritas no mesmo dia em que tinham sido "verificadas". Daí a regra de
conferir com o offset `PDF = livro + 6` antes de escrever, e não depois.

### O comentário acentuado que truncou a query (2026-10)

Um comentário com três acentos acima de uma query em `query.sql` gerou
`WHERE id = ? AND usedAt IS N` — o SQL perdeu três caracteres e ainda compilava.
O sqlc mede a query em bytes e o comentário em runas.

### As três metades da armadilha do sqlc com migração (2026-10)

Medido no sqlc v1.31.1, migração a migração:

- a `00005` criou o tabuleiro como tabela nova porque o `ADD COLUMN` em arquivo
  novo não entra no catálogo; as colunas adicionadas por `ALTER` dentro da
  `00002` continuam valendo porque estão no mesmo arquivo do `CREATE`;
- a `00010` trocou a chave primária do `session_boards` (1:1 com a sessão) criando
  o `open_boards` (uma linha por tabuleiro aberto), copiando e derrubando a
  velha — o rebuild com `RENAME TO` sumia do catálogo como o `ADD COLUMN`;
- a `00015` mostrou que o `DROP COLUMN` só casa em minúsculas: `DROP COLUMN
  hpMax`, com a grafia da `00001`, falhava; `hpmax` passou;
- a `00021` mostrou que o modelo gerado GANHA o campo de um `ADD COLUMN` em
  arquivo novo e mesmo assim toda query que cita a coluna falha — ver o modelo
  não provava nada.

### O gzip que mandava o corpo comprimido sem avisar (2026-10)

A compressão das cenas importa porque todo comando da ficha redesenha a cena
inteira: a aba de Combate viajava 44,7 KB crus e 5,6 KB em gzip, a cada toque no
PV. O primeiro envelope de gzip decidia comprimir só no `Write`, e o
`datastar-go` chama `Flush()` antes, para mandar os cabeçalhos — que saíam sem
`Content-Encoding`, com o corpo comprimido atrás. Foram 27 casos vermelhos no
e2e com os unitários todos verdes: os unitários escreviam o cabeçalho ANTES de
esvaziar, que não é o que a biblioteca faz. A busca não filtrava, a seta não
andava, o diálogo não abria, e nenhuma requisição falhava.

### O `fsync` que custava 139ms por toque (2026-10)

Com o `synchronous=FULL` padrão, o comando que muda o PV levava 121ms de servidor
e 139ms até o número mudar na tela, num prato girante. Com `NORMAL`, no mesmo
disco, 1,7ms e 12ms. O controle que isolou a causa foi trocar só o LUGAR do
arquivo — disco para tmpfs, binário idêntico —, e o número do tmpfs bateu com o
do pragma. A suspeita natural era o tamanho do HTML, e já estava descartada: o
tempo do POST era o mesmo em abas de 15 KB e de 39 KB. Para quem vem do MariaDB:
`FULL` ≈ `innodb_flush_log_at_trx_commit=1`, `NORMAL` ≈ `=2`, e o WAL do SQLite é
parente do redo log do InnoDB, não do binlog.

### O tabuleiro que viveu um dia só em memória (2026-10)

A tabela do tabuleiro (então `session_boards`) sumiu do banco de desenvolvimento
com a migração marcada como aplicada; o goose disse "no migrations to run", e
cada gravação falhou numa linha de log que ninguém lia. Daí o boot conferir o
SCHEMA. A lista de tabelas esperadas é derivada das migrações porque uma lista à
mão já tinha perdido o `TurnsTaken` e o `creatureId` no mesmo dia. E o guarda
precisou aprender o `DROP TABLE`: lendo só os `CREATE`, a derrubada legítima da
`00010` virava "falta a `session_boards`" num banco certo.

### Sem `_txlock=immediate`, o perdedor recebia 500 (2026-10)

Medido com dois pedidos simultâneos numa trava de unicidade: no `DEFERRED`
padrão, o SQLite recusava a escrita do perdedor sobre snapshot mudado (o que está
certo), e isso chegava ao jogador como erro do servidor. Com a trava no BEGIN, o
perdedor espera, relê e recebe a recusa que descreve o que houve. O caso que
abriu isso foi "um personagem por jogador em cada mesa".

### O Ctrl-C no meio do backup (2026-10)

Antes do encerramento ordenado, um Ctrl-C no meio de um `VACUUM INTO` matava a
cópia pela metade, e o `defer database.Close()` nunca rodava — defer não roda
quando o processo morre por sinal.

### O nginx, o compose e o HTTPS saíram (2026-10)

O app chegou a rodar num compose com um nginx na frente terminando HTTPS
autoassinado. Os dois foram apagados por decisão do dono: a infra custava mais
do que comprava. O que se perdeu é o app instalar no telefone; o `COOKIE_SECURE`
desligado é a outra metade — ligado em HTTP, o login não conclui e nada avisa.

### O socket que virou rota por comando, e o gancho que ficou sem dono (2026-10)

O tempo real já foi um socket bidirecional, com caminho próprio no mux, fora do
roteador, e por isso com um `guardSocketOrigin` repetindo o CORS por conta. Ele
virou uma rota por comando mais um `GET` longo, e depois a Mesa ganhou fluxo
próprio pelo `events.Bus`; a rota `/events` e o gateway não existem mais, e o
`SSEHub` ficou só emitindo.

Ao apagar o gateway, o `characterChanged` — então um GANCHO que o
`SocketHandler` preenchia — ficou sem quem o ligasse. O `go test ./...` seguiu
verde (havia até um teste afirmando que nulo era caminho normal), e quem acusou
foi o e2e de dois clientes. Hoje ele é um método, e o `SocketHandler` não existe
mais.

### Quatro mecanismos de aviso com a mesma forma (2026-10)

Antes do `events.Bus`, havia quatro mecanismos de "algo mudou" sem nome em comum,
três deles `chan struct{}`: diziam QUE algo mudou e nunca O QUÊ, e o `select` do
fluxo da Mesa tinha um `case` para cada um só para juntar de volta o que estava
separado por acidente de onde o estado mora. Um `chan struct{}` colapsa — dois
"mudou" pendentes não dizem mais que um —, e é por isso que o descarte no
barramento tipado passou a ser contado.

### O espelho do `targetKey` divergiu em três pontos (2026-10)

O guarda de empilhamento morava no `domain/catalog` com um `switch` que espelhava
o do motor, porque o `targetKey` é privado do `domain/engine`. O espelho lia a
`flag` de `target["flag"]` quando o catálogo a escreve em `name`, e não separava
o `maneuver` e o `displacement` por escopo. Medida antes do conserto, a
divergência era INERTE — mudava 12 baldes de 108 e nenhum veredito, porque os
casos que colidiriam eram condicionais —, mas não era segura. Ao mudar para o
motor, o denominador cresceu de graça: o guarda antigo enumerava nove recursos à
mão e via 189 entidades; o novo via 208.

### O que saiu quando a manobra parou de declarar vencedor (2026-10)

O `ManeuverOutcome` tinha `Won`, `Reroll` e `Imposes`, e um `switch` que
desempatava pelo bônus maior e pedia outra rolagem com bônus iguais — e o guia
afirmava isso como regra do motor muito depois de o código mudar. O
`CommitAttack` chamava um `ImposeCondition` que ramificava pela ficha e emitia
evento próprio; essa função não existe mais. O crachá da faixa dizia "Derrubou" /
"não derrubou". E a primeira versão da faixa escrevia "fica caido" — o id da
condição vazando para a tela, antes de o `book.ConditionName` traduzir.

### O guarda de ícone passava verde com três quebrados (2026-10)

Ele lia os `.templ` do próprio diretório e casava `@icone("X")`, enquanto as
cenas moram em `serve/web/*` e escrevem `@ui.Icon("X")`. Quem achou os três
buracos foi OLHAR a tela — e a resposta não foi olhar melhor, foi reescrever o
guarda para varrer `serve/` inteiro, com as duas grafias e denominador.

### Modificadores que a ficha mostrava e a conta ignorava (2026-10)

O `critRange`, o `critMult` e o `damage` de escopo `this` estavam no catálogo,
apareciam na aba Efeitos e não entravam em conta nenhuma. Cada um custou uma
medição própria, e a forma dos três era a mesma: o rótulo funcionava, a conta
não existia. A varredura que veio depois achou mais três alvos, somando 28
modificadores mostrados e ignorados — metade deles do `maneuver`, que o motor
ainda não resolvia. Daí o guarda de alvos declarados.

O terceiro tipo de inércia veio do livro: o `fearResistance` tinha entrada e
nenhum leitor, e a p229 mostra que ele era o `resistance` com condição — *"uma
criatura com resistência a magia +2 recebe +2 em testes de Fortitude, Reflexos
ou Vontade contra habilidades mágicas"*. Ele não existe mais: a entrada virou
`resistance` + `against: medo`, como o catálogo já escrevia a resistência a
veneno.

Duas armadilhas de instrumento no caminho, as duas com número plausível: uma
busca por `armorPenaltyExpertises` ACHAVA o nome no `breakdowns.go`, mas ali ele
é a variável com a lista de perícias que a armadura penaliza, e não leitor do
alvo; e somar o despejo que o motor lia com os arquivos embutidos contou cada
modificador duas vezes (28 manobras onde havia 14), porque eram cópias. O
despejo foi apagado: o motor lê os embutidos.

### A sabotagem mostrou que a rede de casos do ataque já prendia a ordem (2026-10)

Quando o ataque virou sistemas sobre parcelas, a ordem entre eles foi sabotada
uma aresta por vez: trocar o crítico com o acerto, as parcelas com o descarte do
erro e a soma com a RD saía vermelho em casos nomeados do livro. Só faltava
prender que cada sistema escreveu o componente dele, e esse guarda novo nasceu
vermelho: o bônus de crítico de um ataque que errou saía da conta dizendo "o
ataque não foi um acerto crítico", verdade que não explica nada. Daí "suprimir é
write-once, e a primeira razão é a que fica".

### O `Despawn` saiu por falta de chamador e voltou com o tabuleiro (2026-10)

O núcleo de ECS perdeu `Alive` e `Despawn` por não ter quem os chamasse, e o
comentário do `World` deixou escrito onde voltariam: no dia em que o tabuleiro
entrasse, porque lá a peça sai do mapa de verdade. A redação foi esse dia. O
modo de falha que o `Despawn` descreve (matar a entidade e deixar os
componentes, com a consulta continuando a visitar) ali não é número errado: é a
peça escondida chegando à tela de quem não devia vê-la.

### Guardas do tabuleiro que nasceram de sabotagem verde (2026-10)

- Invertendo a ordem em que a redação visitava as peças, a suíte inteira ficava
  verde: todos os casos afirmavam conteúdo, e conteúdo sobrevive a um
  embaralhamento. A ordem ganhou guarda no dia em que passou a decidir qual peça
  a tela oferece para mover.
- A conta do corpo da peça estava escrita duas vezes, no `occupied` e no
  `TokensInRectangle`, cada uma com o mesmo conserto de footprint zerado na
  frente. Trocando `side <= 0` por `side < 0`, a suíte ficava verde, porque toda
  fixture criava a peça pelo `AddToken`, que já põe 1 no zero; a peça que vem do
  banco não passa por ali.

### O guarda de ordem que não reprovou com uma fonte nova (2026-10)

Sabotando a ordem da coleta, três dos quatro pares adjacentes passavam verdes:
nenhuma fixture do oráculo tinha todas as fontes ao mesmo tempo. Na resolução,
colher o fator antes de julgar a condição passava verde na suíte inteira, porque
os fatores de então vinham da tabela de condições, sem condição. E quando a
devoção entrou como fonte nova, o guarda de ordem da coleta passou verde: o
personagem dele não era devoto de ninguém.

### A regra do poder de classe copiada quatorze vezes (2026-10)

O verbete do poder e a concessão por classe moravam na mesma linha, e a regra era
copiada para cada classe que concedia o poder: o "Aumento de Atributo" tinha
quatorze cópias byte a byte iguais. Não tinham divergido, mas nada impedia — é o
caminho que `general-powers` e `origins` percorreram até ter dezenove poderes
com duas regras. Por um tempo o motor nem seguia o ponteiro, e um guarda proibia
modificador no verbete; ele caiu quando o Aumento de Atributo precisou do
primeiro.

### Os leitores que a memória não listou (2026-10)

Ao separar verbete de concessão, foram mapeados três leitores de
`class-powers`. O quarto, o `FlattenedPowers` do acervo do mestre, só apareceu
quando o `TestPowersComeFromTheThreeCatalogs` reprovou; o quinto, o auditor de
classes (então em Python), estourou em `KeyError: 'name'`. `grep` pelo nome do
recurso responde em um segundo o que a memória responde errado.

### Dois identificadores, contra a recomendação medida (2026-10)

A decisão de ter `id` e `uid` foi do dono, contra uma recomendação medida de
manter só o `id`: o catálogo tinha um renome em mil e tantos nomes de história,
e ele era um hífen. O renome em massa veio horas depois, trocou mais de uma
centena de ids, e nenhum uid mudou. A primeira versão dos guardas de uid
repetia o escopo da cunhagem, e as duas cópias divergiram na hora: a cunhagem
ainda era Python, onde lista vazia é falsa, e 126 itens ficaram de fora de um
lado só.

### O id que era rótulo, e a coincidência que o escondia (2026-10)

Os catálogos de raça e de origem guardavam `{"id": "Humano"}` e
`{"id": "Acólito"}`: o id era o rótulo, e renomear a raça na tela renomeava a
chave. O motor indexava as raças duas vezes, uma pelo id e outra pelo NOME, e as
duas casavam por coincidência. Slugificar desfez a coincidência, e a busca por
nome passou a devolver `nil` — o humano perdendo os `+1` em três atributos,
calado. Quem pegou foram as fichas do oráculo: o diff do renome tinha centenas
de linhas, todas de chave e nenhuma de número. E o Kharvos mostrou o canal que
nenhuma varredura de campo alcança: a escolha da Deformidade, guardada como JSON
dentro de uma string, deixou de ser lida.

### O bônus de recorte fino que subiu o ataque (2026-10)

O `Desejo de Liberdade` (+5 para resistir à manobra) foi modelado primeiro como
`expertise:Luta`, que é literalmente o teste que a p234 manda rolar — e estava
errado: em T20 o ataque também é teste de Luta, e ligar o bônus defensivo subia
o ataque da arma empunhada em 5. Ao fechar uma leva de poderes condicionais,
quase todos couberam no vocabulário que já existia; o que faltava era o dado.

### O `levelStep` errava os patamares (2026-10)

Antes do `per: patamar`, tentou-se o "Coração Heroico" com `levelStep` passo 6:
ele errava cinco dos oito níveis de fronteira — para baixo o 5, o 11 e o 17;
para cima o 1-4 e o 18-20. E procurar o nome do poder no PDF falhou duas vezes:
o versalete volta em caixa baixa do `pdftotext`, e no texto do capítulo inteiro
o nome casa primeiro na tabela de pré-requisitos.

### A devoção só chegava ao PV e ao PM (2026-10)

O `GodPower` entrava no motor por uma porta só, o `vitalGrantMods`, que existe
para PV e PM e descarta todo alvo que não seja um dos dois. O texto de um poder
concedido aparecia na ficha e o número não mudava; a Bênção do Mana funcionava
por ser `maxPm`. A decomposição também mentia: o "PM máximo" de um arcanista
devoto mostrava +1 e o motor dava +2.

### Meio metro de bônus inventado pela conversão (2026-10)

Com o deslocamento guardado em metros, as botas reforçadas (+1,5m, p159) viravam
+2m ao arredondar na fronteira do JSON. Em quadrados a conta fechou exata, e a
cena do mapa parou de converter. Depois, sem a conversão de volta, reler o que o
motor escrevia convertia duas vezes — a armadura ia de −3m para −2 quadrados e
de −2 para −1 —, e quem denunciou foi o `roundTrip` dos casos de paridade, que
mede FORMA e não unidade.

### A isenção que era circunstância (2026-10)

O guia previa que todo fator viria de condição e que o primeiro incondicional
pediria revisão; a Alta Arcana chegou, e o "mais severo" sobreviveu, porque o
que o sustenta é o livro proibir o acúmulo. Também previa uma "imunidade a
fator" pelas Pernas do Mar do bucaneiro, e era leitura da descrição do catálogo,
não da página: a p48 corta o deslocamento só "quando está se equilibrando ou
escalando", circunstância que o app não modela.

### O catálogo era JSON, e a emenda viajava no personagem (2026-10)

O livro era JSON até virar Go, e uma chave escrita errado era ignorada calada por
quem lia. Na versão anterior das emendas de campanha, elas viajavam no
`engine.Character` e um guarda (que não existe mais) varria as portas
cobrando `forCharacter`; foi trocada porque, com as emendas no personagem, campanha-inteira e
personagem-específico chegavam ao motor misturados e a procedência morria no
caminho.

### Dois endereços para o mesmo termo, e a contagem que autorizou a migração (2026-10)

O opt-in do jogador tinha um endereço próprio (`fonte::alvo::nota::valor::tipo`,
gravado em `character_conditionals`) e o mestre tinha outro. Eles convergiram no
`TermID`, e a migração `00019` APAGOU as linhas na forma velha, porque linha que
não casa com nada deixaria o situacional desligado para sempre. A troca foi
medida antes: a tabela tinha zero linhas. O guarda que prendia a forma antiga
dizia que mudá-la exigiria migração — e o que autorizou a migração foi a
contagem, não a vontade.

### Um POST deu Fúria de graça ao bárbaro (2026-10)

O `ToggleSituational` gravava a chave crua do sinal do cliente. Um POST com
`{"conditional":"flag:furia"}` dava ao bárbaro o bônus de Fúria em ataque e dano
com o PM intacto e sem linha de postura — um interruptor que a lista de
situacionais nunca ofereceu. A tela escondia; o servidor aceitava. Daí a lista
de PERMITIDOS (`character.SituationalGroupsOf`) com um dono só.

### Ligar o grupo dobrava um modificador (2026-10)

Enquanto a chave do interruptor de grupo era o endereço do PRIMEIRO membro,
ligar o grupo dobrava um modificador, e o crachá ao lado dizia "3 mods". A
postura tinha o mesmo desenho: gravava uma linha por condicional, calculada ao
entrar e recalculada ao sair. Não era defeito vivo quando foi consertado — as
flags do livro eram todas de postura, que a tela de situacionais exclui —, mas
passaria a ser com as primeiras flags fora delas.

### O catálogo vazio do fixture mediu o contrário do que acontece (2026-10)

O fixture das cenas primava `{"items":[]}`, e regra sumiu do TESTE sem sumir da
produção, duas vezes na mesma épica: um escudo foi medido sendo VESTIDO, porque
o eixo de equipar não achava o item e devolvia "sem opinião"; e a distribuição de
atributo do humano passou com três vezes o mesmo atributo, porque a raça não
estava lá e a validação tratava desconhecido como completo. Nos dois casos o
guarda ficou verde. Depois, com o PV máximo derivado do catálogo, um servidor de
teste sem catálogo passou a dar fichas com os poços em zero — e o primado subiu
para o `newTestServer`.

### Vinte poderes num personagem de nível 1 (2026-10)

As regras de escolha de poder — vagas por nível, benefícios de origem, caminhos
e deuses por classe — só a tela sabia, e a gravação aceitava os blobs de escolha
sem conferir nada. Um pedido montado à mão punha vinte poderes num nível 1, e o
motor somava os modificadores de todos. Quando a validação estrita entrou, dois
testes revelaram fixtures ilegais pelo livro: um poder inventado e um personagem
sem classe com um poder escolhido.

### O schema ficou verde com metade do bestiário errada (2026-10)

44 dos 80 verbetes do bestiário tinham atributo ou resistência trocados, vários
com o número do verbete VIZINHO — a Serpe carregava as resistências do Ogro.
Um inteiro no lugar de outro inteiro é schema válido. Foi o que fez nascer o
primeiro auditor contra o PDF.

### A leitura do PDF virou módulo na terceira cópia (2026-10)

A leitura nasceu no auditor do bestiário, foi copiada para o de magias, e virou
o `bookpdf` quando o terceiro ia copiá-la — as duas cópias já tinham divergido.
O filtro de lixo (`WithoutJunk`) morava em três auditores, também divergentes.
Os auditores nasceram em Python e viraram Go com a saída provada igual linha a
linha; a leitura das tabelas de saque foi extraída com os relatórios dos
auditores existentes BYTE A BYTE iguais antes e depois.

### O auditor de materiais e a página dividida (2026-10)

A Tabela 3-9 é uma matriz — seis materiais por cinco tipos de item —, e o
denominador é a contagem de células. As colunas não se acham pelo cabeçalho
("Aço-Rubi Adamante" chega num `<line>` só), e sim pela linha `Arma`, a única sem
travessão. A prosa ao lado tinha duas armadilhas com resultado plausível: o
BOXE "Fabricando Itens Superiores" divide a coluna com o Mitral e caía, na ordem
de leitura, entre o título do material e as metades dele — o sexto material saía
com esse nome, as três metades certas e a contagem certa, e quem denunciou foi a
outra âncora (o recorte é pelo bloco: a prosa do verbete e a do boxe começam em x
diferentes). E regra na ABERTURA: o "–1 espaço" do mitral e o "–2 em perícias de
Carisma" da matéria vermelha estão no parágrafo de sabor, não em metade nenhuma.
O mitral aplica a margem de ameaça e não o limite de Destreza da armadura
pesada: daí o denominador ser por METADE, não por material.

### A lista de campos à mão descartou o dado extra (2026-10)

O `MarshalJSON`/`UnmarshalJSON` do `Modifier` listavam os campos à mão. Quando o
tipo ganhou `dice`, a lista não ganhou, e todo dado extra do catálogo era
descartado na leitura: seis encantos e um material entraram inertes com a suíte
verde. O `TestTheEnchantReachesTheWeaponCard` montava o `Modifier` em Go e nunca
atravessava o JSON — provava a composição por cima da fronteira quebrada.

### Três itálicos no mesmo verbete, e só um é composição (2026-10)

O verbete de item específico escreve a composição em itálico — "este machado de
guerra *de gelo eterno congelante formidável*" (p337) é base, material e dois
encantos. Mas o itálico também marca o nome do próprio item mais adiante e o de
uma magia ou de um encanto concedido só em certa condição (o Martelo de Doherimm
*adquire* o arremesso na mão de um anão). Duas perguntas separam: a frase é
CONTÍGUA no verbete? Ela nomeia peça que o catálogo conhece? Ausência de itálico
não é ausência de base (a azagaia dos relâmpagos e a espada baronial não têm
encanto, e o auditor confere a base contra o texto). E a página às vezes PERDE o
itálico na quebra de linha — o Escudo do Conjurador —, deslize de tipografia que
os irmãos da mesma tabela e o preço denunciam; a exceção fica escrita no auditor,
com a prova.

### A soma das faixas pegou o leitor perdendo duas linhas (2026-10)

O catálogo de encantos começou VAZIO, e "transcrevi os encantos" e "transcrevi
metade" têm a mesma cara no terminal. O que salvou foi a Tabela 8-8 ser de
rolagem: as faixas têm de ladrilhar 1..100 (e as do catálogo, 1..90; de 91 a 100
a tabela manda rolar noutra). O filtro de lixo descartava o que casa com
`^\d{1,3}$`, o padrão de número de página, e comeu as faixas de um valor só —
Energética (46) e Lancinante (64). Saíram 26 linhas com nomes e faixas certos e
dois efeitos colados no vizinho de cima; a soma não fechou. A 8-10 tinha outras
armadilhas: nome e efeito na mesma célula em duas linhas, e a chamada de rodapé
ora em célula própria, ora grudada no nome ("Esmagador 1"). O leitor de verbete
teve o defeito irmão: sem recortar por coluna, a tabela inteira entrava como
corpo da Magnífica — 939 caracteres num verbete de 370.

### O bloco de uma raça atravessa a página (2026-10)

O Hynne tem título e prosa na p27, o Golem inteiro vem depois, e a linha de
atributos do Hynne só aparece no alto da p28. Na p30-31 o Suraggel e o Trog saem
alternados: a habilidade colada à linha de atributos do Suraggel é do Trog. Nem o
título nem a página servem de âncora; a ligação linha→raça é feita por regras
semânticas.

### Ler Go mal inventa defeito (2026-10)

O auditor de condições compara a HERANÇA que o verbete declara ("Enredado. O
personagem fica lento, vulnerável…") contra o `conditionModifierTable`, em Go. A
primeira versão pegava uma linha por entrada e perdia a continuação, acusando o
Cego de não ficar lento quando ele fica.

### Duas âncoras se conferem; uma só dá tabela plausível (2026-10)

No auditor de perícias, trocar duas colunas na leitura da Tabela 2-1 produz 24
discordâncias nomeadas contra a linha de estado do verbete; com uma leitura só,
produziria uma tabela plausível.

### Casar por título achou a linha da TABELA (2026-10)

A Tabela 1-19 lista as origens uma por linha, então "Batedor" é impresso na p87
antes do verbete na p88. A varredura ingênua acusou 29 das 35 origens com
`bookPage` errado — lista plausível, nomes certos, zero defeitos. O verbete se
reconhece por ter `Benefícios.` abaixo do título, e a tabela é lida PRIMEIRO, com
as linhas dela retiradas da busca; senão o rodapé da coluna da p87 (com o
`Benefícios.` do Artista) dá ao "Trabalhador" um verbete que não é o dele.

### Números e perícias não bastaram para conferir prosa (2026-10)

Nos auditores de origens e de poderes, os números do "Vendedor de Carcaças"
coincidiram por acaso e seis regras trocadas passaram verdes: daí a terceira
dimensão, a cobertura de palavras. E a direção da medida: nos poderes o catálogo
RESUME, e perguntar "a regra do livro está aqui?" dá 20% para a `Torcida`, cuja
regra está perfeita. Medir "todo número do livro está no catálogo" acusaria todo
verbete que o livro ilustra com exemplo trabalhado.

### A linha do PDF juntou colunas vizinhas (2026-10)

Na p119 o elemento `<line>` traz `Furtividade Des • Armadura Iniciativa` —
título, estado e o título da perícia da coluna vizinha na mesma altura. Daí o
`Book.Words`.

### Um campo pode não morar no verbete que o declara (2026-10)

O `paladinoEligible` e o `druidaEligible` do deus não se derivam da lista de
devotos: a Valkaria aceita "membros de todas as classes" e não serve a druidas.
Quem restringe é a CLASSE, numa frase (p61 para druidas, p82 para paladinos).
Derivar teria acusado a Valkaria por um defeito que não existe.

### A marca d'água colou no nome de um poder (2026-10)

A marca d'água do PDF (nome e e-mail do comprador) não ocupa linha própria: ela
cola no texto vizinho, e chegou como parte do nome do último poder de uma lista.

### Seed que mente não quebra (2026-10)

Só `create.items[].catalogId` passava por lookup no `seed-data.json`; raça,
origem, classe, deus, poder concedido, tamanho e magia entravam crus.
`machado-de-batalha` no lugar de `machado-batalha` produzia um personagem sem a
arma, com a ficha abrindo normal. A chave do `classChoices` é um nome de classe,
e essa referência só apareceu quando o `TestEveryCreateFieldOfTheSeedIsClassified`
obrigou a classificar campo por campo.

### A suíte era 99% espera de disco (2026-10)

Migrar do zero eram 7,1 ms em tmpfs e 2.102 ms num HD; a suíte do `api/` levava
15 minutos com 10 s de CPU. O molde migrado uma vez e copiado por teste
(`infra/db/testdb`) tirou o disco da conta.

### Sabotar provou que um teste passava por acidente (2026-10)

Um teste de PV passava porque, em todos os casos, a primeira classe também era a
maior. E os guardas do `convention` foram provados contra a árvore do commit
anterior (`git archive`, o guarda copiado por cima): 775 reprovados num e 128 no
outro — caso negativo real, e nenhuma chance de a sabotagem sair inerte, o que já
tinha enganado duas vezes.

### O comentário entre atributos do templ, de novo (2026-10)

O comentário entre dois atributos derrubou o parser oito vezes — três na mesma
sessão, por quem não leu a armadilha antes de escrever. A mensagem muda com o que
envolve o elemento, e por isso ninguém reconhece o sintoma na segunda vez.

### O `goimports` reprovou a CI com a suíte local verde (2026-10)

Um `goimports -w` num glob de pacote alcançou os `_templ.go` e juntou ao bloco de
imports os dois de runtime que o `templ generate` emite soltos. Doze linhas sem
uma de comportamento, e a CI reprovou em "generated templates were stale".

### Medir contra o servidor que não reiniciou (2026-10)

Uma medição de layout inteira foi feita contra a página velha: 74px de
deslocamento "que não sumiam" depois do conserto, porque o conserto não estava no
ar — o `go run` tinha compilado antes do `templ generate`.

### Os `@source` do Tailwind não mudavam um byte (2026-10)

O guia dizia que o scanner lia só os `.templ` declarados. Tirar os dois `@source`
do `app.src.css` não mudou um byte da folha (105.328 com e sem): a detecção
automática do Tailwind v4 já varria a árvore. Depois, uma conta no leitor de PDF
virou utilidade na folha, e `infra/`, `cmd/` e o livro transcrito ganharam
`@source not`.

### A prosa e um nome de variável entraram na folha (2026-10)

Uma varredura apagou centenas de sítios de classe, a folha foi compilada, e só
DEPOIS dois guias ganharam a explicação da receita: a CI reprovou com uma linha de
diferença na folha. Noutra fatia, uma variável local de Go com o nome do número
que ela representava acrescentou uma utilidade de `font-variant-numeric` à folha,
numa mudança que não tocava tela. O comentário que explicava o caso o disparava
de novo, e foi reescrito nomeando o mecanismo sem o token.

### A camada `components` perdeu para um utilitário (2026-10)

Uma regra da casa em `@layer components` perdeu para o `lg:static` que a coluna
trazia no `class=`, e o mapa cresceu 18px em vez de 728 — só a divisa, sem
concorrente, tinha flutuado. "Cresceu um pouco" parecia regra aplicada com conta
errada; era regra descartada. Tirar o `lg:static` trocou o defeito de lado: o
`fixed` da classe base passou a ganhar e a coluna virou tela cheia em toda
largura. No mesmo trabalho, um painel ancorado com `right: 0` sobre um `inset-0`
foi para o lado errado do mapa.

### O token inventado e o papel semântico fora da lista (2026-10)

`text-grimorio-ink` parecia irmão de `text-grimorio-gold` e não estava na paleta:
o crachá de contagem dos Efeitos saiu dourado sobre dourado (1,53:1) e
atravessou uma fatia inteira. Depois, `text-destructive-foreground` foi escrito
com o token nunca declarado, e o guarda passou por cima porque só olhava a paleta
`grimorio-*`: o botão "Excluir a sessão" saiu a 4,33:1. O guarda passou a cobrir
os papéis semânticos, e o piso do denominador subiu junto. O glob dele, quando
era padrão de NOME, deixou de casar duas vezes — quando o kit mudou de arquivo e
quando os arquivos perderam um prefixo.

### `\bgrimorio\b` levou a classe de escopo junto (2026-10)

Uma varredura trocou `grimorio` por `grimoire` com `\bgrimorio\b`, e o hífen é
fronteira de palavra: `scene-grimorio` foi junto, e com ela o app inteiro perdeu
a cor. A mesma entrada estragou a rota `/grimorio` e o nome de arquivo
`grimorio.js`. `go build`, `templ generate`, `go vet`, a suíte Go e o guarda de
tinta ficaram verdes; quem acusou foi o e2e. A primeira versão do guarda de
escopo varria o arquivo inteiro e reprovava valores de `data-slot`; depois
passou a ler também o `templ.KV`, porque a casca escolhe o escopo por condição.

### O estado da Mesa nascia dentro do `New` (2026-10)

A lente e a aba que cada pessoa escolheu nasciam no `table.New`. Quando a sessão
era apagada, o ciclo da sessão — montado antes da cena — não tinha como
esvaziá-los. Montados no hospedeiro, o mesmo objeto foi para os dois.

### O caso de uso por parâmetro encolheu as portas (2026-10)

Com os casos de uso entrando pelo `New`, a porta da ficha foi de dezoito métodos
para oito, a de campanhas de vinte para seis e a da porta de entrada de nove para
três. A porta das campanhas traduzia sete sentinelas num enum que a cena
declarava só para a classificação atravessar; com as recusas no `app/`, a cena as
lê direto e o tipo do meio sumiu.

### Órfãos guardados para um gesto que não os usou (2026-10)

Numa varredura de cinco funções sem chamador, três eram dívida e dois eram
funcionalidade perdida: a camada de baixo inteira no ar, com teste, e nenhuma
rota chegando nela. Quando o gesto que os esperava finalmente chegou, ele não
usou nada dos três órfãos guardados para ele. Daí "leia o que a coisa FAZ" e
"guardar por 'vai ser útil' é apostar num desenho que não existe".

### A bancada que montava o estado por uma porta morta (2026-10)

Uma mesa inteira passava verde chamando um `Join` que nenhum caminho de produção
chamava desde que a SPA saiu. Em outro caso, o fixture escrevia uma coluna que a
tela nunca escrevia, e toda mesa aberta pela tela nascia com ela nula — com a
suíte verde. Daí "monte o estado pelo caminho de produção".

### A docstring certa que ficou errada (2026-10)

Uma docstring estava certa quando foi escrita, continuou plausível por duas
épicas e estava errada no dia em que alguém a leu para decidir. A sonda que caça
citação velha casava `.Simbolo(` e perdia todo nome que colidia com a biblioteca
padrão; casando também `Simbolo(`, mascarava nome sobrecarregado. Quem reprovou
a sonda foi o controle positivo — um caso conhecido plantado —, não a leitura
dela.

### O mapa medido sem marcador (2026-10)

A cena do mapa estava na lista de visitas dos guardas de leiaute, mas nenhum
visitava um mapa COM marcador: os ícones espremidos e o número de dois dígitos
do acervo atravessaram todas as medições. Daí, também, a nota de que esconder
rótulo por `display:none` o tira do leitor de tela e do `getByLabel`.

### O `*Server` que existia para a união das portas (2026-10)

A repartição do `*Server` apagou mais de cem manipuladores sem chamador. Ele
tinha dezenas de métodos exportados existindo para cumprir a UNIÃO das portas
de todas as cenas, quando quase todas as assinaturas tinham exatamente UMA cena
pedindo. O guia chegou a dizer quantos sobraram, e o `grep` respondeu outro
número — daí a pergunta ficar no guia, e não a resposta. Dois guardas que
varriam rotas de `/characters` morreram com a API, e a invariante mudou de casa
para o `TestNoSheetWriteAcceptsAStranger`.

### A cópia de função pura que perdeu o acento (2026-10)

Quando o `book` saiu do `api`, ele não podia importar o normalizador de acento;
a cópia de nove linhas, reescrita de memória, chamou a função que só faz
`ToLower`, e `KeyOfName("Atuação")` passou a devolver "atuação" com acento — a
classe deixou de ligar a perícia que treina, sem erro e sem log. No `account` a
cópia divergiu na FRASE: a mesma regra de senha recusava em português pela tela
e em inglês pela rota JSON, e o teste que existia prendia a cópia morta.
Aconteceu três vezes na mesma série. Daí "copie o CORPO, e prefira o pacote".

### O renomeador que trocou palavra comum (2026-10)

Exportar o kit de `web/ui` passou os nomes para inglês, e um `re.sub` por
palavra trocou 250 arquivos — inclusive um comentário que virou "o diálogo
Int", e `data-nav-layout` que virou `data-nav-ui.Layout`: o HTML aceitou calado
e só a navegação por teclado daquela cena morreu. O que funcionou foi um
renomeador que pula comentário e string, com o compilador como rede (`ui.Field
:= …` não é declaração válida), e um `grep` por `-ui.` para o hifenizado.

### O `gap` da base que vencia o do tamanho (2026-10)

Medida a ordem na folha compilada, a fonte MENOR sempre vencia (`text-base` antes
de `text-sm`), o `gap` MAIOR vencia, e `hover:text-foreground` vinha depois de
`hover:text-destructive-ink`. A da fonte era latente (o primeiro botão de ícone
a pedir `text-lg` sairia pequeno); a do `gap` estava viva — o `gap-2` da base
vencia o `gap-1.5` do tamanho pequeno, e o botão pequeno nunca teve o respiro
escrito para ele; a do hover é por que o ghost que apaga virou uma variante própria.
Esse botão saiu com a última tela que o usava (ALE-546), e a lição vale para qualquer
receita de classes concatenadas.

### Os medidores que não viam o Tempestade (2026-10)

Quando o segundo sistema de desenho entrou, três medidores passavam VERDE sobre
a cena nova sem medi-la: o de contraste varria só `.scene-grimorio *` (cinco
textos medidos na `/sistema`, os do diálogo do livro), o de face lia só o
`index.css`, e o de escopo lia só o `class="…"`. A migração das telas grandes
(ficha, sessão do jogador, mesa do mestre) correu em paralelo atrás de uma marca
`?tempestade=1` no mesmo endereço, que depois virou cookie na mesa e saiu de vez
quando a última parte migrou.

### O anel de foco que era alcançado em vez de desenhado (2026-10)

Apagar todas as cópias à mão da receita de foco não mudou um pixel: a regra
global não-layerada já decidia. O que mudava era o repouso — o `outline` do
shadcn e o `transition-*` do Tailwind faziam o anel ser interpolado em 150ms, e
quem tabulava na autorrepetição (~33ms por parada) via o meio do caminho, raspando
o piso de 3:1 do WCAG 1.4.11. As três plaquetas de rádio eram uma segunda
aparência de foco (afastamento de 2px), achadas pelo medidor subindo a árvore a
partir do rádio invisível. A primeira versão do medidor isentava tudo dentro de
`[data-nav-region]` e acusou 37 nós de uma vez na forja — os rádios e o
contêiner rolável, que estão no trilho e não são item dele.

### O endereço de página que o guarda confundiria com o de salvar (2026-10)

As notas ganharam quatro lugares (lado a lado, empilhado, flutuando e janela). A
janela entrou no guarda de endereço em vez de ganhar um próprio, porque o
extrator já parava na primeira vírgula de topo. O método ficou GET de propósito:
um POST em `/campanhas/1/sessoes/4/notas` casaria a rota de SALVAR.

### A recusa medida pelo status (2026-10)

A primeira versão do guarda de corpo quebrado reprovou quatro rotas por medir o
STATUS, que é o contrato errado para as que são comando (recusam em 200). Um
teste do pincel apagava de (0,0) a (6,6) e era estruturalmente incapaz de
detectar um `from` que parou de ser lido, porque (0,0) é o valor-zero.

### O guarda de coordenada que escapou pelo nome (2026-10)

A coordenada saiu do caminho para o corpo, e o guarda de rota nasceu para a
família não voltar — e passou VERDE por cima de `/grupo/mover/{dx}/{dy}`, porque
o padrão listava `x`, `y`, `x2`, `mx`. Era a terceira escapada da família: a
`colar` por um `grep` por `base+"…"`, o `/pecas/nova` por um argumento errado.
A conversão levou junto cinco leitores mortos de `chi.URLParam(r, "x")`, sem
rota registrando `{x}` e sem chamador — função de pacote sem uso não é erro em
Go.

O guarda foi refeito porque era duas coisas estreitas. Como lista de PROIBIDOS,
nove grafias sabotadas passaram verdes, entre elas `{col}`/`{lin}` e
`{cx}`/`{cy}`. Como parser de LINHA, perdia o `base :=`, o registro em duas
linhas e o `r.Route` — e sabotando um `r.Route` o denominador SUBIA, porque
contava o filho e ignorava o pai. Com o `chi.Walk`, o denominador ficou exato:
212 rotas contra as 193 que o regex estimava, que incluíam dez `r.Header.Get(`
copiados por toda cena nova.

### O passo que perderia o filtro (2026-10)

Postando `/vitais/pv/-1` com e sem os sinais no corpo: nas perícias filtrando
por "Perce", 19.091 bytes com o termo achado contra 24.122 com a lista inteira de
volta; nos poderes filtrando por "Ataque", 23.247 contra 23.389 com o filtro
perdido. Converter custaria listar os 29 sinais da ficha no payload de cada
botão de passo.

### O `data-show` que piscava meio segundo (2026-10)

Com uma sonda de `requestAnimationFrame` instalada antes de qualquer script: 30
quadros de pano preto nas Perícias, 13 na Mochila, 10 no Combate, 9 nas Magias —
meio segundo a 60fps, que o dono descreveu como "um dialog de 1 frame que some".
A impressão a olho subestima. Eram dezesseis nós: seis sobreposições e dez
conteúdos escondidos por sinal puro.

### A aba escondida pelo aviso igual (2026-10)

O `@post` que perdia o `?tab=` ficou escondido uma fatia inteira porque todas as
abas da ficha desenhavam o mesmo aviso de "ainda vive na ficha antiga" — o
salto não tinha aparência. O primeiro painel de verdade o denunciou no primeiro
clique.

### O foco que fechava a ficha do bestiário (2026-10)

O pedido do foco existia para a prévia da seta e não levava `abrir=1`; chegando
depois do clique, fechava a ficha que o clique abrira. O CI pegou duas vezes o
que a bancada nunca reproduziu; foi reproduzido atrasando só a resposta sem
`abrir`. Quando o bestiário virou pacote próprio, mudar o guarda de casa junto o
teria deixado medindo um dos quatro `.templ` com `data-on:focus` — verde, porque
o piso antigo era "achou algum".

### O botão que não fazia nada (2026-10)

As recusas em 4xx atravessaram três fatias da ficha sem ninguém ver: gastar mais
dinheiro do que se tinha fechava o diálogo e o saldo continuava igual, sem uma
palavra. O `‹` da mesa viveu morto uma issue inteira depois da tradução das
rotas, postando em `initiative/previous-turn`; a primeira versão do guarda de
endereço passou VERDE sobre ele porque a bancada não tinha mesa viva.

### O balão "preencha este campo" na forja (2026-10)

Com o nome do herói `required`, o primeiro clique numa carta de classe só abria
o balão de validação — e a pessoa escolhe a linhagem antes de batizar o herói,
então o balão aparecia sempre.

### O carimbo lido a cada segundo (2026-10)

A primeira versão do aviso de ficha lia o `updatedAt` do personagem em todo
quadro do batimento — uma consulta por segundo por jogador, quase sempre para
saber que nada mudou. A decisão do dono foi escutar o evento. Antes do
barramento, quem não tinha ficha recebia um canal NULO (que num `select` nunca
dispara); não pedir o interesse disse o mesmo sem o truque.

### O véu que o morph apagava (2026-10)

Com o mesmo nó consultado em quatro instantes: ligado ao animar, ligado na
microtarefa, DESLIGADO no primeiro quadro e depois. Esperar um quadro antes de
pendurar resolveu o gesto isolado (o véu viveu 361ms dos 380 pedidos), mas dois
gestos a 60ms matavam o primeiro véu em 17ms. Com o nó em `position:fixed` no
`<body>`, 50 de 50 piscadas completaram, inclusive em rajada; animação no próprio
nó teve zero desconexões em 50 remendos. Um `if (oldValue === 'true') continue`
escrito por medo de repetição nunca foi verdadeiro — provado removendo-o.

### As notas adotadas pelo PiP (2026-10)

Movendo a coluna das notas para uma janela Document Picture-in-Picture, com o
MESMO `input` sintético dos dois lados: 1 POST antes e ZERO depois, com a faixa
dizendo "Salvo". A primeira medição comparava um `fill()` antes com um evento
sintético depois, e responderia sobre o instrumento. Com um documento próprio,
18 de 19 nós com `data-show` foram escondidos pelo Datastar daquela janela.

### O probe sintético que matou o gesto (2026-10)

Na mesma sessão, um `dispatchEvent` de ponteiro levou a "o retângulo não
funciona" (errado: era o probe) e a "o pincel também parou" (certo, mas pelo
ternário). As duas conclusões vieram do mesmo probe.

### O link da Ana sob o nome da Bia (2026-10)

O diálogo de redefinição de senha é um só para todas as linhas, e o token chega
por remendo num `<div>` de id fixo. Gerar o link da Ana, fechar e abrir a caixa
da Bia mostrava o link da Ana sob o nome da Bia — e quem estivesse com pressa
entregaria a chave da conta errada.

### O comando da ficha voltava na primeira aba (2026-10)

O `@post` de cada comando da ficha mandava só o endereço escrito nele, sem o
`?tab=` da página. Mexer no PV com a Mochila aberta devolvia a cena na PRIMEIRA
aba, sem nada falhar. Daí o `sheetPost` escrever o `?` em todo comando, e o
`TestNoSheetCommandLosesTheTab`.
