# Os testes do engine

A bancada, os guardas de `convention` e como apagar sem perder regra. Faz parte do [guia do engine](../CLAUDE.md).

## O fixture do app prima o catálogo DE VERDADE

**Catálogo vazio no fixture é validação desligada em silêncio**: quase toda
checagem diz "não sei" ao desconhecido (recusá-lo travaria fichas antigas), e o
guarda fica verde sem rodar. O `newTestServer` prima o embutido
(`book.EngineCatalogs`); `{"items":[]}` é exceção declarada (ver "O catálogo
chega ao processo por dois caminhos").

## A bancada dos testes: um molde migrado, copiado por teste

O `newTestServer` abre um SQLite de VERDADE por teste, para o `serve/api` provar
composição. O caro é MIGRAR (cada `fsync` do `db.Open` custa o disco do
`TMPDIR`), então o `TestMain` migra UM molde (`infra/db/testdb`) e cada teste o
copia (`testdb.Fresh`). Cada pacote declara:

```go
func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }
```

- **`PRAGMA synchronous=OFF` fica no `newTestServer`, NUNCA no `db.Open`**:
  banco que morre no fim do caso não tem o que proteger; em produção é perda de
  dados do mestre.
- **A cópia FALHA ALTO** (`Close` com erro, `Sync`, TAMANHO comparado): cópia
  parcial parece banco não migrado (`no such table`) e manda caçar migração.
- **O molde é PACOTE, não `_test.go`**, que não exporta: senão a cena que
  virasse pacote escreveria a própria bancada, e o catálogo vazio nasce assim.

A montagem do servidor NÃO se compartilha: precisa do `api.Server`, e a bancada
que o importasse seria importada de volta. O molde é a parte cara; o fixture é
de cada pacote.

## Testes

As faixas, o vermelho antes de confiar e o que não merece teste estão em
[docs/testing.md](../../docs/testing.md). Daqui:

- `go test ./...` — sem flag, sem setup.
- **Teste de regra vive junto da regra e cita a página** (e o exemplo
  trabalhado: `TestBolaDeFogoWorkedExample`). Ele e o ORÁCULO não se substituem:
  o oráculo acusa qualquer número mudado na ficha inteira; o teste explica POR
  QUE o número é aquele.
- **Vermelho em Go**: inverta o operador, rode, confirme o vermelho, reverta.
  Para guarda novo, a árvore de ONTEM é o vermelho real — `git archive` do
  commit anterior, o guarda por cima, `go test` —, e não sai inerte como
  sabotagem.

## `convention`: os guardas que não são de pacote nenhum

`convention/` não tem código de produção. Existe porque "regra mecanizável vira
guarda" e "o guarda mora no pacote que a possui" se contradizem quando a regra é
de TODOS os pacotes, e pôr a regra no `api` seria dono arbitrário. Quantos são:

```
grep -h '^func Test' convention/*_test.go | wc -l
```

**Nasce aqui o guarda** cuja regra é do REPOSITÓRIO, que mediria por acaso o
diretório em que mora se varresse `.`, ou que é da família das CITAÇÕES
(arquivo, teste e símbolo que não existem mais).

- **O terreno é o DIRETÓRIO, nunca padrão de nome**: glob deixa de casar no
  renome, e o guarda segue VERDE medindo menos.
- **Guarda que proíbe a cópia falha quando o original some** — o
  `TestNoHandwrittenFocusRing` afirma que a regra global segue no `index.css`.
- **A LÁPIDE é declarada, não apagada**: `// Aqui morava o TestX, que prendia…`
  diz o que o `git log` esconde; os guardas de citação pedem que entre em
  `tombstones`, `arquivosAusentesDePROPOSITO` ou `simbolosAusentesDePROPOSITO`.

**Idioma não é guarda de pacote nenhum** — ver "Isto NÃO tem guarda, e é decisão
do dono" na [raiz](../../CLAUDE.md). Nome PRÓPRIO do livro não é prosa:
`TestBolaDeFogoWorkedExample` é o nome da magia.

## Código órfão, bancada mentirosa, e raciocínio envelhecido

As três produzem VERDE sobre nada.

### Órfão é dívida OU capacidade esperando gesto

**O que separa as duas é ler o que a coisa FAZ, não contar chamadores.** Uma
camada inteira com teste e sem rota chegando nela é funcionalidade perdida, não
lixo. E **a capacidade esperando gesto pode NÃO servir ao gesto quando ele
chegar**: guardar código por "vai ser útil" aposta num desenho ainda não feito.

### A bancada que escreve o que a produção não escreve

Código que ninguém alcança e teste que arranja o próprio estado parecem
cobertura. Arranjo por uma porta que nenhum caminho de produção usa, ou bancada
que escreve uma coluna que a produção nunca escreve, deixa o verde sobre o
arranjo. **Monte o estado pelo caminho de produção**, não pela porta mais curta.

## Apagar código morto: as armadilhas do RECORTE

Quantos métodos exportados o `*Server` ainda tem se pergunta ao código:

```
grep -c "^func (s \*Server) [A-Z]" serve/api/*.go
```

O compilador pega três armadilhas: recortar função de UMA LINHA procurando `}`
na coluna zero come as vizinhas; o receptor colide com variável do destino;
`s.` dentro de string de formato vira recorte errado. **A quarta ele não pega:
a regra que só aquele teste guardava** — o método sem chamador some com o teste
que o exercitava. Antes de apagar, pergunte o que o teste afirmava, não se
alguém chama.

**Cena se testa pelo `WebRouter()`**, não pelo `Router()` da API JSON. Guarda
que varre rotas morre com o terreno; confira se a invariante mudou de casa (a
escrita de ficha por estranho é do `TestNoSheetWriteAcceptsAStranger`).

