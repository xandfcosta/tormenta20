package sheet

import (
	"slices"
	"testing"
)

// AS CLASSES DA FICHA QUE LANÇAM A MAGIA são a pergunta que o tipo da magia (o corte de custo "só
// arcanas"), o limite de PM por magia (p224) e a saída de uma classe fazem. O Bardo 5/Arcanista 1 de
// evocação lança a Explosão de Chamas pelas duas classes, embora o verbete liste só o Arcanista (p44); sem
// evocação, só pelo Arcanista.
func TestTheCastingClassesFollowTheBardsSchools(t *testing.T) {
	multiclass := func(schools string) CharacterDTO {
		return CharacterDTO{
			Classes:      []ClassDTO{{ClassName: "Bardo", Level: 5}, {ClassName: "Arcanista", Level: 1}},
			PowerChoices: `{"class.bardo.magias-1-circulo":` + schools + `}`,
		}
	}
	for _, c := range []struct {
		name, schools string
		want          []string
	}{
		{"com evocação", `["evocacao"]`, []string{"Bardo", "Arcanista"}},
		{"sem evocação", `["ilusao"]`, []string{"Arcanista"}},
	} {
		dto := multiclass(c.schools)
		got := CastingClasses(dto, dto.Classes, "evocacao", []string{"Arcanista"})
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: lançam a Explosão de Chamas %v, e o livro diz %v", c.name, got, c.want)
		}
		if tradition := SpellTradition(dto.Classes, got); tradition != "arcana" {
			t.Errorf("%s: a tradição saiu %q", c.name, tradition)
		}
	}
}
