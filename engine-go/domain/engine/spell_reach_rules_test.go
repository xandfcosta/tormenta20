package engine

import (
	"encoding/json"
	"testing"

	"t20engine/domain/catalog/transcript"
)

// TODO MODIFICADOR DE MAGIA ALCANÇA UMA MAGIA QUE EXISTE.
//
// A CD, o limite e o custo de PM se perguntam pelo `ForSpell`, que casa o alvo do modificador com a
// magia. Um modificador que não casa com magia nenhuma não reprova nada: ele só não vale, com a linha
// `Modeled` dizendo que vale. Foi assim que o −1 PM do cajado do poder e do símbolo abençoado passaram
// sem ninguém os ler — o `Scope` deles queria dizer tradição, e o custo o lia como alcance.
//
// O guarda varre o catálogo inteiro (`collectModifierOwners`) e pede, para cada modificador de magia:
//   - nada de `Scope`: tradição, escola, alcance e magia têm campos próprios;
//   - a tradição é uma das duas listas (p170);
//   - a escola, o alcance e a magia existem no capítulo de magias, e JUNTOS: o bônus do medalhão das
//     pessoais numa escola sem magia pessoal não alcançaria ninguém.
func TestEverySpellModifierReachesARealSpell(t *testing.T) {
	var spells map[string]struct {
		School string `json:"school"`
		Range  string `json:"range"`
	}
	if err := json.Unmarshal(bookResource(t, "spells"), &spells); err != nil {
		t.Fatalf("o capítulo de magias está ilegível: %v", err)
	}
	if len(spells) < 100 {
		t.Fatalf("só %d magias lidas — o guarda mediria contra um capítulo vazio", len(spells))
	}
	traditions := map[string]bool{"": true, "arcana": true, "divina": true}

	measured := 0
	for _, path := range transcript.Resources {
		var tree any
		if err := json.Unmarshal(bookResource(t, path), &tree); err != nil {
			t.Fatalf("%s ilegível: %v", path, err)
		}
		for _, owner := range collectModifierOwners(tree, path+".json") {
			for _, m := range modifiersOf(t, owner) {
				target := m.Target
				if target.K != "spellDC" && target.K != "pmLimit" && target.K != "pmCost" {
					continue
				}
				measured++
				where := owner.id + " (" + path + ")"
				if target.Scope != "" {
					t.Errorf("%s: o %s usa `scope: %q`. Nas contas de magia o que o modificador alcança tem "+
						"campo próprio: `tradition` (arcana, divina), `school`, `range` ou `spell`.",
						where, target.K, target.Scope)
					continue
				}
				if !traditions[target.Tradition] {
					t.Errorf("%s: o %s pede a tradição %q, e as listas são arcana e divina (p170)",
						where, target.K, target.Tradition)
					continue
				}
				if target.School == "" && target.Range == "" && target.Spell == "" {
					continue
				}
				reaches := false
				for id, spell := range spells {
					if (target.School == "" || target.School == spell.School) &&
						(target.Range == "" || target.Range == spell.Range) &&
						(target.Spell == "" || target.Spell == id) {
						reaches = true
						break
					}
				}
				if !reaches {
					t.Errorf("%s: o %s pede escola %q, alcance %q e magia %q, e nenhuma magia do livro tem "+
						"os três: ele não vale em lugar nenhum", where, target.K, target.School, target.Range, target.Spell)
				}
			}
		}
	}
	// O DENOMINADOR: um piso bem abaixo do que o catálogo tem entre itens, poderes e magias. Abaixo dele, a
	// varredura parou de achar os modificadores, e o verde não mediria nada.
	if measured < 15 {
		t.Fatalf("o guarda mediu só %d modificadores de magia — a varredura não está achando o catálogo", measured)
	}
}
