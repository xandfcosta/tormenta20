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
	const oFunil = "domain/sheet/pools.go"

	// As queries que tocam o estado vital. Eram sete: cinco escreviam as quatro
	// colunas de `characters`, e elas saíram na 00015 junto com as queries. O que
	// resta é o DANO, que é o que se guarda. Mexeu no `query.sql`? Esta lista
	// acompanha, e o `TestEveryVitalQueryIsKnownToTheFunnelGuard` cobra.
	escritasVitais := map[string]bool{
		"SaveCharacterDamage": true, "ClearCharacterDamage": true,
	}

	raiz, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("achar a raiz: %v", err)
	}
	conjunto := token.NewFileSet()
	medidos, chamadasNoFunil := 0, 0
	err = filepath.WalkDir(raiz, func(caminho string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(caminho, ".go") ||
			strings.HasSuffix(caminho, "_templ.go") {
			return err
		}
		rel, _ := filepath.Rel(raiz, caminho)
		if strings.HasPrefix(rel, "infra/db/sqlcgen") || strings.HasSuffix(rel, "vital_write_test.go") {
			return nil
		}
		arquivo, err := parser.ParseFile(conjunto, caminho, nil, 0)
		if err != nil {
			return err
		}
		medidos++
		ast.Inspect(arquivo, func(n ast.Node) bool {
			chamada, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			alvo, ok := chamada.Fun.(*ast.SelectorExpr)
			if !ok || !escritasVitais[alvo.Sel.Name] {
				return true
			}
			if rel == oFunil {
				chamadasNoFunil++
				return true
			}
			t.Errorf("%s:%d chama %s fora do funil.\n"+
				"O que o banco guarda é o DANO: gravar o atual por aqui produz um número\n"+
				"que a próxima leitura do agregado joga fora, sem erro nenhum. Use o\n"+
				"`sheet.ApplyToPools` (ou `ApplyToLoadedPools`, se já tiver a ficha) e\n"+
				"devolva o atual que a regra do seu gesto decidiu — o %s grava.",
				rel, conjunto.Position(chamada.Pos()).Line, alvo.Sel.Name, oFunil)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR, nas duas pontas: uma varredura que não abriu arquivo nenhum
	// e um funil que deixou de escrever se parecem com "nada reprovou".
	if medidos < 200 {
		t.Fatalf("o guarda leu só %d arquivos — ele está medindo a árvore errada", medidos)
	}
	if chamadasNoFunil < 2 {
		t.Fatalf("o funil chama só %d das escritas vitais — se ele parou de gravar, "+
			"este guarda estaria verde sobre um repositório que não persiste PV", chamadasNoFunil)
	}
}

// O CONTROLE do guarda acima: a lista de PERMITIDOS dele é escrita à mão, e uma
// query nova em `query.sql` que toque os mesmos campos nasceria fora dela — sem
// reprovar ninguém, porque o nome que ela não conhece ela também não procura.
//
// Aqui a pergunta é invertida: o que o SQL escreve é conferido contra o que o
// guarda conhece.
func TestEveryVitalQueryIsKnownToTheFunnelGuard(t *testing.T) {
	conhecidas := map[string]bool{
		"SaveCharacterDamage": true, "ClearCharacterDamage": true,
	}
	bruto, err := os.ReadFile(filepath.Join("..", "infra", "db", "query.sql"))
	if err != nil {
		t.Fatalf("ler o query.sql: %v", err)
	}

	medidas := 0
	for _, bloco := range strings.Split(string(bruto), "-- name: ")[1:] {
		nome, _, _ := strings.Cut(bloco, " ")
		corpo := strings.ToLower(bloco)
		escreve := strings.Contains(corpo, "update ") ||
			strings.Contains(corpo, "insert ") || strings.Contains(corpo, "delete ")
		if !escreve || !touchesVitalState(corpo) {
			continue
		}
		medidas++
		if !conhecidas[nome] {
			t.Errorf("a query %q escreve num campo vital e o "+
				"`TestEveryVitalWriteGoesThroughTheFunnel` não a conhece — "+
				"ele varreria a árvore inteira sem procurar por ela.\n"+
				"Ponha o nome na lista de lá (ou aqui, se ela não for um gesto).", nome)
		}
	}
	// Duas, e o piso é o número exato: depois da 00015 não existe mais campo
	// vital em `characters`, então tudo o que o `query.sql` pode escrever é a
	// tabela do dano. Um piso maior só voltaria com uma coluna nova.
	if medidas != 2 {
		t.Fatalf("o guarda achou %d queries vitais no query.sql, e são 2 (o dano) — "+
			"ou nasceu uma escrita nova, ou o formato do arquivo mudou e ele parou de ler", medidas)
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
	for _, campo := range []string{"hpcurrent", "hpmax", "mpcurrent", "mpmax", "hpdamage", "mpspent"} {
		if strings.Contains(lowercaseBody, campo) {
			return true
		}
	}
	return false
}
