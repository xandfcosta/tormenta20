package engine

// A INVESTIDA (p235) e a Investida Imprudente do bárbaro (p42), ALE-499.
//
// O +2 no teste é uma linha da `specialSituationTable` — o mecanismo é o da
// Tabela 5-3, deslocar o ataque e dizer à mesa por quê. O que mora aqui é o
// resto: qual investida foi, a penalidade que ela deixa na Defesa e o dado da
// Imprudente. O avanço em linha reta, o dobro do deslocamento e o terreno
// difícil são do mestre.

// As investidas que um golpe pode ser. Vazio é golpe sem investida.
const (
	ChargePlain    = "charge"
	ChargeReckless = "reckless"
)

// ChargeAttackBonus é o "+2 no teste de ataque" da investida (p235).
const ChargeAttackBonus = 2

// RecklessChargePowerID é o poder que libera a Imprudente.
const RecklessChargePowerID = "class.barbaro.investida-imprudente"

// ChargeEffectID é a chave do efeito que a investida deixa na ficha de quem
// investiu: uma por ficha, e investir de novo reescreve a linha.
const ChargeEffectID = "rule.investida"

// RecklessChargeDice é o "+1d12 na rolagem de dano deste ataque" (p42).
const RecklessChargeDice = "1d12"

// ChargeDefensePenalty é o que a investida tira da Defesa "até o seu próximo
// turno": –2 (p235), ou –5 na Imprudente (p42). Zero para o golpe comum.
//
//	ChargeDefensePenalty(ChargeReckless) // -5
func ChargeDefensePenalty(charge string) int {
	switch charge {
	case ChargePlain:
		return -2
	case ChargeReckless:
		return -5
	}
	return 0
}

// ChargeGuardModifiers são os modificadores do efeito da guarda aberta.
func ChargeGuardModifiers(charge string) []Modifier {
	penalty := ChargeDefensePenalty(charge)
	if penalty == 0 {
		return nil
	}
	return []Modifier{{Target: ModifierTarget{K: "defense"}, Amount: penalty, BonusType: "untyped", Note: "investida"}}
}

// ChargeSituation é a linha da Tabela 5-3 que a investida acende, e se o
// pedido é uma investida conhecida.
func ChargeSituation(charge string) (SpecialSituation, bool) {
	switch charge {
	case ChargePlain:
		return AttackerCharging, true
	case ChargeReckless:
		return AttackerChargingRecklessly, true
	}
	return "", false
}

// StrikeAction é o que o golpe custa do turno: a padrão de agredir (p233), ou a
// completa da investida (p235). Uma conta para a proposta e a confirmação.
//
//	StrikeAction(ChargePlain) // ActionFull
func StrikeAction(charge string) ActionCost {
	if charge != "" {
		return ActionFull
	}
	return ActionStandard
}
