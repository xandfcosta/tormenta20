# Os catálogos

O livro transcrito, as emendas da campanha, a validação de escolha e os auditores. Faz parte do [guia do engine](../CLAUDE.md).

## Catálogos

**As cenas leem o catálogo EMBUTIDO direto** (não há rota que o sirva), **uma
vez, e guardam** (`sync.Once`, como no `race_traits.go`): o livro é compilado no
binário.

**O LIVRO é código Go**, em `domain/catalog/transcript`: cada verbete é um
literal tipado, e é ali que regra transcrita se edita — para o compilador cobrar
a chave errada. Cena, motor e testes leem o mesmo valor; o `catalog.Resource` o
serializa para quem lê JSON (`transcript.ResourceJSON`).

- **A forma dos dados é CANÔNICA** (`transcript.WriteSource`), cobrada pelo
  `TestTheTranscriptIsInCanonicalForm`; depois de editar à mão, `go run
  ./cmd/transcriptfmt`. É o que deixa `cmd/mintuids`, `cmd/bookpages` e
  `cmd/audit/bestiary --aplicar` reescreverem o arquivo sem cirurgia de AST.
- **Comentário dentro de literal não sobrevive à reescrita**: o porquê do
  verbete mora no `modeled`/`unmodeled`, que a ficha lê.
- **Tipos (`*_types.go`) e registro (`layout.go`) se escrevem à mão**; recurso
  novo é uma linha no `layout.go`.
- **`Modifier.Amount`, `Item.Price` e `Item.Slots` não distinguem ausente de
  zero.**

**O que a CAMPANHA acrescenta não mora no livro**, que é compilado, enquanto a
mesa autora em tempo de execução: a `emenda de catálogo` vive em
`campaign_items` e ACRESCENTA modificador, nunca substitui — ver o
[GLOSSARY.md](../../GLOSSARY.md).

**O mundo é um TIPO.** `engine.Catalogs` é o LIVRO e só consulta verbete;
computar ficha pede `engine.Ruleset`, o livro sob as emendas de UMA mesa —
`book.ComputeSheet(…)` não compila. Fora de campanha,
`engine.BookRuleset(book)`, explícito porque elenco, oráculo e fixtures estão
TOMANDO essa decisão. As emendas não viajam no `engine.Character`, onde campanha
e personagem chegariam misturados e a procedência morreria: `sheet.RulesetFor`
resolve UMA vez e o resultado viaja em `CharacterDTO.Ruleset` (`json:"-"`); o
lote da mesa (`sheet.RulesetsFor`) consulta uma vez por campanha DISTINTA.

### As cinco espécies de emenda, e por que não são uma lista só

| espécie | tabela | o que muda | escopo |
|---|---|---|---|
| **verbete** | `campaign_items` | o que uma COISA é | a campanha inteira |
| **concessão** | `campaign_grants` | quem TEM as coisas | seletor |
| **silêncio** | `campaign_silences` | o que esta mesa NÃO aplica | seletor |
| **regra da mesa** | `campaign_rules` | o que soma, o que vale como empunhado e de que deus se escolhem poderes, quando as condições valem | seletor |
| **melhoria da mesa** | `campaign_improvements` | o que se coloca num item | seletor |

Respondem a perguntas diferentes; uma tabela com `kind` as forçaria a uma forma
só. A de verbete não tem escopo: item que significa uma coisa para você e outra
para o vizinho são dois mundos. O **seletor** (`engine.Selector`) é `everyone`
ou `character` (id do CLONE na campanha); espécie desconhecida não alcança
ninguém, e **o id ZERO nunca casa** — senão o `Character{}` dos fixtures poria a
emenda de uma mesa no oráculo.

### O endereço de um termo NÃO carrega o valor, e o opt-in do jogador usa o mesmo

`engine.TermID` é `fonte::alvo::escala::condição`, mais categoria e peça
(`From`) só quando existem. Sem valor, tipo de bônus ou prosa (`note`, `label`):
corrigir o livro trocaria o endereço e soltaria o que estava preso a ele. Escala
e condição separam pares reais (anão: `maxPv +2` e `maxPv +1 por nível`). O
`TestEveryCatalogTermHasAUniqueAddress` prende "é único", não "é único hoje";
nele, a raça é UMA fonte (`raceActiveItems` junta tudo num `ActiveItem` com o
`race.ID`), e o benefício de origem repetido em várias origens conta uma vez
(`getOriginBenefit` lê o primeiro que casa).

O opt-in (`character_conditionals`) e o silêncio gravam o mesmo `TermID`.
**Mudar a forma do endereço é migração**: linha velha não casa, e o situacional
fica desligado para sempre — por isso categoria e peça só entram quando existem.

### JOGADOR NÃO MUDA REGRA — isso é do mestre (decisão do dono)

Vale para toda tela que vier. O jogador alterna o OPT-IN de um condicional que a
ficha já oferece; emenda, concessão e silêncio são do mestre. **Quem cobra é o
servidor**: POST à mão com interruptor que a tela não oferece é recusado pelo
caso de uso. A lista de PERMITIDOS tem UM dono (`character.SituationalGroupsOf`)
— a cena desenha dela, o caso de uso recusa fora dela —, e as posturas ficam de
fora porque custam PM.

**`character_conditionals` não é `campaign_silences`**: o opt-in nasce DESLIGADO
e é do dono da ficha; o silêncio nasce LIGADO e é do mestre. Uma tabela só
juntaria duas permissões de escrita.

### O interruptor de GRUPO tem endereço próprio

Condicionais que dividem uma flag viram UM interruptor, com chave
`engine.FlagGroupID` — não o endereço de um membro, para "metade ligada" não ser
representável. Vale para a POSTURA, o Folião, o terreno do Explorador e a
`marca-da-presa`.


### O interruptor pode desligar sozinho

O `flagOn` declara no `Ends` quando acaba, e o `Catalogs.SwitchesEndingAt` responde a quem gira a vez: `turn`
sai quando a vez passa, e `own-turn` quando ela volta para o dono (p233). Fora de combate não há vez, e o fim
de cena é a rede dos dois. Sem `Ends`, fica ligado até alguém desligar. `Requires` e `Unless` são o "se também
possuir" do livro: o poder levanta uma flag, e o termo pede ou é calado por ela.

### O catálogo chega ao processo por dois caminhos

O `catalog.Resource` serve o `transcript` compilado, que sempre existe. O
`s.catalogs` (`book.EngineCatalogs`) é um CAMPO que a bancada troca — há fixture
com `{"items":[]}` de propósito. **Validação de regra lê o embutido**; o
`s.catalogs` é do motor.

### Regra que só a TELA sabe é fronteira aberta

Sintoma: a tela tranca e o servidor não. Progressão de círculo (`Spellcasting`
em `transcript.Classes`), melhoria/material × item (`aceitaMelhoria`, no comando
da ficha) e limite de texto da campanha (`domain/campaign/rules.go`) moram no
servidor; o filtro da tela é só conveniência.

### O rascunho e a cópia

A ficha sem `campaignId` é o **rascunho**, e ele aceita os mesmos gestos da cópia da campanha. O que
muda é a entrada: a cópia nasce inteira, porque dano, efeitos e condições não atravessam o
`cloneCharacterTx`.

### A validação de ESCOLHA é do servidor, e é estrita

Vagas de poder (uma por nível a partir do 2º, p33), benefícios de origem,
caminhos e deuses por classe se conferem na gravação, não só no diálogo.

- **`sheet.WithChoicesValid`** roda em `saveTheChoices` e é **estrita**: a
  escrita deixa a ficha VÁLIDA, não só "não piora". Não há ficha de mesa real a
  proteger (decisão do dono).
- **`powerChoices`**: só poder que a ficha tem, opção que o livro oferece, mais
  de uma só no repetível, que ocupa uma vaga por ESCOLHA
  (`sheet.UsedPowerSlots`).
- **Poder GERAL com escolha** (Foco em Perícia, Foco em Arma) passa por
  `book.PowerWithChoice`; a escolha vinda da origem é grátis
  (`sheet.FreePicks`).
- **Contada por ATRIBUTO** (`countAttribute`, Especialista, p73): sem vaga,
  perícia treinada, até o atributo sem temporários
  (`Ruleset.PermanentAttribute`), mínimo 1. Gesto que mexe no N ou no treino
  (treino, ofício, guardar/vestir/remover item, raça, forja) aplica numa cópia e
  chama `sheet.ChangeKeepsChosenExpertises` antes de gravar.
- **Contada por NÍVEL** (`countLevels`, terrenos do Explorador, p50): repete sem
  vaga; descer de nível confere `sheet.LevelCountedChoicesFit`.

### Schema pega FORMA, não número errado

Dado transcrito se protege por **schema**
(`domain/catalog/rules_tables_test.go`), não por `expect` por campo: pega
perícia inexistente, faixa com buraco, devoto de raça inexistente. **Número
errado é schema válido** — contra ele, o auditor:

```
cd engine-go
go run ./cmd/audit/bestiary            # relatório
go run ./cmd/audit/bestiary --aplicar  # escreve as correções
```

### Os auditores do livro

`cmd/audit/<nome>`; o livro (gitignorado) se acha por `T20_BOOK_PDF`, por um
`t20-book.pdf` subindo do diretório ou o do checkout principal numa worktree. A
leitura mora em `infra/bookpdf`, com `\s`/`\w` Unicode (`IsPySpace`) por causa
do espaço não separável. Rodar é **ato deliberado**, como o `genoracle`: a
ferramenta PROPÕE e o diff revisado contra o livro decide. **Mexeu num catálogo?
rode o auditor dele.** É um por seção do sumário, e o que os distingue é a
ÂNCORA — os campos que ligam um pedaço do livro a um verbete:

| auditor | catálogo | âncora |
|---|---|---|
| `cmd/audit/bestiary` | `transcript.Bestiary` | Defesa + Pontos de Vida |
| `cmd/audit/spells` | `transcript.Spells` | nome + escola, para os aprimoramentos; para a cobertura, os links em negrito do índice (p174-177) **e** o título do verbete (`Tormenta20` a 24pt) |
| `cmd/audit/classes` | `transcript.Classes`, `transcript.ClassPowers` | nome na prosa, ou nível na tabela |
| `cmd/audit/equipment` | `transcript.Items` | nome + preço na tabela |
| `cmd/audit/races` | `transcript.Races` | linha de atributos, ligada por semântica |
| `cmd/audit/expertises` | `transcript.Expertises` | Tabela 2-1 **e** a linha de estado do verbete |
| `cmd/audit/gods` | `transcript.Gods` | Tabela 1-20 **e** os campos rotulados do verbete |
| `cmd/audit/conditions` | `conditionModifierTable` (Go!) | a HERANÇA que a primeira frase do verbete declara |
| `cmd/audit/origins` | `transcript.OriginSources`, `transcript.Origins` | Tabela 1-19 **e** o verbete, que é o título com `Benefícios.` abaixo |
| `cmd/audit/powers` | `transcript.GeneralPowers`, `transcript.TormentaPowers` | a tabela de pré-requisitos **e** a linha `Pré-requisito:` do verbete |
| `cmd/audit/grantedpowers` | `transcript.GrantedPowers`, `transcript.DivinePowers` | a coluna "Devoto de X" da Tabela da p127 **e** o rótulo de deus do verbete |
| `cmd/audit/enchants` | `transcript.Items` (`weapon-enchant`, `armor-enchant`) | as Tabelas 8-8 e 8-10 (`d%` + encanto + efeito) **e** o verbete `Nome. Regra` |
| `cmd/audit/materials` | `transcript.Items` (`material`) | a Tabela 3-9, uma MATRIZ de preço, **e** o verbete com as metades `Arma.` / `Armadura e Escudo.` / `Esotérico.` |
| `cmd/audit/objects` | `objectExamplesOfTheBook` + as duas escadas (Go!) | a Tabela 5-4, lida por COORDENADA — a p239 tem duas tabelas lado a lado |
| `cmd/audit/specificitems` | `transcript.Items` (os de `composedOf`) | as Tabelas 8-9 e 8-11 (`d%` + nome + preço), o verbete `Nome. Regra` **e o ITÁLICO dele**, que é a composição do item |
| `cmd/audit/potions` | `transcript.Items` (os de `consumable.spell`) | a Tabela 8-12 (`d%` + a MAGIA + preço), resolvida contra o catálogo de magias — a poção não tem verbete |
| `cmd/audit/accessories` | `transcript.Items` (`apparel` com faixa de `d%`) | as Tabelas 8-13/14/15, com UMA lista alfabética de verbetes para as três |
| `cmd/audit/artifacts` | `transcript.Items` (`artifact`), `transcript.GMTables` (`chaosDeck`) | NENHUMA tabela — a p346 tira o artefato da geração aleatória. O denominador vem do TIPO: o cabeçalho de seção e o corpo têm corpos de letra diferentes, e nas p346-349 não há outra seção |

**Dois não leem o livro.** O `cmd/audit/coverage` confere raça, classe e origem
contra o MOTOR — quanto vira número, gesto ou nota do mestre. O balde é
DECLARADO (`modeled`/`unmodeled`), a nota cita a issue da fatia que a resolve,
`--destino` agrupa por ela, e verbete sem efeito nem `unmodeled` reprova (sem
guarda em Go, por decisão do dono). O `cmd/audit/pointers` lista arquivos
citados em comentário de teste que sumiram — candidatos, não veredito.

### O que vale para todo auditor

- **A âncora é o trabalho; o resto é regex.** Só bloco contíguo e completo cujas
  âncoras batam com o catálogo; o que não se liga REPROVA com nome.
- **Coordenada, nunca `pdftotext -layout`**, que junta colunas vizinhas. Quando
  o `<line>` junta título, estado e coluna vizinha, leia por PALAVRA
  (`Book.Words`).
- **As faixas `d%` ladrilham o intervalo**: a soma se afirma antes das parcelas
  (`bookpdf.RangesThatDontClose`).
- **Duas âncoras se conferem entre si ANTES do catálogo**: leitura errada vira
  discordância nomeada, não tabela plausível.
- **O que o motor NÃO aplica sai com denominador, não como lista de proibidos.**
  Todo encanto tem modificador ou diz por que não, nunca os dois (motivo ao lado
  de modificador envelheceu) — cobra o
  `TestEveryEnchantEitherAppliesOrSaysWhyNot`. Nos materiais,
  `modeled`/`unmodeled` são PERMITIDOS por METADE do verbete, e a união dá
  exatamente as partes dele.
- **Medidor de prosa recorta onde a prosa acaba** e imprime a cobertura mínima;
  sem o recorte, outro assunto infla a medida.
- **A DIREÇÃO da medida é do TERRENO**: catálogo EQUIVALENTE pergunta "a regra
  do livro está aqui?"; catálogo que RESUME, "tudo o que ele afirma está na
  página?" — para números, perícias e palavras (`bookpdf.Coverage`). O veredito
  é humano, e as duas saem lado a lado.
- **Filtro de lixo é um só**: `WithoutJunk` (marca d'água, número de página,
  cabeçalho de capítulo).

### A FRONTEIRA JSON do `Modifier`

O `amount` do catálogo está na unidade do LIVRO e o motor guarda deslocamento em
quadrados — é a razão de `MarshalJSON`/`UnmarshalJSON`. **Na leitura o tipo é
EMBUTIDO** (alias local, sem método e sem recursão) e só o `amount` se
redeclara, porque o campo menos profundo vence o embutido: campo novo atravessa
sozinho. **Na escrita os campos ficam à mão**, porque o oráculo compara byte a
byte e embutir jogaria o `amount` para o fim; cobra o
`TestEveryModifierFieldSurvivesTheCatalogJSON`, por reflexão sobre o TIPO. Teste
que monta o `Modifier` em Go não atravessa essa fronteira.

### A LEITURA de tabela e de verbete mora no `bookpdf`

`Book.LootRows`, `RangesThatDontClose` e `Book.EntriesByName` guardam as
armadilhas comuns: linha montada pelo Y, última linha colapsada num `<line>`,
`d%` alinhado à direita (o x é faixa), margem que muda entre página par e ímpar.
**Leitura que um segundo auditor copiaria vira função do `bookpdf`**, e o
refactor se prova com o relatório BYTE A BYTE igual — número que muda é
indistinguível de correção.

### O ITÁLICO é dado, e ler estilo pede outro leitor

O `-bbox` do `pdftotext` **não marca estilo**; o `pdftohtml -xml` marca, noutra
régua. O `Book.Italics` converte pela razão entre as alturas da MESMA página e
devolve na coordenada do resto do módulo; sem ele, comparam-se réguas
diferentes.

### E o buraco INVERSO: o valor existe, mas não se confere que ele APONTA

No `cmd/seed/seed-data.json`, raça, origem, classe, deus, poder concedido,
tamanho, magia e item passam por lookup (`cmd/seed/catalog_refs.go`). **Seed que
mente não quebra**: id de arma errado dá ficha sem a arma, abrindo normal — e o
e2e passa a medir o ambiente. Todo validador de referência:

- **sugere o VIZINHO**, com teto de um terço do comprimento: sugestão errada é
  seguida;
- **junta TUDO antes de falhar**, para cinco ids errados não pedirem cinco
  rodadas;
- **afirma o próprio DENOMINADOR**: o catálogo carrega em `sync.Once` e engole
  erro: com a carga quebrada toda lista vem vazia e todo nome seria acusado.
  Lista vazia é falha de carga, e a mensagem diz isso;
- **trata CHAVE de objeto como referência** (a chave do `classChoices` é nome de
  classe) — o `TestEveryCreateFieldOfTheSeedIsClassified` obriga a classificar
  campo por campo.

