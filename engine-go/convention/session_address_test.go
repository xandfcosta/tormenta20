package convention

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// O ENDEREÇO RESOLVIDO DA SESSÃO TEM UM DONO SÓ: o `routes.Session`.
//
// A prosa desta regra já existia — o `sessionPattern` da Mesa diz, por escrito,
// que "o endereço RESOLVIDO (com os ids) é outra coisa e mora no
// `routes.Session`" — e DEZENOVE sítios a ignoravam (ALE-346). É a lição do
// guia inteira num caso só: uma convenção escrita e não varrida vale para os
// arquivos que alguém apontou, e o vigésimo nasce à mão, em silêncio.
//
// # Por que o modo de falha é silencioso
//
// Nada quebra quando a vigésima grafia diverge. Ela compila, passa na revisão
// de diff e funciona — até o dia em que o prefixo muda (a ALE-345 mudou, em 35
// registros) e sobra uma que aponta para um endereço que ainda existe, responde
// 404, e devolve uma tela que não mudou.
//
// # O que passa
//
//   - o `routes.go` do pacote `routes`, que é onde a regra MORA;
//   - a REGISTRAÇÃO da rota, que é o padrão do chi com os parâmetros nomeados
//     (`{campaignId}`, `{sessionId}`) — ela não é um gesto postando, é a
//     declaração de quem atende.
//
// # E por que AST nos `.go`
//
// Um guarda de linha veria o `/sessoes` de cada comentário que EXPLICA a regra,
// e há cinco deles. Lendo só os literais de string, o comentário sai de graça —
// e é a lição do `TestAWireRouteStartsLowercase`, que passou anos medindo 98
// rotas de 242 porque lia a fonte com regex. O `.templ` não é Go e não tem AST
// aqui, então ele é varrido por texto, com o comentário tirado antes.
func TestNoHandwrittenSessionAddress(t *testing.T) {
	const ondeOEnderecoMora = "serve/web/routes/routes.go"

	raiz, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("achar a raiz: %v", err)
	}
	conjunto := token.NewFileSet()
	go_, templs := 0, 0
	err = filepath.WalkDir(raiz, func(caminho string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(raiz, caminho)
		// O GERADO não conta: ele é a saída do `.templ`, e uma violação nele já
		// foi acusada na fonte. O de TESTE também não — um teste afirma o
		// endereço concreto que o servidor devolveu, e é para isso que ele
		// existe.
		if strings.HasSuffix(caminho, "_templ.go") || strings.HasSuffix(caminho, "_test.go") ||
			rel == ondeOEnderecoMora {
			return nil
		}

		acusa := func(linha int, literal string) {
			t.Errorf("%s:%d escreve o endereço da sessão à mão: %s\n"+
				"Use o `routes.Session(campanha, sessao)` e pendure o verbo nele —\n"+
				"numa cena da Mesa, o `v.SessionBase()` já o resolve.\n"+
				"Uma segunda grafia do mesmo endereço é a que sobra apontando para\n"+
				"o lugar errado no dia em que o prefixo mudar.", rel, linha, literal)
		}

		switch {
		case strings.HasSuffix(caminho, ".go"):
			arquivo, err := parser.ParseFile(conjunto, caminho, nil, 0)
			if err != nil {
				return err
			}
			go_++
			ast.Inspect(arquivo, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING || !isResolvedSessionAddress(lit.Value) {
					return true
				}
				acusa(conjunto.Position(lit.Pos()).Line, lit.Value)
				return true
			})
		case strings.HasSuffix(caminho, ".templ"):
			bruto, err := os.ReadFile(caminho)
			if err != nil {
				return err
			}
			templs++
			for numero, linha := range strings.Split(string(bruto), "\n") {
				// O comentário sai ANTES de medir: sem isto, o bloco que
				// EXPLICA a regra seria o primeiro reprovado por ela.
				if j := strings.Index(linha, "//"); j >= 0 {
					linha = linha[:j]
				}
				if isResolvedSessionAddress(linha) {
					acusa(numero+1, strings.TrimSpace(linha))
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR: uma lista de reprovados vazia e uma varredura que não abriu
	// arquivo nenhum se parecem no terminal.
	if go_ < 200 {
		t.Fatalf("o guarda leu só %d arquivos `.go` — ele está medindo a árvore errada", go_)
	}
	if templs < 20 {
		t.Fatalf("o guarda leu só %d arquivos `.templ` — ele está medindo a árvore errada", templs)
	}
}

// isResolvedSessionAddress separa o endereço RESOLVIDO do padrão de REGISTRO.
//
// O nome sai em inglês porque é identificador, e a prosa em português porque é
// comentário — a linha passa entre o que o compilador consome e o que uma pessoa
// lê.
//
// O que distingue os dois é o parâmetro nomeado: `/campanhas/{campaignId}/sessoes/{sessionId}`
// é o chi declarando quem atende, e `/campanhas/1/sessoes/4` é alguém mandando
// o navegador para lá.
func isResolvedSessionAddress(literal string) bool {
	if !strings.Contains(literal, "/sessoes") {
		return false
	}
	return !strings.Contains(literal, "{campaignId}") && !strings.Contains(literal, "{sessionId}")
}
