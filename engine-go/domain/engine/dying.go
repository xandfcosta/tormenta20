package engine

// CAIR, SANGRAR, ESTABILIZAR E MORRER (T20 p236).
//
//	"Se ficar com 0 PV ou menos, você cai inconsciente e fica sangrando. No
//	início de seu turno, faça um teste de Constituição (CD 15). Se passar, você
//	estabiliza e não precisa mais fazer esse teste (exceto se perder mais PV).
//	Se falhar, você perde 1d6 pontos de vida. [...] Um personagem sangrando pode
//	ser estabilizado com um teste de Cura (CD 15) ou com qualquer efeito que cure
//	pelo menos 1 PV. Um personagem com 0 ou menos pontos de vida que recupere PV
//	até um valor positivo (1 ou mais) [...] recobra a consciência."
//
// ESTÁVEL NÃO É UMA TERCEIRA CONDIÇÃO, e é por isso que não há coluna nova: é
// estar a 0 PV ou menos SEM a condição Sangrando. As duas condições do livro já
// estão no catálogo (p395), e a ficha já as guarda.

// As condições do livro que esta regra liga e desliga.
const (
	ConditionUnconscious = "inconsciente"
	ConditionBleeding    = "sangrando"
)

// BleedingCheckDC é a CD do teste de Constituição de quem sangra.
const BleedingCheckDC = 15

// DeathThreshold é o PV em que se morre: –10 ou menos a metade dos PV totais,
// o que for mais baixo (p236).
//
// Sem total conhecido — um NPC digitado na fila sem PV máximo — o único número
// que o livro dá é o –10. A metade arredonda para baixo, como toda conta do
// livro, e a divisão inteira de positivo já faz isso.
//
// @example DeathThreshold(30) // -15
func DeathThreshold(hpMax int64) int64 {
	return min(-10, -(max(hpMax, 0) / 2))
}

// VitalState é onde o personagem está entre de pé e morto.
type VitalState string

const (
	VitalConscious VitalState = "conscious"
	VitalDying     VitalState = "dying"
	VitalStable    VitalState = "stable"
	VitalDead      VitalState = "dead"
)

// VitalStateOf lê o estado pelo PV e pela condição Sangrando.
func VitalStateOf(hp, hpMax int64, bleeding bool) VitalState {
	switch {
	case hp > 0:
		return VitalConscious
	case hp <= DeathThreshold(hpMax):
		return VitalDead
	case bleeding:
		return VitalDying
	}
	return VitalStable
}

// DyingConditionChange diz que condições uma mudança de PV liga e desliga.
//
// Ela olha só o PV de antes e o de depois, e não o que a ficha já tem: ligar o
// que já está ligado e desligar o que não está são operações de conjunto, e
// quem aplica a troca não precisa perguntar antes.
//
// @example DyingConditionChange(5, 0, 12) // +[inconsciente sangrando]
func DyingConditionChange(before, after, hpMax int64) (add, drop []string) {
	switch {
	case after == before:
		return nil, nil
	case after > 0:
		if before > 0 {
			return nil, nil
		}
		// "recupere PV até um valor positivo […] recobra a consciência"
		return nil, []string{ConditionUnconscious, ConditionBleeding}
	case after <= DeathThreshold(hpMax):
		// quem morreu não rola mais nada
		return nil, []string{ConditionBleeding}
	case before > 0:
		// "Se ficar com 0 PV ou menos, você cai inconsciente e fica sangrando."
		return []string{ConditionUnconscious, ConditionBleeding}, nil
	case after < before:
		// "…não precisa mais fazer esse teste (exceto se perder mais PV)."
		return []string{ConditionBleeding}, nil
	}
	// "…qualquer efeito que cure pelo menos 1 PV" estabiliza.
	return nil, []string{ConditionBleeding}
}

// BleedingCheckPasses diz se o teste de Constituição estabiliza: d20 mais a
// Constituição, contra 15. Alcançar a CD é passar, como em todo teste do livro.
func BleedingCheckPasses(d20, constitution int) bool {
	return d20+constitution >= BleedingCheckDC
}

// actionlessConditions tiram a AÇÃO, e o texto do livro é o critério: "não pode
// fazer ações" (atordoado, pasmo, surpreendido, p394-395). A inconsciência tira
// também a REAÇÃO — "sem ações (incluindo reações)", p395 — e mora à parte.
//
// Três ficam de fora por não serem sim-ou-não, e decidi-las não é desta fatia:
// fascinado ("sem ações exceto observar"), paralisado ("só ações mentais") e
// enjoado (uma ação por rodada).
var actionlessConditions = map[string]bool{"atordoado": true, "pasmo": true, "surpreendido": true}

// MomentFor monta o instante de quem vai agir, a partir do PV e das condições.
//
// A 0 PV ninguém reage, com ou sem a condição gravada: uma ficha que caiu antes
// desta regra existir não tem `inconsciente` na lista, e o PV basta para saber.
//
// @example MomentFor(true, 10, []string{"atordoado"}) // age não, reage sim
func MomentFor(onTurn bool, hp int64, conditions []string) ActionMoment {
	moment := ActionMoment{OnTurn: onTurn, CanAct: hp > 0, CanReact: hp > 0}
	for _, c := range conditions {
		if c == ConditionUnconscious {
			moment.CanAct, moment.CanReact = false, false
		}
		if actionlessConditions[c] {
			moment.CanAct = false
		}
	}
	return moment
}
