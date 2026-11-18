package engine

// A FORMA PRIMAL (p62, druida 18): "você recebe os benefícios de dois tipos de animais (bônus iguais não
// se acumulam; use o que você quiser de cada tipo)". Nos NÚMEROS vale o maior das duas formas (decisão do
// dono): ninguém escolheria o menor de um bônus. A arma natural e o tamanho não se somam nem se comparam
// — são escolhas, e quem escolhe é o jogador, com o padrão na primeira forma.

// PrimalFormPowerID é a Forma Primal.
const PrimalFormPowerID = "class.druida.forma-primal"

// WeaponForm é a forma de onde vem a arma natural: a escolhida, ou a primeira.
func (w WildShape) WeaponForm() WildForm {
	if w.Second != "" && w.WeaponFrom == w.Second {
		return w.Second
	}
	return w.Form
}

// sizeForm é a forma de onde vem o tamanho: a escolhida, ou a primeira.
func (w WildShape) sizeForm() WildForm {
	if w.Second != "" && w.SizeFrom == w.Second {
		return w.Second
	}
	return w.Form
}

// WildShapeStatsOfShape é a linha do quadro da forma que a ficha assumiu: a de uma forma, ou a fera
// primal das duas.
//
//	WildShapeStatsOfShape(WildShape{Form: WildFerocious, Second: WildResilient, Tier: WildTierNormal})
//	// Força +3 (Feroz), Defesa +5 e RD 5 (Resistente), a arma 1d8 da Feroz
func WildShapeStatsOfShape(shape WildShape) (WildShapeStats, bool) {
	first, found := WildShapeStatsOf(shape.Form, shape.Tier)
	if !found || shape.Second == "" {
		return first, found
	}
	second, found := WildShapeStatsOf(shape.Second, shape.Tier)
	if !found {
		return WildShapeStats{}, false
	}
	both := first
	both.Strength = max(first.Strength, second.Strength)
	both.Dexterity = max(first.Dexterity, second.Dexterity)
	both.Defense = max(first.Defense, second.Defense)
	both.Reduction = max(first.Reduction, second.Reduction)
	both.PlusMetres = max(first.PlusMetres, second.PlusMetres)
	both.FlyMetres = max(first.FlyMetres, second.FlyMetres)
	if both.Moves == nil {
		both.Moves = second.Moves
	}
	weapon := first
	if shape.WeaponForm() == shape.Second {
		weapon = second
	}
	both.Damage, both.Weapons, both.CritRange = weapon.Damage, weapon.Weapons, weapon.CritRange
	both.Size = first.Size
	if shape.sizeForm() == shape.Second {
		both.Size = second.Size
	}
	return both, true
}
