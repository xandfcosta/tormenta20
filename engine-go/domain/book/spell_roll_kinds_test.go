package book

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// bookDamageTypes são os tipos de dano da p230, na ordem do livro.
var bookDamageTypes = []string{
	"ácido", "corte", "eletricidade", "essência", "fogo", "frio", "impacto", "luz", "perfuração", "psíquico", "trevas",
}

// TestEverySpellRollKindIsOneTheBookNames confere o tipo de TODA parcela do dado das magias: um dos tipos de dano da
// p230, a cura, os PV temporários, um composto deles ("corte e fogo, metade de cada", "corte, impacto ou
// perfuração"), ou nenhum, quando o livro não diz. A parcela de tipo que o livro não nomeia seria DANO em silêncio
// (`SpellRoll.IsDamage`): um "Cura" com maiúscula ganharia o Arcano de Batalha.
//
// A VARREDURA É PELO JSON, e não pelos campos: ela acha qualquer objeto com `dice` em qualquer lugar da magia (o dado
// da conjuração, os aprimoramentos, o dado da vez, a ação, a marca), e quem guardar parcela num campo novo entra
// sozinho. O denominador impede o guarda de passar vazio se a forma do JSON mudar.
func TestEverySpellRollKindIsOneTheBookNames(t *testing.T) {
	raw, err := json.Marshal(Catalogs().Spells)
	if err != nil {
		t.Fatalf("serializar as magias: %v", err)
	}
	var spells []map[string]any
	if err := json.Unmarshal(raw, &spells); err != nil {
		t.Fatalf("ler as magias de volta: %v", err)
	}
	measured := 0
	for _, spell := range spells {
		for _, kind := range spellRollKindsIn(spell) {
			measured++
			if !kindTheBookNames(kind) {
				t.Errorf("a magia %v tem uma parcela de tipo %q, que não é um tipo de dano da p230 (%s), nem %q, nem %q, "+
					"nem um composto deles; conserte a grafia na transcrição", spell["id"], kind,
					strings.Join(bookDamageTypes, ", "), healingKind, temporaryHitPointsKind)
			}
		}
	}
	// O catálogo tem centenas de parcelas; menos de cem quer dizer que a varredura deixou de achá-las.
	if measured < 100 {
		t.Fatalf("a varredura achou %d parcelas nas magias; o catálogo tem centenas, e a forma do JSON mudou", measured)
	}
}

// spellRollKindsIn são os tipos de toda parcela (objeto com `dice`) dentro de um pedaço do JSON da magia.
func spellRollKindsIn(node any) []string {
	var kinds []string
	switch value := node.(type) {
	case map[string]any:
		if _, isRoll := value["dice"]; isRoll {
			kind, _ := value["kind"].(string)
			kinds = append(kinds, kind)
		}
		for _, child := range value {
			kinds = append(kinds, spellRollKindsIn(child)...)
		}
	case []any:
		for _, child := range value {
			kinds = append(kinds, spellRollKindsIn(child)...)
		}
	}
	return kinds
}

// kindTheBookNames diz se o tipo da parcela é um que o livro nomeia, sozinho ou composto: "corte e fogo, metade de
// cada" e "ácido, eletricidade, fogo ou frio" se partem nos tipos de dano.
func kindTheBookNames(kind string) bool {
	if kind == "" || kind == healingKind || kind == temporaryHitPointsKind {
		return true
	}
	joined := strings.TrimSuffix(kind, ", metade de cada")
	joined = strings.NewReplacer(" e ", ", ", " ou ", ", ").Replace(joined)
	for _, piece := range strings.Split(joined, ", ") {
		if !slices.Contains(bookDamageTypes, piece) {
			return false
		}
	}
	return true
}

// O separador do composto não pode deixar passar um tipo desconhecido escondido no meio.
func TestTheCompoundKindIsSplitIntoBookDamageTypes(t *testing.T) {
	for kind, want := range map[string]bool{
		"corte e fogo, metade de cada":      true,
		"ácido, eletricidade, fogo ou frio": true,
		"Cura":                              false,
		"PV temporario":                     false,
		"fogo e gelo, metade de cada":       false,
		"corte, sônico ou perfuração":       false,
	} {
		if got := kindTheBookNames(kind); got != want {
			t.Errorf("o tipo %q é reconhecido=%v; want %v", kind, got, want)
		}
	}
}
