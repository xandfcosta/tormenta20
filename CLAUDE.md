Gerenciador de mesa e ficha de **Tormenta 20**. O que se LÊ é em português — a
tela, a documentação e os comentários falam a língua da mesa; o que se ESCREVE
em código é em inglês. Ver "Idioma".

São dois pacotes. `engine-go/` é o app: a API HTTP na :3001, o motor de regras,
e as CENAS em `.templ` servidas com Datastar — mais a folha e as ilhas de JS
delas, em `api/piloto/src`. `e2e/` é a suíte de Playwright, que dirige o app
rodando. Um processo serve tudo, e ele sobe por `docker compose up -d --build`
com o banco em bind mount no hospedeiro — um serviço só, sem proxy na frente
(ALE-273).

Houve uma SPA em SolidJS (`frontend/`) e um motor compilado para WASM que rodava
no navegador. Os dois saíram na ALE-272: as cenas são renderizadas no servidor,
e a regra tem um lugar só.

## Antes de mexer

- **Não assuma.** Se a regra do livro ou a decisão de produto não estiver clara
  no código, no `.md` do pacote ou no [GLOSSARIO.md](GLOSSARIO.md), **pergunte**.
  Adivinhar regra de T20 custa mais caro que esperar a resposta, e o erro sai na
  ficha de alguém.
- **Regra do livro se confere no livro**, com a página citada. O offset entre PDF
  e livro e o histórico de citações erradas são do
  [engine-go/CLAUDE.md](engine-go/CLAUDE.md) — é lá que as regras moram, e o
  número não é repetido aqui de propósito (ver "Documentação").
- **Mexeu em `.templ`?** `go tool templ generate`, e **leia a saída DELE**, não a
  do `go build`. **Classe CSS nova no piloto?**
  `engine-go/scripts/build-piloto-css.sh`. As duas armadilhas — e as dez do
  Datastar que não deixam erro para trás — estão explicadas no
  [engine-go/CLAUDE.md](engine-go/CLAUDE.md).
- **Antes de commitar:** `go test ./...`, `go vet ./...` e `gofmt` no
  `engine-go/` — mais `pnpm typecheck` lá, quando mexer nas ilhas de JS. Mexeu
  em gesto de ponteiro, leiaute real ou fluxo entre dois clientes? `cd e2e &&
  npx playwright test` (~2 min, sobe o próprio servidor e o próprio banco).
- **Releia a documentação que a sua mudança tocou** — o `.md` do pacote e o
  `GLOSSARIO.md`. Não "atualize se mudou o comportamento": **releia**. Ver
  "Documentação".

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

- **Preserve os comentários que já existem.** Não os apague num refactor — eles
  carregam intenção e procedência.
- Escreva **POR QUÊ**, não O QUÊ. Pule o `// incrementa o contador` sobre o `i++`.
- Docstring em função pública: intenção e um exemplo de uso.
- Cite a issue ou o SHA quando a linha existe por causa de um defeito específico
  ou de uma restrição externa.

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

O formato é sempre o mesmo: a infraestrutura em volta da medição destrói a
medição, e o que sobra parece um dado. Quatro casos num dia só, e nenhum deles
pareceu erro na hora:

- Um `finally` que fecha contextos de browser lançou "Failed to find context" e
  **substituiu o erro de verdade** do teste. Limpeza ganha `catch`, sempre:
  *limpeza não pode falar mais alto que o defeito* (ALE-245).
- A suíte rodada com `| tail -15` deixou 981 bytes de stdout; procurar uma linha
  ali e não achar foi lido como "o evento não aconteceu", quando era "o canal não
  existe" (ALE-238).
- Uma sonda instalou `MutationObserver` em `document.body` num `addInitScript`,
  onde o `body` ainda é `null` — e a ausência de mutação virou conclusão
  (ALE-199).
- Uma sonda de Playwright mediu zero prévias de arrasto e quase virou "a
  funcionalidade não existe". A peça estava **debaixo do trilho de ferramentas**:
  o `boundingBox` devolve a caixa de um elemento COBERTO sem reclamar, e o
  `mouse.down` acertava o trilho (ALE-203).

**O controle é barato e é obrigatório: antes de ler AUSÊNCIA como evidência,
provar que o canal estaria lá se o evento tivesse acontecido.** Procurar no mesmo
arquivo uma linha que sai SEMPRE; conferir que a sonda vê o caso positivo
conhecido. Sem isso, "não reproduzi" não é evidência de ausência — é ausência de
evidência, e as duas se parecem no terminal.

**E o canal pode morrer DEPOIS de instalado: um observador precisa afirmar que o
DOCUMENTO em que ele foi instalado ainda é o mesmo.** Navegação descarta o
documento, e com ele o `MutationObserver` — a lista de mutações volta VAZIA, que
é a mesma coisa que "nada mudou". O guarda `não desanexa a cena` (ALE-238) passa
exatamente no PIOR caso: a cena não desanexou porque a cena deixou de existir.
Não é um teste que falha em detectar; é um teste que **afirma o oposto do que
aconteceu**, e ele só foi descoberto porque um clique estourou antes e denunciou
a navegação.

O mesmo vale para qualquer sonda de vida longa — `addEventListener`,
`PerformanceObserver`, um `page.on(...)` cujo alvo recarregou. Afirme o documento
antes de afirmar o silêncio.

### Um guarda só mede o que ele VISITA

Cobertura de contraste, de tipografia e de leiaute é função de onde o teste
NAVEGA, não de quantas asserções ele tem. Dois defeitos de contraste
sobreviveram anos com o guarda no ar porque ele nunca abria um popover nem
entrava no livro de campanhas (ALE-237); a mesma forma reapareceu na tipografia
(ALE-252).

**Mas "põe a cena na lista" só resolve enquanto as cenas forem contáveis, e vale
saber a diferença.** Um guarda que mede a folha do grimório cobre 43 telas por
AMOSTRAGEM — ele mede uma e vale para todas porque todas passam pelos mesmos
componentes. No dia em que uma tela escreve as classes à mão, o regime vira
ENUMERAÇÃO: uma entrada por cena, para sempre, e a que alguém esquecer nasce sem
medição — em silêncio, que é a marca desta família. Enumerar é remendo; **o que
restaura a amostragem é a tela nova passar pelos componentes da casa.** Escolher
o remendo dá sensação de conserto e deixa o buraco aberto (ALE-252).

**Duas formas a mais de "não visitar", as duas medidas na ALE-272 e nenhuma
parecida com esquecer uma cena.**

A primeira: **o guarda não alcança porque o MEDIDOR não é importável.** A ficha
em Datastar atravessou duas fatias inteiras sem uma única medição de contraste,
e não foi decisão — o medidor era função *privada* de outro arquivo de teste.
Ninguém omitiu a ficha de uma lista; a lista nunca pôde existir. Quando o
medidor virou módulo de `support/`, a primeira execução reprovou sete rótulos de
uma vez, todos herdados da tela antiga. **Instrumento que mora dentro de um
chamador tem exatamente um chamador**, e isso não aparece em nenhuma revisão de
diff.

A segunda: **o guarda visita todas as telas e um só DADO.** O caminhar pelas
sete abas da ficha abre as sete — de um herói, e o primeiro do elenco é um
guerreiro. Metade do painel de Combate (a tripla mágica, com a paleta arcana
inteira) só existe para quem conjura, então ela estava fora da medição com o
guarda passando por cima dela sete vezes. Quando a tela RAMIFICA pelo dado,
percorrer a navegação não é cobertura: é preciso um caso por ramo, e o ramo tem
de ser nomeado (`quem conjura`), com o controle afirmando que ele apareceu.

**O controle que fecha as duas é o DENOMINADOR.** Uma lista de reprovados vazia
e um seletor que não casa com nada se parecem no terminal, e por isso o medidor
devolve `{falhas, medidos}`: quem afirma "nada reprovou" afirma junto quantos
foram olhados. Sem isso, "verde" e "não mediu" são a mesma cor.

> E saiba o que o medidor NÃO vê: ele lê `color` pelo canvas e ignora o canal
> ALFA, então `text-x/50` é medido como se fosse opaco. O erro é sempre para o
> lado seguro — texto translúcido tem contraste PIOR que o medido, nunca melhor
> —, mas isso quer dizer duas coisas: uma variante `/80` nunca foi validada de
> verdade, e **sabotar a opacidade não prova guarda nenhum.** Uma sabotagem
> assim passou verde aqui e quase virou "o guarda está cego"; era sabotagem
> inerte, e quem a desmentiu foi trocar a tinta por uma cor opaca.

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

Este repositório já vive disso e nunca escreveu a regra: são **44 guardas de
varredura** no formato `TestEvery…` / `TestNo…` — toda espécie
de terreno tem desenho, todo ícone pedido existe no gerado, toda classe
posicionada por `--col`/`--lin` tem caixa, toda tinta da casa escrita num
`.templ` existe na folha compilada, toda aba da ficha desenha painel, nenhum nó
junta `data-show` com `data-attr:style`, nenhuma expressão indexa o sinal da
lista, nenhum foco pede ao servidor sem guarda de teclado, todo item do kit
inicial existe no catálogo, nenhuma concessão de origem com escolha nasce fixa,
todo endereço antigo leva ao piloto. Cada um nasceu de um defeito que tinha
irmãos.

> O número é conferido com `grep -rn "func TestEvery\|func TestNo[A-Z]"
> --include=*_test.go .` e estava em 22 por bastante tempo depois de já serem 27
> — a família cresce a cada issue e a linha não. Se ele divergir de novo, o certo
> é o `grep`.
>
> **A gramática mudou na ALE-282**, junto com os outros 773 nomes: os prefixos
> `…Toda`, `…Todo` e `…Nenhum` viraram `…Every` e `…No`. A regra de idioma
> sempre disse "nome de teste" com todas as letras; o que faltava era a varredura,
> e é por isso que a família tinha DUAS grafias escritas nesta mesma linha.
>
> Aqui morava "as duas gramáticas no `grep` não são descuido", explicando que a
> varredura era issue própria e que até ela rodar as duas conviviam. **Ela rodou**,
> e o parágrafo virou mentira sem ninguém mexer nele — que é exatamente o defeito
> descrito na seção "Documentação", agora acontecido no arquivo que o descreve.

- **Uma convenção só foi adotada depois de varrida.** Uma revisão nomeia um
  arquivo; a correção é *todo* arquivo com a mesma forma. Antes de fechar, rode a
  busca que acha os irmãos e diga no commit quantos eram.
- **Se a regra é mecanizável com o que já roda, ela vira guarda** — um `TestEvery…`/`TestNo…`
  no pacote que a possui, e não um parágrafo. Guarda de varredura falha com o
  nome do caso que faltou, que é a diferença entre "conserte isto" e "procure".
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

- **Inglês:** nome de variável, função, tipo, método, campo de struct, constante,
  pacote, arquivo, nome de teste. Também a fronteira, que já era: tabela, coluna,
  campo JSON, evento SSE, rota. **E componente `templ`**, que é função — a regra
  não dizia isso com todas as letras e por isso duas fatias da ALE-272 nasceram
  com os tipos em inglês e os componentes em português (`oPainelDeCombate` ao
  lado de `combatPanel`). Decisão do dono, ALE-272 fatia 5: componente novo é
  inglês; os que já existem ficam, pela regra do parágrafo final desta seção.
- **Português:** comentário, docstring, `.md`, mensagem de commit, tudo que
  aparece na tela, e o texto de mensagem de erro que um humano vai ler.

O conceito continua sendo o do livro — o que muda é a grafia do identificador.
`sheet`, e não `characterData`: a tradução é do TERMO do glossário, não uma
oportunidade de trocar o conceito por um genérico. Termo sem tradução assentada
(`tormenta`, `goblinoide`) fica como está; é nome próprio.

- **[GLOSSARIO.md](GLOSSARIO.md) — uma palavra por conceito, e um conceito por
  palavra.** Leia antes de nomear qualquer coisa que o usuário vá ler ou que vá
  virar identificador. Ele tem a coluna dos termos PROIBIDOS, as colisões abertas
  que não se consertam por palpite, e a tradução de cada termo. Termo novo:
  escreva a linha do glossário ANTES do código.

**RENOMEIE AO ENCONTRAR.** Decisão do dono, e ela substitui o que estava escrito
aqui: identificador em português que você encontrar no caminho vira inglês, e não
só o que você ia escrever. Não é preciso sair caçando — é preciso não passar por
cima.

> Aqui morava "o código existente NÃO é varrido de uma vez", com o argumento de
> que um renome em massa apagaria a procedência de cada linha no `git blame`. O
> argumento continua verdadeiro para uma varredura de mil identificadores num
> commit só, e foi por isso que os nomes de TESTE tiveram issue própria — a
> ALE-282, que **já rodou**: 773 nomes de uma vez, com guarda no fim. O que
> ele não sustentava era a inércia: com "só o que a sua mudança ia tocar", uma
> função como `escrevePagina` atravessou quatro fatias sendo lida em toda uma e
> renomeada em nenhuma.

Na prática: quando uma fatia move ou reescreve um arquivo, os identificadores
dele saem em inglês inteiros — não os do diff. **O nome que você CHAMA de fora e
não vai tocar segue o que está lá**, porque renomear o chamado obriga a varrer
todos os chamadores, e aí é a varredura em massa por outro caminho.

**Nome de teste é a exceção que já foi varrida, e ela tem guarda**
(`convention.TestEveryTestNameIsEnglish`). A varredura foi barata onde o resto não
é, e o motivo é estrutural: um nome de teste **não tem chamador**. Renomear
`escrevePagina` obriga a mexer em todo lugar que a chama; renomear um `TestX`
mexe numa linha. É por isso que esta convenção coube num commit e a dos
identificadores em geral não cabe.

## Commits

**Conventional Commits**, assunto em português, numa linha só.

- `<tipo>(<escopo>): <o que mudou> (ALE-NNN)`. Tipos: `feat`, `fix`, `refactor`,
  `test`, `docs`, `chore`, `ci`, `build`, `style`. Merge usa `merge:`.
- **O escopo é a superfície** (`tabuleiro`, `gabarito`, `mesa`, `ficha`), ou
  `claude` para este arquivo. Não invente um por assunto — escopo é onde se
  procura código.
- **O assunto diz o que MUDOU para quem usa**, não o que foi editado. "a seta diz
  os metros de cada perna e vira vermelha onde o deslocamento acaba" é o assunto;
  "atualiza piloto_mesa_tabuleiro.templ" é o diff parafraseado.
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
- **`e2e/`** (Playwright) **não tem guia próprio**: o que reger e2e está na
  seção "Testes" deste arquivo, e é uma regra só — e2e é a faixa mais cara do
  repositório e cada caso se justifica com um mecanismo que só um navegador tem.

  > Aqui morava a entrada do `frontend/`, dizendo que a SPA não tinha guia
  > próprio porque estava saindo. Ela saiu (ALE-272, fatia 10c), e com ela as
  > armadilhas de renderização do Solid, os contornos do Kobalte e o `.wasm`.
  > O que sobreviveu da SPA está em `engine-go/api/piloto/src`: a folha de
  > tokens, seis componentes e o driver de teclado das cenas.
