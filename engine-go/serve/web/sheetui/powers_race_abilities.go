package sheetui

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// A ESCOLHA DAS HABILIDADES DE RAÇA no diálogo (ALE-493): a Deformidade do
// lefou e o Vanguardista do kliren, no cartão da raça.

// raceAbilityBlock é uma habilidade com escolha: as perícias, os poderes da
// troca quando ela existe, e quantas de quantas.
type raceAbilityBlock struct {
	ID     string
	Name   string
	Detail string
	Chosen int
	Count  int
	Skills []powerOption
	// Swaps são os poderes da Tormenta que tomam o lugar de UM bônus (p24).
	Swaps []powerOption
	// NoCraft diz que a escolha é de Ofício e a ficha não tem nenhum.
	NoCraft bool
}

// Full diz que não cabe mais nenhuma: as não escolhidas se apagam.
func (b raceAbilityBlock) Full() bool { return b.Chosen >= b.Count }

// Written é "1 de 2".
func (b raceAbilityBlock) Written() string {
	return strconv.Itoa(b.Chosen) + " de " + strconv.Itoa(b.Count)
}

// raceAbilityBlocksOf monta os blocos da raça `race`.
func raceAbilityBlocksOf(dto sheet.CharacterDTO, race string) []raceAbilityBlock {
	out := []raceAbilityBlock{}
	for _, choice := range sheet.RaceAbilityChoices(dto) {
		if choice.Race != race {
			continue
		}
		ability := choice.Ability
		block := raceAbilityBlock{
			ID: ability.ID, Name: ability.Name, Detail: ability.Description,
			Chosen: len(choice.Picks), Count: ability.Choice.Count,
		}
		for _, skill := range skillsOffered(dto, ability.Choice.Kind) {
			block.Skills = append(block.Skills, pickedOption(skill, skill, choice.Picks))
		}
		block.NoCraft = ability.Choice.Kind == "craft" && len(block.Skills) == 0
		if ability.Choice.SwapFor == "tormenta-power" {
			for _, power := range tormentaPowers() {
				block.Swaps = append(block.Swaps, pickedOption(power.ID, power.Name, choice.Picks))
			}
		}
		out = append(out, block)
	}
	return out
}

// skillsOffered são as perícias que a escolha aceita: os Ofícios da ficha, ou
// as perícias do livro mais os Ofícios inventados.
func skillsOffered(dto sheet.CharacterDTO, kind string) []string {
	crafts := []string{}
	for _, e := range dto.Expertises {
		if e.Custom && strings.HasPrefix(e.Name, "Ofício") {
			crafts = append(crafts, e.Name)
		}
	}
	if kind == "craft" {
		return crafts
	}
	out := []string{}
	for _, e := range book.Expertises() {
		out = append(out, e.Name)
	}
	for _, craft := range crafts {
		if !slices.Contains(out, craft) {
			out = append(out, craft)
		}
	}
	return out
}

func tormentaPowers() []book.GeneralPower {
	out := []book.GeneralPower{}
	for _, p := range book.GeneralPowers() {
		if p.Kind == "tormenta" {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Name < out[b].Name })
	return out
}

func pickedOption(value, label string, picks []string) powerOption {
	count := 0
	if slices.Contains(picks, value) {
		count = 1
	}
	return powerOption{Value: value, Label: label, Count: count}
}
