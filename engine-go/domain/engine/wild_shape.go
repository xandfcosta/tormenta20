package engine

// A FORMA SELVAGEM do druida (p62-63): "quando usa esta habilidade você adquire as características de uma
// das formas abaixo. […] Características não mencionadas não mudam."
//
// A forma é uma FONTE DA FICHA, como as mãos e as patas: atributo, Defesa, redução de dano, tamanho e
// deslocamento entram pelo caminho de todo modificador. O corpo é que muda inteiro: as armas de mão não
// atacam ("não pode empunhar itens"), e as armas naturais são só as da forma — a forma animal não tem os
// chifres da raça original (decisão do dono).

// WildForm é uma das cinco formas do quadro da p63.
type WildForm string

const (
	WildAgile     WildForm = "agil"
	WildFerocious WildForm = "feroz"
	WildResilient WildForm = "resistente"
	WildSneaky    WildForm = "sorrateira"
	WildSwift     WildForm = "veloz"
)

// WildForms são as formas na ordem do livro, que é a da tela.
var WildForms = []WildForm{WildAgile, WildFerocious, WildResilient, WildSneaky, WildSwift}

// WildTier é a forma normal, a aprimorada ou a superior, cada uma com o seu poder e o seu preço (p62).
type WildTier string

const (
	WildTierNormal   WildTier = "normal"
	WildTierImproved WildTier = "aprimorada"
	WildTierSuperior WildTier = "superior"
)

// WildTiers são os níveis na ordem do livro.
var WildTiers = []WildTier{WildTierNormal, WildTierImproved, WildTierSuperior}

// WildMovement é o deslocamento que a Forma Veloz escolhe.
type WildMovement string

const (
	WildMoveLand  WildMovement = "deslocamento"
	WildMoveClimb WildMovement = "escalada"
	WildMoveSwim  WildMovement = "natacao"
	WildMoveFly   WildMovement = "voo"
)

// WildShape é a forma que o druida assumiu, com o que ele escolheu ao se transformar.
type WildShape struct {
	Form WildForm `json:"form"`
	Tier WildTier `json:"tier"`
	// DamageType é o da arma natural: "escolha seu tipo de dano entre corte, impacto e perfuração".
	DamageType string `json:"damageType"`
	// Movement é a escolha da Forma Veloz, vazia nas outras.
	Movement WildMovement `json:"movement,omitempty"`
}

// WildShapeDamageTypes são os três tipos que a arma da forma pode ter.
var WildShapeDamageTypes = []string{"corte", "impacto", "perfuracao"}

// WildShapeStats é uma linha do quadro: o que a forma, naquele nível, dá.
type WildShapeStats struct {
	Strength, Dexterity, Defense, Reduction int
	// Damage é o da arma natural, e Weapons, quantas (a Ágil tem duas). CritRange zero é o 20.
	Damage    string
	Weapons   int
	CritRange int
	// Size é o tamanho da forma, na forma normalizada (`sizesOnTheSheet`); vazio mantém o da raça.
	Size string
	// PlusMetres soma ao deslocamento (a Ágil); FlyMetres é o voo que a forma dá sem escolha.
	PlusMetres, FlyMetres int
	// Moves são as escolhas da Forma Veloz, em metros.
	Moves map[WildMovement]int
}

// wildShapes é o quadro da p63, forma a forma. "Características não mencionadas não mudam": a Sorrateira
// aprimorada não diz arma, e fica com a da normal.
var wildShapes = map[WildForm]map[WildTier]WildShapeStats{
	WildAgile: {
		WildTierNormal:   {Dexterity: 2, Damage: "1d6", Weapons: 2, CritRange: 19},
		WildTierImproved: {Dexterity: 4, Damage: "1d8", Weapons: 2, CritRange: 19, PlusMetres: 3, Size: "grande"},
		WildTierSuperior: {Dexterity: 6, Damage: "1d10", Weapons: 2, CritRange: 19, PlusMetres: 6, Size: "grande"},
	},
	WildFerocious: {
		WildTierNormal:   {Strength: 3, Defense: 2, Damage: "1d8", Weapons: 1},
		WildTierImproved: {Strength: 5, Defense: 4, Damage: "2d6", Weapons: 1, Size: "grande"},
		WildTierSuperior: {Strength: 10, Defense: 6, Damage: "4d6", Weapons: 1, Size: "enorme"},
	},
	WildResilient: {
		WildTierNormal:   {Defense: 5, Reduction: 5, Damage: "1d6", Weapons: 1},
		WildTierImproved: {Strength: 3, Defense: 8, Reduction: 8, Damage: "1d8", Weapons: 1, Size: "grande"},
		WildTierSuperior: {Strength: 5, Defense: 10, Reduction: 10, Damage: "2d6", Weapons: 1, Size: "enorme"},
	},
	WildSneaky: {
		WildTierNormal:   {Dexterity: 2, Damage: "1d4", Weapons: 1, Size: "pequeno"},
		WildTierImproved: {Dexterity: 4, Damage: "1d4", Weapons: 1, Size: "minusculo"},
		WildTierSuperior: {Dexterity: 6, Damage: "1d4", Weapons: 1, Size: "minusculo", FlyMetres: 18},
	},
	WildSwift: {
		WildTierNormal: {Dexterity: 2, Damage: "1d6", Weapons: 1,
			Moves: map[WildMovement]int{WildMoveLand: 15, WildMoveClimb: 9, WildMoveSwim: 9}},
		WildTierImproved: {Dexterity: 4, Damage: "1d6", Weapons: 1,
			Moves: map[WildMovement]int{WildMoveLand: 18, WildMoveClimb: 12, WildMoveSwim: 12}},
		WildTierSuperior: {Dexterity: 6, Damage: "1d6", Weapons: 1,
			Moves: map[WildMovement]int{WildMoveSwim: 18, WildMoveFly: 24}},
	},
}

// WildShapeStatsOf é a linha do quadro, e se ela existe.
//
//	WildShapeStatsOf(WildFerocious, WildTierNormal) // Força +3, Defesa +2, 1d8
func WildShapeStatsOf(form WildForm, tier WildTier) (WildShapeStats, bool) {
	stats, found := wildShapes[form][tier]
	return stats, found
}

// WildFormName é o nome da forma, que é também o da carta da arma dela.
func WildFormName(form WildForm) string {
	names := map[WildForm]string{
		WildAgile: "Forma Ágil", WildFerocious: "Forma Feroz", WildResilient: "Forma Resistente",
		WildSneaky: "Forma Sorrateira", WildSwift: "Forma Veloz",
	}
	return names[form]
}

// WildTierPowerID e WildTierPM são o poder que libera o nível e o preço dele (p62).
func WildTierPowerID(tier WildTier) string {
	switch tier {
	case WildTierImproved:
		return "class.druida.forma-selvagem-aprimorada"
	case WildTierSuperior:
		return "class.druida.forma-selvagem-superior"
	}
	return "class.druida.forma-selvagem"
}

func WildTierPM(tier WildTier) int {
	switch tier {
	case WildTierImproved:
		return 6
	case WildTierSuperior:
		return 10
	}
	return 3
}

// activeWildShape é a linha do quadro da forma em que a ficha está, ou falso fora dela.
func activeWildShape(ch Character) (WildShapeStats, bool) {
	if ch.WildShape == nil {
		return WildShapeStats{}, false
	}
	return WildShapeStatsOf(ch.WildShape.Form, ch.WildShape.Tier)
}

// wildShapeAsAnItem é a forma como fonte da ficha. O tamanho dela entra aqui, sem o passo de dano da
// Tab. 1-21: o quadro já escreve o dano da arma da forma.
func (r *Ruleset) wildShapeAsAnItem(ch Character) []ActiveItem {
	stats, shaped := activeWildShape(ch)
	if !shaped {
		return nil
	}
	mods := []Modifier{}
	add := func(target ModifierTarget, amount int) {
		if amount != 0 {
			mods = append(mods, Modifier{Target: target, Amount: amount, BonusType: "untyped"})
		}
	}
	add(ModifierTarget{K: "attribute", Name: "strength"}, stats.Strength)
	add(ModifierTarget{K: "attribute", Name: "dexterity"}, stats.Dexterity)
	add(ModifierTarget{K: "defense"}, stats.Defense)
	add(ModifierTarget{K: "damageReduction"}, stats.Reduction)
	add(ModifierTarget{K: "displacement"}, squaresOf(stats.PlusMetres))
	add(ModifierTarget{K: "flySpeed"}, stats.FlyMetres)
	if metres, chosen := stats.Moves[ch.WildShape.Movement]; chosen {
		switch ch.WildShape.Movement {
		case WildMoveLand:
			add(ModifierTarget{K: "displacement"}, squaresOf(metres)-r.raceDisplacement(ch))
		case WildMoveFly:
			add(ModifierTarget{K: "flySpeed"}, metres)
		}
	}
	if row, differs := sizesOnTheSheet[stats.Size]; differs {
		add(ModifierTarget{K: "expertise", Name: "Furtividade"}, row.stealth)
		for _, maneuver := range ManeuversOfTheBook() {
			for _, role := range []ManeuverRole{ManeuverOffense, ManeuverDefense} {
				add(ModifierTarget{K: "maneuver", Name: maneuver, Scope: string(role)}, row.maneuver)
			}
		}
	}
	return []ActiveItem{{SourceID: WildTierPowerID(ch.WildShape.Tier), Source: WildFormName(ch.WildShape.Form),
		Equipped: &vestedWear, Modifiers: mods}}
}

// wildShapeWeapon é a arma natural da forma.
func wildShapeWeapon(ch Character) (ownedNaturalWeapon, bool) {
	stats, shaped := activeWildShape(ch)
	if !shaped {
		return ownedNaturalWeapon{}, false
	}
	return ownedNaturalWeapon{
		sourceID: WildTierPowerID(ch.WildShape.Tier), ofTheShape: true, critRange: stats.CritRange,
		weapon: NaturalWeapon{Name: WildFormName(ch.WildShape.Form), Damage: stats.Damage, CritMult: 2, DamageType: ch.WildShape.DamageType},
	}, true
}

// formSizeReplacesTheRace diz se a forma tem tamanho próprio, e então o da raça não vale.
func formSizeReplacesTheRace(ch Character) bool {
	stats, shaped := activeWildShape(ch)
	return shaped && stats.Size != ""
}
