package sheet

import (
	"fmt"
	"slices"
	"strings"

	"t20engine/domain/book"
)

// O PODER QUE A ORIGEM DEIXA ESCOLHER ("um poder de combate a sua escolha", p91).
//
// A escolha mora no `powerChoices`, sob o id do BENEFÍCIO, e é ali que o motor a lê
// (`engine.Ruleset.OriginPickedPowerIDs`). Ela não é a escolha DE um poder, como a perícia do
// Foco em Perícia: é qual poder geral a origem dá. Por isso tem a leitura, a gravação e a
// conferência próprias, e o `powerPicksFit` a deixa para o `originPicksFit`.

// OriginPowerPick é um benefício de origem marcado que deixa escolher um poder, e o
// escolhido (vazio enquanto ninguém escolheu).
type OriginPowerPick struct {
	Benefit book.OriginBenefit
	Chosen  string
}

// OriginPowerPicks são os benefícios MARCADOS da origem da ficha que deixam escolher um
// poder, na ordem da origem.
//
//	sheet.OriginPowerPicks(dto) // [{Benefit: Poder de Combate (escolha), Chosen: "bloqueio-com-escudo"}]
func OriginPowerPicks(dto CharacterDTO) []OriginPowerPick {
	origin, found := book.Origins()[dto.Origin]
	if !found {
		return nil
	}
	chosen := UnmarshalStrings(dto.OriginChoices)
	picks := PowerPicks(dto.PowerChoices)
	var out []OriginPowerPick
	for _, b := range OriginBenefitsOf(origin) {
		if b.PowerPick == "" || !slices.Contains(chosen, b.ID) {
			continue
		}
		pick := OriginPowerPick{Benefit: b}
		if len(picks[b.ID]) > 0 {
			pick.Chosen = picks[b.ID][0]
		}
		out = append(out, pick)
	}
	return out
}

// GeneralPowersOfKind são os poderes gerais de um tipo (`combate`, `tormenta`), pelo nome.
func GeneralPowersOfKind(kind string) []book.GeneralPower {
	var out []book.GeneralPower
	for _, p := range book.GeneralPowers() {
		if p.Kind == kind {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b book.GeneralPower) int { return strings.Compare(a.Name, b.Name) })
	return out
}

// WithOriginPowerPick grava o poder escolhido no benefício, no lugar do anterior: o
// benefício dá UM poder.
func WithOriginPowerPick(blob, benefitID, powerID string) string {
	picks := PowerPicks(blob)
	picks[benefitID] = []string{powerID}
	return marshalPowerPicks(picks)
}

// WithoutOriginPowerPick tira a escolha do benefício, para quando ele é desmarcado: a
// escolha sem o benefício seria órfã, e a validação a recusaria.
func WithoutOriginPowerPick(blob, benefitID string) string {
	picks := PowerPicks(blob)
	delete(picks, benefitID)
	return marshalPowerPicks(picks)
}

// isOriginPickKey diz se a chave do `powerChoices` é um benefício de origem que deixa
// escolher um poder, de qualquer origem: a chave de outra origem é recusada pelo
// `originPicksFit`, e não confundida com escolha de poder.
func isOriginPickKey(id string) bool {
	for _, origin := range book.Origins() {
		for _, b := range OriginBenefitsOf(origin) {
			if b.ID == id && b.PowerPick != "" {
				return true
			}
		}
	}
	return false
}

// originPicksFit confere as escolhas de poder da origem: o benefício é da origem da ficha
// e está marcado, há UM poder só, e ele é poder geral do tipo que o benefício dá. Os
// pré-requisitos do poder ficam escritos na tela e não barram, como no passo dos poderes
// da classe: quem decide se valem é o mestre.
func originPicksFit(dto CharacterDTO) error {
	marked := map[string]book.OriginBenefit{}
	for _, pick := range OriginPowerPicks(dto) {
		marked[pick.Benefit.ID] = pick.Benefit
	}
	for id, picks := range PowerPicks(dto.PowerChoices) {
		if !isOriginPickKey(id) {
			continue
		}
		benefit, ok := marked[id]
		if !ok {
			return fmt.Errorf("há poder escolhido em %q, e o benefício não está marcado na origem da ficha", id)
		}
		if len(picks) != 1 {
			return fmt.Errorf("%s dá um poder, e há %d escolhidos", benefit.Name, len(picks))
		}
		power, found := book.GeneralPowers()[picks[0]]
		if !found {
			return fmt.Errorf("%q não é um poder geral", picks[0])
		}
		if power.Kind != benefit.PowerPick {
			return fmt.Errorf("%s dá um poder de %s, e %s é poder de %s", benefit.Name, benefit.PowerPick, power.Name, power.Kind)
		}
	}
	return nil
}
