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
