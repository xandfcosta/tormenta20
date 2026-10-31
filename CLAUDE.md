Gerenciador de mesa e ficha de **Tormenta 20**. O que se LÊ é em português — a
tela, a documentação e os comentários falam a língua da mesa; o que se ESCREVE
em código é em inglês. Ver "Idioma".

São dois pacotes. `engine-go/` é o app: a API HTTP na :3001, o motor de regras,
e as CENAS em `.templ` servidas com Datastar — mais a folha e as ilhas de JS
delas, em `serve/web/assets/src`. Ele se divide em cinco grupos — `domain/`,
`app/`, `serve/`, `infra/` e `cmd/` —, e o mapa de onde procurar cada coisa está no
[guia dele](engine-go/CLAUDE.md). `e2e/` é a suíte de Playwright, que dirige o app
rodando. Um processo serve tudo, direto na máquina e em HTTP, comprimindo em
gzip ele mesmo — sem contêiner e sem nada na frente. Em dev é o
`pnpm dev` do `engine-go/`, sobre o `data/t20-dev.db` semeado.

Houve uma SPA em SolidJS (`frontend/`) e um motor compilado para WASM que rodava
no navegador. Os dois saíram: as cenas são renderizadas no servidor,
e a regra tem um lugar só.

## O sistema INFORMA; o mestre DECIDE

**Esta é a pergunta que vem antes de "como implementar": a fatia que você está
prestes a propor TIRA uma decisão do mestre?** Se tira, ela não existe — por mais
fiel ao livro que seja.

O que o app faz é **levar do ponto A ao ponto B**: do *quero fazer alguma coisa*
— e aqui está, na mão, o que isso pede — até o *isto foi o que aconteceu* — e
aqui está, na mão, o que o mestre precisa para aplicar. Ele existe para
**diminuir atrito de cálculo e de decoreba de livro**, e para nada além disso. O
que ele compra com esse atrito é tempo de interpretação.

### O que ele DÁ

- **Dado**, no sentido de informação: atributo, perícia, dano de arma, Defesa de
  escudo, preço de item.
- **A conta difícil**, pronta e decomposta: o que uma condição faz com as
  perícias, o que material e melhoria fazem com um item, o que uma habilidade faz
  com o personagem.
- **O mapa**, e o que o personagem pode fazer a partir de onde está.
- **O que foi rolado**, para o jogador saber o que rolar e o mestre saber o que
  aconteceu.

### O que ele NÃO faz

**Não arbitra.** Um jogador não escolhe outro, os dois rolam e o sistema anuncia
o vencedor e aplica os efeitos em cada um. Quem diz quem venceu, e o que isso
causa, é o mestre — na voz.

Isto cortou fatias inteiras que o livro descreve e que seriam fiéis a ele: o
TESTE OPOSTO e o AJUDAR da p220-221, e os TESTES ESTENDIDOS da p222-223. As três
pressupõem um turno dentro do sistema com embate de dados e um vencedor
anunciado, e é exatamente essa a linha. Elas ficaram fora de escopo de
propósito, e não se reabrem achando que foram esquecidas.

### A fronteira não é "rolar"; é "julgar"

Rolar um d20 e mostrar `d20 14 + 7 = 21` é dar a conta — o 7 é a soma de treino,
atributo e penalidade de armadura que ninguém quer fazer de cabeça. **Dizer se
21 passou é que é do mestre**, e é por isso que o teste de perícia não tem CD nem
veredicto: quem põe a CD a diz em voz, muitas vezes em segredo.

E o app **recebe o dado da mesa** tão de primeira classe quanto rola o dele: quem
rolou na mão digita o número e ganha a conta do mesmo jeito. Tirar o dado de
quem gosta de rolar seria cobrar o atrito que este guia existe para remover.

### O veredito é do mestre, e só ele o vê

Acertar ou errar é sempre interpretação do mestre. No ATAQUE e na MANOBRA, a
faixa do MESTRE mostra as rolagens, o desfecho que os números sugerem — acerto
ou erro, sucesso ou falha, crítico ou não — e a consequência que o livro prevê,
para facilitar a leitura. A palavra final é dele, em voz.

O JOGADOR não vê esse julgamento, nem a Defesa do alvo, nem a rolagem do outro
lado: no papel do personagem, ele não conhece a ficha de outra entidade. Ele vê
a própria rolagem e ouve o resto do mestre.

## Antes de mexer

- **Não assuma.** Se a regra do livro ou a decisão de produto não estiver clara
  no código, no `.md` do pacote ou no [GLOSSARY.md](GLOSSARY.md), **pergunte**.
  Adivinhar regra de T20 custa mais caro que esperar a resposta, e o erro sai na
  ficha de alguém.
- **Regra do livro se confere no livro**, com a página citada. O offset entre PDF
  e livro é do [engine-go/CLAUDE.md](engine-go/CLAUDE.md) — é lá que as regras
  moram, e o número não é repetido aqui de propósito (ver "Documentação").
- **Mexeu em `.templ`?** `go tool templ generate`, e **leia a saída DELE**, não a
  do `go build`. **Classe CSS nova no app?**
  `engine-go/scripts/build-css.sh`. As duas armadilhas — e as do Datastar,
  que não deixam erro para trás — estão explicadas no
  [engine-go/CLAUDE.md](engine-go/CLAUDE.md).
- **Antes de commitar:** `go test ./...`, `go vet ./...` e `gofmt` no
  `engine-go/` — mais `pnpm typecheck` lá, quando mexer nas ilhas de JS. Mexeu
  em gesto de ponteiro, leiaute real ou fluxo entre dois clientes? `cd e2e &&
  npx playwright test` (sobe o próprio servidor e o próprio banco).
- **`git add` antes do `go test`.** Os guardas varrem `git ls-files`, então um
  arquivo novo fora do índice não é medido, e a suíte fica verde sem ele.
- **Releia a documentação que a sua mudança tocou** — o `.md` do pacote e o
  `GLOSSARY.md`. Não "atualize se mudou o comportamento": **releia**. Ver
  "Documentação".
- **Mexeu em tela? OLHE a tela** antes de fechar — é passo, não zelo, e o
  detalhe está logo abaixo.
- **Tela refeita pede mockup em HTML aprovado antes**, uma tela por vez.
- **Um escopo por vez.** O próximo começa depois do merge do anterior.
- **Achou trabalho entregue contrariando uma regra, ou um defeito fora do
  escopo?** Pergunte ao dono o que fazer antes de consertar ou registrar.

### O último passo é OLHAR

Uma tela pode mentir com a suíte inteira verde: o dado certo e a apresentação
errada. Por isso olhar é passo, e não zelo (ver o [APRENDIZADOS.md](APRENDIZADOS.md)).

**A divisão de trabalho É a regra.** A MÁQUINA julga LIMIAR e LIGAÇÃO: contraste
abaixo do AA, Cinzel abaixo do piso, alvo de toque, sinal escrito sem leitor, id
apontado que não existe. O OLHO julga COMPOSIÇÃO e HIERARQUIA: o que salta
primeiro e se era para saltar, o que ficou apertado, o que ficou solto, o que
desalinha. Nenhum limiar responde a essas quatro, e é por isso que este passo é
humano — não porque a suíte seja fraca.

- **Quais cenas:** as que a mudança TOCA, e os endereços delas estão em
  `engine-go/serve/web/appearance_scenes.json` — o registro que o
  `TestEveryPageSceneIsMeasuredForAppearance` mantém completo. Não escreva uma
  segunda lista aqui: ela envelheceria sozinha.
- **Em que larguras:** **390px e desktop, no mínimo** — a casa chaveia por
  LARGURA, então são as duas pontas que decidem. E cena que ramifica pelo DADO
  (mestre e jogador, quem conjura e quem não) pede uma passada por ramo.
- **Sem digitar senha:** a sessão do e2e já está em `e2e/.auth/`, e um script de
  Playwright com `storageState` entra autenticado sem ninguém digitar nada.
- **O que você viu vira o quê:** conserto no escopo em andamento, ou pergunta
  ao dono. E se o olho pegar
  coisa de LIMIAR, o conserto não é olhar melhor da próxima vez — é escrever o
  guarda, pela mesma razão que a seção "Documentação" dá para não restaurar
  narrativa.

## Ao terminar um escopo: revisar

Com o escopo pronto e a suíte verde, o código que ele mudou passa por uma
revisão antes do merge. A base são três skills de Fabio Akita
([`simplify`](https://github.com/akitaonrails/my-skills/tree/master/simplify),
[`improve-codebase-architecture`](https://github.com/akitaonrails/my-skills/tree/master/improve-codebase-architecture)
e [`post-refactor`](https://github.com/akitaonrails/my-skills/blob/master/post-refactor/SKILL.md)),
e o que vale delas aqui é isto:

- **O alcance é o diff do escopo**, e não o repositório. Revisão não reabre
  trabalho já mergeado.
- **Primeiro o portão:** `go test`, `go vet`, `gofmt`, o `deadcode` e, se o
  escopo tocou tela ou fluxo, o e2e. Ferramenta que não rodou é lacuna, não
  aprovação.
- **O que procurar no que mudou:** regressão (comportamento, padrão, migração,
  autorização), erro engolido ou recuo silencioso, helper duplicado, abstração
  com um chamador só, número mágico, comentário ou `.md` que deixou de ser
  verdade, acoplamento escondido entre pacotes, e teste que afirma
  implementação em vez de comportamento ou que depende de tempo e de ordem.
- **Simplificar preserva o comportamento exato**, segue a convenção do
  repositório, prefere o claro ao esperto e não troca nome que carrega sentido
  por menos linhas. Uma simplificação por vez, com os testes rodando.
- **Arquitetura se lê por PROFUNDIDADE.** Um módulo é fundo quando muito
  comportamento mora atrás de uma interface pequena, e raso quando a interface
  é quase tão complexa quanto o corpo. O **teste da deleção**: apague o módulo em
  pensamento — se a complexidade some, ele era repasse; se reaparece nos N
  chamadores, ele se pagava. A interface é a superfície de teste. Costura com
  um adaptador só é hipotética; com dois, é real.
- **O resultado é uma lista, e quem decide é o dono.** Cada achado com arquivo e
  linha, por que importa e o que fazer, classificado em BLOQUEANTE,
  DEVE-CORRIGIR, DETALHE ou OBSERVAÇÃO. Candidata de arquitetura vem numerada,
  sem interface proposta, com a pergunta "qual destas explorar?". Nada é editado
  sem o dono pedir.
- **Candidata recusada por um motivo que vale guardar** vai para o
  [APRENDIZADOS.md](APRENDIZADOS.md), para a próxima revisão não sugeri-la de
  novo.

## Estilo de código

- **Uma função se divide quando tem mais de uma RAZÃO PARA MUDAR, nunca por
  contagem de linhas.** Aqui morava "funções de 4 a 20 linhas", e ela foi tirada
  por atrapalhar mais do que ajudava: uma tarefa que é uma coisa só, picada em
  dez funções para caber no teto, obriga quem lê a remontar a sequência saltando
  pelo arquivo — e cada nome inventado no caminho é um nome a mais que não
  identifica nada. Um passo a passo linear e longo se lê de cima para baixo;
  cinco chamadas com nome genérico, não.
- **Arquivos abaixo de 500 linhas** — este teto fica, e por outro motivo: arquivo
  é unidade de RESPONSABILIDADE e de conflito de merge, não de leitura. Ele tem
  CATRACA (`TestNoNewFileGoesOverTheLineCeiling`): arquivo novo acima do teto
  reprova com o nome dele, e a dívida antiga mora numa linha de base que **só
  pode encolher**. Arquivo baselinado que encolheu sai da lista, e não sair
  também reprova.
- **Extrair é certo quando o pedaço tem sentido sozinho**: quando ele tem OUTRO
  chamador, outro motivo para mudar, ou quando dar nome a ele explica o que o
  corpo não explicava. Picar por picar produz módulo RASO — ver o teste da
  deleção em "Ao terminar um escopo".
- Retorno cedo em vez de `if` aninhado.
- **Nomes específicos e únicos.** Nada de `data`, `handler`, `Manager`. Prefira
  nomes com menos de 5 ocorrências no `grep` — um nome que já existe em cinco
  lugares não identifica coisa nenhuma.
- **Tipos explícitos.** `any` só na fronteira JSON, onde a forma é do outro lado
  — os sinais do Datastar, o corpo de uma API; o resto é struct. O código ainda
  tem `any` fora dela, e isso é dívida a limpar, não exemplo.
- Sem lógica repetida: extraia para função ou módulo.
- **Diante de duas opções que resolvem a tarefa igualmente bem hoje, escolha a
  mais fácil de mudar depois.** Não é prever o futuro — é não pagar a mesma
  decisão duas vezes. Menos lugares para editar, menos acoplamento, decisão
  adiável em vez de decisão travada.
- **Mexa só no que o escopo toca.** O que estiver tosco por perto não entra de
  carona: ele é achado da revisão de fim de escopo (ver "Ao terminar um
  escopo"), e o dono decide se vira trabalho.
- **Mensagem de exceção carrega o valor ofensor e o formato esperado.** "Caminho
  inválido" não ajuda ninguém no meio de uma sessão; "o caminho começa em (3,1) e
  a peça está em (0,0)" ajuda.

## Comentários

- **Escreva o POR QUÊ, não o O QUÊ.** Pule o `// incrementa o contador` sobre o
  `i++`.
- **O que mais paga é o CAMINHO NÃO TOMADO.** "Por que não o óbvio" é o único
  conteúdo que o código não diz sozinho, e é o que impede a decisão de ser
  refeita por quem chegar depois achando que foi descuido. Um cabeçalho de
  guarda longo se justifica assim — o que ele protege é o ESCOPO da varredura.
- **A DECISÃO mora aqui; o CASO mora no [APRENDIZADOS.md](APRENDIZADOS.md).**
  Não cite issue do Linear para explicar o porquê: o Linear tem limite de issues,
  elas são apagadas, e a citação vira ponteiro morto. Quem quiser a história tem
  onde buscá-la, e ela deixa de custar contexto a cada leitura do arquivo.
- **Explique na MENSAGEM DE FALHA antes do cabeçalho.** Um guarda que falha
  dizendo o nome do caso e o que fazer entrega a explicação no instante em que
  ela importa; o cabeçalho só alcança quem já foi abrir o arquivo.
- **Comentário envelhece como código, e se relê como código.** Um que deixou de
  ser verdade é defeito entregue, igual a um `.md` errado — e quem o pega é a
  releitura, não o compilador. O `TestNoCitationNamesAMissingSymbol` cobre uma
  fatia: citação a símbolo que não existe mais.
- **Não repita no comentário a regra que o guia já tem.** Cópias envelhecem em
  velocidades diferentes, e a do guia é a única que alguém procura. No código
  fica o que é específico DAQUELE arquivo; a regra fica no guia, e o comentário
  no máximo aponta para ele.
- Docstring em função pública: intenção e um exemplo de uso.

**Comentário não se preserva por preservar: ele se RELÊ**, como a seção
"Documentação" exige dos `.md`. E **não existe guarda de tamanho de comentário,
e não deve existir**: ele reprovaria os cabeçalhos de guarda, que são os
melhores comentários do repositório.

## Testes

**O objetivo é confiança de que o app produz o resultado que a gente quer — não
cobertura de cada pedacinho de código.** Um teste ganha o lugar dele protegendo
um resultado que alguém notaria quebrar. Tudo abaixo decorre disso.

- **Prefira INTEGRAÇÃO.** O teste padrão monta uma página de verdade (ou bate num
  handler de verdade, pelo roteador de verdade) com o I/O trocado na borda, e
  afirma o que a pessoa ou o chamador recebe. É a faixa que pega defeito de
  COMPOSIÇÃO — que é onde os defeitos deste repositório de fato estiveram — e é
  para lá que a cobertura nova vai primeiro.
- **Unitário para o que carrega REGRA**, não para o que carrega encanamento.
  Empilhamento de modificador, PV/PM, arredondamento, limites, rollback otimista,
  formato de fio: sim. Getter, formatador de uma linha, um `Set` + `sort` que a
  asserção reimplementa, e tudo que o typechecker já garante: não.
- **E2E é o menor conjunto possível.** Um teste de Playwright precisa se
  justificar com um mecanismo que só um navegador real tem — linha do tempo de
  animação, leiaute e overflow reais, lista virtualizada que mede zero no jsdom,
  gesto de ponteiro com passos intermediários, fluxo ao vivo entre dois
  servidores. **"É uma jornada do usuário" NÃO é justificativa**: jornada é mais
  barata e mais firme como teste de integração. E2E é a coisa mais cara e mais
  frágil deste repositório; gaste com intenção.
- **Empurre cada garantia para a camada mais barata que a segura.** Regra de
  servidor pertence a um teste de handler, não a uma asserção de que o botão
  sumiu — travar na UI é UX, a fronteira de segurança é o servidor.
- **Uma regra, uma camada.** Uma regra é prendida UMA vez, onde ela mora; as
  outras camadas afirmam presença e ligação, nunca a fronteira de novo. **Se
  apagar um teste não muda nada que outra camada já não acuse, apague.**
- **Correção de defeito ganha teste de regressão, e ele nasce VERMELHO.** Um
  teste que nunca foi visto falhando é um palpite. Quando o conserto e o teste
  entram juntos, o commit diz como o teste foi provado falho.
- **Nunca derive o esperado do código sob teste.** Não importe o helper que está
  sendo testado para montar o valor que se espera dele, e não espelhe a
  implementação na asserção — os dois andam juntos com o defeito. Escreva o
  número e a data na mão. Medido: um teste de componente que não passava o
  tamanho comparava o valor do servidor com o **default** da SPA, e passava verde
  sobre nada.
- **Unitário prova DECISÃO, não consulta.** Um spec que troca a borda protege
  exatamente duas coisas: a ramificação em volta da chamada e os argumentos dela.
  Um spec que não afirma nenhuma das duas é um *mock echo* — arranje o dublê para
  devolver X, afirme que o resultado é X — e não deve ser escrito. Ele prova que
  a linguagem devolve valores.
- **Quando o PREDICADO decide quem é afetado, prenda o predicado.** O `where` de
  uma consulta, o papel num `BoardForRole`, o filtro que escolhe QUAIS peças a
  mesa vê: isso não é encanamento, é a regra. Arranjar o resultado por ordem de
  chamada diz o que acontece *com* as linhas achadas e nada sobre *quais* linhas
  são essas.

### O INSTRUMENTO MENTE COM CARA DE RESULTADO

A infraestrutura em volta da medição destrói a medição, e o que sobra parece um
dado. **A pergunta que abre todas: "o que este instrumento DESLIGA para
funcionar, e o que ele não mede?"** Os casos que produziram cada linha estão no
[APRENDIZADOS.md](APRENDIZADOS.md).

- **Limpeza não fala mais alto que o defeito**: `finally` que estoura esconde o
  erro de verdade. Limpeza ganha `catch`.
- **Nunca rode a suíte com `| tail`**: não achar a linha no que sobrou parece
  "não aconteceu".
- **Sonda instalada cedo demais mede o nada**: um `MutationObserver` num
  `addInitScript` observa um `body` que ainda é `null`.
- **`boundingBox` devolve a caixa de um elemento coberto**, e o clique acerta
  quem está por cima.
- **Parser que ignora o que não entende inventa achados.** Falhe no que não
  sabe ler.
- **Comparar tamanho responde sobre tamanho.** A pergunta costuma ser o LIMIAR
  da norma.
- **Captura de tela não diz nada sobre animação**: o `screenshot()` finaliza toda
  animação finita antes de fotografar.
- **Sonda de vida longa mede tudo na janela dela**: armada antes do gesto, ela
  conta a mão. A janela é parte do desenho.
- **"Existe agora" e "existe quando importa" são perguntas diferentes.**
- **Varredura de um nível não é varredura**: o que vale é o fecho transitivo.
- **Leia a norma até o fim**: o WCAG tem exceções que medir tamanho não enxerga.
- **Meça o que é DESENHADO, não o que foi escrito**: herança é a regra do CSS.
- **Transição em curso faz o computado mentir.** Pergunte com `getAnimations()`
  se há transição, em vez de escolher outro instante.
- **Mostrador cujo repouso é igual ao sucesso não testemunha o sucesso.** Ele
  mostra algo diferente antes e depois?
- **Lista de PROIBIDOS subconta em silêncio.** Use uma de PERMITIDOS e falhe no
  que não conhece.
- **O medidor de contraste ignora o ALFA**: sabotar a opacidade é sabotagem
  inerte, e não prova guarda nenhum.

**Antes de ler AUSÊNCIA como evidência, prove que o canal estaria lá se o evento
tivesse acontecido**: uma linha que sai sempre, ou o caso positivo conhecido. E
o canal pode morrer depois de instalado — navegação descarta o documento e o
observador junto, e a lista vazia diz "nada mudou". **Afirme o documento antes
de afirmar o silêncio.**

### Instrumento que DECOMPÕE tem denominador de graça: a soma

**Toda decomposição afirma a soma antes das parcelas.** Uma tabela de parcelas
com nomes e alturas certas pode somar o dobro do todo sem nada reclamar; a soma
que não fecha é o que denuncia. Monte a medição de forma que cada parte tenha um
dono só (uma coluna de sondas `elementFromPoint` fecha por construção). E o
EFEITO de um corte se mede cortando de verdade (`display:none` e medir de
novo), nunca subtraindo.

### Um guarda só mede o que ele VISITA

Cobertura de contraste, tipografia e leiaute depende de onde o teste NAVEGA, não
de quantas asserções ele tem. Quatro formas de não visitar:

1. **A cena não está na lista.** Enumerar é remendo. O que restaura a amostragem
   é um guarda que force a VARREDURA ("alguém escreveu a receita à mão?"), que
   vale para a cena que nascer amanhã.
2. **O medidor não é importável.** Instrumento privado de um arquivo de teste
   tem exatamente um chamador.
3. **A tela tem UM item.** O defeito de N>1 é invisível com N=1. Quantos itens o
   caso põe na tela, e de quantos o defeito precisa?
4. **Todas as telas, um só DADO.** Tela que ramifica pelo dado pede um caso por
   ramo, com o ramo nomeado e o controle afirmando que ele apareceu.

**O controle que fecha as quatro é o DENOMINADOR**: todo medidor devolve
`{falhas, medidos}`. Sem isso, "verde" e "não mediu" são a mesma cor.

### O resto

- **Apague teste que custa mais do que protege**: asserção sobre nome de classe e
  forma de DOM que ninguém prometeu, teste que redescobre o esperado rodando a
  implementação, teste sobre código morto ou fora do bundle. Teste verde sobre
  código que ninguém usa é a pior dívida — cobra manutenção e não protege nada.
- Dado transcrito do livro (catálogos) é validado por SCHEMA, não por
  um `expect` por campo repetindo o mesmo número. Prenda a *exceção* (a armadilha
  da tabela), nunca a tabela inteira.
- Troque I/O externo (API, banco, sistema de arquivos) por classes dublê nomeadas, não por stub inline. Testes F.I.R.S.T: rápidos, independentes,
  repetíveis, auto-verificáveis, oportunos.
- **Teste nasce antes do código.** Sem o teto de linha por função (ver "Estilo
  de código"), uma função pode ser o comportamento inteiro, e o teste que a
  define cabe ser escrito primeiro — na camada que a seção já manda usar
  (integração por padrão, unitário só pra regra). Caçando defeito, reproduza no
  navegador ou num handler antes: o defeito reproduzido É o teste vermelho, e
  ele nasce antes do conserto por definição.
- **Ordem do corte: escreva o substituto, veja-o verde, DEPOIS apague.** Apagar
  primeiro abre uma janela cega.

## Como uma convenção passa a valer

Uma convenção escrita e não varrida é aplicada exatamente aos arquivos que alguém
apontou. O mecanismo que a faz valer não é o guarda pegar o erro — é o guarda
**forçar a varredura**: a suíte só fica verde quando o *último* caso foi tratado.

Este repositório vive disso. Quantos guardas `TestEvery…`/`TestNo…` existem hoje
se pergunta ao código, nunca a esta linha:

```
grep -rn "func TestEvery\|func TestNo[A-Z]" --include=*_test.go .
```

Um número escrito à mão sobre uma família que cresce envelhece; o `grep` é a
fonte.

- **Uma convenção só foi adotada depois de varrida.** Uma revisão nomeia um
  arquivo; a correção é *todo* arquivo com a mesma forma. Antes de fechar, rode a
  busca que acha os irmãos e diga no commit quantos eram.
- **Se a regra é mecanizável, a primeira pergunta é se uma FERRAMENTA faz** —
  linter, formatador, script de CI. Se faz, é configuração, não código.
  **Guarda em Go é para REGRA**: do livro, de produto, de segurança, de quem vê
  o quê. Ele é um `TestEvery…`/`TestNo…` no pacote que a possui, e falha com o
  NOME do caso que faltou, que é a diferença entre "conserte isto" e "procure".
- **Guarda vale o que vale o terreno que ele varre, e o terreno pode sumir.**
  Quando uma rota morre, o guarda que a cobria morre junto — confira se a
  invariante mudou de casa antes de dar por perdida.
- **Comentário não é correção.** Docstring explicando por que a violação está ali
  é dívida registrada, não desenho — e registrar faz parecer resolvido.
- **Regra mora aqui ou não existe.** Corpo de commit, comentário no Linear e
  docstring não vinculam: o próximo autor lê o `CLAUDE.md`, conclui que está em
  conformidade, e escreve a mesma coisa de novo.
## Dependências

- Injete dependência por construtor ou parâmetro, não por global ou import.
- Embrulhe biblioteca de terceiro atrás de uma interface fina, deste projeto.

## Formatação

- Use o formatador padrão da linguagem: `gofmt` no Go, `biome` no TypeScript
  (`pnpm lint`). Não discuta estilo além disso.

## Logs

- JSON estruturado quando o log é para depuração ou observabilidade. Logue
  objetos, não strings interpoladas — ponha o valor num campo para ele continuar
  pesquisável.
- Texto puro só na saída de CLI que uma pessoa lê.

## Idioma

**Identificador é em INGLÊS. Texto que uma pessoa lê é em PORTUGUÊS.** A linha
passa entre o que o compilador consome e o que um humano consome, e não entre
domínio e infraestrutura.

- **Inglês:** variável, função, tipo, método, campo de struct, constante, pacote,
  arquivo, nome de teste, **componente `templ`** (que é função). E a fronteira:
  tabela, coluna, campo JSON, evento SSE.
- **Português:** comentário, docstring, `.md`, mensagem de commit, tudo que
  aparece na tela, e o texto de erro que um humano vai ler.

**A ROTA é a exceção da fronteira, e fica em PORTUGUÊS** (decisão do
dono): o endereço é a única parte da fronteira que **o cliente VÊ**. Duas
ressalva: `/health` e `/static/*` ficam em inglês — são a sonda de saúde e a
pasta dos estáticos, convenção que ferramenta de fora reconhece.

Os quatro casos que ficavam de fora da lista, e cada um já foi decidido por
palpite pelo menos uma vez:

| o quê | idioma | por quê |
|---|---|---|
| nome do arquivo de spec (`board-drag.spec.ts`) | **inglês** | é nome de arquivo |
| a **descrição do teste** — o texto dentro de `test('…')` | **português** | é frase que uma pessoa lê no relatório |
| classe CSS (`.board-token`) | **inglês** | identificador que o código escreve e casa |
| id de elemento (`id="finder-field"`) | **inglês** | idem, e o `getElementById` o casa por texto |
| sinal do Datastar (`$creature_search`) | **inglês, `snake_case`** | é FRONTEIRA — e a forma tem razão própria, abaixo |

**O arquivo e a descrição são coisas diferentes**, e o mesmo spec leva as duas
línguas: `board-drag.spec.ts` contendo `test('arrastar a peça propõe a parada')`.
Ela se chama **descrição do teste** — uma palavra por conceito.

### Sinal: `snake_case`, inglês, uma grafia em todos os canais

A forma foi MEDIDA no navegador, e o mecanismo É a regra:

- **camelCase quebra em silêncio.** Nome de atributo é minusculado pelo parser de
  HTML: `data-bind:buscaCriatura` chega como `data-bind:buscacriatura` e o
  Datastar liga um sinal NOVO, deixando o declarado intocado. O fio leva os DOIS
  e o servidor lê o errado — sem erro em lugar nenhum.
- **kebab-case vira outra coisa.** O Datastar transforma `-[a-z]` em maiúscula
  por padrão, então `data-bind:creature-search` liga um sinal em camelCase: duas
  grafias para um conceito, que é a raiz do defeito acima.
- **`_` atravessa intacto** — não há caixa para perder, e o `-` é o único
  caractere que a transformação toca.

Prefixo por FAMÍLIA quando o sinal pertence a um grupo (`template_*`, `ruler_*`),
para o `grep` achar a família inteira.

Nenhum teste de Go pega isso — a minusculação acontece no NAVEGADOR, e o HTML
servido ainda tem a caixa que o autor escreveu. A garantia desceu para o TEXTO do
atributo: `TestNoDatastarAttributeKeyCarriesUppercase` varre a forma, e
`TestNoNewSignalBreaksTheNamingStandard` varre com linha de base VAZIA.

#### Renomear um sinal são SETE canais, e nada liga um ao outro

Isto é checklist, não história: renomear é ato humano, nenhum guarda o cobre, e
**cinco dos sete são invisíveis para um `grep` de `$nome`**.

| # | canal | como aparece |
|---|---|---|
| 1 | expressão | `$creature_search` |
| 2 | chave de atributo | `data-bind:creature_search` |
| 3 | **valor** de atributo | `data-ref="delete_dialog"` |
| 4 | declaração | `data-signals="{…}"`, e as strings montadas em Go |
| 5 | tag JSON do servidor | `json:"creature_search"` |
| 6 | filtro de remendo | `data-on-signal-patch-filter="{include: /^sheet_version$/}"` |
| 7 | **argumento de string** | `toggleBadge(aba, key, …)`, que monta `"$" + key` |

Os canais 3 e 7 deixaram quatro diálogos sem abrir com a suíte de Go inteira
verde: **a expressão passa a ler `undefined`, que em JavaScript não é erro** — é
o gesto não fazer nada. O canal 4 tem a armadilha inversa: `undefined != ''` é
VERDADEIRO, então um `<p>` que devia nascer escondido nasce mostrado.

**Nem toda tag JSON é sinal** — leia o diff das tags antes de aceitar uma
varredura. E há um oitavo leitor que não é canal de escrita: a **constante**
(`const brushSignal = "pincelando"`, lida como `"$" + brushSignal`), que metade
das vezes mora num bloco `const (…)` com a palavra na linha de cima.

Quem fecha: `TestEverySignalDeclaredByValueHasAReader` (3),
`TestEverySignalTheTableDeclaresHasAReader` (4), e a catraca para a FORMA. **O 7
não tem guarda** — hoje é um sítio só, o `toggleBadge` do acervo do mestre, e
está nomeado aqui por isso.

#### O id de elemento tem NOVE canais

`id="x"` · `getElementById('x')` · `aria-labelledby` · `aria-describedby` ·
`for=` · `popovertarget` · `commandfor` · `querySelector('#x')` · e
**`el.id === 'x'`**, que foi o que escapou: o renome passou por cima, o Enter
deixou de abrir o primeiro achado, e nada mais mudou na tela.

Quem cobra é o `TestEveryReferencedElementIdExists` — todo id APONTADO existe em
algum `.templ`, escrito como `id="x"` ou entregue a uma peça do kit que o
escreve (`ds.Dialog("x", …)`). Ele não cobra o contrário: um id pode existir só
para o CSS.

### Renomear ao encontrar

**Identificador em português que você encontrar no caminho vira inglês**, e não
só o que você ia escrever. Não é preciso sair caçando — é preciso não passar por
cima. Quando uma fatia move ou reescreve um arquivo, os identificadores dele saem
em inglês inteiros, não os do diff. **O nome que você CHAMA de fora e não vai
tocar segue o que está lá**, porque renomear o chamado obriga a varrer todos os
chamadores.

**Parâmetro, variável local e campo também são identificador**, e nome de teste
também — um nome de teste não tem chamador, então ele é o mais barato de todos
para sair certo na primeira vez.

### Isto NÃO tem guarda, e é decisão do dono

Houve guardas de idioma — para declaração de topo, nome local, nome de teste e
nome de arquivo —, e **eles foram apagados**.

A razão é o que eles mediam. Idioma não é uma propriedade que um `map[string]bool`
decida: os três primeiros perguntavam "esta palavra está numa lista de palavras
portuguesas que eu conheço?", e a resposta era não para quase toda palavra
portuguesa que ainda não tinha mordido alguém. Uma lista de PROIBIDOS subconta em
silêncio — é a armadilha que a seção "O INSTRUMENTO MENTE" já nomeia —, e aqui
ela subcontava sem nem ter um denominador para denunciar a subcontagem.

O de nome de arquivo invertia o erro: como lista de PERMITIDOS, reprovava toda
palavra INGLESA que ele ainda não conhecesse, e cada uma custava uma linha numa
lista que ninguém lê.

**A regra continua valendo inteira; o que mudou é quem a cobra.** Ela se confere
no OLHO, na revisão, como a composição de uma tela — e pela mesma razão: a
máquina julga LIMIAR e LIGAÇÃO, e "isto é português?" não é nenhum dos dois.

O conceito continua sendo o do livro — o que muda é a grafia. `sheet`, e não
`characterData`: a tradução é do TERMO do glossário, não uma oportunidade de
trocar o conceito por um genérico. Termo sem tradução assentada (`tormenta`,
`goblinoide`) fica; é nome próprio.

- **[GLOSSARY.md](GLOSSARY.md) — uma palavra por conceito, e um conceito por
  palavra.** Leia antes de nomear qualquer coisa que o usuário vá ler ou que vá
  virar identificador. Termo novo: escreva a linha do glossário ANTES do código.
## Commits

**Conventional Commits**, assunto em português, numa linha só.

- `<tipo>(<escopo>): <o que mudou> (ALE-NNN)`. Tipos: `feat`, `fix`, `refactor`,
  `test`, `docs`, `chore`, `ci`, `build`, `style`. Merge usa `merge:`.
- **O escopo é a superfície** (`tabuleiro`, `gabarito`, `mesa`, `ficha`), ou
  `claude` para este arquivo. Não invente um por assunto — escopo é onde se
  procura código.
- **O assunto diz o que MUDOU para quem usa**, não o que foi editado. "a seta diz
  os metros de cada perna e vira vermelha onde o deslocamento acaba" é o assunto;
  "atualiza board.templ" é o diff parafraseado.
- Cite a issue do Linear. Varreu? Diga quantos eram. Provou o teste vermelho?
  Diga como.
- **Commit pequeno e frequente bate commit grande e raro.** Se a mudança já se
  divide em passos que fazem sentido sozinhos, divida — cada commit é uma
  decisão revisável isoladamente, não um lote de tarefas diferentes
  economizando revisão.
- **Sem `Co-Authored-By` e sem rodapé de ferramenta.**

## Documentação

**Toda documentação deste projeto é escrita em português.** Um idioma só, porque
documentação que alterna obriga quem lê a traduzir no meio da frase. Os dois
guias que existem hoje — este e o do `engine-go/` — já são.

**A regra mora no guia que a possui.** Este arquivo é sobre como escrever código
neste repositório; o que cada pacote faz é do `.md` dele. Uma regra descrita aqui
e no pacote diverge — e quando diverge, ninguém sabe qual está certa.

**Toda mudança termina relendo a documentação que ela tocou.** Reler, e não
"atualizar se mudei o comportamento": as duas coisas são diferentes, e é a
segunda que falha.

Um `.md` fica errado sem ninguém mexer nele. Renomear um símbolo deixa a
explicação falando de um nome que não existe; mover uma regra de lugar deixa o
texto apontando para onde ela não está; juntar duas coisas deixa a frase que as
contrastava dizendo que A difere de A. Nenhum desses aparece no diff do código, e
nenhum é pego por teste — só releitura pega.

O critério: se depois da sua mudança um `.md` afirma algo que deixou de ser
verdade, ele é um defeito entregue igual a qualquer outro. **Documentação errada
é pior que documentação ausente, porque a ausente ninguém segue.**

**Contagem não mora em `.md`.** "Quinze cenas", "restam 14", "os 39
parâmetros": o número envelhece no primeiro commit que mexe no terreno, e ninguém
relê a linha. Quando a quantidade importa, o `.md` diz a pergunta (o `grep`, o
teste) e o código responde.

**Planejamento não vira `.md`**: fase, roadmap e plano de migração são issue no
Linear (org ALE, projeto Tormenta20). Um `.md` descreve o que o sistema **faz** —
plano executado vira mentira e fica, descrição executada vira verdade e fica
certa.

### Regra que tem GUARDA não precisa de história

Um guia carrega a REGRA e o GATILHO: o que fazer, e quando. O CASO que a produziu
mora no [APRENDIZADOS.md](APRENDIZADOS.md), que não custa contexto toda sessão e
é revisado periodicamente com o dono.

**Procedência só paga quando as duas coisas valem ao mesmo tempo:** a regra não
tem guarda, **e** sem o mecanismo ela lê como arbitrária e vai ser derrubada pelo
próximo. "Sinal em `snake_case`" sozinho parece gosto; com o mecanismo — *o
parser de HTML minuscula o nome do atributo e o Datastar liga um sinal novo* —
vira física, e ninguém discute física.

Fora disso, a narrativa cobra caro em dois lugares. Ela faz o arquivo **crescer
monotonicamente**, porque tirar história parece apagar lição — um guia assim só
tem uma direção. E ela **apodrece**: duas passagens deste arquivo viraram mentira
sem ninguém mexer nelas, e uma delas era justamente a que explicava por que duas
grafias conviviam depois de a varredura já ter rodado.

> O corte da narrativa troca legibilidade por memorabilidade, e essa troca tem
> um risco que vale saber: **uma lição de uma linha pode não disparar
> reconhecimento no momento em que você está dentro da armadilha** — que é
> quando ela vale. Se acontecer, o conserto não é restaurar a narrativa: é
> escrever o guarda que torna a lição desnecessária.

## Referência

- O livro: [Tormenta 20](/t20-book.pdf). Toda regra citada com a página, e a
  página conferida antes de escrever — como conferir é do
  [engine-go/CLAUDE.md](engine-go/CLAUDE.md).

## Guias por pacote

- **`engine-go/`** (Go): [engine-go/CLAUDE.md](engine-go/CLAUDE.md) — regenerar
  oráculo é ato deliberado, citação de página conferida, validação de schema dos
  catálogos, as armadilhas do `templ` e as do Datastar que não deixam erro
  para trás, os dois defeitos silenciosos do `sqlc`, e por que a bancada copia um
  molde migrado.
- **`e2e/`** (Playwright) **não tem guia próprio**: o que rege e2e está na seção
  "Testes" deste arquivo, e é uma regra só — e2e é a faixa mais cara do
  repositório e cada caso se justifica com um mecanismo que só um navegador tem.