package sheetui

import (
	"fmt"
	"net/http"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// OS GESTOS DA DEVOÇÃO, no passo do Escolher: o deus da ficha e os poderes concedidos. A regra deles é a
// da devoção (p96), e não a dos poderes da classe, e por isso eles moram fora do powers_commands.go.

// noDevotion é o valor do gesto de ficar sem deus.
const noDevotion = "nenhum"

// pickDevotion grava o deus da ficha, que é a fonte única da devoção (ALE-518), e tira os poderes que ele não
// concede, por decisão do dono. O passo avisa disso antes de chegar aqui.
func pickDevotion(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	god := pathParam(r, "deus")
	if god == noDevotion {
		god = ""
	}
	return s.saveTheChoices(r, row, func(dto *sheet.CharacterDTO) {
		dto.God = &god
		ofTheGod := sheet.GrantedPowerNames(*dto, god)
		kept := slices.DeleteFunc(sheet.UnmarshalStrings(dto.GodPowers), func(power string) bool {
			return !slices.Contains(ofTheGod, power)
		})
		dto.GodPowers = sheet.MarshalStrings(&kept)
	})
}

// pickGrantedPower marca ou desmarca um poder concedido. O marcado sai, seja de que deus for; o novo tem de
// ser do deus da ficha, e cabe nas vagas dele (`sheet.GrantedPowerLimit`). Numa vaga só, ele troca o outro,
// como numa escolha de um; com duas, tirar é um gesto, para ninguém perder um poder sem pedir.
func pickGrantedPower(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	power := pathParam(r, "poder")
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	held := sheet.UnmarshalStrings(dto.GodPowers)
	if slices.Contains(held, power) {
		held = slices.DeleteFunc(held, func(name string) bool { return name == power })
		return s.saveTheChoices(r, row, func(dto *sheet.CharacterDTO) { dto.GodPowers = sheet.MarshalStrings(&held) })
	}
	if dto.God == nil || *dto.God == "" {
		return fmt.Errorf("sem deus, não há poder concedido para escolher: %s", power)
	}
	god := *dto.God
	offered := sheet.GrantedPowerNames(dto, god)
	if !slices.Contains(offered, power) {
		return fmt.Errorf("%s não é um poder que %s concede", power, book.DevotionName(god))
	}
	limit := sheet.GrantedPowerLimit(dto, god)
	ofTheGod := slices.DeleteFunc(slices.Clone(held), func(name string) bool { return !slices.Contains(offered, name) })
	switch {
	case len(ofTheGod) < limit:
	case limit == 1:
		held = slices.DeleteFunc(held, func(name string) bool { return slices.Contains(offered, name) })
	default:
		return fmt.Errorf("esta ficha recebe %d poderes concedidos de %s: tire um antes de escolher %s", limit, book.DevotionName(god), power)
	}
	held = append(held, power)
	return s.saveTheChoices(r, row, func(dto *sheet.CharacterDTO) { dto.GodPowers = sheet.MarshalStrings(&held) })
}
