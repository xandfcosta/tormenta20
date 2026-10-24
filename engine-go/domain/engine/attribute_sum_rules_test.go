package engine

import (
	"encoding/json"
	"testing"

	"t20engine/domain/catalog/transcript"
)

// AS HABILIDADES QUE SOMAM UM ATRIBUTO (cap. 1, ALE-485).
//
// "Você soma seu X em Y" é uma forma só no livro, e o teto é opcional: a p226
// diz que "ALGUMAS habilidades são limitadas pelo seu nível. Para classes, use
// seu nível naquela classe" — vale onde o verbete escreve "limitado pelo seu
// nível", e só ali. Cada caso traz o texto da página e o número escrito à mão.
//
// O atributo NEGATIVO subtrai (decisão do dono na ALE-485): o livro não tem
// regra própria para isso, e o uso dele em todo lugar é somar o valor como é —
// "Sua Defesa é 10 + sua Destreza" (p106) com a Destreza negativa.

func attributeSumHero(class string, level int, powers string) Character {
	return Character{
		Level: level, Classes: []CharacterClass{{ClassName: class, Level: level}},
		ClassPowers: powers, PowerChoices: `{}`, ActiveConditions: `[]`,
	}
}

func wielding(ch Character, ids ...string) Character {
	for i, id := range ids {
		slot := "wielded"
		if i > 0 {
			slot = "wielded2"
		}
		ch.Items = append(ch.Items, CharacterItem{CatalogID: &id, Name: id, Quantity: 1, Equipped: &slot})
	}
	return ch
}

func wearing(ch Character, id string) Character {
	vested := "vested"
	ch.Items = append(ch.Items, CharacterItem{CatalogID: &id, Name: id, Quantity: 1, Equipped: &vested})
	return ch
}

// defenseFrom é quanto a habilidade `source` pôs na Defesa — pela linha com o
// nome dela, que é o que a tela mostra. `Defesa − 10` contaria também o que a
// classe ganha por nível (o bucaneiro de 3º nível já tem outra parcela).
func defenseFrom(t *testing.T, world *Ruleset, ch Character, source string) int {
	t.Helper()
	total := 0
	for _, c := range world.ComputeSheet(ch, nil).Defense.Contributions {
		if c.Source == source {
			total += c.Amount
		}
	}
	return total
}

func expertiseTotal(t *testing.T, world *Ruleset, ch Character, name string) int {
	t.Helper()
	for _, e := range world.ComputeSheet(ch, nil).Expertises {
		if e.Name == name {
			return e.Total
		}
	}
	t.Fatalf("a ficha não tem a perícia %q", name)
	return 0
}

func damageOwnOf(t *testing.T, world *Ruleset, ch Character, weapon string) int {
	t.Helper()
	for _, c := range world.ComputeWeaponCards(ch, nil) {
		if c.Name == weapon {
			return c.DamageOwn.Total
		}
	}
	t.Fatalf("nenhuma carta de arma para %q", weapon)
	return 0
}

func TestTheInsolenceSumsCharismaCappedByTheBuccaneerLevel(t *testing.T) {
	world := BookRuleset(primeBook(t))
	buccaneer := func(level, charisma int) Character {
		ch := attributeSumHero("Bucaneiro", level, `[]`)
		ch.Charisma = charisma
		return ch
	}
	// p226: "um bucaneiro de 2º nível com Car 3 soma +2 na Defesa. Quando subir
	// para o 3º nível, passará a somar +3."
	if got := defenseFrom(t, world, buccaneer(2, 3), "Insolência"); got != 2 {
		t.Errorf("nível 2, Car 3: a Insolência soma %d, want 2 (o exemplo da p226)", got)
	}
	if got := defenseFrom(t, world, buccaneer(3, 3), "Insolência"); got != 3 {
		t.Errorf("nível 3, Car 3: a Insolência soma %d, want 3", got)
	}
	if got := defenseFrom(t, world, buccaneer(10, 3), "Insolência"); got != 3 {
		t.Errorf("nível 10, Car 3: soma %d, want 3 — o teto é o menor dos dois", got)
	}
	// O negativo subtrai: Car −2 tira 2.
	if got := defenseFrom(t, world, buccaneer(5, -2), "Insolência"); got != -2 {
		t.Errorf("Car −2: a Insolência soma %d, want −2", got)
	}
	// p47 "você não pode usá-la se estiver de armadura pesada ou na condição imóvel"
	heavy := wearing(buccaneer(5, 3), "armadura-completa")
	if got := defenseFrom(t, world, heavy, "Insolência"); got != 0 {
		t.Errorf("de armadura completa a Insolência ainda soma %d na Defesa", got)
	}
	still := buccaneer(5, 3)
	still.ActiveConditions = `["imovel"]`
	if got := defenseFrom(t, world, still, "Insolência"); got != 0 {
		t.Errorf("imóvel, a Insolência ainda soma %d na Defesa", got)
	}
	// p226 "Para classes, use seu nível naquela classe."
	multi := buccaneer(10, 5)
	multi.Classes = []CharacterClass{{ClassName: "Guerreiro", Level: 8}, {ClassName: "Bucaneiro", Level: 2}}
	if got := defenseFrom(t, world, multi, "Insolência"); got != 2 {
		t.Errorf("Guerreiro 8 / Bucaneiro 2 com Car 5: soma %d, want 2 — o teto é o nível NA CLASSE", got)
	}
}

func TestTheFighterSumsConstitutionOrStrengthInTheDefense(t *testing.T) {
	world := BookRuleset(primeBook(t))

	// p77 "No 3º nível, você soma sua Constituição na Defesa, limitado pelo seu
	// nível e apenas se não estiver usando armadura pesada."
	tough := attributeSumHero("Lutador", 3, `[]`)
	tough.Constitution = 5
	if got := defenseFrom(t, world, tough, "Casca Grossa (Con)"); got != 3 {
		t.Errorf("Lutador 3, Con 5: a Casca Grossa soma %d, want 3 (teto no nível)", got)
	}
	if got := defenseFrom(t, world, wearing(tough, "armadura-couro"), "Casca Grossa (Con)"); got != 3 {
		t.Errorf("de couro (leve): a Casca Grossa soma %d, want 3 — só a pesada desliga", got)
	}
	if got := defenseFrom(t, world, wearing(tough, "armadura-completa"), "Casca Grossa (Con)"); got != 0 {
		t.Errorf("de armadura completa a Casca Grossa ainda soma %d", got)
	}

	// p76 "Se você não estiver usando armadura, soma sua Força na Defesa,
	// limitado pelo seu nível."
	calloused := attributeSumHero("Lutador", 2, `["class.lutador.bracos-calejados"]`)
	calloused.Strength = 4
	if got := defenseFrom(t, world, calloused, "Braços Calejados"); got != 2 {
		t.Errorf("Lutador 2, For 4: Braços Calejados soma %d, want 2", got)
	}
	if got := defenseFrom(t, world, wearing(calloused, "armadura-couro"), "Braços Calejados"); got != 0 {
		t.Errorf("de armadura LEVE os Braços Calejados ainda somam %d", got)
	}
}

func TestTheUncappedSumsAddTheWholeAttribute(t *testing.T) {
	world := BookRuleset(primeBook(t))

	// p48 "Você soma seu Carisma em Iniciativa" — sem teto: Car 5 no 4º nível.
	presence := attributeSumHero("Bucaneiro", 4, `["class.bucaneiro.presenca-paralisante"]`)
	presence.Charisma = 5
	presence.Dexterity = 0
	presence.Expertises = []CharacterExpertise{{Name: "Iniciativa", Attribute: "dexterity"}}
	without := presence
	without.ClassPowers = `[]`
	if got := expertiseTotal(t, world, presence, "Iniciativa") - expertiseTotal(t, world, without, "Iniciativa"); got != 5 {
		t.Errorf("Presença Paralisante, Car 5 no nível 4: soma %d na Iniciativa, want 5 (sem teto)", got)
	}

	// p74 "Você soma sua Inteligência em Ladinagem e Furtividade."
	mind := attributeSumHero("Ladino", 2, `["class.ladino.mente-criminosa"]`)
	mind.Intelligence = 4
	mind.Expertises = []CharacterExpertise{
		{Name: "Ladinagem", Attribute: "dexterity"}, {Name: "Furtividade", Attribute: "dexterity"},
	}
	plain := mind
	plain.ClassPowers = `[]`
	for _, name := range []string{"Ladinagem", "Furtividade"} {
		if got := expertiseTotal(t, world, mind, name) - expertiseTotal(t, world, plain, name); got != 4 {
			t.Errorf("Mente Criminosa, Int 4 no nível 2: soma %d em %s, want 4 (sem teto)", got, name)
		}
	}
}

func TestTheWeaponSumsLandOnlyOnTheWeaponsTheBookNames(t *testing.T) {
	world := BookRuleset(primeBook(t))

	// p48 "Quando usa uma arma corpo a corpo leve ou ágil, você soma sua
	// Inteligência nas rolagens de dano (limitado pelo seu nível)."
	fencer := attributeSumHero("Guerreiro", 2, `["class.guerreiro.esgrimista"]`)
	fencer.Intelligence = 4
	fencer = wielding(fencer, "adaga", "espada-longa")
	if got := damageOwnOf(t, world, fencer, "adaga"); got != 2 {
		t.Errorf("Esgrimista na adaga (leve), Int 4 no nível 2: soma %d, want 2", got)
	}
	if got := damageOwnOf(t, world, fencer, "espada-longa"); got != 0 {
		t.Errorf("Esgrimista na espada longa (nem leve nem ágil): soma %d, want 0", got)
	}

	// p50 "Se estiver usando uma arma de ataque à distância, você soma sua
	// Sabedoria nas rolagens de dano (limitado pelo seu nível)." — e ataque à
	// distância é arremesso E disparo (p142).
	archer := attributeSumHero("Caçador", 5, `["class.cacador.arqueiro"]`)
	archer.Wisdom = 3
	archer = wielding(archer, "arco-curto", "machadinha")
	if got := damageOwnOf(t, world, archer, "arco-curto"); got != 3 {
		t.Errorf("Arqueiro no arco curto (disparo), Sab 3: soma %d, want 3", got)
	}
	if got := damageOwnOf(t, world, archer, "machadinha"); got != 3 {
		t.Errorf("Arqueiro na machadinha (arremesso), Sab 3: soma %d, want 3", got)
	}
	melee := wielding(attributeSumHero("Caçador", 5, `["class.cacador.arqueiro"]`), "espada-longa")
	melee.Wisdom = 3
	if got := damageOwnOf(t, world, melee, "espada-longa"); got != 0 {
		t.Errorf("Arqueiro na espada longa (corpo a corpo): soma %d, want 0", got)
	}
}

// TestNoAttributeSumTargetsAnAttribute prende a premissa da segunda passada do
// `sheetEffects`: os atributos que ela lê são os da primeira, e uma soma que
// mirasse um atributo os mudaria depois de lidos. O `isAttributeSum` a ignora
// — e por isso ela tem de reprovar aqui, com o nome, e não valer zero em
// silêncio na ficha.
func TestNoAttributeSumTargetsAnAttribute(t *testing.T) {
	measured := 0
	for _, resource := range transcript.Resources {
		var tree any
		if err := json.Unmarshal(bookResource(t, resource), &tree); err != nil {
			t.Fatalf("%s: %v", resource, err)
		}
		for _, m := range attributeSumsIn(tree) {
			measured++
			if k, _ := m["target"].(map[string]any)["k"].(string); k == "attribute" {
				t.Errorf("%s: uma soma de atributo mira o atributo %v — a ficha leria o "+
					"atributo antes de a soma existir", resource, m["target"])
			}
		}
	}
	// O DENOMINADOR: são as nove da ALE-485 (a Mente Criminosa soma em duas
	// perícias); menos que isso é a varredura que parou de achar.
	if measured < 9 {
		t.Fatalf("só %d somas de atributo no catálogo — a varredura parou de achar", measured)
	}
}

// attributeSumsIn são os modificadores com `scale.per = attribute` fora dos
// vitais, em qualquer profundidade.
func attributeSumsIn(node any) []map[string]any {
	var out []map[string]any
	switch v := node.(type) {
	case map[string]any:
		if scale, ok := v["scale"].(map[string]any); ok && scale["per"] == "attribute" {
			if target, ok := v["target"].(map[string]any); ok && target["k"] != "maxPv" && target["k"] != "maxPm" {
				out = append(out, v)
			}
		}
		for _, child := range v {
			out = append(out, attributeSumsIn(child)...)
		}
	case []any:
		for _, child := range v {
			out = append(out, attributeSumsIn(child)...)
		}
	}
	return out
}
