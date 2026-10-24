package engine

// O TAMANHO DA CRIATURA NA FICHA (p106–107, Tab. 1-21).
//
// A tabela tem três colunas que a ficha lê: o modificador de Furtividade, o de
// manobras — o mesmo número com o sinal trocado — e, do texto da p106, o passo
// da arma que a criatura usa ("Minúsculas usam armas reduzidas, que causam um
// passo a menos de dano"; Grandes e Enormes, um a mais; Colossais, dois).
//
// O tamanho vem da RAÇA PRIMÁRIA, como o deslocamento (ALE-383), e não da
// coluna `size` da ficha: a coluna é um espelho escrito no nascimento, e foi um
// espelho assim que fez dois anões andarem 9m.
//
// O espaço ocupado é a quarta coluna, e ela não está aqui: é do TABULEIRO
// (`FootprintForSize`).

// sizeRow é uma linha da Tab. 1-21 vista pela ficha.
type sizeRow struct {
	stealth, maneuver, weaponSteps int
}

// sizesOnTheSheet, pela forma normalizada do nome (`normalizeSizeKey`). O
// Médio não tem linha porque não tem modificador nenhum (p106).
var sizesOnTheSheet = map[string]sizeRow{
	"minusculo": {stealth: 5, maneuver: -5, weaponSteps: -1},
	"pequeno":   {stealth: 2, maneuver: -2},
	"grande":    {stealth: -2, maneuver: 2, weaponSteps: 1},
	"enorme":    {stealth: -5, maneuver: 5, weaponSteps: 1},
	"colossal":  {stealth: -10, maneuver: 10, weaponSteps: 2},
}

// raceSizeItem é o tamanho da raça primária como fonte de modificador, ou nil
// para o Médio e para a raça que o catálogo não conhece.
//
//	r.raceSizeItem(goblin) // Tamanho: Pequeno — Furtividade +2, manobras –2
func (r *Ruleset) raceSizeItem(ch Character) *ActiveItem {
	if len(ch.Races) == 0 {
		return nil
	}
	race := r.raceEntryByID(ch.Races[0].Race)
	if race == nil {
		return nil
	}
	row, differs := sizesOnTheSheet[normalizeSizeKey(race.Size)]
	if !differs {
		return nil
	}
	mods := []Modifier{{
		Target: ModifierTarget{K: "expertise", Name: "Furtividade"}, Amount: row.stealth, BonusType: "untyped",
	}}
	// As cinco manobras, dos DOIS lados do teste oposto: quem é pequeno derruba
	// pior e resiste pior a ser derrubado (p234).
	for _, maneuver := range ManeuversOfTheBook() {
		for _, role := range []ManeuverRole{ManeuverOffense, ManeuverDefense} {
			mods = append(mods, Modifier{
				Target: ModifierTarget{K: "maneuver", Name: maneuver, Scope: string(role)},
				Amount: row.maneuver, BonusType: "untyped",
			})
		}
	}
	if row.weaponSteps != 0 {
		mods = append(mods, Modifier{
			Target: ModifierTarget{K: "damageStep", Scope: "all"}, Amount: row.weaponSteps, BonusType: "untyped",
		})
	}
	return &ActiveItem{
		SourceID: "tamanho-" + normalizeSizeKey(race.Size), Source: "Tamanho: " + race.Size,
		Equipped: &vestedWear, Modifiers: mods,
	}
}
