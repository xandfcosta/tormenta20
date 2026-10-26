package catalog_test

import (
	"strings"
	"testing"

	"t20engine/domain/catalog/transcript"
	"t20engine/domain/sheet"
)

// TODA FLAG DE DESCANSO DO CATÁLOGO É UMA QUE A NOITE LÊ (ALE-488).
//
// O casamento é por TEXTO: uma flag `rest-*` escrita errado num verbete é uma
// habilidade que nunca muda a noite, sem erro nenhum. A varredura vai aos
// valores tipados do livro transcrito — raça, origem (benefício e poder único)
// e poderes de classe e gerais —, porque é lá que as seis moram.
//
// O DENOMINADOR é escrito à mão: Rato das Ruas, Vida Rústica, Estoico, Pajem,
// golem e osteon. A sereia não levanta flag (o PM sem água é do mestre).
func TestEveryRestFlagTheCatalogRaisesIsOneTheNightReads(t *testing.T) {
	known := map[string]bool{
		sheet.FlagRestOneCategoryUp: true, sheet.FlagRestNeverBelowLevel: true, sheet.FlagRestAlwaysNormal: true,
	}
	found := map[string]string{}
	check := func(owner string, mods []transcript.Modifier) {
		for _, m := range mods {
			if m.Target.Kind != "flag" || !strings.HasPrefix(m.Target.Name, "rest-") {
				continue
			}
			found[owner] = m.Target.Name
			if !known[m.Target.Name] {
				t.Errorf("%s levanta a flag %q, e a noite só lê %v — a habilidade nunca mudaria o descanso",
					owner, m.Target.Name, known)
			}
		}
	}
	for _, race := range transcript.RaceDefinitions {
		for _, a := range race.Abilities {
			check(a.ID, a.Modifiers)
		}
	}
	for _, o := range transcript.Origins {
		check(o.UniquePower.ID, o.UniquePower.Modifiers)
	}
	for _, p := range transcript.ClassPowers {
		check(p.ID, p.Modifiers)
	}
	for _, p := range transcript.GeneralPowers {
		check(p.ID, p.Modifiers)
	}
	const versesThatChangeTheNight = 6
	if len(found) != versesThatChangeTheNight {
		t.Errorf("%d verbetes mudam a noite (%v), e o livro dá %d: Rato das Ruas, Vida Rústica, "+
			"Estoico, Pajem, golem e osteon", len(found), found, versesThatChangeTheNight)
	}
}
