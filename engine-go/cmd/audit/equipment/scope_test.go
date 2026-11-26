package main

import (
	"testing"

	"t20engine/domain/catalog/transcript"
)

// QUEM MEDE CADA ITEM: este auditor lê as tabelas do capítulo 3 (p138-167), e o que é de outro capítulo, ou de
// outra tabela, tem auditor irmão. Item do capítulo 8 que nenhum irmão alcança sai como "nenhum", que é o achado.
func TestEachItemHasTheAuditorThatMeasuresIt(t *testing.T) {
	for _, c := range []struct {
		name string
		item transcript.Item
		want string
	}{
		{"a corda, das tabelas do capítulo 3", transcript.Item{Category: "apparel", BookPage: 156}, ""},
		{"o material, da Tabela 3-9", transcript.Item{Category: "material", BookPage: 166}, "materials"},
		{"o encanto de arma", transcript.Item{Category: "weapon-enchant", BookPage: 334}, "enchants"},
		{"o encanto de armadura", transcript.Item{Category: "armor-enchant", BookPage: 340}, "enchants"},
		{"o item específico", transcript.Item{Category: "weapon-martial", BookPage: 336, ComposedOf: &transcript.ItemComposition{}}, "specificitems"},
		{"a poção", transcript.Item{Category: "consumable", BookPage: 342, Consumable: &transcript.Consumable{Spell: "curar-ferimentos"}}, "potions"},
		{"o acessório", transcript.Item{Category: "apparel", BookPage: 345}, "accessories"},
		{"o artefato", transcript.Item{Category: "weapon-martial", BookPage: 347, Artifact: true}, "artifacts"},
		{"o do capítulo 8 que ninguém mede", transcript.Item{Category: "weapon-martial", BookPage: 336}, "nenhum"},
	} {
		if got := measuredBy(c.item); got != c.want {
			t.Errorf("%s: quem mede é %q, want %q", c.name, got, c.want)
		}
	}
}

// O NOME CASA SEM O QUALIFICADOR entre parênteses, dos dois lados: o catálogo guarda "Corda (10 m)" porque os 10 m
// importam na mesa, e a tabela diz "Corda"; a tabela diz "Ração de viagem (por dia)", e o catálogo "Ração de Viagem".
func TestTheNameMatchesWithoutTheQualifier(t *testing.T) {
	for _, pair := range [][2]string{
		{"Corda (10 m)", "Corda"},
		{"Coleção de livros (Guerra)", "Coleção de livros"},
		{"Harmonizada (arma)", "Harmonizada"},
		{"Ração de Viagem", "Ração de viagem (por dia)"},
	} {
		if a, b := nameKey(pair[0]), nameKey(pair[1]); a != b || a == "" {
			t.Errorf("%q e %q não casam: %q contra %q", pair[0], pair[1], a, b)
		}
	}
	if nameKey("Corda") == nameKey("Cordas de alaúde") {
		t.Error("o controle falhou: nomes diferentes casaram")
	}
	if nameKey("(por dia)") != "" {
		t.Errorf("uma célula só de qualificador virou a chave %q, e casaria com qualquer coisa vazia", nameKey("(por dia)"))
	}
}
