package engine

// O DANO NÃO LETAL (p236).
//
//	"Dano não letal conta para determinar quando você cai inconsciente, mas não
//	 para determinar quando você começa a sangrar ou morre. Efeitos de cura
//	 recuperam primeiro pontos de vida perdidos por dano não letal."
//
// São TRÊS regras numa frase, e cada uma puxa um lado diferente do mesmo
// número. A forma que dá conta das três é guardar o não letal como uma PARCELA
// do dano, e não como um PV paralelo:
//
//   - para a INCONSCIÊNCIA, o PV é o de sempre — o não letal já está descontado
//     nele, e por isso não há nada a fazer;
//   - para o SANGRAMENTO e a MORTE, o PV que vale é o de sempre MAIS a parcela
//     não letal, que é o PV que o personagem teria se aquele dano não tivesse
//     acontecido;
//   - para a CURA, a parcela é paga primeiro.
//
// Um segundo pool de PV daria a mesma resposta nos três e abriria a pergunta
// "qual deles a ficha mostra?" — e a resposta é UM número, porque é um só que o
// jogador tem.

// lethalHp é o PV que conta para sangrar e para morrer: o atual mais o que foi
// tirado por dano não letal.
//
// É o PV que o personagem teria se aquele dano não tivesse acontecido, e é
// exatamente o que a p236 descreve ao dizer que o não letal "não determina
// quando você começa a sangrar ou morre".
func lethalHp(hp, nonLethal int64) int64 { return hp + max(nonLethal, 0) }

// VitalStateWith é o `VitalStateOf` sabendo quanto do dano foi não letal.
//
// A INCONSCIÊNCIA olha o PV de verdade e a MORTE olha o letal, que é a divisão
// da p236. Quem está a 0 por dano não letal apagou e não está morrendo.
//
// @example VitalStateWith(-18, 12, 30, true) // "stable" — nada daquilo foi letal
func VitalStateWith(hp, hpMax, nonLethal int64, bleeding bool) VitalState {
	switch {
	case hp > 0:
		return VitalConscious
	case lethalHp(hp, nonLethal) <= DeathThreshold(hpMax):
		return VitalDead
	case bleeding:
		return VitalDying
	}
	return VitalStable
}

// DyingConditionChangeWith é o `DyingConditionChange` sabendo do não letal.
//
// A diferença está num lugar só, e ela é a METADE QUE SEPARA esta regra de
// todas as outras: a mesma queda a 0 PV derruba os dois, e um deles está
// morrendo enquanto o outro só apagou.
//
// @example DyingConditionChangeWith(12, 0, 12, 12) // +[inconsciente], sem sangrar
func DyingConditionChangeWith(before, after, hpMax, nonLethal int64) (add, drop []string) {
	add, drop = DyingConditionChange(before, after, hpMax)
	if lethalHp(after, nonLethal) > 0 {
		add, drop = semSangrar(add), comSangrar(drop)
	}
	return add, drop
}

// semSangrar tira o Sangrando da lista de condições a LIGAR.
func semSangrar(add []string) []string {
	out := make([]string, 0, len(add))
	for _, c := range add {
		if c != ConditionBleeding {
			out = append(out, c)
		}
	}
	return out
}

// comSangrar garante o Sangrando na lista de condições a DESLIGAR.
//
// Desligar e não só deixar de ligar: quem já estava sangrando de dano letal e
// foi curado até o PV letal ficar positivo para de sangrar, e é o mesmo
// movimento do "qualquer efeito que cure pelo menos 1 PV" da p236.
func comSangrar(drop []string) []string {
	for _, c := range drop {
		if c == ConditionBleeding {
			return drop
		}
	}
	return append(drop, ConditionBleeding)
}

// HealingSpends reparte a cura entre as duas parcelas, e o NÃO LETAL vem
// primeiro (p236).
//
// Devolve o que SOBRA de cada uma, com piso em zero: cura não cria PV do nada,
// e um não letal negativo viraria PV de brinde na conta do `lethalHp`.
//
// @example HealingSpends(4, 8, 4) // 4 não letal e 4 letal, que é só o não letal pago
func HealingSpends(healed, nonLethal, lethal int64) (restaNonLethal, restaLethal int64) {
	restaNonLethal = max(nonLethal-healed, 0)
	sobra := max(healed-nonLethal, 0)
	return restaNonLethal, max(lethal-sobra, 0)
}
