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

// TODA ESCRITA DE PV E PM PASSA PELO FUNIL, e o funil é o `domain/sheet/pools.go`.
//
// # O mecanismo, que é o que faz disto guarda e não parágrafo
//
// Desde que o máximo virou DERIVADO (ALE-355), o que o banco guarda é o DANO —
// quanto o personagem apanhou — e o atual sai de `máximo derivado − dano`. Um
// gesto que escreva `hpCurrent` direto não quebra nada na hora: ele grava um
// número plausível, a tela o mostra, e a próxima leitura do agregado o DESCARTA
// e desenha o atual derivado de um dano que ninguém atualizou. O dano some sem
// erro em lugar nenhum, que é a assinatura desta família.
//
// Sete sítios computavam o atual novo por conta própria, e três deles liam o
// máximo da linha — a coluna que deixou de ser verdade e depois deixou de
// existir. O funil dá a todos o mesmo material e guarda para si a única escrita.
//
// # O que ele procura
//
// As duas queries do DANO, que é tudo o que se grava de vital desde a 00015 —
// as quatro colunas de `characters` não existem mais. Elas são PERMITIDOS por
// nome e não proibidos por forma: uma query nova que mexa no mesmo estado cai
// aqui sozinha, porque a lista abaixo é de quem PODE e o teste falha no que não
// conhece (ALE-301).
func TestEveryVitalWriteGoesThroughTheFunnel(t *testing.T) {
	const funnelFile = "domain/sheet/pools.go"

	// As queries que tocam o estado vital. Eram sete: cinco escreviam as quatro
	// colunas de `characters`, e elas saíram na 00015 junto com as queries. O que
	// resta é o DANO, que é o que se guarda. Mexeu no `query.sql`? Esta lista
	// acompanha, e o `TestEveryVitalQueryIsKnownToTheFunnelGuard` cobra.
	vitalWrites := map[string]bool{
		"SaveCharacterDamage": true, "ClearCharacterDamage": true,
	}

	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("achar a raiz: %v", err)
	}
	set := token.NewFileSet()
	measured, funnelCalls := 0, 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_templ.go") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if strings.HasPrefix(rel, "infra/db/sqlcgen") || strings.HasSuffix(rel, "vital_write_test.go") {
			return nil
		}
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			return err
		}
		measured++
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			target, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !vitalWrites[target.Sel.Name] {
				return true
			}
			if rel == funnelFile {
				funnelCalls++
				return true
			}
			t.Errorf("%s:%d chama %s fora do funil.\n"+
				"O que o banco guarda é o DANO: gravar o atual por aqui produz um número\n"+
				"que a próxima leitura do agregado joga fora, sem erro nenhum. Use o\n"+
				"`sheet.ApplyToPools` (ou `ApplyToLoadedPools`, se já tiver a ficha) e\n"+
				"devolva o atual que a regra do seu gesto decidiu — o %s grava.",
				rel, set.Position(call.Pos()).Line, target.Sel.Name, funnelFile)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR, nas duas pontas: uma varredura que não abriu arquivo nenhum
	// e um funil que deixou de escrever se parecem com "nada reprovou".
	if measured < 200 {
		t.Fatalf("o guarda leu só %d arquivos — ele está medindo a árvore errada", measured)
	}
	if funnelCalls < 2 {
		t.Fatalf("o funil chama só %d das escritas vitais — se ele parou de gravar, "+
			"este guarda estaria verde sobre um repositório que não persiste PV", funnelCalls)
	}
}

// O CONTROLE do guarda acima: a lista de PERMITIDOS dele é escrita à mão, e uma
// query nova em `query.sql` que toque os mesmos campos nasceria fora dela — sem
// reprovar ninguém, porque o nome que ela não conhece ela também não procura.
//
// Aqui a pergunta é invertida: o que o SQL escreve é conferido contra o que o
// guarda conhece.
func TestEveryVitalQueryIsKnownToTheFunnelGuard(t *testing.T) {
	known := map[string]bool{
		"SaveCharacterDamage": true, "ClearCharacterDamage": true,
	}
	raw, err := os.ReadFile(filepath.Join("..", "infra", "db", "query.sql"))
	if err != nil {
		t.Fatalf("ler o query.sql: %v", err)
	}

	measured := 0
	for _, block := range strings.Split(string(raw), "-- name: ")[1:] {
		name, _, _ := strings.Cut(block, " ")
		body := strings.ToLower(block)
		writes := strings.Contains(body, "update ") ||
			strings.Contains(body, "insert ") || strings.Contains(body, "delete ")
		if !writes || !touchesVitalState(body) {
			continue
		}
		measured++
		if !known[name] {
			t.Errorf("a query %q escreve num campo vital e o "+
				"`TestEveryVitalWriteGoesThroughTheFunnel` não a conhece — "+
				"ele varreria a árvore inteira sem procurar por ela.\n"+
				"Ponha o nome na lista de lá (ou aqui, se ela não for um gesto).", name)
		}
	}
	// Duas, e o piso é o número exato: depois da 00015 não existe mais campo
	// vital em `characters`, então tudo o que o `query.sql` pode escrever é a
	// tabela do dano. Um piso maior só voltaria com uma coluna nova.
	if measured != 2 {
		t.Fatalf("o guarda achou %d queries vitais no query.sql, e são 2 (o dano) — "+
			"ou nasceu uma escrita nova, ou o formato do arquivo mudou e ele parou de ler", measured)
	}
}

// touchesVitalState procura o estado vital, e a TABELA entra junto com os
// campos: o `ClearCharacterDamage` apaga a linha inteira e não nomeia campo
// nenhum — procurar só por nome de coluna o deixaria de fora, que é a forma de
// um guarda subcontar em silêncio.
//
// Os quatro campos de `characters` continuam na lista mesmo tendo saído na
// 00015: é o que faz uma coluna vital RESSUSCITADA cair aqui.
func touchesVitalState(lowercaseBody string) bool {
	if strings.Contains(lowercaseBody, "character_damage") {
		return true
	}
	for _, field := range []string{"hpcurrent", "hpmax", "mpcurrent", "mpmax", "hpdamage", "mpspent"} {
		if strings.Contains(lowercaseBody, field) {
			return true
		}
	}
	return false
}
