package api

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// A FORMA SELVAGEM EM VIGOR É UM EFEITO COMO OS OUTROS. A aba Efeitos a mostrava em
// "Em vigor", mas a coluna da ficha e o número do ícone de Efeitos não a contavam: no
// desktop o druida transformado não a via na coluna, e no celular o ícone dizia que não
// havia nada em vigor.
func TestTheWildShapeInPlayIsInTheRailAndInTheCount(t *testing.T) {
	f, id := wildDruid(t)
	count := regexp.MustCompile(`class="ds-nav-count" aria-label="(\d+) ativos"`)
	// O id do diálogo vem montado do código (`activeEffects`), como os das condições.
	shapeDialog := "effect-wild-shape"

	// O CONTROLE: humano, a coluna não tem a forma e o ícone não conta nada dela.
	before := sheetTabWith(t, f, id, "skills", `{}`)
	if strings.Contains(before, `commandfor="`+shapeDialog+`"`) {
		t.Fatal("a coluna mostra uma forma antes de o druida se transformar")
	}
	countBefore := 0
	if m := count.FindStringSubmatch(before); m != nil {
		fmt.Sscan(m[1], &countBefore)
	}

	shape := `{"wild_form":"feroz","wild_tier":"normal","wild_damage":"perfuracao","wild_move":""}`
	if refusal := powerCommand(t, f, id, "forma-selvagem", shape); refusal != "" {
		t.Fatalf("assumir a forma: %q", refusal)
	}
	after := sheetTabWith(t, f, id, "skills", `{}`)
	if !strings.Contains(after, `commandfor="`+shapeDialog+`"`) {
		t.Error("a coluna da ficha não mostra a Forma Selvagem em vigor")
	}
	if !regexp.MustCompile(`(?s)<dialog[^>]*id="` + shapeDialog + `".*?/efeitos/forma-selvagem`).MatchString(after) {
		t.Error("a descrição da forma em vigor não oferece voltar ao normal")
	}
	m := count.FindStringSubmatch(after)
	if m == nil {
		t.Fatal("o ícone de Efeitos não tem número com a forma em vigor")
	}
	var countAfter int
	fmt.Sscan(m[1], &countAfter)
	if countAfter != countBefore+1 {
		t.Errorf("o ícone de Efeitos diz %d ativos, e com a forma eram para ser %d", countAfter, countBefore+1)
	}
}
