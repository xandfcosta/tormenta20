package engine

import (
	"slices"
	"testing"
)

// OS PODERES GERAIS COM CONTA somam por interruptor, e só na carta da arma que o livro nomeia (p124-128).
// O caso compara a mesma ficha com o interruptor desligado e ligado, com os números escritos à mão.

func switchHero(powers string, dex int, weapon string) Character {
	return Character{
		Level: 5, Dexterity: dex, ClassPowers: powers,
		Classes: []CharacterClass{{ClassName: "Guerreiro", Level: 5}},
		Items:   []CharacterItem{{CatalogID: &weapon, Name: weapon, Equipped: ptrTo("wielded"), Improvements: "[]"}},
	}
}

func ptrTo(s string) *string { return &s }

func TestTheGeneralPowerSwitchesLandOnTheWeaponsTheBookNames(t *testing.T) {
	world := BookRuleset(primeBook(t))
	for _, c := range []struct {
		name, power, flag, weapon        string
		dex                              int
		attack, damage, threat, multiple int
		defense                          int
	}{
		{name: "Ataque Poderoso na espada (p124)", power: "ataque-poderoso", flag: "ataque-poderoso", weapon: "espada-longa", attack: -2, damage: 5},
		{name: "Ataque Poderoso não vale no arco", power: "ataque-poderoso", flag: "ataque-poderoso", weapon: "arco-curto"},
		{name: "Combate Defensivo (p125)", power: "combate-defensivo", flag: "combate-defensivo", weapon: "espada-longa", attack: -2, defense: 5},
		{name: "Mira Apurada no arco (p128)", power: "mira-apurada", flag: "mira-apurada", weapon: "arco-curto", attack: 2, threat: 2},
		{name: "Mira Apurada não vale na espada", power: "mira-apurada", flag: "mira-apurada", weapon: "espada-longa"},
		{name: "Disparo Rápido (p125)", power: "disparo-rapido", flag: "disparo-rapido", weapon: "arco-curto", attack: -2},
		{name: "Estilo de Duas Armas (p125)", power: "estilo-de-duas-armas", flag: "estilo-de-duas-armas", weapon: "espada-longa", attack: -2},
		{name: "Estilo de Arma e Escudo (p125)", power: "estilo-de-arma-e-escudo", flag: "estilo-de-arma-e-escudo", weapon: "espada-longa", defense: 2},
		{name: "Estilo de Duas Mãos no montante (p128)", power: "estilo-de-duas-maos", flag: "estilo-de-duas-maos", weapon: "montante", damage: 5},
		{name: "Estilo de Duas Mãos não vale na espada", power: "estilo-de-duas-maos", flag: "estilo-de-duas-maos", weapon: "espada-longa"},
		{name: "Estilo de Disparo soma a Destreza (p125)", power: "estilo-de-disparo", flag: "estilo-de-disparo", weapon: "arco-curto", dex: 3, damage: 3},
		{name: "Estilo de Disparo não vale em arremesso", power: "estilo-de-disparo", flag: "estilo-de-disparo", weapon: "azagaia", dex: 3},
		{name: "Estilo de Arremesso (p125)", power: "estilo-de-arremesso", flag: "estilo-de-arremesso", weapon: "azagaia", damage: 2},
		{name: "Estilo de Arremesso com Saque Rápido", power: "estilo-de-arremesso\",\"saque-rapido", flag: "estilo-de-arremesso", weapon: "azagaia", attack: 2, damage: 2},
		{name: "Estilo de Uma Arma (p128)", power: "estilo-de-uma-arma", flag: "estilo-de-uma-arma", weapon: "espada-longa", attack: 2, defense: 2},
		{name: "Estilo de Uma Arma não vale em duas mãos", power: "estilo-de-uma-arma", flag: "estilo-de-uma-arma", weapon: "montante"},
		{name: "Ataque Preciso (p124)", power: "ataque-preciso", flag: "ataque-preciso", weapon: "espada-longa", threat: 2, multiple: 1},
		{name: "Ataque Preciso não vale em duas mãos", power: "ataque-preciso", flag: "ataque-preciso", weapon: "montante"},
	} {
		t.Run(c.name, func(t *testing.T) {
			hero := switchHero(`["`+c.power+`"]`, c.dex, c.weapon)
			on := map[string]bool{FlagGroupID(c.flag): true}
			offCards, onCards := world.ComputeWeaponCards(hero, nil), world.ComputeWeaponCards(hero, on)
			if len(offCards) == 0 || offCards[0].CatalogID != c.weapon {
				t.Fatalf("a primeira carta não é %s: %+v", c.weapon, offCards)
			}
			off, lit := offCards[0], onCards[0]
			if got := lit.Attack - off.Attack; got != c.attack {
				t.Errorf("o ataque mudou %+d, e o livro dá %+d", got, c.attack)
			}
			if got := lit.DamageBonus - off.DamageBonus; got != c.damage {
				t.Errorf("o dano mudou %+d, e o livro dá %+d", got, c.damage)
			}
			if got := off.CritRange - lit.CritRange; got != c.threat {
				t.Errorf("a margem cresceu %+d, e o livro dá %+d", got, c.threat)
			}
			if got := lit.CritMult - off.CritMult; got != c.multiple {
				t.Errorf("o multiplicador cresceu %+d, e o livro dá %+d", got, c.multiple)
			}
			defenseOff := world.ComputeSheet(hero, nil).Defense.Total
			defenseOn := world.ComputeSheet(hero, on).Defense.Total
			if got := defenseOn - defenseOff; got != c.defense {
				t.Errorf("a Defesa mudou %+d, e o livro dá %+d", got, c.defense)
			}
		})
	}
}

// O desarmado é corpo a corpo (p143), então o Ataque Poderoso vale nele; e o Estilo de Uma Arma o exclui
// com todas as letras (p128).
func TestTheUnarmedCardReadsTheSwitchesAsTheBookSays(t *testing.T) {
	world := BookRuleset(primeBook(t))
	unarmed := func(hero Character, flag string) (int, int) {
		cards := world.ComputeWeaponCards(hero, map[string]bool{FlagGroupID(flag): true})
		base := world.ComputeWeaponCards(hero, nil)
		last, lastBase := cards[len(cards)-1], base[len(base)-1]
		return last.Attack - lastBase.Attack, last.DamageBonus - lastBase.DamageBonus
	}
	if attack, damage := unarmed(switchHero(`["ataque-poderoso"]`, 0, "espada-longa"), "ataque-poderoso"); attack != -2 || damage != 5 {
		t.Errorf("Ataque Poderoso no desarmado: %+d e %+d", attack, damage)
	}
	if attack, _ := unarmed(switchHero(`["estilo-de-uma-arma"]`, 0, "espada-longa"), "estilo-de-uma-arma"); attack != 0 {
		t.Errorf("Estilo de Uma Arma no desarmado: %+d", attack)
	}
}

// Com Ambidestria, o Estilo de Duas Armas não tem penalidade (p125), e não há o que ligar.
func TestAmbidextrousHasNoPenaltyWithTwoWeaponStyle(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := switchHero(`["estilo-de-duas-armas","class.guerreiro.ambidestria"]`, 0, "espada-longa")
	on := map[string]bool{FlagGroupID("estilo-de-duas-armas"): true}
	if got := world.ComputeWeaponCards(hero, on)[0].Attack - world.ComputeWeaponCards(hero, nil)[0].Attack; got != 0 {
		t.Errorf("com Ambidestria, o ataque mudou %+d", got)
	}
}

// A carga soma 2d8 ao dano (p124), e dado não entra no bônus numérico.
func TestCavalryChargeAddsTheDice(t *testing.T) {
	world := BookRuleset(primeBook(t))
	hero := switchHero(`["carga-de-cavalaria"]`, 0, "espada-longa")
	cards := world.ComputeWeaponCards(hero, map[string]bool{FlagGroupID("carga-de-cavalaria"): true})
	if !slices.ContainsFunc(cards[0].ExtraDamage, func(e ExtraDamage) bool { return e.Dice == "2d8" }) {
		t.Errorf("a carga não somou 2d8: %+v", cards[0].ExtraDamage)
	}
}

// A duração se pergunta ao catálogo: o que desliga quando a vez passa, e o que desliga quando ela volta.
func TestTheSwitchesDeclareWhenTheyEnd(t *testing.T) {
	catalogs := primeBook(t)
	turn, ownTurn := catalogs.SwitchesEndingAt(SwitchEndsTurn), catalogs.SwitchesEndingAt(SwitchEndsOwnTurn)
	slices.Sort(turn)
	slices.Sort(ownTurn)
	if want := []string{"flag:ataque-poderoso", "flag:carga-de-cavalaria", "flag:mira-apurada"}; !slices.Equal(turn, want) {
		t.Errorf("desligam ao passar a vez: %v, e devia ser %v", turn, want)
	}
	if want := []string{"flag:combate-defensivo", "flag:disparo-rapido", "flag:estilo-de-duas-armas"}; !slices.Equal(ownTurn, want) {
		t.Errorf("desligam quando a vez volta: %v, e devia ser %v", ownTurn, want)
	}
}

// "Se estiver usando uma arma corpo a corpo em uma das mãos e nada na outra" (p128): o escudo e a
// segunda arma ocupam a outra, e o interruptor ligado não soma nada.
func TestTheOneWeaponStyleAsksForAnEmptyHand(t *testing.T) {
	world := BookRuleset(primeBook(t))
	on := map[string]bool{FlagGroupID("estilo-de-uma-arma"): true}
	sword := func(other string) (attack, defense int) {
		hero := switchHero(`["estilo-de-uma-arma"]`, 0, "espada-longa")
		if other != "" {
			hero.Items = append(hero.Items, CharacterItem{CatalogID: &other, Name: other, Equipped: ptrTo("wielded"), Improvements: "[]"})
		}
		return world.ComputeWeaponCards(hero, on)[0].Attack - world.ComputeWeaponCards(hero, nil)[0].Attack,
			world.ComputeSheet(hero, on).Defense.Total - world.ComputeSheet(hero, nil).Defense.Total
	}
	if attack, defense := sword(""); attack != 2 || defense != 2 {
		t.Fatalf("o controle falhou: com a outra mão livre, ataque %+d e Defesa %+d", attack, defense)
	}
	for _, other := range []string{"escudo-leve", "adaga"} {
		if attack, defense := sword(other); attack != 0 || defense != 0 {
			t.Errorf("com %s na outra mão: ataque %+d e Defesa %+d, e o livro pede a mão vazia", other, attack, defense)
		}
	}
}
