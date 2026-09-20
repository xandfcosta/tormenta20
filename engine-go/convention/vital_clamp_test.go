package convention

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// UMA GRAFIA SÓ PARA "PRENDA ESTE VITAL ENTRE ZERO E O TETO".
//
// # Por que um guarda, e não a contagem no comentário
//
// O bloco do `sheet.WithinPool` dizia, por extenso, quantas grafias tinham sido
// colapsadas nele. **A fatia que escreveu essa frase criou a sexta**, num
// arquivo a um diretório de distância: um ajudante que não existe mais, com o
// mesmo `min(max(0, …), teto)` por dentro (ALE-355 → ALE-356).
//
// É a lição que o `CLAUDE.md` já registra sobre o número de guardas: uma
// contagem escrita à mão sobre uma família que cresce envelhece, e quem a
// escreve é quem menos desconfia dela.
//
// # O que ele procura, e por que a FORMA basta
//
// O aninhamento `min(max(…))` ou `max(min(…))`. Quem for escrever este clamp de
// novo vai escrevê-lo assim — é a forma mais curta em Go, e foi a forma das três
// grafias à mão que já existiram. Não é uma prova de equivalência semântica: é a
// forma que a próxima cópia terá, que é o que um guarda de varredura precisa.
//
// # O que fica de FORA, e a semelhança é só de forma
//
//   - o `live.ClampVital` prende a entrada do RASTREADOR, cujo máximo é opcional
//     porque um NPC pode não ter nenhum — e ele não usa o aninhamento;
//   - o `BalanceAfterMoneyGesture` do dinheiro NÃO prende, RECUSA, com a frase
//     que diz quanto a pessoa tem. Um clamp ali gastaria até o fundo em silêncio;
//   - o `domain/engine` inteiro, e essa isenção é ESTRUTURAL e não concessão.
//     Ele computa teto de BÔNUS DO LIVRO — a Insolência do Bucaneiro é "+Carisma
//     na Defesa, até o nível de Bucaneiro" (p47), `max(0, min(car, nível))` —, e
//     não poço de ficha. E ele NÃO PODE chamar o `sheet.WithinPool`: a direção
//     de import é `sheet → engine`, então exigir isso dali seria exigir um ciclo.
//
// Este terceiro item saiu do próprio guarda: ele foi escrito achando que só
// havia uma cópia a consertar e apontou a do Bucaneiro na primeira corrida.
func TestNoSecondSpellingOfTheVitalClamp(t *testing.T) {
	const oDono = "domain/sheet/pools.go"

	raiz, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("achar a raiz: %v", err)
	}
	conjunto := token.NewFileSet()
	medidos, noDono := 0, 0
	err = filepath.WalkDir(raiz, func(caminho string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(caminho, ".go") ||
			strings.HasSuffix(caminho, "_templ.go") {
			return err
		}
		rel, _ := filepath.Rel(raiz, caminho)
		if strings.HasSuffix(rel, "vital_clamp_test.go") ||
			strings.HasPrefix(rel, "domain/engine") {
			return nil
		}
		arquivo, err := parser.ParseFile(conjunto, caminho, nil, 0)
		if err != nil {
			return err
		}
		medidos++
		ast.Inspect(arquivo, func(n ast.Node) bool {
			if !isNestedMinMax(n) {
				return true
			}
			if rel == oDono {
				noDono++
				return true
			}
			t.Errorf("%s:%d escreve um `min(max(…))` — é a segunda grafia de "+
				"\"prenda este vital entre zero e o teto\".\n"+
				"Use o `sheet.WithinPool`. A primeira vez que esta regra se duplicou, as duas\n"+
				"cópias ficaram a um diretório de distância e o comentário de uma delas\n"+
				"afirmava que a contagem estava fechada.",
				rel, conjunto.Position(n.Pos()).Line)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR nas duas pontas: uma varredura que não abriu arquivo e um
	// dono que perdeu a forma se parecem com "nada reprovou".
	if medidos < 200 {
		t.Fatalf("o guarda leu só %d arquivos — ele está medindo a árvore errada", medidos)
	}
	if noDono != 1 {
		t.Fatalf("o `%s` tem %d aninhamentos e devia ter exatamente 1 (o `WithinPool`) — "+
			"ou ele mudou de forma, e aí este guarda procura o que não existe mais", oDono, noDono)
	}
}

// isNestedMinMax reconhece `min(max(…), …)` e `max(min(…), …)` — o clamp de duas
// pontas escrito em uma linha.
func isNestedMinMax(n ast.Node) bool {
	fora, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	nomeFora, ok := fora.Fun.(*ast.Ident)
	if !ok || (nomeFora.Name != "min" && nomeFora.Name != "max") {
		return false
	}
	oOutro := "max"
	if nomeFora.Name == "max" {
		oOutro = "min"
	}
	for _, arg := range fora.Args {
		dentro, ok := arg.(*ast.CallExpr)
		if !ok {
			continue
		}
		if nomeDentro, ok := dentro.Fun.(*ast.Ident); ok && nomeDentro.Name == oOutro {
			return true
		}
	}
	return false
}
