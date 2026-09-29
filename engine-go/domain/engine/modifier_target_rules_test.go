package engine

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

// O CATÁLOGO SÓ USA ALVO QUE O MOTOR APLICA (ALE-418).
//
// # O defeito que ele repõe, e ele apareceu três vezes
//
// O `critRange`, o `critMult` e o `damage` de escopo `this` estavam no catálogo,
// apareciam na aba Efeitos e NÃO entravam em conta nenhuma (ALE-411). Cada um
// custou uma medição própria para ser achado, e a forma dos três é a mesma:
// metade do caminho funciona — o rótulo — e a outra metade não existe.
//
// **O rótulo não conta como leitor.** Ele é exatamente a metade que já funciona
// no defeito, e tratá-lo como prova faria este guarda aprovar o que veio medir.
//
// # As duas listas, e por que elas são PERMITIDOS
//
// Uma lista de proibidos subconta em silêncio. Aqui as duas juntas têm de
// cobrir EXATAMENTE os alvos que o `targetKey` nomeia — alvo novo sem entrada
// em nenhuma delas reprova pelo nome, e alvo que sai do `targetKey` sem sair da
// lista também.
//
// A lista de inertes é linha de base, e ela **só pode encolher**.
var alvosQueOMotorAplica = map[string]string{
	"attribute":            "breakdowns.go, na soma de cada atributo",
	"expertise":            "breakdowns.go, no total da perícia",
	"expertiseAll":         "breakdowns.go, em toda perícia",
	"expertiseByAttribute": "breakdowns.go, nas perícias do atributo",
	"defense":              "breakdowns.go, na Defesa",
	"attack":               "weapons.go, no ataque da carta",
	"damage":               "weapons.go e attack_ecs.go, no dano",
	"critRange":            "weapons.go, na margem de ameaça (ALE-411)",
	"critMult":             "weapons.go, no multiplicador (ALE-411)",
	"damageReduction":      "breakdowns.go, na RD",
	"armorPenalty":         "breakdowns.go e load.go",
	"displacement":         "breakdowns.go, no deslocamento",
	"flySpeed":             "breakdowns.go, no voo",
	"maxPv":                "vitals_catalog.go",
	"maxPm":                "vitals_catalog.go",
	"tempHp":               "sheet/temp_hp.go, na reserva que o dano gasta antes do PV",
	"pmLimit":              "breakdowns_magic.go, no limite de PM por magia",
	"pmCost":               "breakdowns_magic.go, no custo da magia",
	"spellDC":              "breakdowns_magic.go, na CD",
	"inventorySlots":       "breakdowns_magic.go, nos espaços da mochila",
	"catalyst":             "bag_improvements.go, no catalisador",
	"flag":                 "collect.go, como interruptor",
}

// alvosDeclaradosInertes é a LINHA DE BASE: o motor conhece o alvo e não o
// aplica. Cada um diz por quê, e a lista só pode encolher.
//
// Os três primeiros são DEFEITO — há modificador escrito no catálogo que a
// ficha mostra e a conta ignora. O quarto não é: ele está declarado por escrito
// no `withTempHp`, e a decisão de não desenhar é deliberada.
var alvosDeclaradosInertes = map[string]string{
	"maneuver": "o motor não resolve MANOBRA nenhuma — não há gesto onde somar o " +
		"+2 do Derrubar Aprimorado. É a forma do `critRange` antes da ALE-364: " +
		"um número que nada consome",
	"resistance": "carrega DOIS conceitos do livro — \"testes de resistência\" e " +
		"\"resistência a magia\" —, e escolher um leitor só faria metade das " +
		"entradas mentirem. Passa pelo GLOSSARY.md antes do código",
	"fearResistance": "uma entrada no catálogo e nenhum gesto que teste medo",
	"tempMp": "DELIBERADO, e escrito no `withTempHp`: o livro tem PM temporário " +
		"(p106) e este app não o modela — nada os gasta, e desenhar um número " +
		"que nada consome seria pior que não desenhá-lo",
}

func TestTheCatalogOnlyUsesTargetsTheEngineApplies(t *testing.T) {
	doSwitch := alvosQueOTargetKeyNomeia(t)
	declarados := map[string]bool{}
	for k := range alvosQueOMotorAplica {
		declarados[k] = true
	}
	for k := range alvosDeclaradosInertes {
		if declarados[k] {
			t.Errorf("%q está nas DUAS listas: ou o motor o aplica, ou não", k)
		}
		declarados[k] = true
	}

	for _, k := range doSwitch {
		if !declarados[k] {
			t.Errorf("o `targetKey` nomeia %q e nenhuma das duas listas o declara.\n"+
				"Alvo novo entra JUNTO: em `alvosQueOMotorAplica` com o arquivo que o "+
				"lê, ou em `alvosDeclaradosInertes` com o motivo. Sem isso ele nasce "+
				"podendo ser escrito no catálogo e ignorado pela conta, que é o "+
				"defeito que este guarda existe para não deixar acontecer de novo.", k)
		}
	}
	nomeados := map[string]bool{}
	for _, k := range doSwitch {
		nomeados[k] = true
	}
	for k := range declarados {
		if !nomeados[k] {
			t.Errorf("%q está declarado e o `targetKey` não o nomeia mais — "+
				"a lista ficou para trás de um corte", k)
		}
	}

	// O CATÁLOGO, e é aqui que a lista de inertes vira consequência.
	porAlvo := alvosQueOCatalogoUsa(t)
	for alvo, quantos := range porAlvo {
		if motivo, inerte := alvosDeclaradosInertes[alvo]; inerte {
			t.Logf("INERTE COM USO: %s tem %d modificadores no catálogo que a ficha "+
				"mostra e a conta ignora — %s", alvo, quantos, motivo)
			continue
		}
		if _, aplica := alvosQueOMotorAplica[alvo]; !aplica {
			t.Errorf("o catálogo usa o alvo %q em %d modificadores e ele não está em "+
				"lista nenhuma", alvo, quantos)
		}
	}
	t.Logf("alvos medidos: %d nomeados pelo `targetKey`, %d aplicados, %d inertes",
		len(doSwitch), len(alvosQueOMotorAplica), len(alvosDeclaradosInertes))
}

// alvosQueOTargetKeyNomeia lê os `case` do `switch` do `targetKey`.
//
// Pela ÁRVORE e não por regex: o `targetKey` tem `case` com dois valores na
// mesma linha e `case` dentro de `if` aninhado, e um regex que erre um deles
// deixa o alvo fora do denominador — que é a forma de este guarda mentir.
func alvosQueOTargetKeyNomeia(t *testing.T) []string {
	t.Helper()
	arquivo, err := parser.ParseFile(token.NewFileSet(), "itemeffects.go", nil, 0)
	if err != nil {
		t.Fatalf("ler o itemeffects.go: %v", err)
	}
	achados := map[string]bool{}
	for _, decl := range arquivo.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "targetKey" {
			continue
		}
		ast.Inspect(fn, func(n ast.Node) bool {
			clause, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, expr := range clause.List {
				lit, ok := expr.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				if valor, err := strconv.Unquote(lit.Value); err == nil {
					achados[valor] = true
				}
			}
			return true
		})
	}
	if len(achados) == 0 {
		t.Fatal("nenhum `case` lido do `targetKey` — o guarda ficou cego")
	}
	fora := make([]string, 0, len(achados))
	for k := range achados {
		fora = append(fora, k)
	}
	sort.Strings(fora)
	return fora
}

// alvosQueOCatalogoUsa conta os `target.k` que o catálogo escreve.
//
// Ele lê os arquivos EMBUTIDOS, e não o despejo. A razão é a contagem: as duas
// cópias do catálogo são a MESMA coisa para as seis coleções que o despejo tem,
// e somar os dois contava cada modificador duas vezes — 28 manobras onde há 14,
// com cara de resultado.
//
// Os embutidos são o conjunto COMPLETO: o despejo não tem magias, e os
// modificadores delas chegam à ficha pelo efeito consumido. Que as seis
// coleções compartilhadas batem entre as duas cópias é o que o
// `TestDumpAgreesWithEmbeddedCatalog` garante — sem ele, ler só um lado deixaria
// o outro fora do denominador.
func alvosQueOCatalogoUsa(t *testing.T) map[string]int {
	t.Helper()
	arquivos, err := filepath.Glob(filepath.Join("..", "catalog", "data", "*.json"))
	if err != nil {
		t.Fatalf("listar o catálogo: %v", err)
	}
	if len(arquivos) < 10 {
		t.Fatalf("só %d arquivos de catálogo — o guarda mediria quase nada", len(arquivos))
	}

	contagem := map[string]int{}
	var desce func(no any)
	desce = func(no any) {
		switch v := no.(type) {
		case map[string]any:
			if alvo, ok := v["target"].(map[string]any); ok {
				if k, ok := alvo["k"].(string); ok {
					contagem[k]++
				}
			}
			for _, filho := range v {
				desce(filho)
			}
		case []any:
			for _, filho := range v {
				desce(filho)
			}
		}
	}
	for _, caminho := range arquivos {
		bruto, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		var qualquer any
		if err := json.Unmarshal(bruto, &qualquer); err != nil {
			t.Fatalf("%s não é JSON: %v", caminho, err)
		}
		desce(qualquer)
	}
	if len(contagem) == 0 {
		t.Fatal("nenhum alvo lido do despejo — o guarda ficou cego")
	}
	return contagem
}
