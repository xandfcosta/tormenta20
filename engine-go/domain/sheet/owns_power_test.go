package sheet

import (
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// UM PODER ESCOLHIDO SÓ É DA FICHA COM O NÍVEL NA CLASSE DELE (p40). O Totem
// Espiritual pede bárbaro 4: escolhido por um bárbaro 3, ou por um ladino 5 que
// é bárbaro 1, ele não soma no motor — e a postura, o cumulativo e o Usar
// perguntam à mesma lista que a aba mostra, então ele não pode estar nela. O
// motor é perguntado junto porque é a discordância dos dois que este caso pegou:
// a ficha dava o poder, e as regras da mesa não.
func TestAChosenClassPowerAboveTheClassLevelIsNotOwned(t *testing.T) {
	const totem = "class.barbaro.totem-espiritual"
	cases := []struct {
		name    string
		classes []ClassDTO
		owns    bool
	}{
		{"bárbaro 4", []ClassDTO{{ClassName: "Bárbaro", Level: 4}}, true},
		{"bárbaro 3", []ClassDTO{{ClassName: "Bárbaro", Level: 3}}, false},
		{"ladino 5 e bárbaro 1", []ClassDTO{{ClassName: "Ladino", Level: 5}, {ClassName: "Bárbaro", Level: 1}}, false},
	}
	catalogs, err := book.EngineCatalogs()
	if err != nil {
		t.Fatalf("carregar os catálogos do motor: %v", err)
	}
	for _, c := range cases {
		dto := CharacterDTO{Classes: c.classes, ClassPowers: `["` + totem + `"]`}
		ec, err := EngineCharacterFrom(dto)
		if err != nil {
			t.Fatalf("%s: converter a ficha: %v", c.name, err)
		}
		if got := engine.BookRuleset(catalogs).OwnsPower(ec, totem); got != c.owns {
			t.Errorf("%s com o Totem Espiritual escolhido: o motor diz %v, e o livro diz %v", c.name, got, c.owns)
		}
	}
}
