package master

import (
	"regexp"
	"strings"
	"testing"

	"t20engine/serve/web/bookui"
)

// O guarda dos DESCRITORES: cada catálogo responde nos três gestos da coleção.
//
// A lista garante que nenhum catálogo fique de fora de um gesto; o que ela não
// garante é a DECLARAÇÃO certa. Um `ID` que devolve o nome quebra o elo, e
// `Fields` que não varrem o nome tiram o catálogo da busca unificada — e nos dois
// casos a aba continua desenhando a lista inteira. Por AMOSTRAGEM sobre a lista,
// para o catálogo que entrar amanhã nascer medido.
func TestEveryCatalogAnswersInTheThreeGestures(t *testing.T) {
	for _, catalog := range collectionCatalogs {
		aba := catalog.Stop.Slug
		whole := catalog.group(nil)
		if whole.Count() == 0 {
			t.Errorf("o catálogo %q não tem verbete nenhum", aba)
			continue
		}
		sample := whole.Entries[whole.Count()-1]

		// O ID é a CHAVE com que os catálogos se citam ("upgradesTo": "apavorado"),
		// e não o nome: um elo de outro cartão pede a chave, e um descritor que
		// devolvesse o nome acharia a si mesmo e a mais ninguém.
		for _, e := range whole.Entries {
			if !entryKey.MatchString(e.ID) {
				t.Errorf("o verbete %q da aba %q tem o id %q, que não é chave", e.Name, aba, e.ID)
				break
			}
		}

		// A ABA mostra o catálogo dela, e não o da queda.
		tab := loadCollection(collectionCriteria{Aba: aba}, bookui.BookAddress{})
		if tab.Groups[0].Label != catalog.Stop.Label {
			t.Errorf("a aba %q desenhou o grupo %q", aba, tab.Groups[0].Label)
		}

		// O ELO acha o verbete pelo id, e só ele.
		entry := groupForEntry(aba, sample.ID)
		if entry.Count() != 1 || entry.entryName() != sample.Name {
			t.Errorf("o elo para %q na aba %q achou %d verbetes (%q)", sample.ID, aba, entry.Count(), entry.entryName())
		}

		// A BUSCA unificada acha o verbete pelo nome, no grupo do catálogo.
		found := false
		for _, g := range loadCollection(collectionCriteria{Term: sample.Name}, bookui.BookAddress{}).Groups {
			if g.Label != catalog.Stop.Label {
				continue
			}
			for _, e := range g.Entries {
				found = found || e.ID == sample.ID
			}
		}
		if !found {
			t.Errorf("a busca por %q não achou o verbete no grupo %q", sample.Name, catalog.Stop.Label)
		}
	}
}

// entryKey é a forma das chaves do livro transcrito: minúscula, sem espaço e sem
// acento, com o ponto dos poderes ("class.ladino.…").
var entryKey = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)

// O TRILHO é alfabético pelo nome sem acento, com o bestiário na frente. Pelo
// byte, "Perigos" viria antes de "Perícias".
func TestTheCatalogRailIsAlphabeticalWithTheBestiaryFirst(t *testing.T) {
	stops := catalogStops()
	if stops[0].Slug != "bestiario" {
		t.Fatalf("o trilho dos catálogos começa em %q", stops[0].Slug)
	}
	var labels []string
	for _, s := range stops[1:] {
		labels = append(labels, s.Label)
	}
	written := strings.Join(labels, ", ")
	if !strings.Contains(written, "Perícias, Perigos") {
		t.Errorf("o trilho não está em ordem alfabética: %s", written)
	}
}
