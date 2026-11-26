package auditscope

import (
	"sort"
	"testing"

	"t20engine/domain/catalog/transcript"
)

// TODO ITEM DO CATÁLOGO TEM O AUDITOR QUE O MEDE, e isto se prova SEM o PDF: os auditores precisam do livro e
// rodam à mão, e o item que nenhum deles alcança só aparecia para quem os rodasse — foi assim com o Macabro, cuja
// página errada o tirava do capítulo 3. Reprova com o nome do órfão.
func TestEveryCatalogItemHasTheAuditorThatMeasuresIt(t *testing.T) {
	// O DENOMINADOR: o catálogo tem centenas de itens, e um que não carregasse mediria o vazio.
	if len(transcript.Items) < 300 {
		t.Fatalf("o catálogo tem %d itens, e o livro tem mais de 300: o guarda mediria o vazio", len(transcript.Items))
	}
	byAuditor := map[Auditor]int{}
	for _, it := range transcript.Items {
		auditor := Of(it)
		byAuditor[auditor]++
		if auditor == Nobody {
			t.Errorf("%s (%s, p%d, %q) não é medido por auditor nenhum: confira a página e a categoria contra o livro, "+
				"ou dê a ele um auditor em `auditscope.Of`", it.Name, it.ID, it.BookPage, it.Category)
		}
	}
	// CADA AUDITOR TEM OS SEUS: uma regra que esvaziasse um deles mandaria os itens dele para outro sem órfão nenhum.
	var empty []string
	for _, auditor := range All {
		if byAuditor[auditor] == 0 {
			empty = append(empty, string(auditor))
		}
	}
	sort.Strings(empty)
	if len(empty) > 0 {
		t.Errorf("os auditores %v ficaram sem item nenhum: a regra os esvaziou (contagem: %v)", empty, byAuditor)
	}
}

// QUEM MEDE CADA ITEM: a página do capítulo 3 decide primeiro, e no capítulo 8 o tipo do verbete diz a tabela.
func TestEachItemHasTheAuditorOfItsTable(t *testing.T) {
	for _, c := range []struct {
		name string
		item transcript.Item
		want Auditor
	}{
		{"a corda, das tabelas do capítulo 3", transcript.Item{Category: "apparel", BookPage: 156}, Equipment},
		{"o material, da Tabela 3-9", transcript.Item{Category: "material", BookPage: 166}, Materials},
		{"o encanto de arma", transcript.Item{Category: "weapon-enchant", BookPage: 334}, Enchants},
		{"o encanto de armadura", transcript.Item{Category: "armor-enchant", BookPage: 340}, Enchants},
		{"o item específico", transcript.Item{Category: "weapon-martial", BookPage: 336, ComposedOf: &transcript.ItemComposition{}}, SpecificItems},
		{"a poção", transcript.Item{Category: "consumable", BookPage: 342, Consumable: &transcript.Consumable{Spell: "curar-ferimentos"}}, Potions},
		{"o acessório", transcript.Item{Category: "apparel", BookPage: 345}, Accessories},
		{"o artefato, mesmo composto", transcript.Item{Category: "weapon-martial", BookPage: 346, Artifact: true, ComposedOf: &transcript.ItemComposition{}}, Artifacts},
		{"o do capítulo 8 que ninguém mede", transcript.Item{Category: "weapon-martial", BookPage: 336}, Nobody},
	} {
		if got := Of(c.item); got != c.want {
			t.Errorf("%s: quem mede é %q, want %q", c.name, got, c.want)
		}
	}
}
