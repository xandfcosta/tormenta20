package engine

import (
	"encoding/json"
	"fmt"
)

// FrozenStrike é o golpe como ele foi proposto: a arma, o alvo e a Tabela 5-3 daquele
// instante (decisão do dono). A decisão do mestre, o rolar de novo e o dano leem daqui,
// e não da ficha: ligar uma postura depois de propor não muda o que foi proposto, e
// guardar a arma não troca o golpe por um soco.
//
// O regime guarda isto sem abrir, porque ele não conhece o motor.
type FrozenStrike struct {
	Weapon     WeaponCard
	Target     AttackTarget
	Situations []SpecialSituation
	// ShootsIntoMelee fica fora do JSON da carta, e a Tabela 5-3 o lê. Os outros campos
	// fora do fio não entram na conta do golpe.
	ShootsIntoMelee bool
}

// FreezeStrike escreve o golpe para viajar na proposta.
//
//	frozen, err := engine.FreezeStrike(card, target, situations)
func FreezeStrike(card WeaponCard, target AttackTarget, situations []SpecialSituation) (json.RawMessage, error) {
	frozen, err := json.Marshal(FrozenStrike{
		Weapon: card, Target: target, Situations: situations, ShootsIntoMelee: card.ShootsIntoMelee,
	})
	if err != nil {
		return nil, fmt.Errorf("congelar o golpe com %s: %w", card.Name, err)
	}
	return frozen, nil
}

// ThawStrike lê de volta o que o `FreezeStrike` escreveu.
func ThawStrike(raw json.RawMessage) (FrozenStrike, error) {
	var frozen FrozenStrike
	if err := json.Unmarshal(raw, &frozen); err != nil {
		return FrozenStrike{}, fmt.Errorf("ler o golpe proposto: %w", err)
	}
	frozen.Weapon.ShootsIntoMelee = frozen.ShootsIntoMelee
	return frozen, nil
}
