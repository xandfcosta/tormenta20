Gerenciador de mesa e ficha de **Tormenta 20**. O que se LÊ é em português — a
tela, a documentação e os comentários falam a língua da mesa; o que se ESCREVE
em código é em inglês. Ver "Idioma".

São dois pacotes. `engine-go/` é o app: a API HTTP na :3001, o motor de regras,
e as CENAS em `.templ` servidas com Datastar — mais a folha e as ilhas de JS
delas, em `serve/api/assets/src`. Ele se divide em quatro grupos — `domain/`,
`serve/`, `infra/` e `cmd/` —, e o mapa de onde procurar cada coisa está no
[guia dele](engine-go/CLAUDE.md). `e2e/` é a suíte de Playwright, que dirige o app
rodando. Um processo serve tudo, e ele sobe por `docker compose up -d --build`
com o banco em bind mount no hospedeiro — um serviço só, sem proxy na frente
(ALE-273).

Houve uma SPA em SolidJS (`frontend/`) e um motor compilado para WASM que rodava
no navegador. Os dois saíram na ALE-272: as cenas são renderizadas no servidor,
e a regra tem um lugar só.

## Antes de mexer

- **Não assuma.** Se a regra do livro ou a decisão de produto não estiver clara
  no código, no `.md` do pacote ou no [GLOSSARY.md](GLOSSARY.md), **pergunte**.
  Adivinhar regra de T20 custa mais caro que esperar a resposta, e o erro sai na
  ficha de alguém.
- **Regra do livro se confere no livro**, com a página citada. O offset entre PDF
  e livro e o histórico de citações erradas são do
  [engine-go/CLAUDE.md](engine-go/CLAUDE.md) — é lá que as regras moram, e o
  número não é repetido aqui de propósito (ver "Documentação").
- **Mexeu em `.templ`?** `go tool templ generate`, e **leia a saída DELE**, não a
  do `go build`. **Classe CSS nova no app?**
  `engine-go/scripts/build-css.sh`. As duas armadilhas — e as dez do
  Datastar que não deixam erro para trás — estão explicadas no
  [engine-go/CLAUDE.md](engine-go/CLAUDE.md).
- **Antes de commitar:** `go test ./...`, `go vet ./...` e `gofmt` no
  `engine-go/` — mais `pnpm typecheck` lá, quando mexer nas ilhas de JS. Mexeu
  em gesto de ponteiro, leiaute real ou fluxo entre dois clientes? `cd e2e &&
  npx playwright test` (~2 min, sobe o próprio servidor e o próprio banco).
- **Releia a documentação que a sua mudança tocou** — o `.md` do pacote e o
  `GLOSSARY.md`. Não "atualize se mudou o comportamento": **releia**. Ver
  "Documentação".
- **Mexeu em tela? OLHE a tela** antes de fechar — é passo, não zelo, e o
  detalhe está logo abaixo.

### O último passo é OLHAR

A fatura está na ALE-319: a decomposição da perícia mentiu na tela por meses,
com 193 casos verdes e 75 guardas no ar. O dado estava certo e o defeito era
100% de APRESENTAÇÃO — e não havia, em lugar nenhum deste repositório, um passo
que mandasse abrir a cena e julgar.

**A divisão de trabalho É a regra.** A MÁQUINA julga LIMIAR e LIGAÇÃO: contraste
abaixo do AA, Cinzel abaixo do piso, alvo de toque, sinal escrito sem leitor, id
apontado que não existe. O OLHO julga COMPOSIÇÃO e HIERARQUIA: o que salta
primeiro e se era para saltar, o que ficou apertado, o que ficou solto, o que
desalinha. Nenhum limiar responde a essas quatro, e é por isso que este passo é
humano — não porque a suíte seja fraca.

- **Quais cenas:** as que a mudança TOCA, e os endereços delas estão em
  `engine-go/serve/web/appearance_scenes.json` — o registro que o
  `TestEveryPageSceneIsMeasuredForAppearance` mantém completo. Não escreva uma
  segunda lista aqui: ela envelheceria sozinha, que é o remendo da ALE-252.
- **Em que larguras:** **390px e desktop, no mínimo** — a casa chaveia por
  LARGURA, então são as duas pontas que decidem. E cena que ramifica pelo DADO
  (mestre e jogador, quem conjura e quem não) pede uma passada por ramo.
- **Sem digitar senha:** a sessão do e2e já está em `e2e/.auth/`, e um script de
  Playwright com `storageState` entra autenticado sem ninguém digitar nada.
- **O que você viu vira o quê:** conserto, ou linha na issue. E se o olho pegar
  coisa de LIMIAR, o conserto não é olhar melhor da próxima vez — é escrever o
  guarda, pela mesma razão que a seção "Documentação" dá para não restaurar
  narrativa.

## Estilo de código

- **Uma função se divide quando tem mais de uma RAZÃO PARA MUDAR, nunca por
  contagem de linhas.** Aqui morava "funções de 4 a 20 linhas", e ela foi tirada
  por atrapalhar mais do que ajudava: uma tarefa que é uma coisa só, picada em
  dez funções para caber no teto, obriga quem lê a remontar a sequência saltando
  pelo arquivo — e cada nome inventado no caminho é um nome a mais que não
  identifica nada. Um passo a passo linear e longo se lê de cima para baixo;
  cinco chamadas com nome genérico, não.
- **Arquivos abaixo de 500 linhas** — este teto fica, e por outro motivo: arquivo
  é unidade de RESPONSABILIDADE e de conflito de merge, não de leitura.
- Uma responsabilidade por módulo. Extrair continua sendo o certo quando o
  pedaço tem sentido sozinho: quando ele tem OUTRO chamador, outro motivo para
  mudar, ou quando dar nome a ele explica o que o corpo não explicava.
- Retorno cedo em vez de `if` aninhado. No máximo dois níveis de indentação.
- **Nomes específicos e únicos.** Nada de `data`, `handler`, `Manager`. Prefira
  nomes com menos de 5 ocorrências no `grep` — um nome que já existe em cinco
  lugares não identifica coisa nenhuma.
- Tipos explícitos. Sem `any`, sem `Dict`, sem função sem tipo.
- Sem lógica repetida: extraia para função ou módulo.
- **Diante de duas opções que resolvem a tarefa igualmente bem hoje, escolha a
  mais fácil de mudar depois.** Não é prever o futuro — é não pagar a mesma
  decisão duas vezes. Menos lugares para editar, menos acoplamento, decisão
  adiável em vez de decisão travada.
- **Refatore de passagem, não só quando um renome esbarra em você.** Se a
  tarefa já abriu o arquivo, aproveite para corrigir o que estiver tosco por
  perto — nome ruim, duplicação, função que já devia ter sido dividida.
  "Depois" costuma virar "nunca"; a limpeza incidental é mais barata que a
  faxina separada.
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
- **A DECISÃO mora aqui; a INVESTIGAÇÃO mora na issue.** Cite `ALE-NNN` e siga.
  Quem quiser a história tem onde buscá-la, e ela deixa de custar contexto a cada
  leitura do arquivo.
- **Explique na MENSAGEM DE FALHA antes do cabeçalho.** Um guarda que falha
  dizendo o nome do caso e o que fazer entrega a explicação no instante em que
  ela importa; o cabeçalho só alcança quem já foi abrir o arquivo.
- **Comentário envelhece como código, e se relê como código.** Um que deixou de
  ser verdade é defeito entregue, igual a um `.md` errado — e quem o pega é a
  releitura, não o compilador. O `TestNoCitationNamesAMissingSymbol` cobre uma
  fatia: citação a símbolo que não existe mais.
- **Não repita no comentário a regra que o guia já tem.** Medido na ALE-333:
  "uma porta que devolve tipo do hospedeiro não é porta" estava escrita por
  extenso em QUATRO `deps.go` — e no `engine-go/CLAUDE.md`, que é o dono dela.
  Cinco cópias envelhecem em cinco velocidades, e a do guia é a única que alguém
  procura. No código fica o que é específico DAQUELE arquivo; a regra fica no
  guia, e o comentário no máximo aponta para ele.
- Docstring em função pública: intenção e um exemplo de uso.

> Aqui mandava **preservar** os comentários existentes e nunca apagá-los num
> refactor, "porque carregam intenção e procedência". Essa frase era a CATRACA:
> somada a "cite a issue", cada conserto acrescentava prosa e nada nunca
> removia. O resultado medido foram 766 blocos de 10+ linhas no `engine-go`,
> 11.586 linhas ao todo, e 37% do `web/` em comentário (ALE-324). O substituto
> não manda apagar — manda RELER, que é o que a seção "Documentação" já exige
> dos `.md`.
>
> **Não existe guarda de tamanho de comentário, e não deve existir**: ele
> reprovaria justamente os cabeçalhos de guarda, que são os melhores comentários
> do repositório. Contar linhas responderia sobre linhas.

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
dado. Cada linha abaixo custou uma investigação inteira; a história está na
issue, e o que fica aqui é a forma de reconhecer a armadilha na próxima vez.

**A pergunta que abre todas elas: "o que este instrumento DESLIGA para
funcionar, e o que ele não mede?"**

- **Limpeza não pode falar mais alto que o defeito** — um `finally` que estoura
  substitui o erro de verdade. Limpeza ganha `catch`, sempre (ALE-245).
- **Nunca rode a suíte com `| tail`** — procurar uma linha no que sobrou e não
  achar vira "o evento não aconteceu", quando era "o canal não existe" (ALE-238).
- **Sonda instalada cedo demais mede o nada** — `MutationObserver` num
  `addInitScript` observa um `body` que ainda é `null` (ALE-199).
- **`boundingBox` devolve a caixa de um elemento COBERTO sem reclamar** — e o
  clique acerta quem está por cima (ALE-203).
- **Ramo que ignora o que não entende produz lista de falhas com cara de
  descoberta** — um parser que descarta o seletor desconhecido acusou 24 botões
  vivos. Hoje ele FALHA no que não sabe ler (ALE-294).
- **Instrumento que compara TAMANHO responde sobre tamanho** — a diferença de
  2px era real e a pergunta era outra: o LIMITE de contraste do WCAG 1.4.11
  (ALE-250).
- **Captura de tela não responde nada sobre linha do tempo** — o `screenshot()`
  do Playwright FINALIZA toda animação finita antes de fotografar (ALE-174).
- **Sonda de vida longa mede tudo o que acontece na janela dela** — armada antes
  do gesto, ela conta a MÃO e não a animação. A janela é parte do desenho
  (ALE-174).
- **"Existe agora" e "existe quando importa" são perguntas diferentes** — o nó
  medido some no quadro seguinte, e a animação some pedida (ALE-174).
- **Varredura de um nível não é varredura** — o defeito morava em `for` →
  `@expertiseDetail` → `@overlay`, e só o último tinha o atributo. O conserto é o
  fecho transitivo (ALE-298).
- **Ler a norma até o fim é parte de construir o medidor** — contar `< 24px`
  inflou o defeito em 25×, porque o WCAG 2.5.8 tem duas exceções que tamanho não
  enxerga: espaçamento e equivalente (ALE-177).
- **Herança é a REGRA do CSS** — quem mede tipografia pelo texto do código mede
  o que foi ESCRITO, e a pergunta é sobre o que é DESENHADO. Dois tokens em
  `class=` diferentes, e regex nenhum os junta (ALE-252).
- **Transição em curso faz o computado mentir, e escolher outro instante não
  conserta** — `getAnimations()` responde SE existe transição; ler "no instante
  certo" é trocar o erro de lugar (ALE-318).
- **Mostrador cujo REPOUSO é igual ao sucesso não testemunha o sucesso** — a
  faixa dizia "Salvo" antes de qualquer digitação. Pergunte sempre: *ele mostra
  algo diferente antes e depois do que eu vim medir?* (ALE-218).
- **Lista de PROIBIDOS subconta em silêncio** — e, ao contrário de uma
  decomposição, não tem denominador embutido. Inverta: tenha um PERMITIDOS e
  falhe no que não conhece (ALE-301).

**O controle é barato e é obrigatório: antes de ler AUSÊNCIA como evidência,
prove que o canal estaria lá se o evento tivesse acontecido.** Procure no mesmo
arquivo uma linha que sai SEMPRE; confira que a sonda vê o caso positivo
conhecido. Sem isso, "não reproduzi" não é evidência de ausência — é ausência de
evidência, e as duas se parecem no terminal.

**E o canal pode morrer DEPOIS de instalado.** Navegação descarta o documento e
com ele o `MutationObserver`; a lista de mutações volta VAZIA, que é a mesma
coisa que "nada mudou". O guarda `não desanexa a cena` passava no PIOR caso — a
cena não desanexou porque deixou de existir. Não é um teste que falha em
detectar: é um teste que **afirma o oposto do que aconteceu** (ALE-238). Vale
para toda sonda de vida longa: **afirme o documento antes de afirmar o
silêncio.**

### Instrumento que DECOMPÕE tem denominador de graça: a soma

Medir "quanto cada camada come dos 390px" na ALE-230 exigiu três instrumentos, e
**os dois primeiros produziram uma tabela plausível com o número errado**:

- **792 de 390** — o instrumento subia a árvore somando os irmãos de cada
  ancestral, e contou a textura de fundo (`absolute inset-0`, 390px) como camada
  empilhada. O que está FORA DO FLUXO não empilha.
- **412 de 390** — o mesmo instrumento contou o `gap` de um filho
  `display:none`. Um filho que não desenha não separa nada de nada.

Nenhum dos dois reclamou. As duas tabelas tinham nomes de camada certos, alturas
certas para as camadas que existiam, e uma linha de total que ninguém confere
quando a lista parece boa. O que os denunciou foi **a soma não fechar na altura
da janela** — e é isso que faz de uma decomposição um instrumento barato de
validar: ela já tem denominador embutido, ao contrário de uma lista de falhas,
que precisa de um `medidos` escrito à mão.

O terceiro fechou em 390 **por construção**: uma coluna de sondas
`elementFromPoint` de y=0 a y=389, agrupada em bandas. Não há como uma camada ser
contada duas vezes nem um overlay virar empilhamento, porque cada pixel da coluna
tem exatamente um dono.

**A regra: toda decomposição afirma a soma antes de afirmar as parcelas.** E
quando o que se mede é o EFEITO de um corte, corte de verdade — a ALE-230 mediu
"quanto esta camada devolve" escondendo-a com `display:none` e remedindo, em vez
de subtrair a altura dela. Somar teria errado pelos mesmos dois motivos acima.

### Um guarda só mede o que ele VISITA

Cobertura de contraste, de tipografia e de leiaute é função de onde o teste
NAVEGA, não de quantas asserções ele tem. Dois defeitos de contraste
sobreviveram anos com o guarda no ar porque ele nunca abria um popover nem
entrava na cena de campanhas (ALE-237).

**Quatro formas de "não visitar", e só a primeira parece esquecimento:**

1. **A cena não está na lista.** Enumerar é remendo: uma entrada por cena, para
   sempre, e a que alguém esquecer nasce sem medição — em silêncio, que é a
   marca desta família. **O que restaura a AMOSTRAGEM é um guarda que force a
   varredura**, e não visitar mais cenas: o `TestNoHandwrittenLabelRecipe` não
   pergunta "esta cena está na lista?" e sim "alguém escreveu a receita à mão?",
   e a resposta dele vale para a cena que nascer amanhã (ALE-252, ALE-295).
2. **O MEDIDOR não é importável.** A ficha atravessou duas fatias sem uma única
   medição de contraste porque o medidor era função *privada* de outro arquivo
   de teste. Ninguém a omitiu de uma lista; a lista nunca pôde existir.
   **Instrumento que mora dentro de um chamador tem exatamente um chamador**, e
   isso não aparece em revisão de diff (ALE-272).
3. **O guarda visita a tela com UM ITEM.** Uma tela que desenha N nós iguais tem
   um comportamento com N=1 e outro com N>1, e medir N=1 é medir a metade em que
   o defeito é invisível por construção — com uma peça, o primeiro do DOM É o
   arrastado, e o gesto certo e o errado dão o mesmo resultado. **A pergunta é
   "quantos itens o caso põe na tela, e o defeito precisa de quantos?"**
   (ALE-299).
4. **O guarda visita todas as telas e um só DADO.** As sete abas da ficha, de um
   guerreiro — e metade do painel de Combate só existe para quem conjura. Quando
   a tela RAMIFICA pelo dado, percorrer a navegação não é cobertura: é um caso
   por ramo, com o ramo NOMEADO e o controle afirmando que ele apareceu
   (ALE-272).

**O controle que fecha as quatro é o DENOMINADOR.** Uma lista de reprovados
vazia e um seletor que não casa com nada se parecem no terminal, e por isso todo
medidor devolve `{falhas, medidos}`: quem afirma "nada reprovou" afirma junto
quantos olhou. Sem isso, **"verde" e "não mediu" são a mesma cor**.

> E saiba o que o medidor NÃO vê: ele lê `color` pelo canvas e ignora o ALFA,
> então `text-x/50` é medido como se fosse opaco. O erro é para o lado seguro,
> mas **sabotar a opacidade não prova guarda nenhum** — é sabotagem inerte, e
> quase virou "o guarda está cego".
### O resto

- **Apague teste que custa mais do que protege**: asserção sobre nome de classe e
  forma de DOM que ninguém prometeu, teste que redescobre o esperado rodando a
  implementação, teste sobre código morto ou fora do bundle. Teste verde sobre
  código que ninguém usa é a pior dívida — cobra manutenção e não protege nada.
- Dado transcrito do livro (catálogos) é validado por SCHEMA no despejo, não por
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

> Aqui morava a lista dos guardas, um a um, e mais o NÚMERO deles. O número ficou
> obsoleto **três vezes** sem ninguém mexer na linha — e uma vez ele DESCEU, de
> 46 para 44, porque o terreno que dois deles varriam foi apagado. A lição não é
> "atualizar com mais cuidado": é que **um número escrito à mão sobre uma família
> que cresce envelhece**, e a lista é a coisa que apodrece. O `grep` é a fonte.

- **Uma convenção só foi adotada depois de varrida.** Uma revisão nomeia um
  arquivo; a correção é *todo* arquivo com a mesma forma. Antes de fechar, rode a
  busca que acha os irmãos e diga no commit quantos eram.
- **Se a regra é mecanizável com o que já roda, ela vira guarda** — um
  `TestEvery…`/`TestNo…` no pacote que a possui, e não um parágrafo. Guarda de
  varredura falha com o NOME do caso que faltou, que é a diferença entre
  "conserte isto" e "procure".
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

## Estrutura

- Siga a convenção do framework.
- Prefira módulos pequenos e focados a arquivos-deus.
- Caminhos previsíveis: controller/model/view, src/lib/test.

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

**A ROTA é a exceção da fronteira, e fica em PORTUGUÊS** (decisão do dono,
ALE-303): o endereço é a única parte da fronteira que **o cliente VÊ**. Duas
ressalva: `/health` e `/static/*` ficam em inglês — são a sonda do compose e a
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
| 7 | **argumento de string** | `@pickerDialog("condition_dialog", …)`, que monta `"$" + sinal` |

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
não tem guarda** — são três sítios, e estão nomeados aqui por isso.

#### O id de elemento tem OITO canais

`id="x"` · `getElementById('x')` · `aria-labelledby` · `aria-describedby` ·
`for=` · `popovertarget` · `querySelector('#x')` · e **`el.id === 'x'`**, que foi
o que escapou: o renome passou por cima, o Enter deixou de abrir o primeiro
achado, e nada mais mudou na tela.

Quem cobra é o `TestEveryReferencedElementIdExists` — todo id APONTADO existe em
algum `.templ`. Ele não cobra o contrário: um id pode existir só para o CSS.

### Renomear ao encontrar

**Identificador em português que você encontrar no caminho vira inglês**, e não
só o que você ia escrever. Não é preciso sair caçando — é preciso não passar por
cima. Quando uma fatia move ou reescreve um arquivo, os identificadores dele saem
em inglês inteiros, não os do diff. **O nome que você CHAMA de fora e não vai
tocar segue o que está lá**, porque renomear o chamado obriga a varrer todos os
chamadores.

O resto tem CATRACA (`TestNoNewIdentifierIsWrittenInPortuguese`): a dívida antiga
mora numa linha de base que **só pode encolher** — nome novo reprova com o nome
dele, e nome baselinado que sumiu reprova também, senão o arquivo vira mentira
sozinho.

> A diferença entre a metade com guarda e a sem não foi cuidado, foi varredura: a
> com guarda saiu 100% em inglês, a sem produziu 39 identificadores em português
> em sete fatias seguidas (ALE-300).

**Nome de teste foi varrido de uma vez** (`TestEveryTestNameIsEnglish`), e o
motivo é estrutural: um nome de teste **não tem chamador**.

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

**Planejamento não vira `.md`**: fase, roadmap e plano de migração são issue no
Linear (org ALE, projeto Tormenta20). Um `.md` descreve o que o sistema **faz** —
plano executado vira mentira e fica, descrição executada vira verdade e fica
certa.

### Regra que tem GUARDA não precisa de história

Um guia carrega a REGRA e o GATILHO: o que fazer, e quando. O CASO que a produziu
mora na issue, que é onde ele pode ser verificado e onde não custa contexto toda
sessão.

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

> Este arquivo tinha 847 linhas quando a regra foi escrita, e 42% delas eram duas
> seções contando casos. O corte trocou legibilidade por memorabilidade, e essa
> troca tem um risco que vale saber: **uma lição de uma linha pode não disparar
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
  catálogos, as armadilhas do `templ` e as dez do Datastar que não deixam erro
  para trás, os dois defeitos silenciosos do `sqlc`, e por que a bancada copia um
  molde migrado.
- **`e2e/`** (Playwright) **não tem guia próprio**: o que rege e2e está na seção
  "Testes" deste arquivo, e é uma regra só — e2e é a faixa mais cara do
  repositório e cada caso se justifica com um mecanismo que só um navegador tem.