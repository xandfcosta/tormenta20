package character

import (
	"context"
	"errors"
	"fmt"

	"t20engine/domain/live"
	"t20engine/domain/sheet"
)

// FireArcaneBolt dispara o Raio Arcano da ficha (ver GLOSSARY, **raio arcano**): rola o dado — ou confere o total que
// a mesa rolou na mão —, cobra o 1 PM do Raio Elemental quando ele muda o tipo, e devolve o raio para a faixa da
// magia, com as parcelas dos poderes que beneficiam as magias, porque ele "conta como uma magia" (p39).
//
// A ação padrão é de quem chama, ANTES e DEPOIS, como a da ação da magia: confere que cabe, dispara e cobra.
//
//	cast, err := plays.FireArcaneBolt(ctx, dto, "fogo", nil)
func (p Plays) FireArcaneBolt(ctx context.Context, dto sheet.CharacterDTO, element string, fromTheTable []int) (SpellCast, error) {
	computed, err := sheet.Compute(dto)
	if err != nil {
		return SpellCast{}, fmt.Errorf("computar a ficha para o Raio Arcano: %w", err)
	}
	bolt, has := sheet.ArcaneBoltOf(dto, computed)
	if !has {
		return SpellCast{}, errors.New("a ficha não tem o Raio Arcano (p39)")
	}
	parts, err := bolt.Roll(element)
	if err != nil {
		return SpellCast{}, err
	}
	// O PM DO ELEMENTAL é conferido antes de rolar, pela regra de toda conjuração: a recusa não gasta nada.
	if element != "" && dto.ManaAvailable() < sheet.ArcaneBoltElementPM {
		return SpellCast{}, fmt.Errorf("faltam PM: o Raio Elemental custa %d e restam %d", sheet.ArcaneBoltElementPM, dto.ManaAvailable())
	}
	cast := SpellCast{
		Name: "Raio Arcano", Save: "Reflexos reduz à metade", DC: bolt.DC, Range: arcaneBoltRangeWritten(bolt.Range),
		Kind: live.RollOfTheCast,
	}
	if cast.Parts, err = p.rollTheParts(cast.Name, parts, fromTheTable); err != nil {
		return SpellCast{}, err
	}
	cast.Bonuses = sheet.SpellRollPowersOf(dto, computed).CastBonuses(parts)
	if element != "" {
		if err := sheet.SpendLoadedMana(ctx, p.queries, &dto, sheet.ArcaneBoltElementPM); err != nil {
			return SpellCast{}, err
		}
	}
	return cast, nil
}

// arcaneBoltRangeWritten é o alcance do raio como a faixa o escreve.
func arcaneBoltRangeWritten(reach string) string {
	if reach == "medio" {
		return "médio"
	}
	return reach
}
