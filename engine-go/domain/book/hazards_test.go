package book

import "testing"

// Os perigos do Capítulo 7 chegam ao app nos três tipos, com os números que a mesa usa (p317-319).

func TestTheHazardsComeInTheThreeKindsOfTheBook(t *testing.T) {
	count := map[string]int{}
	for _, h := range Hazards() {
		count[h.Kind]++
		if h.BookPage < 317 || h.BookPage > 319 {
			t.Errorf("%s diz estar na p%d, fora das p317-319", h.Name, h.BookPage)
		}
		if (h.Kind == HazardKindTrap) != (h.Trap != nil) || (h.Kind == HazardKindDisease) != (h.Disease != nil) {
			t.Errorf("%s é %q e a ficha não é a do tipo dele", h.Name, h.Kind)
		}
	}
	want := map[string]int{HazardKindTrap: 23, HazardKindDisease: 8, HazardKindAmbient: 12}
	for kind, n := range want {
		if count[kind] != n {
			t.Errorf("%d perigos do tipo %q, e o livro tem %d", count[kind], kind, n)
		}
	}
}

func TestTheTrapCarriesWhatTheMasterRolls(t *testing.T) {
	needle, ok := HazardByID("agulha-envenenada")
	if !ok {
		t.Fatal("a Agulha Envenenada (p317) não está no catálogo")
	}
	trap := needle.Trap
	if trap.ND != 0.25 || trap.Save == nil || trap.Save.Test != "Reflexos" || trap.Save.DC != 20 ||
		trap.Investigation != 25 || trap.Thievery != 20 {
		t.Errorf("a Agulha Envenenada é ND 1/4, Reflexos CD 20, Investigação 25 e Ladinagem 20 (p317), e veio %+v", trap)
	}
	if rune, _ := HazardByID("runa-de-protecao"); !rune.Trap.Magical {
		t.Error("a Runa de Proteção tem o nome em itálico (p317), e veio sem a marca mágica")
	}
}
