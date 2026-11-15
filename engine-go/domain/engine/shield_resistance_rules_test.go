package engine

import "testing"

// A SOLIDEZ LEVA O ESCUDO AOS TESTES DE RESISTÊNCIA (ALE-650).
//
//	"Se estiver usando um escudo, você soma o bônus na Defesa recebido pelo
//	 escudo em testes de resistência." (cavaleiro, p55; guerreiro, p66)
//
// "O bônus na Defesa recebido pelo escudo" é o do ITEM inteiro, e a conta o lê
// das parcelas da Defesa que vêm do escudo — ver `wieldedShieldDefense`.
func TestTheSolidityAddsTheShieldDefenseToTheSavingThrows(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		why    string
		class  string
		power  string
		shield string
		want   int
	}{
		{"o guerreiro com escudo pesado", "Guerreiro", "class.guerreiro.solidez", "escudo-pesado", 2},
		{"o guerreiro com escudo leve", "Guerreiro", "class.guerreiro.solidez", "escudo-leve", 1},
		{"o cavaleiro com escudo pesado", "Cavaleiro", "class.cavaleiro.solidez", "escudo-pesado", 2},
		{"o guerreiro sem escudo", "Guerreiro", "class.guerreiro.solidez", "", 0},
		{"o guerreiro com escudo, sem a Solidez", "Guerreiro", "", "escudo-pesado", 0},
	} {
		powers := `[]`
		if c.power != "" {
			powers = `["` + c.power + `"]`
		}
		ch := saving(c.class, powers)
		if c.shield != "" {
			ch = wielding(ch, c.shield)
		}
		for _, save := range []string{"Fortitude", "Reflexos", "Vontade"} {
			if got := expertiseFrom(t, world, ch, save, "Solidez"); got != c.want {
				t.Errorf("%s: a Solidez pôs %+d em %s, e o escudo dá %+d na Defesa", c.why, got, save, c.want)
			}
		}
	}
}

// O ESCUDO NA MOCHILA NÃO CONTA: "se estiver usando um escudo" é empunhá-lo.
func TestTheSolidityIgnoresAShieldInTheBackpack(t *testing.T) {
	world := BookRuleset(primeBook(t))
	id := "escudo-pesado"
	ch := saving("Guerreiro", `["class.guerreiro.solidez"]`)
	ch.Items = append(ch.Items, CharacterItem{CatalogID: &id, Name: id, Quantity: 1})
	if got := expertiseFrom(t, world, ch, "Fortitude", "Solidez"); got != 0 {
		t.Errorf("a Solidez pôs %+d em Fortitude com o escudo na mochila", got)
	}
}

// saving é o herói com as três perícias de resistência, que a ficha só desenha quando a pessoa as tem.
func saving(class string, powers string) Character {
	ch := attributeSumHero(class, 6, powers)
	ch.Expertises = []CharacterExpertise{
		{Name: "Fortitude", Attribute: "constitution"},
		{Name: "Reflexos", Attribute: "dexterity"},
		{Name: "Vontade", Attribute: "wisdom"},
	}
	return ch
}

// expertiseFrom é quanto a habilidade `source` pôs na perícia, pela linha com o nome dela.
func expertiseFrom(t *testing.T, world *Ruleset, ch Character, name, source string) int {
	t.Helper()
	for _, e := range world.ComputeSheet(ch, nil).Expertises {
		if e.Name != name {
			continue
		}
		total := 0
		for _, c := range e.ItemContributions {
			if c.Source == source {
				total += c.Amount
			}
		}
		return total
	}
	t.Fatalf("a ficha não tem a perícia %q", name)
	return 0
}
