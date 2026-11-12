package sheetui

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"t20engine/app/character"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// A FORMA SELVAGEM na ficha (p62-63): o diálogo da escolha na aba Poderes, e a forma em vigor na aba
// Efeitos. Os números são do motor (`engine.WildShapeStatsOf`); aqui mora só o que a tela escreve.

// wildShapeView é o que as duas abas precisam: as opções do diálogo, para quem tem a Forma Selvagem, e a
// linha da forma em que o druida está.
type wildShapeView struct {
	// Offered diz que a ficha tem a Forma Selvagem, e Tiers são os níveis que ela pode pagar.
	Offered bool
	Tiers   []wildTierOption
	// Summaries é o resumo de cada forma em cada nível, por "forma:nível", que o diálogo lê do sinal.
	Summaries map[string]string
	// Moves são as escolhas da Forma Veloz em cada nível.
	Moves map[engine.WildTier][]wildMoveOption
	// Active é a forma em vigor, ou nil.
	Active *wildShapeLine
	// AgileBoth é o "atacar com ambas" da Forma Ágil (p63), quando o druida está nela; Declared diz que
	// ele já declarou.
	AgileBoth *agileBothState
}

type agileBothState struct {
	Declared bool
}

type wildTierOption struct {
	Tier  engine.WildTier
	Label string
	PM    int
}

type wildMoveOption struct {
	Move  engine.WildMovement
	Label string
}

// wildShapeLine é a forma em vigor: o nome e o que ela está dando.
type wildShapeLine struct {
	Name string
	Meta string
}

// wildShapePowerIDs são os três poderes que abrem o mesmo diálogo: a normal, a aprimorada e a superior.
var wildShapePowerIDs = map[string]bool{
	engine.WildTierPowerID(engine.WildTierNormal):   true,
	engine.WildTierPowerID(engine.WildTierImproved): true,
	engine.WildTierPowerID(engine.WildTierSuperior): true,
}

// opensWildShape diz se a linha é a da Forma Selvagem normal, que leva o botão. A aprimorada e a superior
// são níveis do mesmo gesto, e não gestos à parte: as linhas delas ficam sem botão.
func opensWildShape(row powerRow) bool {
	return row.ID == engine.WildTierPowerID(engine.WildTierNormal)
}

// wildShapeDialogID é o id do diálogo da escolha.
const wildShapeDialogID = "wild-shape"

// wildShapeOpens são os sinais que abrir o diálogo zera: a primeira forma e o nível normal, que todo
// druida com a Forma Selvagem tem.
const wildShapeOpens = "$wild_form = 'agil'; $wild_tier = 'normal'; $wild_damage = 'corte'; $wild_move = ''"

func wildShapeViewOf(dto sheet.CharacterDTO) wildShapeView {
	view := wildShapeView{Summaries: map[string]string{}, Moves: map[engine.WildTier][]wildMoveOption{}}
	for _, tier := range engine.WildTiers {
		if !sheet.OwnsClassPower(dto, engine.WildTierPowerID(tier)) {
			continue
		}
		view.Offered = true
		pm := engine.WildTierPM(tier)
		view.Tiers = append(view.Tiers, wildTierOption{Tier: tier, Label: wildTierLabel(tier) + " · " + strconv.Itoa(pm) + " PM", PM: pm})
		for _, form := range engine.WildForms {
			stats, _ := engine.WildShapeStatsOf(form, tier)
			view.Summaries[string(form)+":"+string(tier)] = strings.Join(wildShapeParts(stats, "", ""), " · ")
		}
		swift, _ := engine.WildShapeStatsOf(engine.WildSwift, tier)
		for _, move := range []engine.WildMovement{engine.WildMoveLand, engine.WildMoveClimb, engine.WildMoveSwim, engine.WildMoveFly} {
			if metres, found := swift.Moves[move]; found {
				view.Moves[tier] = append(view.Moves[tier], wildMoveOption{Move: move, Label: wildMoveLabel(move, metres)})
			}
		}
	}
	if dto.WildShape != nil {
		if stats, found := engine.WildShapeStatsOf(dto.WildShape.Form, dto.WildShape.Tier); found {
			parts := wildShapeParts(stats, dto.WildShape.DamageType, dto.WildShape.Movement)
			parts = append(parts, strconv.FormatInt(dto.WildShapePaid, 10)+" PM pagos")
			view.Active = &wildShapeLine{Name: wildShapeName(*dto.WildShape), Meta: strings.Join(parts, " · ")}
		}
	}
	if character.OffersTheDeclaration(dto, engine.WildAgileBothFlag) {
		view.AgileBoth = &agileBothState{Declared: slices.Contains(dto.Conditionals, engine.FlagGroupID(engine.WildAgileBothFlag))}
	}
	return view
}

// agileBothPath é o gesto de declarar as duas armas naturais.
var agileBothPath = "/declara/" + engine.WildAgileBothFlag

// agileBothRows é a linha das Ações, entre as livres: a declaração se faz antes do golpe, e é ali que se
// agride.
func agileBothRows(w wildShapeView) []actionRow {
	if w.AgileBoth == nil {
		return nil
	}
	return []actionRow{{
		Name: "Forma Ágil · as duas armas naturais", Detail: "–2 em todos os ataques até o seu próximo turno · p63",
		Can: true, Live: w.AgileBoth.Declared,
		Rolls: []actionRoll{{Label: "Usar", Path: agileBothPath}},
	}}
}

// wildShapeName é "Forma Feroz", ou "Forma Feroz aprimorada".
func wildShapeName(shape engine.WildShape) string {
	name := engine.WildFormName(shape.Form)
	if shape.Tier != engine.WildTierNormal {
		name += " " + string(shape.Tier)
	}
	return name
}

// wildFormShortName é o nome no botão do diálogo: "Feroz", sem o "Forma".
func wildFormShortName(form engine.WildForm) string {
	return strings.TrimPrefix(engine.WildFormName(form), "Forma ")
}

func wildTierLabel(tier engine.WildTier) string {
	switch tier {
	case engine.WildTierImproved:
		return "Aprimorada"
	case engine.WildTierSuperior:
		return "Superior"
	}
	return "Normal"
}

func wildMoveLabel(move engine.WildMovement, metres int) string {
	words := map[engine.WildMovement]string{
		engine.WildMoveLand: "deslocamento", engine.WildMoveClimb: "escalada",
		engine.WildMoveSwim: "natação", engine.WildMoveFly: "voo",
	}
	return fmt.Sprintf("%s %dm", words[move], metres)
}

// wildDamageLabels são os três tipos da arma da forma, como a tela os escreve.
var wildDamageLabels = map[string]string{"corte": "Corte", "impacto": "Impacto", "perfuracao": "Perfuração"}

// wildShapeParts é o que a forma dá, em pedaços: "Força +3 · Defesa +2 · arma natural 1d8". Com o tipo de
// dano e o deslocamento escolhidos, eles entram também; sem, é o resumo do diálogo.
func wildShapeParts(stats engine.WildShapeStats, damageType string, move engine.WildMovement) []string {
	parts := []string{}
	signed := func(label string, n int) {
		if n != 0 {
			parts = append(parts, fmt.Sprintf("%s %+d", label, n))
		}
	}
	signed("Força", stats.Strength)
	signed("Destreza", stats.Dexterity)
	signed("Defesa", stats.Defense)
	if stats.Reduction != 0 {
		parts = append(parts, fmt.Sprintf("RD %d", stats.Reduction))
	}
	weapon := "arma natural " + stats.Damage
	if stats.Weapons == 2 {
		weapon = "duas armas naturais " + stats.Damage
	}
	if stats.CritRange != 0 {
		weapon += fmt.Sprintf(", crítico %d", stats.CritRange)
	}
	if damageType != "" {
		weapon += ", " + strings.ToLower(wildDamageLabels[damageType])
	}
	parts = append(parts, weapon)
	if stats.PlusMetres != 0 {
		parts = append(parts, fmt.Sprintf("deslocamento +%dm", stats.PlusMetres))
	}
	if stats.FlyMetres != 0 {
		parts = append(parts, fmt.Sprintf("voo %dm", stats.FlyMetres))
	}
	if metres, chosen := stats.Moves[move]; chosen {
		parts = append(parts, wildMoveLabel(move, metres))
	}
	if size := wildSizeLabel(stats.Size); size != "" {
		parts = append(parts, size)
	}
	return parts
}

func wildSizeLabel(size string) string {
	labels := map[string]string{"minusculo": "Minúsculo", "pequeno": "Pequeno", "grande": "Grande", "enorme": "Enorme"}
	return labels[size]
}

// wildSummaryExpr é a expressão do diálogo que escreve o resumo da forma e do nível escolhidos.
func (w wildShapeView) wildSummaryExpr() string {
	table, err := json.Marshal(w.Summaries)
	if err != nil {
		return "''"
	}
	return "(" + string(table) + ")[$wild_form + ':' + $wild_tier] || ''"
}

// wildCostExpr é o "Transformar · N PM" do pé, pelo nível escolhido.
func (w wildShapeView) wildCostExpr() string {
	prices := map[string]int{}
	for _, tier := range w.Tiers {
		prices[string(tier.Tier)] = tier.PM
	}
	table, err := json.Marshal(prices)
	if err != nil {
		return "'Transformar'"
	}
	return "'Transformar · ' + (" + string(table) + ")[$wild_tier] + ' PM'"
}
