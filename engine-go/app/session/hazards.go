package session

import (
	"context"
	"fmt"
	"strconv"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O PERIGO NA MESA (ver GLOSSARY, **perigo**): o mestre dispara a armadilha, ou expõe ao ácido e à lava, da página
// Perigos. A mesa rola o que a ficha do perigo manda — o dano da armadilha e as rodadas que ele dura — e guarda o
// disparo, que só o mestre vê (`live.HazardRoll`). O que dura vira a marca na fila que ele põe em quem está no
// perigo. Quem passou e o que se aplica é dele.

// FireHazard dispara a armadilha (`exposureID` vazio) ou expõe ao perigo do ambiente pela forma pedida
// ("exposicao", "imersao").
//
//	state, err := store.FireHazard(ctx, sessionID, "gas-venenoso", "", engine.RollDie)
func (st *Store) FireHazard(
	ctx context.Context, sessionID int64, hazardID, exposureID string, rollDie func(faces int) (int, error),
) (*live.SessionRuntimeState, error) {
	hazard, found := book.HazardByID(hazardID)
	if !found {
		return nil, fmt.Errorf("o perigo %q não está no catálogo", hazardID)
	}
	var (
		roll live.HazardRoll
		err  error
	)
	switch {
	case hazard.Trap != nil:
		if exposureID != "" {
			return nil, fmt.Errorf("%s é armadilha e não tem exposição (pedida: %q): ela se dispara", hazard.Name, exposureID)
		}
		roll, err = firedTrap(hazard, rollDie)
	case len(hazard.Exposures) > 0:
		roll, err = exposedTo(hazard, exposureID)
	default:
		return nil, fmt.Errorf("%s não se dispara: só a armadilha e o ácido e a lava rolam na mesa", hazard.Name)
	}
	if err != nil {
		return nil, err
	}
	return st.RecordRoll(ctx, sessionID, roll)
}

// firedTrap rola o dano da armadilha, uma parcela por dado da ficha, e as rodadas que ele dura. A rodada do disparo
// é a primeira, e por isso a marca conta as outras: o gás de 6 rodadas rola agora e em mais 5 vezes de quem está
// nele.
func firedTrap(h book.Hazard, rollDie func(faces int) (int, error)) (live.HazardRoll, error) {
	t := h.Trap
	roll := live.HazardRoll{Hazard: h.Name, Kind: h.Kind, BookPage: h.BookPage, Description: h.Description}
	if t.Save != nil {
		roll.Save, roll.DC = t.Save.Test+" "+t.Save.Effect, t.Save.DC
	}
	for _, d := range t.Damage {
		part, err := rolledPart(h.Name, d.Dice, d.Kind, rollDie)
		if err != nil {
			return live.HazardRoll{}, err
		}
		roll.Parts = append(roll.Parts, part)
	}
	if t.Rounds == "" {
		return roll, nil
	}
	rounds, err := rolledPart(h.Name, t.Rounds, "", rollDie)
	if err != nil {
		return live.HazardRoll{}, err
	}
	roll.Rounds, roll.RoundsDice, roll.RoundsRolls = rounds.Total, t.Rounds, rounds.Rolls
	mark := &live.QueueMark{Source: h.Name, At: "inicio", Counted: true, TurnsLeft: rounds.Total - 1}
	for _, d := range t.Damage {
		mark.Parts = append(mark.Parts, live.QueueMarkPart{Dice: d.Dice, Kind: d.Kind, Note: roll.Save})
	}
	roll.Lingering = mark
	return roll, nil
}

// exposedTo é a exposição ao perigo do ambiente: nada se rola agora, e o dano de cada rodada é a marca que o mestre
// põe em quem está nele, até tirá-la.
func exposedTo(h book.Hazard, exposureID string) (live.HazardRoll, error) {
	e, found := h.ExposureByID(exposureID)
	if !found {
		return live.HazardRoll{}, fmt.Errorf("%s se expõe por exposição ou imersão total, e o pedido foi %q", h.Name, exposureID)
	}
	return live.HazardRoll{
		Hazard: h.Name, Kind: h.Kind, BookPage: h.BookPage, Exposure: e.Name, Lingers: e.Lingers,
		Description: h.Description,
		Lingering: &live.QueueMark{
			Source: h.Name + " · " + e.Name, At: "inicio",
			Parts: []live.QueueMarkPart{{Dice: e.Dice, Kind: e.Kind}},
		},
	}, nil
}

// rolledPart rola uma parcela da ficha do perigo. O número sem dado é dano fixo — o "1 ponto de dano de
// perfuração" da Agulha Envenenada (p317) — e entra como total.
func rolledPart(hazard, dice, kind string, rollDie func(faces int) (int, error)) (live.SpellCastPart, error) {
	if fixed, err := strconv.Atoi(dice); err == nil {
		return live.SpellCastPart{Dice: dice, Kind: kind, Total: fixed}, nil
	}
	sum, err := engine.ParseDiceSum(dice)
	if err != nil {
		return live.SpellCastPart{}, fmt.Errorf("o dado de %s: %w", hazard, err)
	}
	rolls, total, err := sum.RollOrTake(nil, rollDie)
	if err != nil {
		return live.SpellCastPart{}, err
	}
	return live.SpellCastPart{Dice: dice, Kind: kind, Rolls: rolls, Total: total}, nil
}
