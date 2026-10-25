package engine

import "testing"

// O ATAQUE DESARMADO É DE TODO MUNDO (p143, ALE-486).
//
// "Um ataque desarmado é considerado uma arma leve corpo a corpo que causa dano
// de impacto não letal (1d3 pontos de dano para criaturas Pequenas e Médias)",
// e "todas as criaturas são proficientes em ataques desarmados" (p142). Toda
// ficha ganha a carta dele, depois das armas empunhadas (decisão do dono).
//
// O LUTADOR troca o dado pela Briga (p75-77): 1d6 no 1º nível, 1d8 no 5º,
// 1d10 no 9º, 2d6 no 13º, 2d8 no 17º e 2d10 no 20º (Dono da Rua), "e podem
// causar dano letal ou não letal". Minúsculo diminui um passo (p76).

func unarmedCard(t *testing.T, world *Ruleset, ch Character) WeaponCard {
	t.Helper()
	for _, c := range world.ComputeWeaponCards(ch, nil) {
		if c.Name == "Ataque desarmado" {
			return c
		}
	}
	t.Fatalf("a ficha não tem a carta de ataque desarmado")
	return WeaponCard{}
}

func TestEveryoneHasAnUnarmedAttack(t *testing.T) {
	world := BookRuleset(primeBook(t))

	plain := unarmedCard(t, world, attributeSumHero("Guerreiro", 3, `[]`))
	if plain.Damage != "1d3" || !plain.NonLethal || plain.DamageType != "impacto" || plain.Skill != "Luta" {
		t.Errorf("desarmado de quem não é lutador: %s %s não letal=%v %s, want 1d3 impacto não letal de Luta",
			plain.Damage, plain.DamageType, plain.NonLethal, plain.Skill)
	}
	if plain.CritRange != 20 || plain.CritMult != 2 {
		t.Errorf("crítico do desarmado: %d/x%d, want 20/x2", plain.CritRange, plain.CritMult)
	}

	// As armas empunhadas vêm antes, e a carta desarmada não toma o lugar delas.
	armed := world.ComputeWeaponCards(wielding(attributeSumHero("Guerreiro", 3, `[]`), "espada-longa"), nil)
	if len(armed) != 2 || armed[0].Name != "espada-longa" || armed[1].Name != "Ataque desarmado" {
		t.Errorf("com espada longa: cartas %v, want [espada-longa, Ataque desarmado]", cardNames(armed))
	}

	// p76: Minúsculo diminui o dado em um passo — 1d3 vira 1d2.
	tiny := attributeSumHero("Guerreiro", 3, `[]`)
	tiny.Races = []CharacterRace{{Race: "silfide"}}
	if got := unarmedCard(t, world, tiny).Damage; got != "1d2" {
		t.Errorf("sílfide (Minúscula): desarmado %s, want 1d2", got)
	}

	// A Manopla levanta `lethal-unarmed`: o desarmado deixa de ser não letal.
	gauntlet := wearing(attributeSumHero("Guerreiro", 3, `[]`), "manopla")
	if unarmedCard(t, world, gauntlet).NonLethal {
		t.Errorf("de manopla, o desarmado ainda é não letal")
	}

	// "arma leve corpo a corpo": o Esgrimista soma nele (p48).
	fencer := attributeSumHero("Guerreiro", 4, `["class.guerreiro.esgrimista"]`)
	fencer.Intelligence = 3
	if got := unarmedCard(t, world, fencer).DamageOwn.Total; got != 3 {
		t.Errorf("Esgrimista, Int 3 no nível 4: o desarmado soma %d, want 3", got)
	}
}

func cardNames(cards []WeaponCard) []string {
	out := make([]string, len(cards))
	for i, c := range cards {
		out[i] = c.Name
	}
	return out
}

func TestTheFighterBrawlGrowsTheUnarmedDie(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, tt := range []struct {
		level int
		die   string
	}{{1, "1d6"}, {4, "1d6"}, {5, "1d8"}, {9, "1d10"}, {13, "2d6"}, {17, "2d8"}, {20, "2d10"}} {
		card := unarmedCard(t, world, attributeSumHero("Lutador", tt.level, `[]`))
		if card.Damage != tt.die {
			t.Errorf("Lutador %d: desarmado %s, want %s", tt.level, card.Damage, tt.die)
		}
		if card.NonLethal {
			t.Errorf("Lutador %d: o desarmado é não letal, e a Briga deixa escolher o letal", tt.level)
		}
	}

	// A Briga também desce um passo para Minúsculo: 1d6 vira 1d4.
	tiny := attributeSumHero("Lutador", 1, `[]`)
	tiny.Races = []CharacterRace{{Race: "silfide"}}
	if got := unarmedCard(t, world, tiny).Damage; got != "1d4" {
		t.Errorf("Lutador 1 sílfide: desarmado %s, want 1d4", got)
	}
}

func TestTheFighterUnarmedCriticalAndTrincadoStayOnTheUnarmedCard(t *testing.T) {
	world := BookRuleset(primeBook(t))

	// p77 "Golpe Cruel. No 5º nível […] Sua margem de ameaça com ataques
	// desarmados aumenta em +1." / "Golpe Violento. No 9º nível […] seu
	// multiplicador de crítico com ataques desarmados aumenta em +1."
	fighter := wielding(attributeSumHero("Lutador", 9, `[]`), "adaga")
	cards := world.ComputeWeaponCards(fighter, nil)
	unarmed := unarmedCard(t, world, fighter)
	if unarmed.CritRange != 19 || unarmed.CritMult != 3 {
		t.Errorf("Lutador 9 desarmado: %d/x%d, want 19/x3 (Golpe Cruel e Golpe Violento)", unarmed.CritRange, unarmed.CritMult)
	}
	if cards[0].Name != "adaga" || cards[0].CritRange != 19 || cards[0].CritMult != 2 {
		t.Errorf("a adaga do lutador: %s %d/x%d, want 19/x2 — o Golpe Cruel e o Violento são só do desarmado",
			cards[0].Name, cards[0].CritRange, cards[0].CritMult)
	}

	// p77 "Trincado. […] Você soma sua Constituição nas rolagens de dano
	// desarmado." — sem teto: o verbete não o limita.
	trincado := attributeSumHero("Lutador", 10, `["class.lutador.trincado"]`)
	trincado.Constitution = 4
	if got := unarmedCard(t, world, trincado).DamageOwn.Total; got != 4 {
		t.Errorf("Trincado, Con 4: o desarmado soma %d, want 4", got)
	}
}

// wieldedCards são só as cartas das armas EMPUNHADAS, sem a do desarmado — para
// os casos que medem uma arma e contavam as cartas antes de ela existir.
func wieldedCards(cards []WeaponCard) []WeaponCard {
	out := []WeaponCard{}
	for _, c := range cards {
		if c.Name != UnarmedName {
			out = append(out, c)
		}
	}
	return out
}
