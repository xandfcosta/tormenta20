package sheetui

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/search"
	"t20engine/domain/sheet"
)

// A ABA EFEITOS DA FICHA (ALE-473). O dado é o do painel
// (`effectsPanel`); daqui sai só o que a aba lê. As descrições da condição,
// da postura e do efeito aplicado são as da coluna (`activeEffects`), uma só
// por efeito.

// effectsOfKind são os efeitos ativos de um tipo, na ordem da coluna.
func effectsOfKind(v View, kinds ...string) []activeEffect {
	var out []activeEffect
	for _, e := range activeEffects(v) {
		for _, kind := range kinds {
			if e.Kind == kind {
				out = append(out, e)
			}
		}
	}
	return out
}

var notIDSafe = regexp.MustCompile(`[^a-z0-9-]+`)

// situationalSheetID é o `id` da folha de um situacional. A chave pode ter
// dois-pontos ("flag:terreno"), e o id fica só com o que o seletor aceita.
func situationalSheetID(row situationalRow) string {
	return "situational-" + notIDSafe.ReplaceAllString(row.Key, "-")
}

// situationalMeta é a nota da linha: de onde vem e, com degrau pago, quanto.
//
//	situationalMeta(row) // "Machado silvestre · +4 · 2 PM", com o degrau
func situationalMeta(row situationalRow) string {
	meta := row.Source
	for _, d := range row.Degrees {
		if d.Chosen && row.Active {
			meta += " · " + d.Label + " · " + d.Pm
		}
	}
	if row.Warning != "" {
		meta += " · " + row.Warning
	}
	return meta
}

// twoWeaponStyleWarning avisa que o Estilo de Duas Armas pede uma arma leve: "se
// estiver empunhando duas armas (e pelo menos uma delas for leve)" (p125). O Arma
// Secundária Grande dispensa a leve (p124).
//
// Item feito à mão não diz se é leve, e aí a tela cala em vez de acusar.
func twoWeaponStyleWarning(key string, dto sheet.CharacterDTO) string {
	if key != engine.FlagGroupID(engine.TwoWeaponStylePowerID) && key != engine.FlagGroupID(engine.AmbidextrousSwitch) {
		return ""
	}
	owned := sheet.OwnedPowers(dto)
	if slices.ContainsFunc(owned, func(p sheet.OwnedPower) bool { return p.ID == "arma-secundaria-grande" }) {
		return ""
	}
	for _, item := range dto.Items {
		if item.Equipped == nil || sheet.HandsFor(*item.Equipped) == 0 {
			continue
		}
		if item.CatalogID == nil {
			return ""
		}
		if fromBook := book.ItemByID(*item.CatalogID); fromBook == nil || fromBook.Weapon == nil {
			continue
		} else if fromBook.Weapon.Hand == "light" || slices.Contains(fromBook.Weapon.Traits, engine.DoubleTrait) {
			return ""
		}
	}
	return "Nenhuma arma empunhada é leve (p125)"
}

// rapidShotWarning avisa que o Disparo Rápido pede arma que recarregue como ação livre (p125), e só
// o arco faz isso: besta e funda recarregam com movimento, e pistola e mosquete, com a padrão
// (p146-151). Não trava pela mesma razão do aviso acima.
func rapidShotWarning(key string, dto sheet.CharacterDTO) string {
	if key != engine.FlagGroupID(engine.RapidShotPowerID) {
		return ""
	}
	for _, item := range dto.Items {
		if item.Equipped == nil || sheet.HandsFor(*item.Equipped) == 0 {
			continue
		}
		if item.CatalogID == nil || strings.HasPrefix(*item.CatalogID, "arco-") {
			return ""
		}
	}
	return "Só o arco recarrega como ação livre (p125)"
}

// oneWeaponStyleWarning avisa que o Estilo de Uma Arma pede "uma arma corpo a corpo em uma das mãos e nada
// na outra" (p128). O interruptor liga do mesmo jeito, e o número só se mexe com as mãos assim.
func oneWeaponStyleWarning(key string, dto sheet.CharacterDTO) string {
	if key != engine.FlagGroupID("estilo-de-uma-arma") || dto.Ruleset == nil {
		return ""
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil || dto.Ruleset.HoldsALoneWeapon(ec) {
		return ""
	}
	return "Pede uma arma corpo a corpo em uma das mãos e nada na outra (p128)"
}

// situationalToggles é o gesto do interruptor: a chave vai no sinal.
func situationalToggles(v View, row situationalRow) string {
	return "$conditional = '" + row.Key + "'; " + sheetPost(v, "/efeitos/situacao")
}

// degreeChooses escolhe o degrau E liga o situacional: escolher o degrau é
// dizer que o PM foi gasto.
func degreeChooses(v View, row situationalRow, steps int) string {
	return "$conditional = '" + row.Key + "'; $conditional_degrees = " + strconv.Itoa(steps) + "; " +
		sheetPost(v, "/efeitos/situacao/degraus")
}

// effectPickFilters são as buscas dos dois escolhedores da aba, filtradas no
// servidor como as do grimório.
type effectPickFilters struct {
	ConditionSearch string
	BuffSearch      string
	BuffCircle      string
}

// pickerOptionsShown são as opções que passam pela busca do nome e, quando há
// um, pelo círculo.
//
//	pickerOptionsShown(v.Effects.BuffOptions, "escudo", "") // só o Escudo da Fé
func pickerOptionsShown(options []pickerOption, query, circle string) []pickerOption {
	var shown []pickerOption
	for _, o := range options {
		if circle != "" && o.Circle != circle {
			continue
		}
		if strings.TrimSpace(query) != "" && !strings.Contains(search.Fold(o.Label), search.Fold(query)) {
			continue
		}
		shown = append(shown, o)
	}
	return shown
}

// buffCircleOptions são as opções do filtro do escolhedor de magia: "Todos" e
// os círculos que têm magia aplicável, como no grimório.
func buffCircleOptions(v View) []filterOption {
	has := map[string]bool{}
	for _, o := range v.Effects.BuffOptions {
		has[o.Circle] = true
	}
	options := []filterOption{{Value: "", Label: "Todos", Active: v.EffectPicks.BuffCircle == ""}}
	for circle := 1; circle <= 5; circle++ {
		if name := circleName(circle); has[name] {
			options = append(options, filterOption{Value: name, Label: name, Active: v.EffectPicks.BuffCircle == name})
		}
	}
	return options
}
