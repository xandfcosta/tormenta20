package sheetui

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/search"
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
	// PowerGroups são os poderes gerais da Memória Póstuma, por grupo (p130),
	// já filtrados pela busca (ALE-502).
	PowerGroups []powerGroup
	Search      string
	// RaceGroups são as habilidades das raças humanoides que a Memória
	// Póstuma pode tomar, por raça (ALE-504).
	RaceGroups []powerGroup
}

// powerGroup é um grupo de poder geral — Combate, Destino, Magia, Tormenta.
type powerGroup struct {
	Title   string
	Options []powerOption
}

// generalPowerGroups são os quatro grupos que a Memória Póstuma aceita, na
// ordem do livro (p130). Os Concedidos ficam de fora: pedem devoção.
var generalPowerGroups = []struct{ Kind, Title string }{
	{"combate", "Combate"}, {"destino", "Destino"}, {"magia", "Magia"}, {"tormenta", "Tormenta"},
}

// Full diz que não cabe mais nenhuma: as não escolhidas se apagam.
func (b raceAbilityBlock) Full() bool { return b.Chosen >= b.Count }

// SkillsLabel é o rótulo das perícias quando há outra lista embaixo: o +2 da
// Deformidade, o treino da Memória Póstuma.
func (b raceAbilityBlock) SkillsLabel() string {
	switch {
	case len(b.Swaps) > 0:
		return "Perícias · +2"
	case b.PowerGroups != nil || b.Search != "":
		return "Uma perícia · fica treinada"
	}
	return ""
}

// Written é "1 de 2".
func (b raceAbilityBlock) Written() string {
	return strconv.Itoa(b.Chosen) + " de " + strconv.Itoa(b.Count)
}

// raceAbilityBlocksOf monta os blocos da raça `race`. A busca filtra os
// poderes gerais da Memória Póstuma.
func raceAbilityBlocksOf(dto sheet.CharacterDTO, race, query string) []raceAbilityBlock {
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
		if ability.Choice.Kind == "variant" {
			for _, v := range ability.Variants {
				block.Skills = append(block.Skills, pickedOption(v.ID, v.Name, choice.Picks))
			}
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
		if sheet.IsTrainedOrPower(*ability.Choice) {
			block.Search = query
			block.PowerGroups = generalPowerGroupsOf(choice.Picks, query)
			block.RaceGroups = borrowableRaceGroupsOf(choice.Picks)
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
	if kind == "variant" {
		return nil
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

// generalPowerGroupsOf são os poderes gerais por grupo, os que casam com a
// busca — o escolhido sempre aparece, para poder ser tirado.
func generalPowerGroupsOf(picks []string, query string) []powerGroup {
	term := search.Fold(strings.TrimSpace(query))
	out := []powerGroup{}
	for _, g := range generalPowerGroups {
		group := powerGroup{Title: g.Title}
		for _, p := range book.GeneralPowers() {
			if p.Kind != g.Kind || (!casaComABusca(p.Name, term) && !slices.Contains(picks, p.ID)) {
				continue
			}
			group.Options = append(group.Options, pickedOption(p.ID, p.Name, picks))
		}
		sort.Slice(group.Options, func(a, b int) bool { return group.Options[a].Label < group.Options[b].Label })
		if len(group.Options) > 0 {
			out = append(out, group)
		}
	}
	return out
}

// borrowableRaceGroupsOf são as habilidades das raças humanoides, uma raça por
// grupo; a raça que não é Média diz o tamanho, porque ele vem junto (p29).
func borrowableRaceGroupsOf(picks []string) []powerGroup {
	sizes := map[string]string{}
	races, _, _ := book.CharacterCatalogs()
	for _, r := range races {
		sizes[r.ID] = r.Size
	}
	out := []powerGroup{}
	for _, race := range sheet.BorrowableRaces() {
		title := race.Name
		if size := sizes[race.ID]; size != "" && size != "Médio" {
			title += " · " + size
		}
		group := powerGroup{Title: title}
		for _, ability := range race.Abilities {
			group.Options = append(group.Options, pickedOption(ability.ID, ability.Name, picks))
		}
		out = append(out, group)
	}
	return out
}
