package sheet

import "testing"

// O ENCANTO ATIVÁVEL É A SEXTA PROCEDÊNCIA DE PODER (ALE-433).
//
// Um encanto que diz "gaste uma ação de movimento e 1 PM para fazer a arma
// flutuar e atacar" (Dançarina, p335) é uma coisa que se FAZ. O lugar onde se
// procura o que fazer é o acervo de poderes — é ele que a aba Poderes lista e
// que a superfície Ações reagrupa —, e era justamente ali que o encanto não
// existia: ele aparecia só como texto no cartão da Mochila.
//
// SÓ O ATIVÁVEL, e só no item EQUIPADO. As duas metades são regra:
//
//   - encanto passivo já está no número da arma e no cartão da Mochila, e
//     listá-lo aqui seria a mesma coisa duas vezes;
//   - espada guardada na mochila não dança — o gesto pede o item na mão.
func TestTheActivatableEnchantEntersTheOwnedPowers(t *testing.T) {
	wielded := "wielded"
	dancarina := "espada-longa"
	comEspada := func(equipped *string, enchants ...string) CharacterDTO {
		return CharacterDTO{Items: []ItemDTO{{
			Name: "Espada longa", CatalogID: &dancarina, Equipped: equipped, Enchants: enchants,
		}}}
	}

	owned := OwnedPowers(comEspada(&wielded, "encanto-dancarina"))
	var found *OwnedPower
	for i := range owned {
		if owned[i].ID == "encanto-dancarina" {
			found = &owned[i]
		}
	}
	if found == nil {
		t.Fatalf("a Dançarina não entrou no acervo: a aba Poderes não ofereceria o "+
			"gesto dela, e o servidor recusaria quem o pedisse. Veio %+v", owned)
	}
	if found.Source != PowerFromItem {
		t.Errorf("a procedência veio %q, e o encanto vem de ITEM", found.Source)
	}
	if found.Owner != "Espada longa" {
		t.Errorf("o dono veio %q, e é a arma que carrega o encanto — sem o nome "+
			"dela, duas espadas encantadas viram duas linhas iguais", found.Owner)
	}

	// O PASSIVO NÃO ENTRA: a Flamejante soma +1d6 de fogo sozinha, e ela já
	// aparece no cartão da arma.
	if temPassivo(OwnedPowers(comEspada(&wielded, "encanto-flamejante")), "encanto-flamejante") {
		t.Error("a Flamejante entrou no acervo de poderes, e ela não tem gesto nenhum")
	}

	// E A GUARDADA NÃO DANÇA: sem empunhar, não há o que ativar.
	if temPassivo(OwnedPowers(comEspada(nil, "encanto-dancarina")), "encanto-dancarina") {
		t.Error("a espada guardada ofereceu o gesto do encanto")
	}
}

func temPassivo(owned []OwnedPower, id string) bool {
	for _, p := range owned {
		if p.ID == id {
			return true
		}
	}
	return false
}

// E O ITEM COMPOSTO TEM GESTO PRÓPRIO (p337, ALE-445).
//
// A Avalanche não é um encanto ativável: é o VERBETE que diz "gaste uma ação
// padrão e 6 PM para invocar uma tempestade de gelo". O gesto é do item, e sem
// ele a aba Poderes não ofereceria nada — a tempestade existiria só na prosa do
// cartão da Mochila, que é exatamente onde o encanto ativável estava antes da
// ALE-433.
func TestTheComposedItemOffersItsOwnGesture(t *testing.T) {
	wielded := "wielded"
	avalanche := "avalanche"
	owned := OwnedPowers(CharacterDTO{Items: []ItemDTO{{
		Name: "Avalanche", CatalogID: &avalanche, Equipped: &wielded,
	}}})

	var found *OwnedPower
	for i := range owned {
		if owned[i].Name == "Tempestade de gelo" {
			found = &owned[i]
		}
	}
	if found == nil {
		t.Fatalf("a tempestade da Avalanche não entrou no acervo. Veio %+v", owned)
	}
	if found.Source != PowerFromItem || found.Owner != "Avalanche" {
		t.Errorf("o gesto veio de %q com dono %q, e ele é do ITEM que a pessoa "+
			"empunha", found.Source, found.Owner)
	}

	// E A GUARDADA NÃO INVOCA: o gesto pede o machado na mão.
	guardada := OwnedPowers(CharacterDTO{Items: []ItemDTO{{
		Name: "Avalanche", CatalogID: &avalanche,
	}}})
	for _, p := range guardada {
		if p.Name == "Tempestade de gelo" {
			t.Error("o machado guardado ofereceu a tempestade")
		}
	}
}
